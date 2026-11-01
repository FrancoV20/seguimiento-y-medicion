package web

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/pdf"
	"github.com/FrancoV20/seguimiento-y-medicion/src/report"
)

// HU-09, T028 y T029: "Reportes" con el servidor, el contexto de HU-20b, el catálogo, report.Service
// (HU-07 y HU-10 reales, reloj fijo) y PostgreSQL reales, y los datos de report_fixtures_test.go.

var reportTestNow = time.Date(2026, 10, 28, 10, 30, 0, 0, time.UTC)

type reportConfig struct {
	catalog   func(ReportCatalog) ReportCatalog
	generator func(ReportGenerator) ReportGenerator
}

type reportOption func(*reportConfig)

func withReportCatalog(wrap func(ReportCatalog) ReportCatalog) reportOption {
	return func(c *reportConfig) { c.catalog = wrap }
}

func withReportGenerator(wrap func(ReportGenerator) ReportGenerator) reportOption {
	return func(c *reportConfig) { c.generator = wrap }
}

type reportApp struct {
	*ctxApp
	fx *reportWebFixture
}

// newReportApp arma el servidor con el contexto de trabajo y "Reportes" registrados.
func newReportApp(t *testing.T, opts ...reportOption) *reportApp {
	t.Helper()
	var cfg reportConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	db := openContextTestDB(t)
	var catalog ReportCatalog = NewPostgreSQLReportCatalog(db)
	if cfg.catalog != nil {
		catalog = cfg.catalog(catalog)
	}
	var generator ReportGenerator = report.NewService(report.NewPostgreSQLStore(db),
		metrics.NewService(metrics.NewPostgreSQLMetricsStore(db)),
		metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db)),
		func() time.Time { return reportTestNow })
	if cfg.generator != nil {
		generator = cfg.generator(generator)
	}
	app := newCtxApp(t, nil, func(s *Server) {
		if err := RegisterReports(s, catalog, generator); err != nil {
			t.Fatalf("no se pudo registrar Reportes: %v", err)
		}
	})
	return &reportApp{ctxApp: app, fx: newReportWebFixture(t, db)}
}

// open devuelve un navegador (que no sigue redirecciones) con el proyecto ya elegido.
func (a *reportApp) open(t *testing.T, project string) *http.Client {
	t.Helper()
	c := a.client(false)
	if resp, _ := a.selectProject(t, c, id64(a.fx.projects[project]), "/reportes"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto %s: se obtuvo %d", project, resp.StatusCode)
	}
	return c
}

func (a *reportApp) page(t *testing.T, project, path string) (*http.Response, string) {
	t.Helper()
	return a.get(t, a.open(t, project), path)
}

// pdfText exige que la respuesta sea un PDF descargable y devuelve su texto.
func reportPDFText(t *testing.T, resp *http.Response, body, fileName string) string {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d:\n%s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, se esperaba application/pdf", ct)
	}
	if cd, want := resp.Header.Get("Content-Disposition"), `attachment; filename="`+fileName+`"`; cd != want {
		t.Errorf("Content-Disposition = %q, se esperaba %q", cd, want)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, se esperaba no-store", cc)
	}
	if cl := resp.Header.Get("Content-Length"); cl != strconv.Itoa(len(body)) {
		t.Errorf("Content-Length = %q, el cuerpo tiene %d bytes", cl, len(body))
	}
	if !strings.HasPrefix(body, "%PDF-1.4") {
		t.Fatalf("el cuerpo no es un PDF: %.40q", body)
	}
	text, err := pdf.ExtractText([]byte(body))
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	return text
}

func reportWantText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q", want)
		}
	}
}

func reportWantNoText(t *testing.T, body string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(body, u) {
			t.Errorf("no debía aparecer %q", u)
		}
	}
}

func reportWantInOrder(t *testing.T, body string, wants ...string) {
	t.Helper()
	pos := 0
	for _, want := range wants {
		i := strings.Index(body[pos:], want)
		if i < 0 {
			t.Errorf("falta %q (o aparece fuera de orden)", want)
			continue
		}
		pos += i + len(want)
	}
}

// reportWantNoPDF exige una página HTML sin bytes del PDF ni encabezados de descarga.
func reportWantNoPDF(t *testing.T, resp *http.Response, body string) {
	t.Helper()
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "application/pdf") {
		t.Errorf("no debía responder un PDF (Content-Type %q)", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		t.Errorf("no debía haber Content-Disposition: %q", cd)
	}
	if strings.Contains(body, "%PDF") {
		t.Error("el cuerpo no debía contener bytes de un PDF")
	}
}

// --- T028 ---

// Escenario 11 y FR-002, FR-023.
func TestHU09_ListaDeSprints(t *testing.T) {
	app := newReportApp(t)

	resp, body := app.page(t, "Alfa", "/reportes")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	reportWantInOrder(t, body,
		"Reportes",
		`href="/reportes/proyecto/pdf"`, "Exportar reporte del proyecto",
		"Sprint 3", "Exportar e informar", "En curso", fmt.Sprintf(`href="%s"`, app.fx.sprintPath("Alfa 3")), "Exportar PDF",
		"Sprint 2", "Gestionar perfiles y reportes", "Finalizado", fmt.Sprintf(`href="%s"`, app.fx.sprintPath("Alfa 2")), "Exportar PDF",
		"Sprint 1", "Dar de alta usuarios", "Finalizado", fmt.Sprintf(`href="%s"`, app.fx.sprintPath("Alfa 1")), "Exportar PDF",
	)
	reportWantNoText(t, body, "<script", "<form", MsgReportNoSprints, "Probar otro proyecto")

	resp, body = app.page(t, "Beta", "/reportes")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("«Beta»: se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	reportWantText(t, body, "Aún no hay Sprints para exportar", `href="/reportes/proyecto/pdf"`, "Exportar reporte del proyecto")
	reportWantNoText(t, body, "<script", "<form", "Exportar PDF")
}

// Escenario 1.
func TestHU09_ExportarFinalizado(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Alfa", app.fx.sprintPath("Alfa 2"))
	reportPDFText(t, resp, body, "reporte-sprint-2.pdf")
}

// Escenarios 1 a 6.
func TestHU09_ContenidoDelPDF(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Alfa", app.fx.sprintPath("Alfa 2"))
	text := reportPDFText(t, resp, body, "reporte-sprint-2.pdf")
	reportWantInOrder(t, text,
		"Proyecto: Alfa", "Sprint 2", "Objetivo: Gestionar perfiles y reportes",
		"Fechas: 01/09/2026 al 14/09/2026", "Estado: Finalizado", "Generado: 28/10/2026 10:30",
		"Login Hecho 5 10 11", "Tablero Hecho 3 6 6", "Perfil Hecho 2 Sin estimar 4",
		"Reportes En progreso – volvió al Backlog 8 12 3",
		"3 de 4 historias completadas (75,00 %)",
		"18 Story Points planificados y 10 completados",
		"28 horas estimadas en total (1 historia sin horas estimadas)",
		"24 horas registradas en total", "Ana: 17 h", "Beto: 7 h",
		"Velocidad: 10 Story Points", "Desviación de esfuerzo: +6,25 %",
		"Cálculo parcial: basado en 2 de 3 historias",
		"El login no valida la contraseña Alta Resuelto Login Sprint 2",
		"La foto de perfil no se guarda Media Abierto Perfil —",
		"2 defectos detectados y 1 resuelto en este Sprint",
		"Página 1 de 1",
	)
}

// Escenario 7.
func TestHU09_SinDefectos(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Alfa", app.fx.sprintPath("Alfa 1"))
	text := reportPDFText(t, resp, body, "reporte-sprint-1.pdf")
	reportWantText(t, text, "No se registraron defectos en este Sprint", "Desviación de esfuerzo: 0,00 %")
	reportWantNoText(t, text, "Cálculo parcial")
}

// Caso límite: defectos resueltos en el Sprint que se detectaron antes.
func TestHU09_ResueltosDetectadosAntes(t *testing.T) {
	app := newReportApp(t)
	c := app.open(t, "Gamma")

	resp, body := app.get(t, c, app.fx.sprintPath("Gamma 2"))
	text := reportPDFText(t, resp, body, "reporte-sprint-2.pdf")
	reportWantText(t, text, "0 defectos detectados y 1 resuelto en este Sprint", "0 horas registradas en total")
	reportWantNoText(t, text, "Descripción", "La migración pierde acentos", " h Ana", "Ana:")

	resp, body = app.get(t, c, app.fx.sprintPath("Gamma 1"))
	text = reportPDFText(t, resp, body, "reporte-sprint-1.pdf")
	reportWantText(t, text, "La migración pierde acentos Baja Resuelto G1 Sprint 2",
		"1 defecto detectado y 0 resueltos en este Sprint", "0 horas registradas en total")
	reportWantNoText(t, text, "Ana:")
}

// Caso límite: Sprint finalizado sin registro de cierre.
func TestHU09_SprintSinRegistroDeCierre(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Gamma", app.fx.sprintPath("Gamma 3"))
	text := reportPDFText(t, resp, body, "reporte-sprint-3.pdf")
	reportWantText(t, text, "No hay historias registradas en este Sprint",
		"0 de 0 historias completadas (No disponible)", "0 Story Points planificados y 0 completados")
}

// Escenarios 8 y 10.
func TestHU09_AdvertenciaSprintEnCurso(t *testing.T) {
	app := newReportApp(t)
	c := app.open(t, "Alfa")
	path := app.fx.sprintPath("Alfa 3")
	for _, query := range []string{"", "?confirmar=SI", "?confirmar=1"} {
		resp, body := app.get(t, c, path+query)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: se esperaba 200, se obtuvo %d", query, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s: se esperaba una página HTML, Content-Type %q", query, ct)
		}
		reportWantNoPDF(t, resp, body)
		reportWantText(t, body,
			"El Sprint 3 sigue en curso: el reporte puede estar incompleto",
			fmt.Sprintf(`href="%s?confirmar=si"`, path), "Generar igual",
			`href="/reportes"`, "Cancelar",
		)
		reportWantNoText(t, body, "<script", "<form")
	}
	if n := app.fx.metricsRows(t, "Alfa"); n != 0 {
		t.Errorf("la advertencia no debía escribir en sprint_metrics; hay %d filas", n)
	}
}

// Escenario 9.
func TestHU09_GenerarIgual(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Alfa", app.fx.sprintPath("Alfa 3")+"?confirmar=si")
	text := reportPDFText(t, resp, body, "reporte-sprint-3.pdf")
	reportWantInOrder(t, text,
		"Reporte parcial: Sprint en curso", "Estado: En curso",
		"Exportar En progreso 5 6 2", "Ayuda En Sprint Sin estimar Sin estimar 0",
		"Historias completadas: Se calculan al cerrar el Sprint",
		"Story Points planificados y completados: Se calculan al cerrar el Sprint",
		"6 horas estimadas en total (1 historia sin horas estimadas)",
		"Ana: 2 h",
		"Velocidad y desviación: Se calculan al cerrar el Sprint",
		"El PDF sale en blanco Crítica Abierto Exportar —",
		"1 defecto detectado hasta el momento",
	)
	reportWantNoText(t, text, "Velocidad:")
}

// countingReportCatalog cuenta las llamadas al catálogo.
type countingReportCatalog struct {
	ReportCatalog
	calls int
}

func (c *countingReportCatalog) ReportSprint(ctx context.Context, projectID, sprintID int64) (ReportSprintOption, bool, error) {
	c.calls++
	return c.ReportCatalog.ReportSprint(ctx, projectID, sprintID)
}

// Escenario 12.
func TestHU09_SprintInexistenteOAjeno(t *testing.T) {
	counting := &countingReportCatalog{}
	app := newReportApp(t, withReportCatalog(func(c ReportCatalog) ReportCatalog {
		counting.ReportCatalog = c
		return counting
	}))
	c := app.open(t, "Alfa")

	for _, id := range []string{"abc", "0", "-1", "007", "2147483648"} {
		resp, body := app.get(t, c, "/reportes/sprints/"+id+"/pdf")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("id %q: se esperaba 404, se obtuvo %d", id, resp.StatusCode)
		}
		reportWantNoPDF(t, resp, body)
		reportWantText(t, body, LinkBackHome)
	}
	if counting.calls != 0 {
		t.Errorf("un id inválido no debía consultar el catálogo (%d llamadas)", counting.calls)
	}

	for _, path := range []string{"/reportes/sprints/2147483647/pdf", app.fx.sprintPath("Delta 1")} {
		resp, body := app.get(t, c, path)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: se esperaba 404, se obtuvo %d", path, resp.StatusCode)
		}
		reportWantNoPDF(t, resp, body)
	}
}

// --- T029 ---

type failingReportCatalog struct {
	ReportCatalog
	err error
}

func (c failingReportCatalog) ReportSprints(context.Context, int64) ([]ReportSprintOption, error) {
	return nil, c.err
}

func (c failingReportCatalog) ReportSprint(context.Context, int64, int64) (ReportSprintOption, bool, error) {
	return ReportSprintOption{}, false, c.err
}

type failingReportGenerator struct {
	ReportGenerator
	err error
}

func (g failingReportGenerator) SprintPDF(context.Context, int64, int64) ([]byte, error) {
	return nil, g.err
}

// Escenario 13.
func TestHU09_ErrorAlGenerar(t *testing.T) {
	cause := errors.New("detalle interno de prueba")

	t.Run("catálogo que falla", func(t *testing.T) {
		app := newReportApp(t, withReportCatalog(func(c ReportCatalog) ReportCatalog {
			return failingReportCatalog{ReportCatalog: c, err: cause}
		}))
		c := app.open(t, "Alfa")
		for _, path := range []string{"/reportes", app.fx.sprintPath("Alfa 2")} {
			resp, body := app.get(t, c, path)
			if resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("%s: se esperaba 500, se obtuvo %d", path, resp.StatusCode)
			}
			reportWantNoPDF(t, resp, body)
			reportWantNoText(t, body, cause.Error())
		}
	})

	cases := []struct {
		name string
		err  error
		want int
	}{
		{"generador que falla", cause, http.StatusInternalServerError},
		{"datos incoherentes", fmt.Errorf("build: %w", report.ErrInconsistentReport), http.StatusInternalServerError},
		{"cierre inválido de HU-07", fmt.Errorf("sprint metrics 2: %w",
			metrics.ClosureValidationError{Reasons: []string{"historia sin Story Points"}}), http.StatusInternalServerError},
		{"base de datos caída", fmt.Errorf("load sprint report 2: %w", driver.ErrBadConn), http.StatusServiceUnavailable},
		{"Sprint borrado mientras se arma", report.ErrSprintNotFound, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newReportApp(t, withReportGenerator(func(g ReportGenerator) ReportGenerator {
				return failingReportGenerator{ReportGenerator: g, err: tc.err}
			}))
			resp, body := app.page(t, "Alfa", app.fx.sprintPath("Alfa 2"))
			if resp.StatusCode != tc.want {
				t.Errorf("se esperaba %d, se obtuvo %d", tc.want, resp.StatusCode)
			}
			reportWantNoPDF(t, resp, body)
			reportWantNoText(t, body, cause.Error(), "historia sin Story Points", "driver")
			reportWantText(t, body, LinkBackHome)
		})
	}
}

// FR-020, FR-001 y research.md §14.
func TestHU09_SinProyectoYMetodo(t *testing.T) {
	app := newReportApp(t)
	download := app.fx.sprintPath("Alfa 2")

	for path, want := range map[string]string{
		"/reportes": "/contexto/proyecto?volver_a=%2Freportes",
		download:    "/contexto/proyecto?volver_a=" + url.QueryEscape(download),
	} {
		resp, _ := app.get(t, app.client(false), path)
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("%s sin proyecto: se esperaba 303, se obtuvo %d", path, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != want {
			t.Errorf("%s sin proyecto: Location %q, se esperaba %q", path, loc, want)
		}
	}

	for _, path := range []string{"/reportes", download} {
		resp, body := app.doWithHeaders(t, http.MethodPost, path, url.Values{}, map[string]string{"Origin": app.ts.URL})
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: se esperaba 405, se obtuvo %d", path, resp.StatusCode)
		}
		reportWantText(t, body, LinkBackHome)
	}

	c := app.open(t, "Alfa")
	getResp, getBody := app.get(t, c, download)
	req, err := http.NewRequest(http.MethodHead, app.ts.URL+download, nil)
	if err != nil {
		t.Fatal(err)
	}
	headResp, err := c.Do(req)
	if err != nil {
		t.Fatalf("HEAD %s: %v", download, err)
	}
	headBody, _ := io.ReadAll(headResp.Body)
	headResp.Body.Close()
	if headResp.StatusCode != http.StatusOK || len(headBody) != 0 {
		t.Errorf("HEAD: estado %d y %d bytes de cuerpo; se esperaba 200 sin cuerpo", headResp.StatusCode, len(headBody))
	}
	for _, h := range []string{"Content-Type", "Content-Disposition", "Cache-Control", "Content-Length"} {
		if got, want := headResp.Header.Get(h), getResp.Header.Get(h); got != want {
			t.Errorf("HEAD %s = %q, GET = %q", h, got, want)
		}
	}
	if getResp.Header.Get("Content-Length") != strconv.Itoa(len(getBody)) {
		t.Error("el Content-Length del GET no coincide con el cuerpo")
	}
}

// SC-005 y research.md §16.
func TestHU09_Rendimiento(t *testing.T) {
	app := newReportApp(t)
	c := app.open(t, "Largo")

	start := time.Now()
	resp, body := app.get(t, c, app.fx.sprintPath("Largo 1"))
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("el PDF del Sprint 1 de «Largo» tardó %v; se esperaba menos de 1 s", elapsed)
	}
	text := reportPDFText(t, resp, body, "reporte-sprint-1.pdf")
	reportWantText(t, text,
		"90 de 120 historias completadas (75,00 %)",
		"270 Story Points planificados y 180 completados",
		"480 horas estimadas en total",
		"360 horas registradas en total",
		"Ana: 360 h",
		"Desviación de esfuerzo: 0,00 %",
		"50 defectos detectados y 20 resueltos en este Sprint",
		largoLongTitle,
	)
	reportWantNoText(t, text, "Cálculo parcial")
	wantAllPagesNumbered(t, text)
}

// wantAllPagesNumbered exige más de una página y "Página X de Y" en todas.
func wantAllPagesNumbered(t *testing.T, text string) {
	t.Helper()
	total := strings.Count(text, "Página ")
	if total < 2 {
		t.Fatalf("se esperaba más de una página, se encontraron %d pies", total)
	}
	for page := 1; page <= total; page++ {
		if want := fmt.Sprintf("Página %d de %d", page, total); !strings.Contains(text, want) {
			t.Errorf("falta %q", want)
		}
	}
}

// T034: con un argumento nil no se registra nada.
func TestRegisterReports_ArgumentosNil(t *testing.T) {
	srv := NewServer(nil)
	catalog := NewPostgreSQLReportCatalog(nil)
	generator := report.NewService(nil, nil, nil, nil)
	for name, register := range map[string]func() error{
		"servidor":  func() error { return RegisterReports(nil, catalog, generator) },
		"catálogo":  func() error { return RegisterReports(srv, nil, generator) },
		"generador": func() error { return RegisterReports(srv, catalog, nil) },
	} {
		if err := register(); err == nil {
			t.Errorf("sin %s se esperaba un error", name)
		}
	}
	if srv.registry.Has(pathReports) {
		t.Error("no se debía registrar la sección")
	}
}
