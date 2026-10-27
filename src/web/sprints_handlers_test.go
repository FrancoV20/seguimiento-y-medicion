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
{{if .Flash}}<div>{{.Flash.Text}}</div>{{end}}
{{if .Data.Notice}}<div>{{.Data.Notice}}</div>{{end}}
{{if .Data.HasActive}}<section>{{.Data.Active.Label}} {{.Data.Active.Goal}}</section>{{end}}
{{range .Data.StoryOptions}}<div data-story-id="{{.ID}}" data-available="{{.Available}}"><input type="checkbox" name="historia" value="{{.ID}}"{{if .Selected}} checked{{end}}{{if not .Available}} disabled{{end}}><span>{{.Title}}</span><span>{{.UnavailableReason}}</span></div>{{end}}
{{if .Form}}<form>CREATE_FORM_MARKER
<span id="goal-value">{{index .Form.Values "objetivo"}}</span><span id="objetivo-error">{{index .Form.Errors "objetivo"}}</span>
<span id="start-value">{{index .Form.Values "fecha_inicio"}}</span><span id="fecha_inicio-error">{{index .Form.Errors "fecha_inicio"}}</span>
<span id="end-value">{{index .Form.Values "fecha_fin"}}</span><span id="fecha_fin-error">{{index .Form.Errors "fecha_fin"}}</span>
<span id="story-value">{{index .Form.Values "historia"}}</span><span id="story-error">{{index .Form.Errors "historia"}}</span></form>{{end}}
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
	createError error
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
	if a.createError != nil {
		return sprint.SprintCreationResult{}, a.createError
	}
	if strings.TrimSpace(request.Goal) == "" {
		return sprint.SprintCreationResult{}, sprint.ErrGoalRequired
	}
	if request.StartDate.IsZero() || request.EndDate.IsZero() || !request.EndDate.After(request.StartDate) {
		return sprint.SprintCreationResult{}, sprint.ErrInvalidSprintPeriod
	}
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
	server   *httptest.Server
	client   *http.Client
	projects map[int64]string
	catalog  *sprintHandlerCatalog
	actions  *sprintHandlerActions
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
	return &sprintHandlerTestApp{server: ts, client: client, projects: projects, catalog: catalog, actions: actions}
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

func (a *sprintHandlerTestApp) setProjectCookie(t *testing.T, value string) {
	t.Helper()
	u, err := url.Parse(a.server.URL)
	if err != nil {
		t.Fatalf("no se pudo analizar la URL de prueba: %v", err)
	}
	a.client.Jar.SetCookies(u, []*http.Cookie{{Name: CookieProject, Value: value, Path: "/"}})
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

func TestSprintsHandler_ListDisablesUnavailableStoriesWithReason(t *testing.T) {
	const projectID = 606
	options := []StoryOption{
		{ID: 61, Title: "Pendiente", Status: sprint.Pendiente, Available: true},
		{ID: 62, Title: "Asignada", Status: sprint.EnSprint, AssignedSprintNumber: 2, UnavailableReason: "Ya está asignada al Sprint 2"},
		{ID: 63, Title: "En progreso", Status: sprint.InProgress, UnavailableReason: "No está pendiente"},
	}
	app := newSprintHandlerTestApp(t, map[int64]string{projectID: "Proyecto estados"},
		map[int64][]StoryOption{projectID: options}, map[int64]*SprintView{})
	app.selectProject(t, projectID)

	_, body := app.get(t)
	if !strings.Contains(body, `data-story-id="61" data-available="true"`) {
		t.Error("la historia Pendiente debía estar disponible")
	}
	for _, expected := range []string{
		`data-story-id="62" data-available="false"`,
		`data-story-id="63" data-available="false"`,
		"Ya está asignada al Sprint 2",
		"No está pendiente",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("la lista debía mostrar el motivo/estado %q: %s", expected, body)
		}
	}
	if !strings.Contains(body, `name="historia" value="62" disabled`) || !strings.Contains(body, `name="historia" value="63" disabled`) {
		t.Error("las historias no disponibles debían tener sus casillas deshabilitadas")
	}
}

func TestSprintsHandler_StoryTakenWhileFillingFormPreservesValues(t *testing.T) {
	const projectID, storyID = 607, 71
	app := newSprintHandlerTestApp(t,
		map[int64]string{projectID: "Proyecto concurrente"},
		map[int64][]StoryOption{projectID: sprintHandlerOptions(projectID, storyID)},
		map[int64]*SprintView{},
	)
	app.selectProject(t, projectID)
	app.actions.createError = fmt.Errorf("%w: historia %d", sprint.ErrStoryNotAvailable, storyID)

	response, body := app.post(t, url.Values{
		FieldGoal:      {"Planificar entrega"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {strconv.FormatInt(storyID, 10)},
	})
	if response.StatusCode != http.StatusConflict {
		t.Errorf("se esperaba 409 para una historia que dejó de estar disponible, se obtuvo %d", response.StatusCode)
	}
	if !strings.Contains(body, `<span id="story-error">la historia no está disponible para asignación: historia 71</span>`) {
		t.Errorf("el motivo exacto debía aparecer en el error de la lista de historias: %s", body)
	}
	for _, expected := range []string{"71", "Planificar entrega", "2026-11-01", "2026-11-14", `name="historia" value="71" checked`} {
		if !strings.Contains(body, expected) {
			t.Errorf("la respuesta debía informar/conservar %q: %s", expected, body)
		}
	}
	if app.catalog.active[projectID] != nil {
		t.Error("no debía crearse Sprint si la historia dejó de estar disponible")
	}
}

func TestSprintsHandler_RejectsEmptyGoalAndInvalidPeriods(t *testing.T) {
	validStart := "2026-11-01"
	cases := []struct {
		name          string
		goal          string
		startDate     string
		endDate       string
		errorID       string
		expectedError string
	}{
		{name: "empty Goal", goal: "", startDate: validStart, endDate: "2026-11-14", errorID: "objetivo-error", expectedError: sprint.ErrGoalRequired.Error()},
		{name: "Goal with whitespace only", goal: " \t ", startDate: validStart, endDate: "2026-11-14", errorID: "objetivo-error", expectedError: sprint.ErrGoalRequired.Error()},
		{name: "end before start", goal: "Sprint válido", startDate: validStart, endDate: "2026-10-31", errorID: "fecha_fin-error", expectedError: sprint.ErrInvalidSprintPeriod.Error()},
		{name: "end equals start", goal: "Sprint válido", startDate: validStart, endDate: validStart, errorID: "fecha_fin-error", expectedError: sprint.ErrInvalidSprintPeriod.Error()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const projectID, storyID = 608, 81
			app := newSprintHandlerTestApp(t,
				map[int64]string{projectID: "Proyecto validación"},
				map[int64][]StoryOption{projectID: sprintHandlerOptions(projectID, storyID)},
				map[int64]*SprintView{},
			)
			app.selectProject(t, projectID)

			response, body := app.post(t, url.Values{
				FieldGoal:      {tc.goal},
				FieldStartDate: {tc.startDate},
				FieldEndDate:   {tc.endDate},
				FieldStory:     {strconv.FormatInt(storyID, 10)},
			})
			if response.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422, se obtuvo %d: %s", response.StatusCode, body)
			}
			if !strings.Contains(body, tc.expectedError) {
				t.Errorf("falta el error de campo %q: %s", tc.expectedError, body)
			}
			if expectedFieldError := fmt.Sprintf(`<span id="%s">%s</span>`, tc.errorID, tc.expectedError); !strings.Contains(body, expectedFieldError) {
				t.Errorf("el mensaje debía estar asociado al campo %q: %s", tc.errorID, body)
			}
			for _, preserved := range []string{tc.goal, tc.startDate, tc.endDate, `name="historia" value="81" checked`} {
				if preserved != "" && !strings.Contains(body, preserved) {
					t.Errorf("no se conservó %q en el formulario: %s", preserved, body)
				}
			}
			if app.actions.createCalls > 1 || app.catalog.active[projectID] != nil {
				t.Errorf("un error de validación no debe persistir un Sprint; calls=%d", app.actions.createCalls)
			}
		})
	}
}

func TestSprintsHandler_RejectsMissingOrMalformedDates(t *testing.T) {
	cases := []struct {
		name          string
		startDate     string
		endDate       string
		errorID       string
		expectedError string
	}{
		{name: "missing start", startDate: "", endDate: "2026-11-14", errorID: "fecha_inicio-error", expectedError: MsgMissingStartDate},
		{name: "missing end", startDate: "2026-11-01", endDate: "", errorID: "fecha_fin-error", expectedError: MsgMissingEndDate},
		{name: "malformed start", startDate: "01/11/2026", endDate: "2026-11-14", errorID: "fecha_inicio-error", expectedError: MsgInvalidDateFormat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const projectID, storyID = 609, 91
			app := newSprintHandlerTestApp(t,
				map[int64]string{projectID: "Proyecto fechas"},
				map[int64][]StoryOption{projectID: sprintHandlerOptions(projectID, storyID)},
				map[int64]*SprintView{},
			)
			app.selectProject(t, projectID)
			values := url.Values{
				FieldGoal:      {"Conservar formulario"},
				FieldStartDate: {tc.startDate},
				FieldEndDate:   {tc.endDate},
				FieldStory:     {strconv.FormatInt(storyID, 10)},
			}

			response, body := app.post(t, values)
			if response.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422, se obtuvo %d: %s", response.StatusCode, body)
			}
			if !strings.Contains(body, tc.expectedError) {
				t.Errorf("falta el mensaje %q: %s", tc.expectedError, body)
			}
			if expectedFieldError := fmt.Sprintf(`<span id="%s">%s</span>`, tc.errorID, tc.expectedError); !strings.Contains(body, expectedFieldError) {
				t.Errorf("el mensaje debía estar asociado al campo %q: %s", tc.errorID, body)
			}
			for _, preserved := range []string{"Conservar formulario", tc.startDate, tc.endDate, `name="historia" value="91" checked`} {
				if preserved != "" && !strings.Contains(body, preserved) {
					t.Errorf("no se conservó %q: %s", preserved, body)
				}
			}
			if app.actions.createCalls != 0 || app.catalog.active[projectID] != nil {
				t.Errorf("no debía invocarse el servicio ni persistir; calls=%d", app.actions.createCalls)
			}
		})
	}
}

func TestSprintsHandler_RejectsNonNumericOrForeignStoryIDs(t *testing.T) {
	cases := []struct {
		name    string
		storyID string
	}{
		{name: "non numeric ID", storyID: "abc"},
		{name: "story from another project", storyID: "122"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const projectAID, projectBID = 612, 613
			app := newSprintHandlerTestApp(t,
				map[int64]string{projectAID: "Proyecto activo", projectBID: "Otro proyecto"},
				map[int64][]StoryOption{
					projectAID: sprintHandlerOptions(projectAID, 111),
					projectBID: sprintHandlerOptions(projectBID, 122),
				},
				map[int64]*SprintView{},
			)
			app.selectProject(t, projectAID)

			response, body := app.post(t, url.Values{
				FieldGoal:      {"Planificar historias del proyecto activo"},
				FieldStartDate: {"2026-11-01"},
				FieldEndDate:   {"2026-11-14"},
				FieldStory:     {tc.storyID},
			})
			if response.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422 para %s, se obtuvo %d", tc.name, response.StatusCode)
			}
			if !strings.Contains(body, `<span id="story-error">`) || !strings.Contains(body, "no es válido") && !strings.Contains(body, sprint.ErrStoryNotAvailable.Error()) {
				t.Errorf("se esperaba un error seguro junto al campo de historias: %s", body)
			}
			if app.actions.createCalls != 0 || app.catalog.active[projectAID] != nil {
				t.Errorf("el ID inválido o ajeno no debe llegar a creación; calls=%d", app.actions.createCalls)
			}
			if strings.Contains(body, "INSERT INTO") || strings.Contains(body, "SELECT ") {
				t.Error("la respuesta no debe exponer detalles SQL")
			}
		})
	}
}

func TestSprintsHandler_StoreFailureShowsGenericErrorAndPreservesForm(t *testing.T) {
	const projectID, storyID = 610, 101
	app := newSprintHandlerTestApp(t,
		map[int64]string{projectID: "Proyecto con falla"},
		map[int64][]StoryOption{projectID: sprintHandlerOptions(projectID, storyID)},
		map[int64]*SprintView{},
	)
	app.selectProject(t, projectID)
	app.actions.createError = fmt.Errorf("postgres: INSERT INTO sprints secret_table")

	response, body := app.post(t, url.Values{
		FieldGoal:      {"Conservar tras fallo"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {strconv.FormatInt(storyID, 10)},
	})
	if response.StatusCode != http.StatusInternalServerError {
		t.Errorf("se esperaba 500, se obtuvo %d: %s", response.StatusCode, body)
	}
	if !strings.Contains(body, MsgOperationFailed) {
		t.Errorf("falta el mensaje genérico %q: %s", MsgOperationFailed, body)
	}
	for _, preserved := range []string{"Conservar tras fallo", "2026-11-01", "2026-11-14", `name="historia" value="101" checked`} {
		if !strings.Contains(body, preserved) {
			t.Errorf("no se conservó %q: %s", preserved, body)
		}
	}
	if strings.Contains(body, "secret_table") || strings.Contains(body, MsgSprintCreated) || strings.Contains(body, "alert-success") {
		t.Error("no se deben exponer detalles SQL ni un mensaje de éxito")
	}
	if app.catalog.active[projectID] != nil {
		t.Error("el fallo de persistencia no debe dejar un Sprint activo")
	}
}

func TestSprintsHandler_RequiresValidProjectContextForPost(t *testing.T) {
	cases := []struct {
		name            string
		selectorMessage string
	}{
		{name: "no project selected", selectorMessage: MsgProjectRequired},
		{name: "deleted project", selectorMessage: MsgProjectGone},
		{name: "invalid project cookie", selectorMessage: MsgProjectRequired},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			const projectID = 611
			projects := map[int64]string{projectID: "Proyecto contextual"}
			app := newSprintHandlerTestApp(t, projects, map[int64][]StoryOption{}, map[int64]*SprintView{})
			switch scenario.name {
			case "deleted project":
				app.selectProject(t, projectID)
				delete(projects, projectID)
			case "invalid project cookie":
				app.setProjectCookie(t, "no-es-numerico")
			}

			response, _ := app.post(t, url.Values{
				FieldGoal:      {"No debe crearse"},
				FieldStartDate: {"2026-11-01"},
				FieldEndDate:   {"2026-11-14"},
				"project_id":   {"999"},
				"sprint_id":    {"777"},
			})
			if response.StatusCode != http.StatusSeeOther || !strings.HasPrefix(response.Header.Get("Location"), pathSelectProject) {
				t.Errorf("sin proyecto válido se esperaba 303 al selector, status=%d location=%q", response.StatusCode, response.Header.Get("Location"))
			}
			selectionResponse, err := app.client.Get(app.server.URL + response.Header.Get("Location"))
			if err != nil {
				t.Fatalf("no se pudo abrir la selección de proyecto: %v", err)
			}
			selectionBody := sprintHandlerReadBody(t, selectionResponse)
			if !strings.Contains(selectionBody, scenario.selectorMessage) {
				t.Errorf("el selector debía mostrar el mensaje %q: %s", scenario.selectorMessage, selectionBody)
			}
			if app.actions.createCalls != 0 || len(app.catalog.active) != 0 {
				t.Errorf("no debía invocarse CreateSprint ni crear Sprints; calls=%d", app.actions.createCalls)
			}
		})
	}
}

var _ ContextStore = sprintHandlerContextStore{}
var _ SprintCatalog = (*sprintHandlerCatalog)(nil)
var _ SprintActions = (*sprintHandlerActions)(nil)
var _ fs.FS = fstest.MapFS{}
