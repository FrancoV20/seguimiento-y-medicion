package web

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-16, T011, T018 y T019: "Sprints anteriores" con el servidor, el contexto de HU-20b, el
// catálogo, HU-07 y PostgreSQL reales, y los datos de past_sprint_fixtures_test.go.

type pastConfig struct {
	catalog func(PastSprintCatalog) PastSprintCatalog
	calc    func(PastSprintMetrics) PastSprintMetrics
}

type pastOption func(*pastConfig)

// withPastCatalog reemplaza el catálogo (para contar llamadas o simular fallas).
func withPastCatalog(wrap func(PastSprintCatalog) PastSprintCatalog) pastOption {
	return func(c *pastConfig) { c.catalog = wrap }
}

// withPastMetrics reemplaza HU-07 (para contar llamadas o simular fallas).
func withPastMetrics(wrap func(PastSprintMetrics) PastSprintMetrics) pastOption {
	return func(c *pastConfig) { c.calc = wrap }
}

type pastApp struct {
	*ctxApp
	fx *pastSprintFixture
}

// newPastApp arma el servidor con el contexto de trabajo y "Sprints anteriores" registrados.
func newPastApp(t *testing.T, opts ...pastOption) *pastApp {
	t.Helper()
	var cfg pastConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	db := openContextTestDB(t)
	var catalog PastSprintCatalog = NewPostgreSQLPastSprintCatalog(db)
	if cfg.catalog != nil {
		catalog = cfg.catalog(catalog)
	}
	var calc PastSprintMetrics = metrics.NewService(metrics.NewPostgreSQLMetricsStore(db))
	if cfg.calc != nil {
		calc = cfg.calc(calc)
	}
	app := newCtxApp(t, nil, func(s *Server) {
		if err := RegisterPastSprints(s, catalog, calc); err != nil {
			t.Fatalf("no se pudo registrar Sprints anteriores: %v", err)
		}
	})
	return &pastApp{ctxApp: app, fx: newPastSprintFixture(t, db)}
}

// open devuelve un navegador (que no sigue redirecciones) con el proyecto ya elegido.
func (a *pastApp) open(t *testing.T, projectID int64) *http.Client {
	t.Helper()
	c := a.client(false)
	if resp, _ := a.selectProject(t, c, id64(projectID), "/sprints-anteriores"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto: se obtuvo %d", resp.StatusCode)
	}
	return c
}

// page elige el proyecto y pide la dirección.
func (a *pastApp) page(t *testing.T, projectID int64, path string) (*http.Response, string) {
	t.Helper()
	return a.get(t, a.open(t, projectID), path)
}

func pastWantText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en la pantalla", want)
		}
	}
}

func pastWantNoText(t *testing.T, body string, unwanted ...string) {
	t.Helper()
	for _, text := range unwanted {
		if strings.Contains(body, text) {
			t.Errorf("la pantalla no debe contener %q", text)
		}
	}
}

func pastWantStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("se esperaba %d, se obtuvo %d", want, resp.StatusCode)
	}
}

// --- T011: lista ---

// Escenarios 1 y 2.
func TestHU16_Listado(t *testing.T) {
	app := newPastApp(t)
	resp, body := app.page(t, app.fx.alfa, "/sprints-anteriores")

	pastWantStatus(t, resp, http.StatusOK)
	pastWantText(t, body,
		"Sprints anteriores",
		"Sprint 2", "Gestionar perfiles y reportes", "Del 01/09/2026 al 14/09/2026",
		"Sprint 1", "Dar de alta usuarios", "Del 18/08/2026 al 31/08/2026",
		fmt.Sprintf(`href="/sprints-anteriores/%d"`, app.fx.sprints["Alfa 2"]),
		fmt.Sprintf(`href="/sprints-anteriores/%d"`, app.fx.sprints["Alfa 1"]),
	)
	if strings.Index(body, "Sprint 2") > strings.Index(body, "Sprint 1") {
		t.Error("el Sprint 2 debía aparecer antes que el Sprint 1")
	}
	pastWantNoText(t, body, "Sprint 3", "Exportar e informar", "Migrar datos", "Auditar accesos", MsgPastSprintsEmpty)
}

// Escenario 6.
func TestHU16_SinSprintsAnteriores(t *testing.T) {
	app := newPastApp(t)
	resp, body := app.page(t, app.fx.beta, "/sprints-anteriores")

	pastWantStatus(t, resp, http.StatusOK)
	pastWantText(t, body, "Aún no hay Sprints anteriores")
}

// T014: con un argumento nil no se registra nada.
func TestRegisterPastSprints_ArgumentosNil(t *testing.T) {
	srv := NewServer(nil)
	calc := metrics.NewService(nil)
	catalog := NewPostgreSQLPastSprintCatalog(nil)
	for name, register := range map[string]func() error{
		"servidor": func() error { return RegisterPastSprints(nil, catalog, calc) },
		"catálogo": func() error { return RegisterPastSprints(srv, nil, calc) },
		"métricas": func() error { return RegisterPastSprints(srv, catalog, nil) },
	} {
		if err := register(); err == nil {
			t.Errorf("sin %s se esperaba un error", name)
		}
	}
}

// FR-013: sin proyecto activo, HU-20b pide elegirlo y vuelve a la dirección pedida.
func TestHU16_SinProyecto(t *testing.T) {
	app := newPastApp(t)
	paths := map[string]string{"/sprints-anteriores": "/contexto/proyecto?volver_a=%2Fsprints-anteriores"}
	for path, want := range paths {
		resp, _ := app.get(t, app.client(false), path)
		pastWantStatus(t, resp, http.StatusSeeOther)
		if loc := resp.Header.Get("Location"); loc != want {
			t.Errorf("%s: se esperaba %q, se obtuvo %q", path, want, loc)
		}
	}
}

// --- T018: detalle ---

func (a *pastApp) detailPath(key string) string {
	return fmt.Sprintf("/sprints-anteriores/%d", a.fx.sprints[key])
}

// pastWantTableRow comprueba una fila de tabla: celdas en orden, con espacios libres entre etiquetas.
func pastWantTableRow(t *testing.T, body string, cells ...string) {
	t.Helper()
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = `<td[^>]*>\s*` + regexp.QuoteMeta(c) + `\s*</td>`
	}
	if !regexp.MustCompile(`(?s)<tr>\s*` + strings.Join(parts, `\s*`) + `\s*</tr>`).MatchString(body) {
		t.Errorf("falta la fila %q", cells)
	}
}

// pastSection devuelve el texto entre el encabezado h2 dado y el siguiente h2 (o el final).
func pastSection(body, heading string) string {
	start := strings.Index(body, "<h2>"+heading+"</h2>")
	if start < 0 {
		return ""
	}
	rest := body[start+len(heading)+9:]
	if end := strings.Index(rest, "<h2>"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// Escenario 3.
func TestHU16_Detalle(t *testing.T) {
	app := newPastApp(t)
	resp, body := app.page(t, app.fx.alfa, app.detailPath("Alfa 2"))

	pastWantStatus(t, resp, http.StatusOK)
	pastWantText(t, body, "Sprint 2", "Objetivo: Gestionar perfiles y reportes", "Del 01/09/2026 al 14/09/2026",
		"Velocidad: 10 Story Points", "Completadas", "No completadas", LinkPastSprintsBack)
	done, pending := pastSection(body, "Completadas"), pastSection(body, "No completadas")
	pastWantTableRow(t, done, "Login", "5", "Hecho")
	pastWantTableRow(t, done, "Tablero", "3", "Hecho")
	pastWantTableRow(t, done, "Perfil", "2", "Hecho")
	pastWantTableRow(t, pending, "Reportes", "8", "En progreso", "volvió al Backlog")
	pastWantNoText(t, done, "Reportes", "volvió al Backlog")
	pastWantNoText(t, pending, "Login", "Tablero", "Perfil")
	if i, j, k := strings.Index(done, "Login"), strings.Index(done, "Tablero"), strings.Index(done, "Perfil"); !(i < j && j < k) {
		t.Error("las completadas debían estar en orden de creación: «Login», «Tablero», «Perfil»")
	}
	pastWantNoText(t, body, MsgPastSprintAllCompleted, MsgPastSprintNoneCompleted, MsgPastSprintNoRecord)
}

// Escenario 4.
func TestHU16_TodasCompletadas(t *testing.T) {
	app := newPastApp(t)
	_, body := app.page(t, app.fx.alfa, app.detailPath("Alfa 1"))

	pastWantText(t, body, "Velocidad: 3 Story Points", "Todas las historias se completaron")
	pastWantTableRow(t, pastSection(body, "Completadas"), "Alta", "3", "Hecho")
	pastWantNoText(t, body, "volvió al Backlog", "Ninguna historia se completó")
	if strings.Count(body, "<table") != 1 {
		t.Errorf("debía haber una sola tabla (Completadas), hay %d", strings.Count(body, "<table"))
	}
}

// Escenario 5.
func TestHU16_NingunaCompletada(t *testing.T) {
	app := newPastApp(t)
	_, body := app.page(t, app.fx.gamma, app.detailPath("Gamma 1"))

	pastWantText(t, body, "Velocidad: 0 Story Points", "Ninguna historia se completó")
	pending := pastSection(body, "No completadas")
	pastWantTableRow(t, pending, "Migración", "5", "En progreso", "volvió al Backlog")
	pastWantTableRow(t, pending, "Respaldo", "Sin estimar", "En Sprint", "volvió al Backlog")
	if strings.Count(body, "<table") != 1 {
		t.Errorf("debía haber una sola tabla (No completadas), hay %d", strings.Count(body, "<table"))
	}
}

// Nota 5: Sprint finalizado sin registro de cierre.
func TestHU16_SinRegistro(t *testing.T) {
	app := newPastApp(t)
	_, body := app.page(t, app.fx.gamma, app.detailPath("Gamma 2"))

	pastWantText(t, body, "No hay registro de las historias de este Sprint", "Velocidad: 2 Story Points")
	pastWantNoText(t, body, "<table", MsgPastSprintAllCompleted, MsgPastSprintNoneCompleted)
}

// SC-002: la velocidad de HU-07 coincide con la suma de las filas "Hecho" del registro de cierre.
func TestHU16_VelocidadCoherente(t *testing.T) {
	app := newPastApp(t)
	for _, c := range []struct {
		key     string
		project int64
		want    int
	}{{"Alfa 1", app.fx.alfa, 3}, {"Alfa 2", app.fx.alfa, 10}, {"Gamma 1", app.fx.gamma, 0}} {
		_, body := app.page(t, c.project, app.detailPath(c.key))
		record := app.fx.storyPointsOfDone(t, app.fx.sprints[c.key])
		if record != c.want || !strings.Contains(body, fmt.Sprintf("Velocidad: %d Story Points", record)) {
			t.Errorf("%s: el registro suma %d y se esperaba «Velocidad: %d Story Points» en la pantalla", c.key, record, c.want)
		}
	}
}

// countingPastCatalog cuenta las llamadas a SprintDetail.
type countingPastCatalog struct {
	PastSprintCatalog
	details *int
}

func (c countingPastCatalog) SprintDetail(ctx context.Context, projectID, sprintID int64) (PastSprintDetail, error) {
	*c.details++
	return c.PastSprintCatalog.SprintDetail(ctx, projectID, sprintID)
}

// countingPastMetrics cuenta las llamadas a HU-07.
type countingPastMetrics struct {
	PastSprintMetrics
	calls *int
}

func (c countingPastMetrics) CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error) {
	*c.calls++
	return c.PastSprintMetrics.CalculateMetrics(sprintID)
}

// Escenario 8 (FR-010): 404 decidido antes de calcular; un id inválido, sin consultar la base.
func TestHU16_DetalleNoEncontrado(t *testing.T) {
	var details, calls int
	app := newPastApp(t,
		withPastCatalog(func(c PastSprintCatalog) PastSprintCatalog { return countingPastCatalog{c, &details} }),
		withPastMetrics(func(m PastSprintMetrics) PastSprintMetrics { return countingPastMetrics{m, &calls} }),
	)
	c := app.open(t, app.fx.alfa)
	found := []string{id64(app.fx.sprints["Alfa 3"]), id64(app.fx.sprints["Gamma 1"]), "2147483647"}
	invalid := []string{"abc", "0", "-1", "01"}
	for _, id := range append(found, invalid...) {
		resp, body := app.get(t, c, "/sprints-anteriores/"+id)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: se esperaba 404, se obtuvo %d", id, resp.StatusCode)
		}
		pastWantText(t, body, LinkBackHome)
		pastWantNoText(t, body, "Velocidad:", "Exportar e informar", "Migrar datos")
	}
	if calls != 0 {
		t.Errorf("no debía calcularse la velocidad de HU-07: hubo %d llamadas", calls)
	}
	if details != len(found) {
		t.Errorf("los id inválidos no debían consultar la base: se esperaban %d consultas, hubo %d", len(found), details)
	}
}

// --- T019: solo lectura, errores y bordes ---

// Escenario 7.
func TestHU16_SoloLectura(t *testing.T) {
	app := newPastApp(t)
	c := app.open(t, app.fx.alfa)
	path := app.detailPath("Alfa 2")
	_, body := app.get(t, c, path)
	pastWantNoText(t, body, "<form", "<button", "<script")

	before := app.fx.storyPointsOfDone(t, app.fx.sprints["Alfa 2"])
	for _, req := range []struct{ method, path string }{
		{http.MethodPost, path}, {http.MethodPut, path}, {http.MethodDelete, path}, {http.MethodPost, "/sprints-anteriores"},
	} {
		resp, body := app.doWithHeaders(t, req.method, req.path, url.Values{"goal": {"cambiado"}}, map[string]string{"Origin": app.ts.URL})
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: se esperaba 405, se obtuvo %d", req.method, req.path, resp.StatusCode)
		}
		pastWantText(t, body, LinkBackHome)
	}
	var goal string
	if err := app.db.QueryRow(`SELECT goal FROM sprints WHERE id = $1`, app.fx.sprints["Alfa 2"]).Scan(&goal); err != nil || goal != "Gestionar perfiles y reportes" {
		t.Errorf("el Sprint no debía cambiar: %q, %v", goal, err)
	}
	if after := app.fx.storyPointsOfDone(t, app.fx.sprints["Alfa 2"]); after != before {
		t.Errorf("el registro de cierre no debía cambiar: %d → %d", before, after)
	}
}

type failingPastCatalog struct{ err error }

func (f failingPastCatalog) FinishedSprints(context.Context, int64) ([]PastSprintSummary, error) {
	return nil, f.err
}

func (f failingPastCatalog) SprintDetail(context.Context, int64, int64) (PastSprintDetail, error) {
	return PastSprintDetail{}, f.err
}

type failingPastMetrics struct{ err error }

func (f failingPastMetrics) CalculateMetrics(int64) (metrics.SprintMetrics, error) {
	return metrics.SprintMetrics{}, f.err
}

const pastInternalDetail = `pq: connection refused; SELECT goal FROM sprint_story_results at C:\internal\past.go:9`

// Escenario 9.
func TestHU16_ErrorAlConsultar(t *testing.T) {
	internal := errors.New(pastInternalDetail)
	unavailable := fmt.Errorf("leer los Sprints: %w", driver.ErrBadConn)
	failCatalog := func(err error) pastOption {
		return withPastCatalog(func(PastSprintCatalog) PastSprintCatalog { return failingPastCatalog{err} })
	}
	failMetrics := func(err error) pastOption {
		return withPastMetrics(func(PastSprintMetrics) PastSprintMetrics { return failingPastMetrics{err} })
	}
	cases := []struct {
		name   string
		opt    pastOption
		detail bool
		want   int
	}{
		{"falla del catálogo en la lista", failCatalog(internal), false, http.StatusInternalServerError},
		{"falla del catálogo en el detalle", failCatalog(internal), true, http.StatusInternalServerError},
		{"falla de HU-07 en el detalle", failMetrics(internal), true, http.StatusInternalServerError},
		{"base no disponible en la lista", failCatalog(unavailable), false, http.StatusServiceUnavailable},
		{"base no disponible en el detalle", failCatalog(unavailable), true, http.StatusServiceUnavailable},
		{"base no disponible en HU-07", failMetrics(unavailable), true, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newPastApp(t, tc.opt)
			path := "/sprints-anteriores"
			if tc.detail {
				path = app.detailPath("Alfa 2")
			}
			resp, body := app.page(t, app.fx.alfa, path)

			if resp.StatusCode != tc.want {
				t.Errorf("se esperaba %d, se obtuvo %d", tc.want, resp.StatusCode)
			}
			pastWantText(t, body, LinkBackHome)
			pastWantNoText(t, body, "Gestionar perfiles y reportes", "Login", "Velocidad:", "Sprint 2")
			lower := strings.ToLower(body)
			for _, leak := range []string{"select ", "pq:", "sql:", ".go:", `c:\`, "internal", "connection refused", "sprint_story_results"} {
				if strings.Contains(lower, leak) {
					t.Errorf("la página expone un detalle interno (contiene %q)", leak)
				}
			}
		})
	}
}

// FR-013 (detalle): sin proyecto activo, HU-20b pide elegirlo y vuelve al detalle.
func TestHU16_SinProyectoDetalle(t *testing.T) {
	app := newPastApp(t)
	path := app.detailPath("Alfa 2")
	resp, _ := app.get(t, app.client(false), path)
	pastWantStatus(t, resp, http.StatusSeeOther)
	want := "/contexto/proyecto?volver_a=" + url.QueryEscape(path)
	if loc := resp.Header.Get("Location"); loc != want {
		t.Errorf("se esperaba %q, se obtuvo %q", want, loc)
	}
}

// FR-002, FR-006: los textos ingresados se muestran escapados en la lista y en el detalle.
func TestHU16_TextosEscapados(t *testing.T) {
	app := newPastApp(t)
	f := app.fx
	project := f.addProject(t, "Escapes")
	f.story(t, project, "<b>Login & Co</b>", "Pendiente", 3)
	sprintID := f.sprint(t, project, "Escapes 1", 1, "<script>x</script>", "2026-09-01", "2026-09-14", "Finalizado")
	f.closed(t, sprintID, "<b>Login & Co</b>", "Hecho", 3, true)

	_, list := app.page(t, project, "/sprints-anteriores")
	_, detail := app.page(t, project, app.detailPath("Escapes 1"))
	pastWantText(t, list, "&lt;script&gt;x&lt;/script&gt;")
	pastWantText(t, detail, "&lt;script&gt;x&lt;/script&gt;", "&lt;b&gt;Login &amp; Co&lt;/b&gt;", "Velocidad: 3 Story Points")
	pastWantNoText(t, list+detail, "<script", "<b>Login")
}

// SC-004: 50 Sprints finalizados y un registro de 200 filas, en menos de 500 ms cada pantalla.
func TestHU16_Rendimiento(t *testing.T) {
	app := newPastApp(t)
	f := app.fx
	project := f.addProject(t, "Rendimiento")
	var big int64
	for n := 1; n <= 50; n++ {
		id := f.sprint(t, project, fmt.Sprintf("Rendimiento %d", n), n, fmt.Sprintf("Objetivo %d", n), "2026-01-01", "2026-01-14", "Finalizado")
		if n == 50 {
			big = id
		}
	}
	if _, err := f.db.Exec(`
		WITH s AS (
			INSERT INTO user_stories (project_id, title, description, priority, status, story_points)
			SELECT $1, 'Historia ' || g, 'd', 'Media', 'Pendiente', 1 FROM generate_series(1, 200) g
			RETURNING id, title
		), r AS (
			INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
			SELECT $2, id, CASE WHEN substring(title from 10)::int <= 150 THEN 'Hecho' ELSE 'En progreso' END, 1 FROM s
			RETURNING story_id, final_status
		)
		INSERT INTO sprint_story_assignments (sprint_id, story_id) SELECT $2, story_id FROM r WHERE final_status = 'Hecho'`,
		project, big); err != nil {
		t.Fatalf("no se pudo cargar el registro grande: %v", err)
	}
	f.exec(t, `UPDATE user_stories SET status = 'Hecho', active_sprint_id = $1
		WHERE id IN (SELECT story_id FROM sprint_story_assignments WHERE sprint_id = $1)`, big)

	c := app.open(t, project)
	app.get(t, c, "/sprints-anteriores") // calienta conexiones y plantillas
	for _, path := range []string{"/sprints-anteriores", fmt.Sprintf("/sprints-anteriores/%d", big)} {
		start := time.Now()
		resp, body := app.get(t, c, path)
		elapsed := time.Since(start)
		pastWantStatus(t, resp, http.StatusOK)
		if elapsed >= 500*time.Millisecond {
			t.Errorf("%s tardó %v (límite 500 ms)", path, elapsed)
		}
		if strings.HasSuffix(path, "es") {
			if n := strings.Count(body, `href="/sprints-anteriores/`); n != 50 {
				t.Errorf("se esperaban 50 Sprints en la lista, hay %d", n)
			}
		} else {
			pastWantText(t, body, "Velocidad: 150 Story Points")
		}
		t.Logf("%s: %v", path, elapsed)
	}
}
