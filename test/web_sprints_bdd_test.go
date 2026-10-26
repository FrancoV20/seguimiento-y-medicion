package test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

type sprintBDDContextStore struct {
	projects map[int64]string
}

func (s sprintBDDContextStore) ListProjects(context.Context) ([]web.ProjectSummary, error) {
	projects := make([]web.ProjectSummary, 0, len(s.projects))
	for id, name := range s.projects {
		projects = append(projects, web.ProjectSummary{ID: id, Name: name})
	}
	return projects, nil
}

func (s sprintBDDContextStore) ListMembers(context.Context, int64) ([]web.MemberSummary, error) {
	return nil, nil
}

func (s sprintBDDContextStore) ResolveContext(_ context.Context, projectID, _ int64) (web.ResolvedContext, error) {
	name, exists := s.projects[projectID]
	return web.ResolvedContext{ProjectExists: exists, ProjectName: name}, nil
}

type sprintBDDCatalog struct {
	active  map[int64]*web.SprintView
	options map[int64][]web.StoryOption
}

func (c *sprintBDDCatalog) ActiveSprint(_ context.Context, projectID int64) (*web.SprintView, error) {
	return c.active[projectID], nil
}

func (c *sprintBDDCatalog) ListStoryOptions(_ context.Context, projectID int64) ([]web.StoryOption, error) {
	return append([]web.StoryOption(nil), c.options[projectID]...), nil
}

type sprintBDDActions struct {
	catalog     *sprintBDDCatalog
	createCalls int
}

func (a *sprintBDDActions) CreateSprint(
	_ context.Context,
	request sprint.SprintCreationRequest,
	project sprint.Project,
	stories []sprint.UserStory,
) (sprint.SprintCreationResult, error) {
	if active := a.catalog.active[project.ID]; active != nil {
		return sprint.SprintCreationResult{}, &sprint.ActiveSprintExistsError{Number: active.Number}
	}
	a.createCalls++
	sprintID := int64(900 + a.createCalls)
	created := sprint.Sprint{
		ID:        sprintID,
		ProjectID: project.ID,
		Number:    1,
		DisplayID: "Sprint 1",
		Goal:      request.Goal,
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Status:    sprint.SprintActive,
		StoryIDs:  append([]int64(nil), request.StoryIDs...),
	}
	view := &web.SprintView{
		ID:        created.ID,
		Number:    created.Number,
		Label:     created.DisplayID,
		Goal:      created.Goal,
		StartDate: created.StartDate,
		EndDate:   created.EndDate,
		Stories:   make([]web.SprintStoryRow, 0, len(request.StoryIDs)),
	}
	selected := make(map[int64]struct{}, len(request.StoryIDs))
	for _, id := range request.StoryIDs {
		selected[id] = struct{}{}
	}
	for _, story := range stories {
		if _, chosen := selected[story.ID]; chosen {
			view.Stories = append(view.Stories, web.SprintStoryRow{ID: story.ID, Title: fmt.Sprintf("Historia %d", story.ID), Status: sprint.EnSprint})
		}
	}
	a.catalog.active[project.ID] = view
	result := sprint.SprintCreationResult{Sprint: created}
	if len(created.StoryIDs) == 0 {
		result.Warning = "El Sprint se creó sin historias asignadas."
	}
	return result, nil
}

func (*sprintBDDActions) AssignStories(context.Context, int64, []int64) error { return nil }
func (*sprintBDDActions) RemoveStory(context.Context, int64, int64) error     { return nil }

type sprintBDDApp struct {
	server  *httptest.Server
	client  *http.Client
	catalog *sprintBDDCatalog
	actions *sprintBDDActions
}

func newSprintBDDApp(t *testing.T, projectID int64, name string, active *web.SprintView, options []web.StoryOption) *sprintBDDApp {
	t.Helper()
	server := web.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	web.RegisterContext(server, sprintBDDContextStore{projects: map[int64]string{projectID: name}})
	catalog := &sprintBDDCatalog{
		active:  map[int64]*web.SprintView{projectID: active},
		options: map[int64][]web.StoryOption{projectID: options},
	}
	actions := &sprintBDDActions{catalog: catalog}
	if err := web.RegisterSprints(server, catalog, actions); err != nil {
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
	return &sprintBDDApp{server: ts, client: client, catalog: catalog, actions: actions}
}

func (a *sprintBDDApp) chooseProject(t *testing.T, projectID int64) {
	t.Helper()
	response, err := a.client.PostForm(a.server.URL+"/contexto/proyecto", url.Values{
		"proyecto_id": {fmt.Sprintf("%d", projectID)},
		"volver_a":    {"/sprints"},
	})
	if err != nil {
		t.Fatalf("no se pudo elegir proyecto: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("se esperaba 303 al elegir proyecto, se obtuvo %d", response.StatusCode)
	}
}

func (a *sprintBDDApp) postCreate(t *testing.T, values url.Values) (*http.Response, string) {
	t.Helper()
	response, err := a.client.PostForm(a.server.URL+"/sprints", values)
	if err != nil {
		t.Fatalf("POST /sprints: %v", err)
	}
	return response, sprintBDDReadBody(t, response)
}

func (a *sprintBDDApp) getSprints(t *testing.T) (*http.Response, string) {
	t.Helper()
	response, err := a.client.Get(a.server.URL + "/sprints")
	if err != nil {
		t.Fatalf("GET /sprints: %v", err)
	}
	return response, sprintBDDReadBody(t, response)
}

func sprintBDDReadBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return string(body)
}

func TestWebSprintsBDD_CreateWithSelectedStories(t *testing.T) {
	// Given: elijo un proyecto con historias disponibles y sin Sprint activo.
	const projectID, storyID = 601, 61
	app := newSprintBDDApp(t, projectID, "Proyecto BDD", nil, []web.StoryOption{{
		ID: storyID, Title: "Historia BDD", Status: sprint.Pendiente, Available: true,
	}})
	app.chooseProject(t, projectID)

	// When: completo el Goal, las fechas y marco una historia.
	response, _ := app.postCreate(t, url.Values{
		web.FieldGoal:      {"Entregar capacidad de Sprint"},
		web.FieldStartDate: {"2026-11-01"},
		web.FieldEndDate:   {"2026-11-14"},
		web.FieldStory:     {"61"},
	})
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/sprints" {
		t.Fatalf("se esperaba redirección 303 a /sprints; estado=%d location=%q", response.StatusCode, response.Header.Get("Location"))
	}

	// Then: veo la confirmación y el Sprint activo con la historia asignada.
	_, body := app.getSprints(t)
	for _, expected := range []string{web.MsgSprintCreated, "Sprint 1", "Entregar capacidad de Sprint", "Historia 61", "En Sprint"} {
		if !strings.Contains(body, expected) {
			t.Errorf("la pantalla debía contener %q: %s", expected, body)
		}
	}
	if app.actions.createCalls != 1 || app.catalog.active[projectID] == nil {
		t.Errorf("se esperaba exactamente un Sprint creado; llamadas=%d", app.actions.createCalls)
	}
}

func TestWebSprintsBDD_CreateWithoutStoriesShowsWarning(t *testing.T) {
	// Given: elijo un proyecto sin Sprint activo y sin historias marcadas.
	const projectID = 602
	app := newSprintBDDApp(t, projectID, "Proyecto vacío", nil, nil)
	app.chooseProject(t, projectID)

	// When: envío un Goal y un período válido sin marcar historias.
	response, _ := app.postCreate(t, url.Values{
		web.FieldGoal:      {"Preparar backlog"},
		web.FieldStartDate: {"2026-11-01"},
		web.FieldEndDate:   {"2026-11-14"},
	})
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("se esperaba redirección 303, se obtuvo %d", response.StatusCode)
	}

	// Then: el Sprint se crea vacío y muestra la advertencia específica.
	_, body := app.getSprints(t)
	if !strings.Contains(body, web.MsgSprintWithoutStories) {
		t.Errorf("falta la advertencia de Sprint vacío %q: %s", web.MsgSprintWithoutStories, body)
	}
	if app.catalog.active[projectID] == nil || len(app.catalog.active[projectID].Stories) != 0 {
		t.Errorf("se esperaba un Sprint activo sin historias: %+v", app.catalog.active[projectID])
	}
}

func TestWebSprintsBDD_ExistingActiveSprintBlocksAnother(t *testing.T) {
	// Given: el proyecto ya tiene el Sprint activo Sprint 4.
	const projectID = 603
	active := &web.SprintView{ID: 64, Number: 4, Label: "Sprint 4", Goal: "Sprint vigente"}
	app := newSprintBDDApp(t, projectID, "Proyecto con Sprint", active, nil)
	app.chooseProject(t, projectID)

	// When: abro Sprints y luego envío directamente otro formulario de creación.
	_, page := app.getSprints(t)
	response, body := app.postCreate(t, url.Values{
		web.FieldGoal:      {"Sprint duplicado"},
		web.FieldStartDate: {"2026-11-01"},
		web.FieldEndDate:   {"2026-11-14"},
	})

	// Then: se informa el Sprint existente, no se ofrece crear otro ni se persiste uno nuevo.
	if strings.Contains(page, "CREATE_FORM_MARKER") {
		t.Error("no debe ofrecerse el formulario con un Sprint activo")
	}
	if !strings.Contains(page, "Sprint 4") || !strings.Contains(page, "Ya hay un Sprint activo (Sprint 4)") {
		t.Errorf("debía mostrarse Sprint 4 y su aviso: %s", page)
	}
	if response.StatusCode != http.StatusConflict || !strings.Contains(body, "Ya hay un Sprint activo (Sprint 4)") {
		t.Errorf("POST debía responder 409 con el conflicto del Sprint 4; status=%d body=%s", response.StatusCode, body)
	}
	if app.actions.createCalls != 0 {
		t.Errorf("no debía guardarse otro Sprint; llamadas=%d", app.actions.createCalls)
	}
}
