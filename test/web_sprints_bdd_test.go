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
	"os"
	"strings"
	"testing"
	"time"

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

// MemberCredential completa el ContextStore ampliado por HU-27 (T016).
func (s sprintBDDContextStore) MemberCredential(context.Context, int64, int64) (web.MemberCredential, error) {
	return web.MemberCredential{}, nil
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
	createError error
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
	if a.createError != nil {
		return sprint.SprintCreationResult{}, a.createError
	}
	if strings.TrimSpace(request.Goal) == "" {
		return sprint.SprintCreationResult{}, sprint.ErrGoalRequired
	}
	if request.StartDate.IsZero() || request.EndDate.IsZero() || !request.EndDate.After(request.StartDate) {
		return sprint.SprintCreationResult{}, sprint.ErrInvalidSprintPeriod
	}
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

func TestWebSprintsBDD_UnavailableStoryAndValidationErrors(t *testing.T) {
	t.Run("historia no disponible en la lista y al guardar", func(t *testing.T) {
		const projectID, storyID = 701, 71
		unavailable := web.StoryOption{
			ID: storyID, Title: "Historia ya asignada", Status: sprint.EnSprint,
			AssignedSprintNumber: 2, Available: false, UnavailableReason: "Ya está asignada al Sprint 2",
		}
		app := newSprintBDDApp(t, projectID, "Proyecto con historia asignada", nil, []web.StoryOption{unavailable})
		app.chooseProject(t, projectID)

		// Given: la historia ya aparece deshabilitada con el motivo del Sprint que la contiene.
		_, list := app.getSprints(t)
		if !strings.Contains(list, "Ya está asignada al Sprint 2") || !strings.Contains(list, "disabled") {
			t.Errorf("la lista debía deshabilitar la historia con su motivo: %s", list)
		}

		// When: la historia se vuelve no disponible después de abrir el formulario.
		app.actions.createError = fmt.Errorf("%w: historia %d", sprint.ErrStoryNotAvailable, storyID)
		response, body := app.postCreate(t, url.Values{
			web.FieldGoal:      {"Planificar historias disponibles"},
			web.FieldStartDate: {"2026-11-01"},
			web.FieldEndDate:   {"2026-11-14"},
			web.FieldStory:     {"71"},
		})

		// Then: no se crea, se identifica la historia y se conservan campos y selección.
		if response.StatusCode != http.StatusConflict {
			t.Errorf("se esperaba 409, se obtuvo %d", response.StatusCode)
		}
		storyError := `<p class="field-error"><span class="field-error-label">Error:</span> la historia no está disponible para asignación: historia 71</p>`
		if !strings.Contains(body, storyError) {
			t.Errorf("el motivo exacto debía mostrarse junto al campo de historias: %s", body)
		}
		for _, expected := range []string{"Planificar historias disponibles", "2026-11-01", "2026-11-14", `name="historia" value="71" checked`} {
			if !strings.Contains(body, expected) {
				t.Errorf("la respuesta debía mostrar/conservar %q: %s", expected, body)
			}
		}
		if app.catalog.active[projectID] != nil {
			t.Error("no debía quedar un Sprint creado tras rechazar la historia")
		}
	})

	validationCases := []struct {
		name          string
		goal          string
		startDate     string
		endDate       string
		errorID       string
		expectedError string
	}{
		{name: "Goal vacío", goal: "", startDate: "2026-11-01", endDate: "2026-11-14", errorID: "objetivo-error", expectedError: sprint.ErrGoalRequired.Error()},
		{name: "Goal con espacios", goal: " \t ", startDate: "2026-11-01", endDate: "2026-11-14", errorID: "objetivo-error", expectedError: sprint.ErrGoalRequired.Error()},
		{name: "fecha de fin anterior", goal: "Sprint válido", startDate: "2026-11-01", endDate: "2026-10-31", errorID: "fecha_fin-error", expectedError: sprint.ErrInvalidSprintPeriod.Error()},
		{name: "fecha de fin igual", goal: "Sprint válido", startDate: "2026-11-01", endDate: "2026-11-01", errorID: "fecha_fin-error", expectedError: sprint.ErrInvalidSprintPeriod.Error()},
		{name: "falta fecha de inicio", goal: "Sprint válido", startDate: "", endDate: "2026-11-14", errorID: "fecha_inicio-error", expectedError: web.MsgMissingStartDate},
		{name: "falta fecha de fin", goal: "Sprint válido", startDate: "2026-11-01", endDate: "", errorID: "fecha_fin-error", expectedError: web.MsgMissingEndDate},
		{name: "formato de fecha inválido", goal: "Sprint válido", startDate: "01/11/2026", endDate: "2026-11-14", errorID: "fecha_inicio-error", expectedError: web.MsgInvalidDateFormat},
	}
	for _, tc := range validationCases {
		t.Run(tc.name, func(t *testing.T) {
			const projectID, storyID = 702, 72
			app := newSprintBDDApp(t, projectID, "Proyecto de validación", nil, []web.StoryOption{{
				ID: storyID, Title: "Historia a conservar", Status: sprint.Pendiente, Available: true,
			}})
			app.chooseProject(t, projectID)

			// Given: un formulario con el dato inválido y los demás datos ingresados.
			response, body := app.postCreate(t, url.Values{
				web.FieldGoal:      {tc.goal},
				web.FieldStartDate: {tc.startDate},
				web.FieldEndDate:   {tc.endDate},
				web.FieldStory:     {"72"},
			})

			// When: el servidor procesa la solicitud.
			if response.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422, se obtuvo %d: %s", response.StatusCode, body)
			}

			// Then: informa el error, conserva los valores y no presenta éxito.
			if !strings.Contains(body, tc.expectedError) {
				t.Errorf("falta el mensaje %q: %s", tc.expectedError, body)
			}
			expectedFieldError := `<p class="field-error" id="` + tc.errorID + `"><span class="field-error-label">Error:</span> ` + tc.expectedError + `</p>`
			if !strings.Contains(body, expectedFieldError) {
				t.Errorf("el mensaje debía renderizarse exactamente junto al campo %q: %s", tc.errorID, body)
			}
			for _, value := range []string{tc.goal, tc.startDate, tc.endDate} {
				if value != "" && !strings.Contains(body, value) {
					t.Errorf("no se conservó el valor %q: %s", value, body)
				}
			}
			if !strings.Contains(body, `name="historia" value="72" checked`) {
				t.Errorf("la casilla marcada debía conservarse tras el rechazo: %s", body)
			}
			if strings.Contains(body, web.MsgSprintCreated) || strings.Contains(body, "alert-success") || app.catalog.active[projectID] != nil {
				t.Error("un error de validación no debe producir éxito ni crear un Sprint")
			}
		})
	}

	t.Run("falla el guardado", func(t *testing.T) {
		const projectID = 703
		app := newSprintBDDApp(t, projectID, "Proyecto de persistencia", nil, []web.StoryOption{{
			ID: 73, Title: "Historia de persistencia", Status: sprint.Pendiente, Available: true,
		}})
		app.chooseProject(t, projectID)
		app.actions.createError = fmt.Errorf("postgres: INSERT INTO sprints internal detail")

		// Given: un formulario válido y una falla interna durante el guardado.
		response, body := app.postCreate(t, url.Values{
			web.FieldGoal:      {"Conservar ante falla"},
			web.FieldStartDate: {"2026-11-01"},
			web.FieldEndDate:   {"2026-11-14"},
			web.FieldStory:     {"73"},
		})

		// When / Then: se informa el error genérico y nunca se confirma el éxito.
		if response.StatusCode != http.StatusInternalServerError {
			t.Errorf("se esperaba 500, se obtuvo %d", response.StatusCode)
		}
		if !strings.Contains(body, web.MsgOperationFailed) || strings.Contains(body, "internal detail") || strings.Contains(body, web.MsgSprintCreated) || strings.Contains(body, "alert-success") {
			t.Errorf("se esperaba solo el error genérico, se obtuvo: %s", body)
		}
		if !strings.Contains(body, "Conservar ante falla") || !strings.Contains(body, `name="historia" value="73" checked`) || app.catalog.active[projectID] != nil {
			t.Error("los datos deben conservarse y el Sprint no debe quedar creado")
		}
	})
}

func TestWebSprintsBDD_ViewActiveSprint(t *testing.T) {
	// Given: el proyecto seleccionado tiene un Sprint activo con historias en distintos estados.
	const projectID = 704
	active := &web.SprintView{
		ID: 96, Number: 7, Label: "Sprint 7", Goal: "Completar autenticación",
		StartDate: time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, time.November, 14, 0, 0, 0, 0, time.UTC),
		Stories: []web.SprintStoryRow{
			{ID: 701, Title: "Implementar inicio de sesión", Status: sprint.EnSprint},
			{ID: 702, Title: "Probar recuperación de cuenta", Status: sprint.InProgress},
		},
	}
	app := newSprintBDDApp(t, projectID, "Proyecto con Sprint activo", active, nil)
	app.chooseProject(t, projectID)

	// When: abro la sección Sprints.
	response, body := app.getSprints(t)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", response.StatusCode)
	}

	// Then: veo los detalles del Sprint y el estado de cada historia.
	for _, expected := range []string{
		"Sprint 7",
		"Completar autenticación",
		"01/11/2026",
		"14/11/2026",
		"Implementar inicio de sesión",
		"Probar recuperación de cuenta",
		string(sprint.EnSprint),
		string(sprint.InProgress),
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("la vista del Sprint debía contener %q: %s", expected, body)
		}
	}
	if strings.Contains(body, "CREATE_FORM_MARKER") {
		t.Error("no debe mostrarse el formulario de creación cuando hay Sprint activo")
	}
}

func TestWebSprintsBDD_NoActiveSprintShowsCreationForm(t *testing.T) {
	// Given: elijo un proyecto que todavía no tiene Sprint activo.
	const projectID = 705
	app := newSprintBDDApp(t, projectID, "Proyecto sin Sprint", nil, nil)
	app.chooseProject(t, projectID)

	// When: abro la sección Sprints.
	response, body := app.getSprints(t)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", response.StatusCode)
	}

	// Then: veo el mensaje de estado vacío y el formulario para crear el primer Sprint.
	for _, expected := range []string{"No hay un Sprint activo", "Sprint Goal", "Fecha de inicio", "Fecha de fin", "Crear Sprint"} {
		if !strings.Contains(body, expected) {
			t.Errorf("la pantalla sin Sprint debía contener %q: %s", expected, body)
		}
	}
}

type sprintAssignmentBDDAdapter struct {
	service sprint.Service
	store   sprint.SprintStore
}

func (a sprintAssignmentBDDAdapter) CreateSprint(
	ctx context.Context,
	request sprint.SprintCreationRequest,
	project sprint.Project,
	stories []sprint.UserStory,
) (sprint.SprintCreationResult, error) {
	return a.service.CreateSprint(ctx, request, project, stories)
}

func (a sprintAssignmentBDDAdapter) AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error {
	return a.store.AssignStories(ctx, sprintID, storyIDs)
}

func (a sprintAssignmentBDDAdapter) RemoveStory(ctx context.Context, sprintID, storyID int64) error {
	return a.store.RemoveStory(ctx, sprintID, storyID)
}

type sprintAssignmentBDDApp struct {
	fixture *seleccionFixture
	server  *httptest.Server
	client  *http.Client
	store   sprint.SprintStore
	sprint  sprint.Sprint
	storyID int64
}

func newSprintAssignmentBDDApp(t *testing.T) *sprintAssignmentBDDApp {
	t.Helper()
	fixture := nuevaSeleccionFixture(t)
	store, err := sprint.NewPostgreSQLSprintStore(os.Getenv(web.EnvDatabaseURL))
	if err != nil {
		t.Fatalf("no se pudo abrir SprintStore PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("no se pudo cerrar SprintStore: %v", err)
		}
	})

	activeSprint, err := store.CreateSprint(context.Background(), sprint.Sprint{
		ProjectID: fixture.proyectoA,
		Goal:      "Sprint activo para BDD",
		StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint activo de fixture: %v", err)
	}
	var storyID int64
	if err := fixture.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, status)
		VALUES ($1, $2, $3, 'Media', 'Pendiente') RETURNING id`,
		fixture.proyectoA,
		fmt.Sprintf("BDD HU23 historia disponible %d", time.Now().UnixNano()),
		"Historia pendiente para probar su asignación desde la web",
	).Scan(&storyID); err != nil {
		t.Fatalf("no se pudo crear la historia de fixture: %v", err)
	}
	t.Cleanup(func() {
		if _, err := fixture.db.Exec(`DELETE FROM user_stories WHERE id = $1`, storyID); err != nil {
			t.Errorf("no se pudo limpiar la historia BDD: %v", err)
		}
	})

	server := web.NewServer(silencioso())
	web.RegisterContext(server, web.NewPostgreSQLContextStore(fixture.db))
	catalog := web.NewPostgreSQLSprintCatalog(fixture.db)
	actions := sprintAssignmentBDDAdapter{service: sprint.NewService(store), store: store}
	if err := web.RegisterSprints(server, catalog, actions); err != nil {
		t.Fatalf("no se pudo registrar la sección Sprints: %v", err)
	}
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("no se pudo crear el cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.PostForm(ts.URL+"/contexto/proyecto", url.Values{
		"proyecto_id": {fmt.Sprintf("%d", fixture.proyectoA)},
		"volver_a":    {"/sprints"},
	})
	if err != nil {
		t.Fatalf("no se pudo seleccionar el proyecto de BDD: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("seleccionar proyecto: se esperaba 303, se obtuvo %d", response.StatusCode)
	}
	return &sprintAssignmentBDDApp{
		fixture: fixture,
		server:  ts,
		client:  client,
		store:   store,
		sprint:  activeSprint,
		storyID: storyID,
	}
}

func (a *sprintAssignmentBDDApp) assign(t *testing.T, storyIDs ...string) (*http.Response, string) {
	t.Helper()
	values := url.Values{}
	for _, storyID := range storyIDs {
		values.Add(web.FieldStory, storyID)
	}
	response, err := a.client.PostForm(a.server.URL+"/sprints/historias", values)
	if err != nil {
		t.Fatalf("POST /sprints/historias: %v", err)
	}
	return response, sprintBDDReadBody(t, response)
}

func (a *sprintAssignmentBDDApp) remove(t *testing.T, storyID int64) (*http.Response, string) {
	t.Helper()
	response, err := a.client.PostForm(a.server.URL+"/sprints/historias/quitar", url.Values{
		web.FieldStoryID: {fmt.Sprintf("%d", storyID)},
	})
	if err != nil {
		t.Fatalf("POST /sprints/historias/quitar: %v", err)
	}
	return response, sprintBDDReadBody(t, response)
}

func (a *sprintAssignmentBDDApp) getSprints(t *testing.T) (*http.Response, string) {
	t.Helper()
	response, err := a.client.Get(a.server.URL + "/sprints")
	if err != nil {
		t.Fatalf("GET /sprints: %v", err)
	}
	return response, sprintBDDReadBody(t, response)
}

func (a *sprintAssignmentBDDApp) storyState(t *testing.T) (string, int64, int) {
	t.Helper()
	var status string
	var activeSprintID int64
	var assignments int
	err := a.fixture.db.QueryRow(`
		SELECT us.status, COALESCE(us.active_sprint_id, 0),
		       (SELECT count(*) FROM sprint_story_assignments WHERE story_id = us.id)
		FROM user_stories us WHERE us.id = $1`, a.storyID).Scan(&status, &activeSprintID, &assignments)
	if err != nil {
		t.Fatalf("no se pudo consultar la historia persistida: %v", err)
	}
	return status, activeSprintID, assignments
}

func TestWebSprintsBDD_AddAvailableStory(t *testing.T) {
	// Given: un Sprint activo y una historia disponible en el Backlog del proyecto.
	app := newSprintAssignmentBDDApp(t)

	// When: marco la historia y solicito agregarla al Sprint activo.
	response, _ := app.assign(t, fmt.Sprintf("%d", app.storyID))
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/sprints" {
		t.Fatalf("se esperaba 303 a /sprints; estado=%d location=%q", response.StatusCode, response.Header.Get("Location"))
	}

	// Then: veo una única confirmación y PostgreSQL refleja la asociación y estado.
	_, body := app.getSprints(t)
	if count := strings.Count(body, web.MsgStoriesAdded); count != 1 {
		t.Errorf("el mensaje exacto debía aparecer una vez; count=%d body=%s", count, body)
	}
	status, activeSprintID, assignments := app.storyState(t)
	if status != string(sprint.EnSprint) || activeSprintID != app.sprint.ID || assignments != 1 {
		t.Errorf("historia no asignada en PostgreSQL: status=%q activeSprintID=%d assignments=%d", status, activeSprintID, assignments)
	}
}

func TestWebSprintsBDD_AddWithoutSelection(t *testing.T) {
	// Given: un Sprint activo y una historia pendiente sin marcar.
	app := newSprintAssignmentBDDApp(t)

	// When: envío la acción de agregar sin seleccionar historias.
	response, body := app.assign(t)
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("se esperaba 422 sin selección, se obtuvo %d", response.StatusCode)
	}

	// Then: aparece el mensaje exacto y la fila de PostgreSQL no cambia.
	if !strings.Contains(body, web.MsgSelectAtLeastOneStory) {
		t.Errorf("falta el mensaje %q: %s", web.MsgSelectAtLeastOneStory, body)
	}
	status, activeSprintID, assignments := app.storyState(t)
	if status != string(sprint.Pendiente) || activeSprintID != 0 || assignments != 0 {
		t.Errorf("la BD no debía mutar sin selección: status=%q activeSprintID=%d assignments=%d", status, activeSprintID, assignments)
	}
}

func TestWebSprintsBDD_RemoveStoryAndReturnItToBacklog(t *testing.T) {
	// Given: una historia Pendiente asignada al Sprint activo.
	app := newSprintAssignmentBDDApp(t)
	if err := app.store.AssignStories(context.Background(), app.sprint.ID, []int64{app.storyID}); err != nil {
		t.Fatalf("no se pudo asignar la historia al Sprint de prueba: %v", err)
	}

	// When: quito la historia desde la pantalla web.
	response, _ := app.remove(t, app.storyID)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/sprints" {
		t.Fatalf("se esperaba 303 a /sprints; estado=%d location=%q", response.StatusCode, response.Header.Get("Location"))
	}

	// Then: veo una sola confirmación y PostgreSQL devuelve la historia al Backlog.
	_, body := app.getSprints(t)
	if count := strings.Count(body, web.MsgStoryRemoved); count != 1 {
		t.Errorf("el mensaje debía mostrarse una vez; count=%d body=%s", count, body)
	}
	status, activeSprintID, assignments := app.storyState(t)
	if status != string(sprint.Pendiente) || activeSprintID != 0 || assignments != 0 {
		t.Errorf("la historia debía volver a Pendiente sin asociación: status=%q activeSprintID=%d assignments=%d", status, activeSprintID, assignments)
	}
	if !strings.Contains(body, fmt.Sprintf(`name="historia" value="%d"`, app.storyID)) {
		t.Errorf("la historia devuelta debía estar disponible en el listado: %s", body)
	}
}

func TestWebSprintsBDD_CannotRemoveStoryAndRejectsDirectSubmission(t *testing.T) {
	// Given: una historia En progreso que sigue asignada al Sprint activo.
	app := newSprintAssignmentBDDApp(t)
	if err := app.store.AssignStories(context.Background(), app.sprint.ID, []int64{app.storyID}); err != nil {
		t.Fatalf("no se pudo asignar la historia al Sprint de prueba: %v", err)
	}
	if _, err := app.fixture.db.Exec(`UPDATE user_stories SET status = $1 WHERE id = $2`, sprint.InProgress, app.storyID); err != nil {
		t.Fatalf("no se pudo poner la historia En progreso: %v", err)
	}

	// When: abro la pantalla e intento quitarla igualmente con un POST directo.
	_, page := app.getSprints(t)
	response, body := app.remove(t, app.storyID)

	// Then: el botón queda deshabilitado con el motivo y el envío no altera la BD.
	if !strings.Contains(page, "Ya está en progreso") || !strings.Contains(page, "disabled") {
		t.Errorf("la vista debía deshabilitar Quitar con el motivo exacto: %s", page)
	}
	if response.StatusCode != http.StatusConflict || !strings.Contains(body, "Ya está en progreso") {
		t.Errorf("el POST directo debía rechazarse con 409 y el motivo: status=%d body=%s", response.StatusCode, body)
	}
	status, activeSprintID, assignments := app.storyState(t)
	if status != string(sprint.InProgress) || activeSprintID != app.sprint.ID || assignments != 1 {
		t.Errorf("el rechazo debía conservar la historia en el Sprint: status=%q activeSprintID=%d assignments=%d", status, activeSprintID, assignments)
	}
}
