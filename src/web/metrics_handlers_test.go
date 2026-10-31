package web

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-25: pruebas de handler de "Métricas del Sprint" con el servidor real, PostgreSQL real, el
// servicio real de HU-07 y un navegador simulado con cookies. El proyecto activo se elige con el
// flujo de HU-20b y los Sprints se cierran con el servicio real de HU-15.

// metricsResponseLimit es el tiempo máximo de respuesta de la pantalla (SC-005).
const metricsResponseLimit = 500 * time.Millisecond

// metricsHandlerApp es el servidor de prueba con el contexto de HU-20b y la sección de HU-25.
type metricsHandlerApp struct {
	*ctxApp
}

func newMetricsHandlerApp(t *testing.T) *metricsHandlerApp {
	t.Helper()
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		if err := RegisterMetrics(server, NewPostgreSQLMetricsCatalog(db), NewHU07MetricsSource(db)); err != nil {
			t.Fatalf("no se pudo registrar la sección Métricas del Sprint: %v", err)
		}
	})
	return &metricsHandlerApp{ctxApp: app}
}

// open elige el proyecto con el flujo de HU-20b (volviendo a /metricas) y abre la ruta pedida.
// Devuelve la respuesta, el cuerpo y cuánto tardó la pantalla de métricas.
func (a *metricsHandlerApp) open(t *testing.T, projectID int64, path string) (*http.Response, string, time.Duration) {
	t.Helper()
	client := a.client(false)
	resp, _ := a.selectProject(t, client, id64(projectID), pathMetrics)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != pathMetrics {
		t.Fatalf("elegir el proyecto: se esperaba 303 a %s, se obtuvo %d a %q",
			pathMetrics, resp.StatusCode, resp.Header.Get("Location"))
	}
	started := time.Now()
	resp, body := a.get(t, client, path)
	return resp, body, time.Since(started)
}

// metricsSection devuelve el bloque de métricas del Sprint elegido: el contenido de
// <section id="metricas-sprint"> de metrics.html.
func metricsSection(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<section id="metricas-sprint"`)
	if start < 0 {
		t.Fatalf("la página no tiene el bloque de métricas del Sprint elegido")
	}
	end := strings.Index(body[start:], "</section>")
	if end < 0 {
		t.Fatalf("el bloque de métricas del Sprint elegido no está cerrado")
	}
	return body[start : start+end]
}

// assertMetricsPageOK verifica que la pantalla respondió 200 sin la página de error ni valores
// inválidos.
func assertMetricsPageOK(t *testing.T, resp *http.Response, body string) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metricas = %d; se esperaba 200", resp.StatusCode)
	}
	for _, invalid := range []string{MsgInternal, MsgUnavailable, "NaN", "Inf", "%!", "<no value>"} {
		if strings.Contains(body, invalid) {
			t.Errorf("la pantalla de métricas contiene %q", invalid)
		}
	}
}

func assertContainsAll(t *testing.T, where, text string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("%s debía contener %q", where, w)
		}
	}
}

func assertContainsNone(t *testing.T, where, text string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(text, u) {
			t.Errorf("%s no debía contener %q", where, u)
		}
	}
}

// T012 (US1, escenario 1; FR-003, FR-004, SC-001, SC-005).
func TestMetricsHandler_SprintFinalizadoMuestraVelocidadYDesviacionDelMasReciente(t *testing.T) {
	app := newMetricsHandlerApp(t)

	// Dado el proyecto A con un Sprint 1 anterior (13 Story Points, 10 estimadas y 5 reales) y el
	// Sprint 2, el más reciente, con 8 Story Points completados, 10 horas estimadas y 12 reales;
	// y el proyecto B con un Sprint finalizado propio.
	a := newMetricsProject(t, app.db, "handler A")
	s1 := a.activeSprint(1, "Sprint anterior", "2026-09-01", "2026-09-14")
	anterior := a.story(s1, "Historia del Sprint anterior", 13, sprint.Done, "10")
	a.effort(anterior, "5")
	a.close(s1)
	s2 := a.activeSprint(2, "Sprint con datos conocidos", "2026-09-15", "2026-09-28")
	h1 := a.story(s2, "Hecha 1", 5, sprint.Done, "4")
	h2 := a.story(s2, "Hecha 2", 3, sprint.Done, "6")
	devuelta := a.story(s2, "Devuelta al Backlog", 13, sprint.InProgress, "50")
	a.effort(h1, "5")
	a.effort(h2, "7")
	a.effort(devuelta, "20")
	a.close(s2)
	b := newMetricsProject(t, app.db, "handler B")
	sb := b.activeSprint(1, "Sprint privado de B", "2026-09-01", "2026-09-14")
	privada := b.story(sb, "Historia privada de B", 21, sprint.Done, "2")
	b.effort(privada, "9")
	b.close(sb)

	// Cuando abro "Métricas del Sprint" con el proyecto A activo, sin elegir Sprint.
	resp, body, elapsed := app.open(t, a.ProjectID, pathMetrics)

	// Entonces veo las métricas del Sprint 2 tal como las calcula HU-07, solo del proyecto A,
	// sin guardar nada y dentro del tiempo de SC-005.
	assertMetricsPageOK(t, resp, body)
	section := metricsSection(t, body)
	assertContainsAll(t, "el bloque de métricas", section,
		"Sprint 2", "Sprint con datos conocidos", "Del 15/09/2026 al 28/09/2026",
		"8 Story Points", "+20,00 %", MsgMetricsDeviationFormula)
	assertContainsNone(t, "el bloque de métricas", section, "13 Story Points", "-50,00 %")
	assertContainsNone(t, "la pantalla del proyecto A", body, "Sprint privado de B", "21 Story Points")
	if elapsed >= metricsResponseLimit {
		t.Errorf("GET /metricas tardó %v; se esperaba menos de %v", elapsed, metricsResponseLimit)
	}
	for _, id := range []int64{s1, s2, sb} {
		assertNoMetricsSnapshot(t, app.db, id)
	}
}

// T012 (SC-005): 20 Sprints finalizados y el más reciente con 49 historias.
func TestMetricsHandler_RespondeEnMenosDe500msCon20SprintsY49Historias(t *testing.T) {
	app := newMetricsHandlerApp(t)

	// Dado un proyecto con 19 Sprints finalizados y un Sprint 20 cerrado con 49 historias "Hecho"
	// de 1 Story Point, 1 hora estimada y 1 hora real cada una.
	p := newMetricsProject(t, app.db, "rendimiento")
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for n := 1; n <= 19; n++ {
		start := day.AddDate(0, 0, (n-1)*7)
		p.finishedSprint(n, fmt.Sprintf("Sprint histórico %d", n),
			start.Format("2006-01-02"), start.AddDate(0, 0, 6).Format("2006-01-02"))
	}
	last := p.activeSprint(20, "Sprint grande", "2026-06-01", "2026-06-14")
	for i := 1; i <= 49; i++ {
		id := p.story(last, fmt.Sprintf("Historia %d", i), 1, sprint.Done, "1")
		p.effort(id, "1")
	}
	p.close(last)

	// Cuando abro la pantalla.
	resp, body, elapsed := app.open(t, p.ProjectID, pathMetrics)

	// Entonces responde en menos de 500 ms con las métricas del Sprint 20.
	assertMetricsPageOK(t, resp, body)
	assertContainsAll(t, "el bloque de métricas", metricsSection(t, body),
		"Sprint 20", "49 Story Points", "0,00 %")
	if elapsed >= metricsResponseLimit {
		t.Errorf("GET /metricas tardó %v; se esperaba menos de %v", elapsed, metricsResponseLimit)
	}
}

// T013 (US1, escenario 2; FR-005, SC-003).
func TestMetricsHandler_SinHorasEstimadasMuestraNoDisponibleYLaVelocidad(t *testing.T) {
	app := newMetricsHandlerApp(t)

	// Dado un Sprint finalizado con una historia "Hecho" de 5 Story Points sin horas estimadas.
	p := newMetricsProject(t, app.db, "sin estimaciones")
	s := p.activeSprint(1, "Sprint sin estimaciones", "2026-10-01", "2026-10-14")
	h := p.story(s, "Hecha sin estimar", 5, sprint.Done, "")
	p.effort(h, "3")
	p.close(s)

	// Cuando abro sus métricas.
	resp, body, _ := app.open(t, p.ProjectID, pathMetrics)

	// Entonces veo "No disponible" en la desviación y la velocidad igual se muestra, sin error.
	assertMetricsPageOK(t, resp, body)
	assertContainsAll(t, "el bloque de métricas", metricsSection(t, body),
		"5 Story Points", hu07NoData(t))
	assertContainsNone(t, "el bloque de métricas", metricsSection(t, body), "0,00 %")
}

// T014 (US1, escenario 3; FR-006, SC-004).
func TestMetricsHandler_DatosIncompletosMuestranLaAlertaDeCalculoParcial(t *testing.T) {
	app := newMetricsHandlerApp(t)

	// Dado un Sprint finalizado con tres historias "Hecho" y solo una con horas estimadas y reales.
	p := newMetricsProject(t, app.db, "parcial")
	s := p.activeSprint(1, "Sprint parcial", "2026-10-01", "2026-10-14")
	completa := p.story(s, "Hecha con datos", 5, sprint.Done, "8")
	sinEstimar := p.story(s, "Hecha sin estimar", 3, sprint.Done, "")
	p.story(s, "Hecha sin horas reales", 2, sprint.Done, "4")
	p.effort(completa, "6")
	p.effort(sinEstimar, "2")
	p.close(s)

	// Cuando abro sus métricas.
	resp, body, _ := app.open(t, p.ProjectID, pathMetrics)

	// Entonces veo la alerta de HU-07 con 1 de 3 historias y las métricas que calcula.
	assertMetricsPageOK(t, resp, body)
	assertContainsAll(t, "el bloque de métricas", metricsSection(t, body),
		hu07PartialWarning(t, 1, 3), "10 Story Points", "-25,00 %")
}

// T015 (US1, escenario 4; FR-007, SC-003).
func TestMetricsHandler_SinHistoriasHechoMuestraVelocidadCeroYNoDisponible(t *testing.T) {
	app := newMetricsHandlerApp(t)

	// Dado un Sprint cerrado por HU-15 sin historias en "Hecho": las incompletas vuelven al Backlog.
	p := newMetricsProject(t, app.db, "velocidad cero")
	s := p.activeSprint(1, "Sprint sin terminar nada", "2026-10-01", "2026-10-14")
	enProgreso := p.story(s, "En progreso", 5, sprint.InProgress, "6")
	p.story(s, "En Sprint", 3, sprint.EnSprint, "2")
	p.effort(enProgreso, "4")
	p.close(s)

	// Cuando abro sus métricas.
	resp, body, _ := app.open(t, p.ProjectID, pathMetrics)

	// Entonces veo 0 Story Points y la desviación "No disponible", sin error.
	assertMetricsPageOK(t, resp, body)
	assertContainsAll(t, "el bloque de métricas", metricsSection(t, body),
		"Sprint 1", "0 Story Points", hu07NoData(t))
	assertContainsNone(t, "el bloque de métricas", metricsSection(t, body), "5 Story Points", "3 Story Points")
}
