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
)

// HU-12 (T015 y T016): pantalla "Estado del proyecto" con el servidor, el catálogo, el Tablero y
// PostgreSQL reales, el reloj fijo y los datos de project_status_fixtures_test.go.

// statusNow es el reloj de las pruebas: 28/10/2026 10:30, hora local.
var statusNow = time.Date(2026, time.October, 28, 10, 30, 0, 0, time.Local)

const statusInternalDetail = `pq: connection refused; SELECT name FROM projects at C:\internal\status.go:9`

type statusConfig struct {
	catalog func(ProjectStatusCatalog) ProjectStatusCatalog
	board   func(BoardCatalog) BoardCatalog
}

type statusOption func(*statusConfig)

// withProjectStatusCatalog reemplaza el catálogo propio (para simular fallas).
func withProjectStatusCatalog(wrap func(ProjectStatusCatalog) ProjectStatusCatalog) statusOption {
	return func(c *statusConfig) { c.catalog = wrap }
}

// withProjectStatusBoard reemplaza el catálogo del Tablero (para simular fallas).
func withProjectStatusBoard(wrap func(BoardCatalog) BoardCatalog) statusOption {
	return func(c *statusConfig) { c.board = wrap }
}

type failingStatusCatalog struct{ err error }

func (f failingStatusCatalog) StatusSummary(context.Context, int64) (ProjectStatusSummary, error) {
	return ProjectStatusSummary{}, f.err
}

type failingStatusBoard struct{ err error }

func (f failingStatusBoard) ActiveBoard(context.Context, int64) (BoardView, bool, error) {
	return BoardView{}, false, f.err
}

type statusApp struct{ *ctxApp }

// newStatusApp arma el servidor con el contexto de trabajo y "Estado del proyecto" registrados.
func newStatusApp(t *testing.T, opts ...statusOption) *statusApp {
	t.Helper()
	var cfg statusConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	db := openContextTestDB(t)
	var catalog ProjectStatusCatalog = NewPostgreSQLProjectStatusCatalog(db)
	if cfg.catalog != nil {
		catalog = cfg.catalog(catalog)
	}
	var board BoardCatalog = NewPostgreSQLBoardCatalog(db)
	if cfg.board != nil {
		board = cfg.board(board)
	}
	app := newCtxApp(t, nil, func(s *Server) {
		if err := RegisterProjectStatus(s, catalog, board, func() time.Time { return statusNow }); err != nil {
			t.Fatalf("no se pudo registrar Estado del proyecto: %v", err)
		}
	})
	return &statusApp{ctxApp: app}
}

// open devuelve un navegador (que no sigue redirecciones) con el proyecto ya elegido.
func (a *statusApp) open(t *testing.T, projectID int64) *http.Client {
	t.Helper()
	c := a.client(false)
	if resp, _ := a.selectProject(t, c, id64(projectID), "/estado"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto: se obtuvo %d", resp.StatusCode)
	}
	return c
}

// page elige el proyecto y pide GET /estado.
func (a *statusApp) page(t *testing.T, projectID int64) (*http.Response, string) {
	t.Helper()
	return a.get(t, a.open(t, projectID), "/estado")
}

func statusWantText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en la pantalla", want)
		}
	}
}

func statusWantNoText(t *testing.T, body string, unwanted ...string) {
	t.Helper()
	for _, text := range unwanted {
		if strings.Contains(body, text) {
			t.Errorf("la pantalla no debe contener %q", text)
		}
	}
}

// statusWantRow comprueba una fila de la tabla Estado / Historias: <th>etiqueta</th><td>cantidad</td>.
func statusWantRow(t *testing.T, body, label string, count int) {
	t.Helper()
	row := regexp.MustCompile(`(?s)<th[^>]*>\s*` + regexp.QuoteMeta(label) + `\s*</th>\s*<td[^>]*>\s*` + fmt.Sprint(count) + `\s*</td>`)
	if !row.MatchString(body) {
		t.Errorf("falta la fila %q con %d historias", label, count)
	}
}

// statusWantStatic comprueba que la pantalla es solo de lectura: sin scripts ni formularios.
func statusWantStatic(t *testing.T, body string) {
	t.Helper()
	statusWantNoText(t, body, "<script", "<form")
}

// --- T015: pantalla con datos ---

// Escenario 1: proyecto en curso con Sprint activo.
func TestHU12_ProyectoEnCurso(t *testing.T) {
	app := newStatusApp(t)
	alfa := seedStatusAlfa(t, app.db)

	resp, body := app.page(t, alfa.ProjectID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	statusWantText(t, body,
		"Estado: En curso", "Del 01/10/2026 al 30/11/2026",
		"Sprint 2 en curso", "Objetivo: Pagos online", "Del 19/10/2026 al 01/11/2026",
		"4 de 8 historias completadas (50,00 %)")
	statusWantRow(t, body, "Pendiente", 2)
	statusWantRow(t, body, "En Sprint", 1)
	statusWantRow(t, body, "En progreso", 1)
	statusWantRow(t, body, "Hecho", 4)
	statusWantStatic(t, body)
}

// Escenarios 2 a 5: el estado sale de las fechas y de "hoy" (28/10/2026), con los bordes incluidos.
func TestHU12_EstadoPorFechas(t *testing.T) {
	app := newStatusApp(t)
	states := []string{"No iniciado", "En curso", "Finalizado"}
	cases := []struct {
		name     string
		seed     func(*testing.T, *statusApp) statusFixture
		want     string
		progress string
	}{
		{"Beta empieza después", func(t *testing.T, a *statusApp) statusFixture { return seedStatusBeta(t, a.db) },
			"No iniciado", "0 de 0 historias completadas (0,00 %)"},
		{"Gamma ya terminó", func(t *testing.T, a *statusApp) statusFixture { return seedStatusGamma(t, a.db) },
			"Finalizado", "3 de 3 historias completadas (100,00 %)"},
		{"Delta empieza hoy", func(t *testing.T, a *statusApp) statusFixture { return seedStatusDelta(t, a.db) },
			"En curso", "0 de 0 historias completadas (0,00 %)"},
		{"Épsilon termina hoy", func(t *testing.T, a *statusApp) statusFixture { return seedStatusEpsilon(t, a.db) },
			"En curso", "0 de 0 historias completadas (0,00 %)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := tc.seed(t, app)

			resp, body := app.page(t, project.ProjectID)

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
			}
			statusWantText(t, body, "Estado: "+tc.want, tc.progress)
			for _, other := range states {
				if other != tc.want {
					statusWantNoText(t, body, "Estado: "+other)
				}
			}
			statusWantStatic(t, body)
		})
	}
}

// Escenario 6: sin historias, las cuatro cantidades son 0 y el avance es 0,00 %, sin error ni NaN.
func TestHU12_SinHistorias(t *testing.T) {
	app := newStatusApp(t)
	beta := seedStatusBeta(t, app.db)

	resp, body := app.page(t, beta.ProjectID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	for _, label := range []string{"Pendiente", "En Sprint", "En progreso", "Hecho"} {
		statusWantRow(t, body, label, 0)
	}
	statusWantText(t, body, "0 de 0 historias completadas (0,00 %)")
	statusWantNoText(t, body, "NaN", "Inf")
	statusWantStatic(t, body)
}

// Escenario 7: sin Sprint activo se avisa y el resto de la pantalla sigue igual.
func TestHU12_SinSprintEnCurso(t *testing.T) {
	app := newStatusApp(t)
	gamma := seedStatusGamma(t, app.db)

	resp, body := app.page(t, gamma.ProjectID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	statusWantText(t, body, "No hay un Sprint en curso", "Estado: Finalizado", "3 de 3 historias completadas (100,00 %)")
	statusWantRow(t, body, "Hecho", 3)
	statusWantNoText(t, body, "Objetivo:", "Sprint 1 en curso")
	statusWantStatic(t, body)
}

// Escenario 8: solo cuentan las historias y el Sprint del proyecto activo.
func TestHU12_OtroProyecto(t *testing.T) {
	app := newStatusApp(t)
	alfa := seedStatusAlfa(t, app.db)
	gamma := seedStatusGamma(t, app.db)

	_, bodyAlfa := app.page(t, alfa.ProjectID)
	_, bodyGamma := app.page(t, gamma.ProjectID)

	statusWantText(t, bodyAlfa, "4 de 8 historias completadas (50,00 %)", "Sprint 2 en curso", "Objetivo: Pagos online")
	statusWantNoText(t, bodyAlfa, "7 de 11", "3 de 3", gamma.Name)
	statusWantText(t, bodyGamma, "3 de 3 historias completadas (100,00 %)", "No hay un Sprint en curso")
	statusWantNoText(t, bodyGamma, "7 de 11", "4 de 8", "Pagos online", alfa.Name)
}

// --- T016: contexto, errores y bordes ---

// Escenario 9: sin proyecto activo válido se redirige a elegirlo.
func TestHU12_SinProyecto(t *testing.T) {
	app := newStatusApp(t)
	cases := []struct {
		name    string
		cookies []*http.Cookie
	}{
		{"sin selección", nil},
		{"cookie inválida", []*http.Cookie{{Name: CookieProject, Value: "abc", Path: "/"}}},
		{"proyecto inexistente", []*http.Cookie{idCookie(CookieProject, MaxID)}},
	}
	want := pathSelectProject + "?" + volverAQuery("/estado")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := app.client(false)
			if tc.cookies != nil {
				app.setCookies(t, c, tc.cookies...)
			}

			resp, body := app.get(t, c, "/estado")

			if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
				t.Errorf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, resp.Header.Get("Location"))
			}
			statusWantNoText(t, body, "Estado:", "historias completadas")
		})
	}
}

// Escenario 10: una falla del catálogo o del Tablero muestra la página de error, sin datos parciales
// ni detalles internos.
func TestHU12_ErrorAlConsultar(t *testing.T) {
	failCatalog := func(err error) statusOption {
		return withProjectStatusCatalog(func(ProjectStatusCatalog) ProjectStatusCatalog { return failingStatusCatalog{err: err} })
	}
	failBoard := func(err error) statusOption {
		return withProjectStatusBoard(func(BoardCatalog) BoardCatalog { return failingStatusBoard{err: err} })
	}
	unavailable := fmt.Errorf("leer el estado: %w", driver.ErrBadConn)
	cases := []struct {
		name string
		opt  statusOption
		want int
	}{
		{"falla del catálogo", failCatalog(errors.New(statusInternalDetail)), http.StatusInternalServerError},
		{"falla del Tablero", failBoard(errors.New(statusInternalDetail)), http.StatusInternalServerError},
		{"base no disponible en el catálogo", failCatalog(unavailable), http.StatusServiceUnavailable},
		{"base no disponible en el Tablero", failBoard(unavailable), http.StatusServiceUnavailable},
		{"proyecto que ya no existe", failCatalog(NotFound(errors.New("el proyecto no existe"))), http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newStatusApp(t, tc.opt)
			alfa := seedStatusAlfa(t, app.db)

			resp, body := app.page(t, alfa.ProjectID)

			if resp.StatusCode != tc.want {
				t.Errorf("se esperaba %d, se obtuvo %d", tc.want, resp.StatusCode)
			}
			statusWantText(t, body, LinkBackHome)
			statusWantNoText(t, body, "Estado:", "historias completadas", "Del 01/10/2026", "Pagos online")
			lower := strings.ToLower(body)
			for _, leak := range []string{"select ", "pq:", "sql:", ".go:", `c:\`, "internal", "connection refused"} {
				if strings.Contains(lower, leak) {
					t.Errorf("la página expone un detalle interno (contiene %q)", leak)
				}
			}
		})
	}
}

// POST (u otro método) en /estado no está permitido.
func TestHU12_PostNoPermitido(t *testing.T) {
	app := newStatusApp(t)
	alfa := seedStatusAlfa(t, app.db)
	c := app.open(t, alfa.ProjectID)

	resp, body := app.post(t, c, "/estado", url.Values{})

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("se esperaba 405, se obtuvo %d", resp.StatusCode)
	}
	statusWantText(t, body, LinkBackHome)
	statusWantNoText(t, body, "Estado:", "historias completadas")
}

// Los textos del proyecto y del Sprint se muestran escapados y la página no tiene scripts.
func TestHU12_TextosEscapados(t *testing.T) {
	app := newStatusApp(t)
	project := newStatusProject(t, app.db, "<b>Alfa & Co</b>", "2026-10-01", "2026-11-30")
	newStatusSprint(t, app.db, project.ProjectID, 1, "<script>x</script>", "2026-10-19", "2026-11-01", "Activo")

	_, body := app.page(t, project.ProjectID)

	statusWantText(t, body, "&lt;b&gt;Alfa &amp; Co&lt;/b&gt;", "&lt;script&gt;x&lt;/script&gt;")
	statusWantNoText(t, body, "<b>Alfa", "<script")
}

// SC-004: 1000 historias y un Sprint activo con 50 se muestran en menos de 500 ms.
func TestHU12_Rendimiento(t *testing.T) {
	app := newStatusApp(t)
	project := newStatusProject(t, app.db, "Rendimiento", "2026-10-01", "2026-11-30")
	old := newStatusSprint(t, app.db, project.ProjectID, 1, "Sprint anterior", "2026-10-05", "2026-10-18", "Finalizado")
	active := newStatusSprint(t, app.db, project.ProjectID, 2, "Pagos online", "2026-10-19", "2026-11-01", "Activo")
	// 250 por estado; el Sprint activo tiene 50. Solo importa la cantidad, no el reparto.
	addStatusStories(t, app.db, project.ProjectID, 0, catPending, 250)
	addStatusStories(t, app.db, project.ProjectID, active, catEnSprint, 50)
	addStatusStories(t, app.db, project.ProjectID, old, catEnSprint, 200)
	addStatusStories(t, app.db, project.ProjectID, old, catInProgress, 250)
	addStatusStories(t, app.db, project.ProjectID, old, catDone, 250)
	c := app.open(t, project.ProjectID)
	_, body := app.get(t, c, "/estado") // calienta la conexión y las plantillas
	statusWantText(t, body, "250 de 1000 historias completadas (25,00 %)")

	const runs = 5
	start := time.Now()
	for i := 0; i < runs; i++ {
		app.get(t, c, "/estado")
	}

	if avg := time.Since(start) / runs; avg >= 500*time.Millisecond {
		t.Errorf("SC-004: el promedio fue %v, debía ser menor a 500 ms", avg)
	}
}
