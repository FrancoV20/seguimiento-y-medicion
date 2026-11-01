package web

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-08 (T021, T022): pantalla GET /dashboard con el catálogo, el Tablero (HU-14), la calidad de
// HU-10 y la fuente de métricas de HU-25 reales, sobre los datos de dashboard_fixtures_test.go.

const hu08Path = "/dashboard"

type hu08Deps struct {
	catalog DashboardCatalog
	board   BoardCatalog
	quality QualityReader
	source  MetricsSource
}

type hu08App struct {
	*ctxApp
	client *http.Client
	f      *dashboardFixture
}

func newHU08App(t *testing.T) *hu08App {
	return newHU08AppWith(t, nil)
}

// newHU08AppWith permite reemplazar alguna dependencia (por ejemplo, para simular una falla).
func newHU08AppWith(t *testing.T, override func(db *sql.DB, deps *hu08Deps)) *hu08App {
	t.Helper()
	app := newCtxApp(t, nil)
	deps := hu08Deps{
		catalog: NewPostgreSQLDashboardCatalog(app.db),
		board:   NewPostgreSQLBoardCatalog(app.db),
		quality: metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(app.db)),
		source:  NewHU07MetricsSource(app.db),
	}
	if override != nil {
		override(app.db, &deps)
	}
	if err := RegisterDashboard(app.srv, deps.catalog, deps.board, deps.quality, deps.source); err != nil {
		t.Fatalf("no se pudo registrar la sección Dashboard: %v", err)
	}
	// El fixture se arma después del contexto: su limpieza corre antes que la del contexto.
	f := newDashboardFixture(t, app.db)
	return &hu08App{ctxApp: app, client: app.client(false), f: f}
}

// open elige el proyecto con el flujo de HU-20b y abre el Dashboard.
func (a *hu08App) open(t *testing.T, projectID int64) (*http.Response, string) {
	t.Helper()
	resp, _ := a.selectProject(t, a.client, id64(projectID), hu08Path)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto %d: estado %d; se esperaba 303", projectID, resp.StatusCode)
	}
	return a.get(t, a.client, hu08Path)
}

// openOK abre el Dashboard y exige 200.
func (a *hu08App) openOK(t *testing.T, projectID int64) string {
	t.Helper()
	resp, body := a.open(t, projectID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s del proyecto %d: estado %d; se esperaba 200", hu08Path, projectID, resp.StatusCode)
	}
	return body
}

var (
	hu08TagPattern  = regexp.MustCompile(`<[^>]*>`)
	hu08RectHeight  = regexp.MustCompile(`<rect[^>]*\sheight="(\d+)"`)
	hu08TableRow    = regexp.MustCompile(`<th scope="row">([^<]*)</th>\s*<td>(\d+)</td>`)
	hu08SVGWithAria = regexp.MustCompile(`<svg role="img" aria-label="([^"]*)"`)
)

// hu08Text devuelve el texto visible del contenido principal, con los espacios normalizados.
func hu08Text(body string) string {
	if start := strings.Index(body, "<main"); start >= 0 {
		body = body[start:]
		if end := strings.Index(body, "</main>"); end >= 0 {
			body = body[:end]
		}
	}
	text := html.UnescapeString(hu08TagPattern.ReplaceAllString(body, " "))
	return strings.Join(strings.Fields(text), " ")
}

// hu08ChartHeights devuelve el alto de las barras del gráfico cuyo aria-label empieza con prefix.
func hu08ChartHeights(t *testing.T, body, prefix string) []int {
	t.Helper()
	start := strings.Index(body, `aria-label="`+prefix)
	if start < 0 {
		t.Fatalf("no se encontró el gráfico %q", prefix)
	}
	chart := body[start:]
	if end := strings.Index(chart, "</svg>"); end >= 0 {
		chart = chart[:end]
	}
	heights := []int{}
	for _, m := range hu08RectHeight.FindAllStringSubmatch(chart, -1) {
		h, _ := strconv.Atoi(m[1])
		heights = append(heights, h)
	}
	return heights
}

// hu08Table devuelve las filas (etiqueta y valor) de la tabla con ese título.
func hu08Table(t *testing.T, body, caption string) [][2]string {
	t.Helper()
	start := strings.Index(body, "<caption>"+caption+"</caption>")
	if start < 0 {
		t.Fatalf("no se encontró la tabla %q", caption)
	}
	table := body[start:]
	if end := strings.Index(table, "</table>"); end >= 0 {
		table = table[:end]
	}
	rows := [][2]string{}
	for _, m := range hu08TableRow.FindAllStringSubmatch(table, -1) {
		rows = append(rows, [2]string{html.UnescapeString(strings.TrimSpace(m[1])), m[2]})
	}
	return rows
}

func hu08Rows(pairs ...string) [][2]string {
	rows := [][2]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		rows = append(rows, [2]string{pairs[i], pairs[i+1]})
	}
	return rows
}

func hu08AssertText(t *testing.T, body string, want ...string) {
	t.Helper()
	text := hu08Text(body)
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("la pantalla debía mostrar %q: %s", w, text)
		}
	}
}

func hu08AssertNoText(t *testing.T, body string, unwanted ...string) {
	t.Helper()
	text := hu08Text(body)
	for _, u := range unwanted {
		if strings.Contains(text, u) {
			t.Errorf("la pantalla no debía mostrar %q: %s", u, text)
		}
	}
}

func hu08AssertStories(t *testing.T, body string, pending, inSprint, inProgress, done string) {
	t.Helper()
	want := hu08Rows("Pendiente", pending, "En Sprint", inSprint, "En progreso", inProgress, "Hecho", done)
	if got := hu08Table(t, body, "Historias por estado"); !reflect.DeepEqual(got, want) {
		t.Errorf("tabla de historias por estado = %v; se esperaba %v", got, want)
	}
}

// E1: proyecto recién creado.
func TestHU08_SinDatos(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.empty)

	text := hu08Text(body)
	if strings.Count(text, MsgDashboardNoData) != 1 {
		t.Errorf("debía verse una sola vez %q: %s", MsgDashboardNoData, text)
	}
	if strings.Contains(body, "<svg") || strings.Contains(body, "<table") {
		t.Error("un proyecto sin datos no debe mostrar gráficos ni tablas vacías")
	}
	hu08AssertNoText(t, body, "No hay un Sprint en curso", "Velocidad acumulada", "defectos abiertos")
}

// E2: historias por estado con gráfico y tabla.
func TestHU08_HistoriasPorEstado(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.alfa)

	hu08AssertStories(t, body, "2", "1", "2", "3")
	if got := hu08ChartHeights(t, body, "Historias por estado"); !reflect.DeepEqual(got, []int{80, 40, 80, 120}) {
		t.Errorf("altos de las barras de historias = %v; se esperaba [80 40 80 120]", got)
	}
}

// E3: Sprint en curso con sus cantidades y Story Points.
func TestHU08_SprintEnCurso(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.alfa)

	hu08AssertText(t, body, "Sprint 3 en curso", "En Sprint: 1", "En progreso: 2", "Hecho: 1",
		"11 Story Points planificados y 2 completados")
}

// E4: sin Sprint activo, el resto se muestra igual.
func TestHU08_SinSprintEnCurso(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.noSprint)

	hu08AssertText(t, body, MsgDashboardNoActiveSprint)
	hu08AssertStories(t, body, "2", "0", "0", "0")
	hu08AssertNoText(t, body, "Story Points planificados")
}

// E5: velocidad por Sprint, acumulada y promedio.
func TestHU08_Velocidad(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.alfa)

	if got, want := hu08Table(t, body, "Velocidad por Sprint"), hu08Rows("Sprint 1", "8", "Sprint 2", "13"); !reflect.DeepEqual(got, want) {
		t.Errorf("tabla de velocidad = %v; se esperaba %v", got, want)
	}
	if got := hu08ChartHeights(t, body, "Velocidad por Sprint"); !reflect.DeepEqual(got, []int{74, 120}) {
		t.Errorf("altos de las barras de velocidad = %v; se esperaba [74 120]", got)
	}
	hu08AssertText(t, body, "Velocidad acumulada: 21 Story Points", "Promedio: 10,50 Story Points por Sprint")
	hu08AssertNoText(t, body, MsgDashboardLastTenSprints)
}

// Con más de 10 Sprints finalizados, el gráfico muestra los últimos 10 y la acumulada usa todos.
func TestHU08_UltimosDiezSprints(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.twelve)

	rows := hu08Table(t, body, "Velocidad por Sprint")
	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, row[0])
	}
	want := []string{"Sprint 3", "Sprint 4", "Sprint 5", "Sprint 6", "Sprint 7", "Sprint 8", "Sprint 9", "Sprint 10", "Sprint 11", "Sprint 12"}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("Sprints del gráfico = %v; se esperaba %v", labels, want)
	}
	if got := hu08ChartHeights(t, body, "Velocidad por Sprint"); len(got) != 10 {
		t.Errorf("barras de velocidad = %d; se esperaban 10", len(got))
	}
	hu08AssertText(t, body, "Velocidad acumulada: 92 Story Points", "Promedio: 7,67 Story Points por Sprint",
		MsgDashboardLastTenSprints)
}

// E6: sin Sprints finalizados, velocidad y desviación muestran el mensaje; el resto se ve.
func TestHU08_SinSprintsFinalizados(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.onlyActive)

	text := hu08Text(body)
	if strings.Count(text, MsgDashboardNoData) != 1 {
		t.Errorf("el bloque de velocidad y desviación debía mostrar %q: %s", MsgDashboardNoData, text)
	}
	hu08AssertStories(t, body, "1", "1", "0", "0")
	hu08AssertText(t, body, "Sprint 1 en curso")
	hu08AssertNoText(t, body, "Velocidad acumulada", "Desviación de esfuerzo")
	if strings.Contains(body, `aria-label="Velocidad por Sprint`) {
		t.Error("sin Sprints finalizados no debe haber gráfico de velocidad")
	}
}

// E7: desviación del último Sprint finalizado.
func TestHU08_Desviacion(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.alfa)

	hu08AssertText(t, body, "Desviación de esfuerzo del Sprint 2: +20,00 %")
	hu08AssertNoText(t, body, "Cálculo parcial")
}

// Aviso de cálculo parcial de HU-07, tal cual.
func TestHU08_DesviacionParcial(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.partial)

	hu08AssertText(t, body, "Desviación de esfuerzo del Sprint 1: +20,00 %", "Cálculo parcial: basado en 1 de 2 historias")
}

// E8: desviación no disponible; la velocidad igual se muestra.
func TestHU08_DesviacionNoDisponible(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.noHours)

	hu08AssertText(t, body, "Desviación de esfuerzo del Sprint 1: No disponible", "Velocidad acumulada: 5 Story Points",
		"Cálculo parcial: basado en 0 de 1 historias")
}

// E9: defectos abiertos.
func TestHU08_DefectosAbiertos(t *testing.T) {
	app := newHU08App(t)

	hu08AssertText(t, app.openOK(t, app.f.alfa), "3 defectos abiertos")
	hu08AssertText(t, app.openOK(t, app.f.beta), "1 defecto abierto")
	hu08AssertText(t, app.openOK(t, app.f.noSprint), "No hay defectos abiertos")
}

// E10: solo se ven los datos del proyecto activo.
func TestHU08_OtroProyecto(t *testing.T) {
	app := newHU08App(t)

	beta := app.openOK(t, app.f.beta)
	hu08AssertStories(t, beta, "1", "0", "0", "1")
	if got, want := hu08Table(t, beta, "Velocidad por Sprint"), hu08Rows("Sprint 1", "5"); !reflect.DeepEqual(got, want) {
		t.Errorf("velocidad de Beta = %v; se esperaba %v", got, want)
	}
	hu08AssertText(t, beta, "1 defecto abierto", MsgDashboardNoActiveSprint)

	alfa := app.openOK(t, app.f.alfa)
	hu08AssertStories(t, alfa, "2", "1", "2", "3")
	hu08AssertText(t, alfa, "3 defectos abiertos", "Velocidad acumulada: 21 Story Points")
}

// Un Sprint finalizado sin historias: barras en 0 y tabla en ceros.
func TestHU08_SprintSinHistorias(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.noStories)

	hu08AssertStories(t, body, "0", "0", "0", "0")
	if got := hu08ChartHeights(t, body, "Historias por estado"); !reflect.DeepEqual(got, []int{0, 0, 0, 0}) {
		t.Errorf("altos de las barras de historias = %v; se esperaba todo en 0", got)
	}
	hu08AssertText(t, body, "Velocidad acumulada: 0 Story Points", "Promedio: 0,00 Story Points por Sprint",
		"Desviación de esfuerzo del Sprint 1: No disponible")
}

// E11: los gráficos no usan JavaScript y cada uno tiene su tabla con los mismos valores.
func TestHU08_GraficosSinJavaScript(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.alfa)

	if strings.Contains(body, "<script") {
		t.Error("el Dashboard no debe usar JavaScript")
	}
	labels := []string{}
	for _, m := range hu08SVGWithAria.FindAllStringSubmatch(body, -1) {
		labels = append(labels, html.UnescapeString(m[1]))
	}
	want := []string{
		"Historias por estado: Pendiente 2, En Sprint 1, En progreso 2, Hecho 3",
		"Velocidad por Sprint: Sprint 1, 8 Story Points; Sprint 2, 13 Story Points",
	}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("gráficos accesibles = %v; se esperaba %v", labels, want)
	}
	hu08AssertStories(t, body, "2", "1", "2", "3")
	if got := hu08Table(t, body, "Velocidad por Sprint"); !reflect.DeepEqual(got, hu08Rows("Sprint 1", "8", "Sprint 2", "13")) {
		t.Errorf("la tabla de velocidad no coincide con el gráfico: %v", got)
	}
}

// Cada barra de velocidad coincide con HU-10 (PointsCompleted) y con HU-07 (Velocity).
func TestHU08_CoherenciaConHU07(t *testing.T) {
	app := newHU08App(t)
	body := app.openOK(t, app.f.alfa)
	rows := hu08Table(t, body, "Velocidad por Sprint")

	quality := metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(app.db))
	hu07 := metrics.NewService(metrics.NewPostgreSQLMetricsStore(app.db))
	for i, sprintID := range []int64{app.f.alfaSprint1, app.f.alfaSprint2} {
		q, err := quality.SprintQuality(context.Background(), app.f.alfa, sprintID)
		if err != nil {
			t.Fatalf("HU-10 del Sprint %d: %v", i+1, err)
		}
		m, err := hu07.CalculateMetrics(sprintID)
		if err != nil {
			t.Fatalf("HU-07 del Sprint %d: %v", i+1, err)
		}
		shown, _ := strconv.Atoi(rows[i][1])
		if shown != q.PointsCompleted || shown != m.Velocity {
			t.Errorf("Sprint %d: Dashboard %d, HU-10 %d, HU-07 %d; se esperaban iguales", i+1, shown, q.PointsCompleted, m.Velocity)
		}
	}
}

// Abrir el Dashboard no escribe en sprint_metrics (usa la fuente de solo lectura de HU-25).
func TestHU08_NoEscribeSprintMetrics(t *testing.T) {
	app := newHU08App(t)
	app.openOK(t, app.f.alfa)

	var rows int
	if err := app.db.QueryRow(`SELECT count(*) FROM sprint_metrics WHERE sprint_id IN ($1, $2)`,
		app.f.alfaSprint1, app.f.alfaSprint2).Scan(&rows); err != nil {
		t.Fatalf("no se pudo leer sprint_metrics: %v", err)
	}
	if rows != 0 {
		t.Errorf("el Dashboard escribió %d filas en sprint_metrics; no debe escribir nada", rows)
	}
}

// --- T022 [E12]: errores y bordes ---

type hu08FailingCatalog struct{ err error }

func (c hu08FailingCatalog) DashboardSummary(context.Context, int64) (DashboardSummary, error) {
	return DashboardSummary{}, c.err
}

type hu08FailingBoard struct{ err error }

func (b hu08FailingBoard) ActiveBoard(context.Context, int64) (BoardView, bool, error) {
	return BoardView{}, false, b.err
}

type hu08FailingQuality struct{ err error }

func (q hu08FailingQuality) SprintQuality(context.Context, int64, int64) (metrics.SprintQuality, error) {
	return metrics.SprintQuality{}, q.err
}

type hu08FailingSource struct{ err error }

func (s hu08FailingSource) Metrics(context.Context, int64, int64) (SprintMetrics, error) {
	return SprintMetrics{}, s.err
}

const hu08InternalError = `pq: relation "dashboard_internal" failed at C:\internal\dashboard.go:42`

// E12: cualquier falla al leer muestra la página de error, sin gráficos ni números parciales.
func TestHU08_ErrorAlLeer(t *testing.T) {
	internal := errors.New(hu08InternalError)
	cases := []struct {
		name       string
		override   func(*sql.DB, *hu08Deps)
		wantStatus int
	}{
		{name: "catálogo", override: func(_ *sql.DB, d *hu08Deps) { d.catalog = hu08FailingCatalog{internal} }, wantStatus: http.StatusInternalServerError},
		{name: "Tablero", override: func(_ *sql.DB, d *hu08Deps) { d.board = hu08FailingBoard{internal} }, wantStatus: http.StatusInternalServerError},
		{name: "calidad de HU-10", override: func(_ *sql.DB, d *hu08Deps) { d.quality = hu08FailingQuality{internal} }, wantStatus: http.StatusInternalServerError},
		{name: "desviación de HU-07", override: func(_ *sql.DB, d *hu08Deps) { d.source = hu08FailingSource{internal} }, wantStatus: http.StatusInternalServerError},
		{name: "validación de cierre de HU-07", override: func(_ *sql.DB, d *hu08Deps) {
			d.source = hu08FailingSource{fmt.Errorf("métricas de HU-07: %w", metrics.ClosureValidationError{Reasons: []string{"Historia 1: Story Points inválidos"}})}
		}, wantStatus: http.StatusInternalServerError},
		{name: "base no disponible", override: func(_ *sql.DB, d *hu08Deps) {
			d.catalog = hu08FailingCatalog{fmt.Errorf("dashboard summary: %w", driver.ErrBadConn)}
		}, wantStatus: http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newHU08AppWith(t, tc.override)
			resp, body := app.open(t, app.f.alfa)

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("estado = %d; se esperaba %d", resp.StatusCode, tc.wantStatus)
			}
			if !strings.Contains(body, TitleInternal) && !strings.Contains(body, TitleUnavailable) {
				t.Errorf("debía verse la página de error de la aplicación")
			}
			if strings.Contains(body, "<svg") {
				t.Error("la página de error no debe mostrar gráficos")
			}
			hu08AssertNoText(t, body, "Story Points", "defectos abiertos", "defecto abierto", "en curso", "Desviación de esfuerzo")
			for _, detail := range []string{"pq:", "dashboard_internal", `C:\internal`, "Story Points inválidos"} {
				if strings.Contains(body, detail) {
					t.Errorf("la respuesta expuso un detalle interno: %q", detail)
				}
			}
		})
	}
}

// Sin proyecto activo, el Dashboard lleva a elegir uno y vuelve después.
func TestHU08_SinProyecto(t *testing.T) {
	app := newHU08App(t)
	resp, _ := app.get(t, app.client, hu08Path)

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/contexto/proyecto?volver_a="+url.QueryEscape(hu08Path) {
		t.Errorf("sin proyecto = %d a %q; se esperaba 303 a /contexto/proyecto?volver_a=%%2Fdashboard",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

// El Dashboard es de solo lectura: POST no está permitido.
func TestHU08_PostNoPermitido(t *testing.T) {
	app := newHU08App(t)
	app.openOK(t, app.f.alfa)

	resp, _ := app.post(t, app.client, hu08Path, url.Values{})
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST %s = %d; se esperaba 405", hu08Path, resp.StatusCode)
	}
}

// Con 20 Sprints finalizados, 200 historias y 100 defectos, la pantalla responde en menos de 500 ms.
func TestHU08_Rendimiento(t *testing.T) {
	app := newHU08App(t)
	f := app.f
	project := f.project(t, "Rendimiento")
	var firstStory int64
	for number := 1; number <= 20; number++ {
		sprintID := f.sprint(t, project, number, "Finalizado")
		for i := 0; i < 5; i++ {
			id := f.closedStory(t, project, sprintID, fmt.Sprintf("R%d-H%d", number, i), 2, "Hecho", "")
			if firstStory == 0 {
				firstStory = id
			}
			f.closedStory(t, project, sprintID, fmt.Sprintf("R%d-P%d", number, i), 2, "En progreso", "")
		}
	}
	firstSprint := f.insertID(t, `SELECT id FROM sprints WHERE project_id = $1 AND number = 1`, project)
	for i := 0; i < 100; i++ {
		resolution := int64(0)
		if i >= 60 {
			resolution = firstSprint
		}
		f.defect(t, project, firstStory, firstSprint, resolution)
	}
	resp, _ := app.selectProject(t, app.client, id64(project), hu08Path)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: estado %d", resp.StatusCode)
	}

	start := time.Now()
	resp, body := app.get(t, app.client, hu08Path)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("estado = %d; se esperaba 200", resp.StatusCode)
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("el Dashboard tardó %v; se esperaba menos de 500 ms", elapsed)
	}
	rows := hu08Table(t, body, "Velocidad por Sprint")
	if len(rows) != 10 || rows[0][0] != "Sprint 11" || rows[9][0] != "Sprint 20" {
		t.Errorf("barras de velocidad = %v; se esperaban los Sprints 11 a 20", rows)
	}
	hu08AssertText(t, body, "Velocidad acumulada: 200 Story Points", "Promedio: 10,00 Story Points por Sprint", "60 defectos abiertos")
}
