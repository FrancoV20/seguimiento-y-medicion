package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-28, T019: las secciones reales (HU-11, HU-15, HU-23, HU-05a, HU-05b, HU-18, HU-14 y HU-24)
// con dobles de sus puertos que cuentan llamadas y el PostgreSQLContextStore real (escenarios 3 a 7
// y 9). Cualquier llamada a un doble indica que la solicitud llegó al handler de la historia.

// hitCounter cuenta las llamadas a los dobles de las secciones.
type hitCounter struct{ n atomic.Int32 }

func (c *hitCounter) hit() { c.n.Add(1) }

type smSprintCatalog struct{ *hitCounter }

func (d smSprintCatalog) ActiveSprint(context.Context, int64) (*SprintView, error) {
	d.hit()
	return nil, nil
}

func (d smSprintCatalog) ListStoryOptions(context.Context, int64) ([]StoryOption, error) {
	d.hit()
	return nil, nil
}

type smSprintActions struct{ *hitCounter }

func (d smSprintActions) CreateSprint(context.Context, sprint.SprintCreationRequest, sprint.Project, []sprint.UserStory) (sprint.SprintCreationResult, error) {
	d.hit()
	return sprint.SprintCreationResult{}, nil
}

func (d smSprintActions) AssignStories(context.Context, int64, []int64) error {
	d.hit()
	return nil
}

func (d smSprintActions) RemoveStory(context.Context, int64, int64) error {
	d.hit()
	return nil
}

type smPokerActions struct{ *hitCounter }

func (d smPokerActions) OpenRound(context.Context, int64, int64, int64) (poker.OpenedRound, error) {
	d.hit()
	return poker.OpenedRound{}, nil
}

func (d smPokerActions) Vote(context.Context, int64, int64, int64, string) (poker.VoteOutcome, error) {
	d.hit()
	return poker.VoteOutcome{}, nil
}

func (d smPokerActions) CloseRound(context.Context, int64, int64, int64) (poker.ClosedRound, error) {
	d.hit()
	return poker.ClosedRound{}, nil
}

func (d smPokerActions) OpenNextRound(context.Context, int64, int64, int64) (poker.OpenedRound, error) {
	d.hit()
	return poker.OpenedRound{}, nil
}

func (d smPokerActions) AgreeEstimate(context.Context, int64, int64, int64, string) (poker.AgreedEstimate, error) {
	d.hit()
	return poker.AgreedEstimate{}, nil
}

type smPokerCatalog struct{ *hitCounter }

func (d smPokerCatalog) PokerBoard(context.Context, int64, int64) (PokerBoard, error) {
	d.hit()
	return PokerBoard{}, nil
}

type smProjectCatalog struct{ *hitCounter }

func (d smProjectCatalog) ListProjects(context.Context) ([]ProjectRow, error) {
	d.hit()
	return nil, nil
}

func (d smProjectCatalog) ListMembers(context.Context) ([]MemberOption, error) {
	d.hit()
	return nil, nil
}

func (d smProjectCatalog) ProjectForEdit(_ context.Context, id int64) (ProjectEdit, error) {
	d.hit()
	return ProjectEdit{ID: id}, nil
}

type smProjectStore struct{ *hitCounter }

func (d smProjectStore) Create(*project.Project) error {
	d.hit()
	return nil
}

func (d smProjectStore) MemberExists(int) (bool, error) {
	d.hit()
	return true, nil
}

func (d smProjectStore) Update(context.Context, project.ProjectUpdate) error {
	d.hit()
	return nil
}

type smBoardCatalog struct{ *hitCounter }

func (d smBoardCatalog) ActiveBoard(context.Context, int64) (BoardView, bool, error) {
	d.hit()
	return BoardView{}, false, nil
}

type smCloseStore struct{ *hitCounter }

func (d smCloseStore) FindSprint(context.Context, int64, int64) (sprint.Sprint, error) {
	d.hit()
	return sprint.Sprint{}, sprint.ErrSprintNotFound
}

func (d smCloseStore) CloseSprint(context.Context, int64, sprint.ClosureCheck) (sprint.CloseResult, error) {
	d.hit()
	return sprint.CloseResult{}, nil
}

type smClosureValidator struct{ *hitCounter }

func (d smClosureValidator) ValidateSprintClosure(context.Context, []sprint.ClosingStory) error {
	d.hit()
	return nil
}

type smStatusChanger struct{ *hitCounter }

func (d smStatusChanger) ChangeStatus(context.Context, int64, int64, sprint.StoryStatus, sprint.StoryStatus, sprint.StoryStatus) (sprint.StatusResult, error) {
	d.hit()
	return sprint.StatusResult{}, nil
}

type smEffortActions struct{ *hitCounter }

func (d smEffortActions) Validate(EffortSubmission) []EffortViolation {
	d.hit()
	return nil
}

func (d smEffortActions) Register(context.Context, EffortSubmission) ([]EffortViolation, error) {
	d.hit()
	return nil, nil
}

type smEffortCatalog struct{ *hitCounter }

func (d smEffortCatalog) ActiveSprintStories(context.Context, int64) (*EffortSprint, error) {
	d.hit()
	return nil, nil
}

func (d smEffortCatalog) StoryHistory(context.Context, int64, int64) (*EffortHistory, error) {
	d.hit()
	return nil, nil
}

// scrumMasterRoutesApp es el servidor con las secciones reales y dobles que cuentan llamadas.
// «Alfa» = proyecto A (Ana Scrum Master, Beto Integrante); «Beta» = proyecto B (Carla, sin Scrum
// Master), del contextFixture de HU-20b.
type scrumMasterRoutesApp struct {
	*ctxApp
	hits *hitCounter
}

func newScrumMasterRoutesApp(t *testing.T) *scrumMasterRoutesApp {
	t.Helper()
	hits := &hitCounter{}
	register := func(name string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("no se pudo registrar %s: %v", name, err)
		}
	}
	app := newCtxApp(t, nil, func(s *Server) {
		register("Sprints", RegisterSprints(s, smSprintCatalog{hits}, smSprintActions{hits}))
		register("Planning Poker", RegisterPlanningPoker(s, smPokerActions{hits}, smPokerCatalog{hits}))
		register("Proyectos", RegisterProjects(s, project.NewProjectService(smProjectStore{hits}), smProjectCatalog{hits}))
		register("Cierre", RegisterClose(s, sprint.NewCloseSprintService(smCloseStore{hits}, smClosureValidator{hits}), smBoardCatalog{hits}))
		register("Tablero", RegisterBoard(s, smStatusChanger{hits}, smBoardCatalog{hits}))
		register("Esfuerzo", RegisterEffort(s, smEffortActions{hits}, smEffortCatalog{hits}, nil))
	})
	setTestScrumMaster(t, app.db, app.fx.projectA, app.fx.ana)
	return &scrumMasterRoutesApp{ctxApp: app, hits: hits}
}

// as devuelve un navegador con el proyecto y el integrante elegidos (cookies firmadas de HU-26).
func (a *scrumMasterRoutesApp) as(t *testing.T, projectID, memberID int64) *http.Client {
	t.Helper()
	c := a.client(false)
	a.setCookies(t, c, idCookie(CookieProject, projectID), idCookie(CookieMember, memberID))
	return c
}

type smRoute struct {
	name   string
	path   func(projectID int64) string
	values url.Values
}

func fixedPath(p string) func(int64) string { return func(int64) string { return p } }

// scrumMasterRoutes son las 9 rutas protegidas presentes en main (T001).
func scrumMasterRoutes() []smRoute {
	return []smRoute{
		{"HU-11 modificar proyecto", func(id int64) string { return fmt.Sprintf("/proyectos/%d/modificar", id) },
			url.Values{"nombre": {"Alfa"}, "fecha_inicio": {"2026-01-01"}, "fecha_fin": {"2026-12-31"}}},
		{"HU-23 crear Sprint", fixedPath("/sprints"),
			url.Values{FieldGoal: {"Objetivo"}, FieldStartDate: {"2026-11-02"}, FieldEndDate: {"2026-11-15"}}},
		{"HU-23 agregar historias", fixedPath("/sprints/historias"), url.Values{FieldStory: {"1"}}},
		{"HU-23 quitar historia", fixedPath("/sprints/historias/quitar"), url.Values{FieldStoryID: {"1"}}},
		{"HU-15 cerrar Sprint", fixedPath("/cierre"), url.Values{"sprint_id": {"1"}}},
		{"HU-05a abrir ronda", fixedPath("/planning-poker/abrir"), url.Values{"historia": {"1"}}},
		{"HU-05b cerrar ronda", fixedPath("/planning-poker/cerrar"), url.Values{"ronda": {"1"}}},
		{"HU-18 nueva ronda", fixedPath("/planning-poker/nueva-ronda"), url.Values{"ronda": {"1"}}},
		{"HU-18 estimación acordada", fixedPath("/planning-poker/estimacion"), url.Values{"ronda": {"1"}, "valor": {"5"}}},
	}
}

func TestScrumMasterRoutes_Protegidas(t *testing.T) {
	app := newScrumMasterRoutesApp(t)
	fx := app.fx

	for _, route := range scrumMasterRoutes() {
		t.Run(route.name, func(t *testing.T) {
			// Beto, Integrante de «Alfa», que tiene Scrum Master.
			app.hits.n.Store(0)
			resp, body := app.post(t, app.as(t, fx.projectA, fx.beto), route.path(fx.projectA), route.values)
			if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, MsgScrumMasterOnly) {
				t.Errorf("Beto: estado %d; se esperaba 403 con %q", resp.StatusCode, MsgScrumMasterOnly)
			}
			if n := app.hits.n.Load(); n != 0 {
				t.Errorf("Beto: la sección recibió %d llamadas; el rechazo debía ser antes del handler", n)
			}

			// Carla en «Beta», que no tiene Scrum Master.
			app.hits.n.Store(0)
			resp, body = app.post(t, app.as(t, fx.projectB, fx.carla), route.path(fx.projectB), route.values)
			if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, MsgNoScrumMaster) {
				t.Errorf("«Beta» sin Scrum Master: estado %d; se esperaba 403 con %q", resp.StatusCode, MsgNoScrumMaster)
			}
			if n := app.hits.n.Load(); n != 0 {
				t.Errorf("«Beta» sin Scrum Master: la sección recibió %d llamadas", n)
			}

			// Ana, Scrum Master de «Alfa»: llega a la acción de la historia.
			app.hits.n.Store(0)
			resp, body = app.post(t, app.as(t, fx.projectA, fx.ana), route.path(fx.projectA), route.values)
			if resp.StatusCode == http.StatusForbidden || strings.Contains(body, TitleScrumMasterOnly) {
				t.Errorf("Ana: estado %d; el Scrum Master no debía ser rechazado", resp.StatusCode)
			}
			if n := app.hits.n.Load(); n == 0 {
				t.Errorf("Ana: la solicitud no llegó al handler de la historia (estado %d)", resp.StatusCode)
			}
		})
	}
}

func TestScrumMasterRoutes_AccionesLibres(t *testing.T) {
	app := newScrumMasterRoutesApp(t)
	fx := app.fx

	free := []smRoute{
		{"HU-05a votar", fixedPath("/planning-poker/votar"), url.Values{"ronda": {"1"}, "valor": {"5"}}},
		{"HU-14 cambiar estado en el Tablero", fixedPath("/tablero/estado"),
			url.Values{"historia_id": {"1"}, "accion": {"comenzar"}, "estado_actual": {"En Sprint"}}},
		{"HU-24 registrar esfuerzo", fixedPath("/esfuerzo"),
			url.Values{"historia": {"1"}, "fecha": {"2026-11-02"}, "actividad": {"Programar"}, "horas": {"2"}}},
		{"HU-21 crear proyecto", fixedPath("/proyectos"), url.Values{"nombre": {"Gamma"}}},
	}
	for _, route := range free {
		t.Run(route.name, func(t *testing.T) {
			app.hits.n.Store(0)
			resp, body := app.post(t, app.as(t, fx.projectA, fx.beto), route.path(fx.projectA), route.values)
			if resp.StatusCode == http.StatusForbidden || strings.Contains(body, TitleScrumMasterOnly) {
				t.Errorf("Beto: estado %d; la acción libre no debía rechazarse por rol", resp.StatusCode)
			}
			if n := app.hits.n.Load(); n == 0 {
				t.Errorf("Beto: la solicitud no llegó a la acción (estado %d)", resp.StatusCode)
			}
		})
	}

	// Las pantallas GET de las mismas secciones no cambian para un Integrante (FR-003, FR-012).
	for _, path := range []string{"/sprints", "/cierre", "/planning-poker", fmt.Sprintf("/proyectos/%d/modificar", fx.projectA)} {
		resp, body := app.get(t, app.as(t, fx.projectA, fx.beto), path)
		if resp.StatusCode != http.StatusOK || strings.Contains(body, TitleScrumMasterOnly) {
			t.Errorf("GET %s para Beto = %d; se esperaba la pantalla sin cambios", path, resp.StatusCode)
		}
	}
}
