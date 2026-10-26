package web

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

const sprintsHandlerTestTemplate = `{{define "content"}}<h1>{{.Title}}</h1>
{{if .Form}}<form>CREATE_FORM_MARKER</form>{{end}}
{{if .Flash}}<div>{{.Flash.Text}}</div>{{end}}
<pre>{{printf "%+v" .Data}}</pre>{{end}}`

type sprintHandlerContextStore struct {
	projects map[int64]string
}

func (s sprintHandlerContextStore) ListProjects(context.Context) ([]ProjectSummary, error) {
	projects := make([]ProjectSummary, 0, len(s.projects))
	for id, name := range s.projects {
		projects = append(projects, ProjectSummary{ID: id, Name: name})
	}
	return projects, nil
}

func (s sprintHandlerContextStore) ListMembers(context.Context, int64) ([]MemberSummary, error) {
	return nil, nil
}

func (s sprintHandlerContextStore) ResolveContext(_ context.Context, projectID, _ int64) (ResolvedContext, error) {
	name, exists := s.projects[projectID]
	return ResolvedContext{ProjectExists: exists, ProjectName: name}, nil
}

type sprintHandlerCatalog struct {
	active  map[int64]*SprintView
	options map[int64][]StoryOption
}

func (c *sprintHandlerCatalog) ActiveSprint(_ context.Context, projectID int64) (*SprintView, error) {
	return c.active[projectID], nil
}

func (c *sprintHandlerCatalog) ListStoryOptions(_ context.Context, projectID int64) ([]StoryOption, error) {
	return append([]StoryOption(nil), c.options[projectID]...), nil
}

type sprintHandlerActions struct {
	catalog     *sprintHandlerCatalog
	createCalls int
	lastRequest sprint.SprintCreationRequest
	lastProject sprint.Project
	lastStories []sprint.UserStory
	nextID      int64
}

func (a *sprintHandlerActions) CreateSprint(
	_ context.Context,
	request sprint.SprintCreationRequest,
	project sprint.Project,
	stories []sprint.UserStory,
) (sprint.SprintCreationResult, error) {
	if active := a.catalog.active[project.ID]; active != nil {
		return sprint.SprintCreationResult{}, &sprint.ActiveSprintExistsError{Number: active.Number}
	}
	a.createCalls++
	a.lastRequest = request
	a.lastProject = project
	a.lastStories = append([]sprint.UserStory(nil), stories...)
	if a.nextID == 0 {
		a.nextID = 1
	}

	created := sprint.Sprint{
		ID:        a.nextID,
		ProjectID: project.ID,
		Number:    1,
		DisplayID: "Sprint 1",
		Goal:      request.Goal,
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Status:    sprint.SprintActive,
		StoryIDs:  append([]int64(nil), request.StoryIDs...),
	}
	view := &SprintView{
		ID:        created.ID,
		Number:    created.Number,
		Label:     created.DisplayID,
		Goal:      created.Goal,
		StartDate: created.StartDate,
		EndDate:   created.EndDate,
		Stories:   make([]SprintStoryRow, 0, len(stories)),
	}
	selected := make(map[int64]struct{}, len(request.StoryIDs))
	for _, id := range request.StoryIDs {
		selected[id] = struct{}{}
	}
	for _, story := range stories {
		if _, chosen := selected[story.ID]; !chosen {
			continue
		}
		story.Status = sprint.EnSprint
		story.ActiveSprintID = created.ID
		view.Stories = append(view.Stories, SprintStoryRow{ID: story.ID, Status: sprint.EnSprint})
		for i := range a.catalog.options[project.ID] {
			if a.catalog.options[project.ID][i].ID == story.ID {
				a.catalog.options[project.ID][i].Status = sprint.EnSprint
				a.catalog.options[project.ID][i].Available = false
				a.catalog.options[project.ID][i].AssignedSprintNumber = created.Number
				a.catalog.options[project.ID][i].UnavailableReason = "Ya está asignada al Sprint 1"
			}
		}
	}
	a.catalog.active[project.ID] = view
	a.nextID++
	result := sprint.SprintCreationResult{Sprint: created}
	if len(created.StoryIDs) == 0 {
		result.Warning = "El Sprint se creó sin historias asignadas."
	}
	return result, nil
}

func (a *sprintHandlerActions) AssignStories(context.Context, int64, []int64) error { return nil }
func (a *sprintHandlerActions) RemoveStory(context.Context, int64, int64) error     { return nil }

type sprintHandlerTestApp struct {
	server  *httptest.Server
	client  *http.Client
	catalog *sprintHandlerCatalog
	actions *sprintHandlerActions
}

func newSprintHandlerTestApp(
	t *testing.T,
	projects map[int64]string,
	options map[int64][]StoryOption,
	active map[int64]*SprintView,
) *sprintHandlerTestApp {
	t.Helper()
	server := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	files := formsTemplates(t).(fstest.MapFS)
	files["templates/sprints.html"] = &fstest.MapFile{Data: []byte(sprintsHandlerTestTemplate)}
	server.renderer.files = files
	RegisterContext(server, sprintHandlerContextStore{projects: projects})
	catalog := &sprintHandlerCatalog{active: active, options: options}
	actions := &sprintHandlerActions{catalog: catalog}
	if err := RegisterSprints(server, catalog, actions); err != nil {
		t.Fatalf("no se pudo registrar la sección Sprints: %v", err)
	}
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("no se pudo crear el cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &sprintHandlerTestApp{server: ts, client: client, catalog: catalog, actions: actions}
}

func (a *sprintHandlerTestApp) selectProject(t *testing.T, projectID int64) {
	t.Helper()
	response, err := a.client.PostForm(a.server.URL+pathSelectProject, url.Values{
		"proyecto_id": {strconv.FormatInt(projectID, 10)},
		"volver_a":    {"/sprints"},
	})
	if err != nil {
		t.Fatalf("POST selección de proyecto: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("seleccionar proyecto: se esperaba 303, se obtuvo %d", response.StatusCode)
	}
}

func (a *sprintHandlerTestApp) get(t *testing.T) (*http.Response, string) {
	t.Helper()
	response, err := a.client.Get(a.server.URL + "/sprints")
	if err != nil {
		t.Fatalf("GET /sprints: %v", err)
	}
	return response, sprintHandlerReadBody(t, response)
}

func (a *sprintHandlerTestApp) post(t *testing.T, values url.Values) (*http.Response, string) {
	t.Helper()
	response, err := a.client.PostForm(a.server.URL+"/sprints", values)
	if err != nil {
		t.Fatalf("POST /sprints: %v", err)
	}
	return response, sprintHandlerReadBody(t, response)
}

func sprintHandlerReadBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return string(body)
}

func sprintHandlerOptions(projectID int64, ids ...int64) []StoryOption {
	options := make([]StoryOption, 0, len(ids))
	for _, id := range ids {
		options = append(options, StoryOption{
			ID:        id,
			Title:     fmt.Sprintf("Historia %d del proyecto %d", id, projectID),
			Priority:  backlog.PriorityMedium,
			Status:    sprint.Pendiente,
			Available: true,
		})
	}
	return options
}

func TestSprintsHandler_CreateSprintSuccess_UsesSelectedProject(t *testing.T) {
	const projectAID, projectBID = 101, 202
	app := newSprintHandlerTestApp(t,
		map[int64]string{projectAID: "Proyecto A", projectBID: "Proyecto B"},
		map[int64][]StoryOption{
			projectAID: sprintHandlerOptions(projectAID, 11),
			projectBID: sprintHandlerOptions(projectBID, 22),
		},
		map[int64]*SprintView{},
	)
	app.selectProject(t, projectBID)

	response, _ := app.post(t, url.Values{
		FieldGoal:      {"Preparar entrega"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {"22"},
		"project_id":   {strconv.FormatInt(projectAID, 10)},
		"sprint_id":    {"999"},
	})
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/sprints" {
		t.Fatalf("se esperaba 303 a /sprints; estado=%d location=%q", response.StatusCode, response.Header.Get("Location"))
	}
	if app.actions.createCalls != 1 {
		t.Fatalf("CreateSprint se llamó %d veces, se esperaba 1", app.actions.createCalls)
	}
	if app.actions.lastProject.ID != projectBID || app.actions.lastRequest.ProjectID != projectBID {
		t.Errorf("el proyecto de la acción debe venir del contexto elegido (%d), se obtuvo project=%d request=%d",
			projectBID, app.actions.lastProject.ID, app.actions.lastRequest.ProjectID)
	}
	if !reflect.DeepEqual(app.actions.lastRequest.StoryIDs, []int64{22}) {
		t.Errorf("StoryIDs = %v, se esperaba solo la historia del proyecto activo", app.actions.lastRequest.StoryIDs)
	}
	if len(app.actions.lastStories) != 1 || app.actions.lastStories[0].ProjectID != projectBID {
		t.Errorf("se enviaron historias ajenas al proyecto activo: %+v", app.actions.lastStories)
	}

	_, body := app.get(t)
	for _, expected := range []string{"Sprint 1", "Preparar entrega", "Historia 22 del proyecto 202", MsgSprintCreated} {
		if !strings.Contains(body, expected) {
			t.Errorf("la pantalla posterior a crear no contiene %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "Historia 11 del proyecto 101") {
		t.Error("no deben mostrarse ni asignarse historias del proyecto no activo")
	}
}

func TestSprintsHandler_CreateSprintWithoutStoriesShowsWarning(t *testing.T) {
	const projectID = 303
	app := newSprintHandlerTestApp(t,
		map[int64]string{projectID: "Proyecto sin historias"},
		map[int64][]StoryOption{projectID: {}},
		map[int64]*SprintView{},
	)
	app.selectProject(t, projectID)

	response, _ := app.post(t, url.Values{
		FieldGoal:      {"Preparar entrega"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
	})
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/sprints" {
		t.Fatalf("se esperaba 303 a /sprints, se obtuvo %d a %q", response.StatusCode, response.Header.Get("Location"))
	}
	_, body := app.get(t)
	if !strings.Contains(body, MsgSprintWithoutStories) {
		t.Errorf("falta la advertencia %q: %s", MsgSprintWithoutStories, body)
	}
	if len(app.catalog.active[projectID].Stories) != 0 {
		t.Errorf("el Sprint debía crearse sin historias: %+v", app.catalog.active[projectID].Stories)
	}
}

func TestSprintsHandler_ActiveSprintBlocksFormAndPost(t *testing.T) {
	const projectID = 404
	active := &SprintView{ID: 44, Number: 3, Label: "Sprint 3", Goal: "Objetivo activo"}
	app := newSprintHandlerTestApp(t,
		map[int64]string{projectID: "Proyecto activo"},
		map[int64][]StoryOption{projectID: sprintHandlerOptions(projectID, 41)},
		map[int64]*SprintView{projectID: active},
	)
	app.selectProject(t, projectID)

	_, body := app.get(t)
	if strings.Contains(body, "CREATE_FORM_MARKER") {
		t.Error("no debe ofrecerse el formulario cuando ya existe un Sprint activo")
	}
	if !strings.Contains(body, "Sprint 3") || !strings.Contains(body, "Ya hay un Sprint activo (Sprint 3)") {
		t.Errorf("la pantalla debe mostrar el Sprint activo y su aviso: %s", body)
	}

	response, body := app.post(t, url.Values{
		FieldGoal:      {"No debe crear"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {"41"},
	})
	if response.StatusCode != http.StatusConflict {
		t.Errorf("POST con Sprint activo: se esperaba 409, se obtuvo %d", response.StatusCode)
	}
	if !strings.Contains(body, "Ya hay un Sprint activo (Sprint 3)") {
		t.Errorf("falta el conflicto con el número del Sprint activo: %s", body)
	}
	if app.actions.createCalls != 0 {
		t.Errorf("no debía crearse otro Sprint; CreateSprint se llamó %d veces", app.actions.createCalls)
	}
}

func TestSprintsHandler_PostRedirectGetDoesNotCreateDuplicateOrRepeatFlash(t *testing.T) {
	const projectID = 505
	app := newSprintHandlerTestApp(t,
		map[int64]string{projectID: "Proyecto PRG"},
		map[int64][]StoryOption{projectID: sprintHandlerOptions(projectID, 51)},
		map[int64]*SprintView{},
	)
	app.selectProject(t, projectID)

	response, _ := app.post(t, url.Values{
		FieldGoal:      {"Preparar entrega"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {"51"},
	})
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("se esperaba una redirección 303, se obtuvo %d", response.StatusCode)
	}
	_, firstBody := app.get(t)
	if !strings.Contains(firstBody, MsgSprintCreated) {
		t.Errorf("la primera vista debía mostrar el mensaje de éxito: %s", firstBody)
	}
	_, reloadBody := app.get(t)
	if strings.Contains(reloadBody, MsgSprintCreated) {
		t.Error("el mensaje de éxito no debe repetirse al recargar")
	}
	if app.actions.createCalls != 1 {
		t.Errorf("recargar no debe crear otro Sprint; llamadas=%d", app.actions.createCalls)
	}
}

var _ ContextStore = sprintHandlerContextStore{}
var _ SprintCatalog = (*sprintHandlerCatalog)(nil)
var _ SprintActions = (*sprintHandlerActions)(nil)
var _ fs.FS = fstest.MapFS{}
