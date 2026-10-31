package web

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
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
	return newMetricsHandlerAppWith(t, nil, nil)
}

// newMetricsHandlerAppWith permite envolver el catálogo o la fuente reales (por ejemplo, para
// contar llamadas o simular una falla); nil deja el real.
func newMetricsHandlerAppWith(
	t *testing.T,
	wrapCatalog func(MetricsCatalog) MetricsCatalog,
	wrapSource func(MetricsSource) MetricsSource,
) *metricsHandlerApp {
	t.Helper()
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		var catalog MetricsCatalog = NewPostgreSQLMetricsCatalog(db)
		source := NewHU07MetricsSource(db)
		if wrapCatalog != nil {
			catalog = wrapCatalog(catalog)
		}
		if wrapSource != nil {
			source = wrapSource(source)
		}
		if err := RegisterMetrics(server, catalog, source); err != nil {
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

// metricsSection devuelve el texto visible del bloque de métricas del Sprint elegido: el contenido
// de <section id="metricas-sprint"> de metrics.html, con las entidades HTML resueltas (html/template
// escribe el "+" como &#43;; el navegador muestra "+20,00 %").
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
	return html.UnescapeString(body[start : start+end])
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

// metricsList devuelve el HTML de la lista de Sprints finalizados (<ul id="sprints-finalizados">),
// o vacío si la página no la tiene.
func metricsList(body string) string {
	start := strings.Index(body, `<ul id="sprints-finalizados"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(body[start:], "</ul>")
	if end < 0 {
		return body[start:]
	}
	return body[start : start+end]
}

var metricsSelectedLinkPattern = regexp.MustCompile(`href="\?sprint=(\d+)" aria-current="true"`)

// metricsSelectedID devuelve el identificador del Sprint marcado como elegido en la lista.
func metricsSelectedID(t *testing.T, body string) string {
	t.Helper()
	list := metricsList(body)
	if list == "" {
		t.Fatalf("la página no muestra la lista de Sprints finalizados")
	}
	matches := metricsSelectedLinkPattern.FindAllStringSubmatch(list, -1)
	if len(matches) != 1 {
		t.Fatalf("la lista debía marcar exactamente un Sprint elegido; marca %d", len(matches))
	}
	return matches[0][1]
}

// countingMetricsSource cuenta los pedidos de métricas a HU-07.
type countingMetricsSource struct {
	MetricsSource
	calls atomic.Int32
}

func (s *countingMetricsSource) Metrics(ctx context.Context, projectID, sprintID int64) (SprintMetrics, error) {
	s.calls.Add(1)
	return s.MetricsSource.Metrics(ctx, projectID, sprintID)
}

// metricsThreeSprints arma el proyecto de US2: Sprints 1, 2 y 3 finalizados con 3, 5 y 8 Story
// Points y, si withActive, el Sprint 4 activo. Devuelve los identificadores en orden de número.
func metricsThreeSprints(t *testing.T, f *metricsFixture, withActive bool) (s1, s2, s3, active int64) {
	t.Helper()
	data := []struct {
		number     int
		goal       string
		start, end string
		points     int
	}{
		{1, "Alta de usuarios", "2026-09-01", "2026-09-14", 3},
		{2, "Pagos online", "2026-09-15", "2026-09-28", 5},
		{3, "Reportes", "2026-09-29", "2026-10-12", 8},
	}
	ids := make([]int64, 0, 3)
	for _, d := range data {
		id := f.activeSprint(d.number, d.goal, d.start, d.end)
		h := f.story(id, fmt.Sprintf("Historia del Sprint %d", d.number), d.points, sprint.Done, "4")
		f.effort(h, "4")
		f.close(id)
		ids = append(ids, id)
	}
	if withActive {
		active = f.activeSprint(4, "Sprint en curso", "2026-10-13", "2026-10-26")
		f.story(active, "Historia en curso", 13, sprint.Done, "2")
	}
	return ids[0], ids[1], ids[2], active
}

// T020 (US2, escenario 1; FR-002, FR-003).
func TestMetricsHandler_ListaDeSprintsFinalizadosYEleccionPorDireccion(t *testing.T) {
	app := newMetricsHandlerApp(t)
	f := newMetricsProject(t, app.db, "lista")
	s1, _, s3, _ := metricsThreeSprints(t, f, false)

	t.Run("lista del más reciente al más antiguo con las métricas del más reciente", func(t *testing.T) {
		resp, body, _ := app.open(t, f.ProjectID, pathMetrics)
		assertMetricsPageOK(t, resp, body)
		list := html.UnescapeString(metricsList(body))
		if list == "" {
			t.Fatalf("la página no muestra la lista de Sprints finalizados")
		}
		order := []string{"Sprint 3", "Sprint 2", "Sprint 1"}
		last := -1
		for _, label := range order {
			i := strings.Index(list, label)
			if i < 0 || i < last {
				t.Fatalf("la lista debía mostrar %v en ese orden; no se encontró %q en su lugar", order, label)
			}
			last = i
		}
		assertContainsAll(t, "la lista", list,
			"Reportes", "Del 29/09/2026 al 12/10/2026",
			"Pagos online", "Del 15/09/2026 al 28/09/2026",
			"Alta de usuarios", "Del 01/09/2026 al 14/09/2026")
		if got := metricsSelectedID(t, body); got != id64(s3) {
			t.Errorf("Sprint elegido = %s; se esperaba el más reciente (%d)", got, s3)
		}
		assertContainsAll(t, "el bloque de métricas", metricsSection(t, body), "Sprint 3", "8 Story Points")
	})

	t.Run("con ?sprint=ID se ven las métricas del Sprint elegido", func(t *testing.T) {
		resp, body, _ := app.open(t, f.ProjectID, pathMetrics+"?sprint="+id64(s1))
		assertMetricsPageOK(t, resp, body)
		section := metricsSection(t, body)
		assertContainsAll(t, "el bloque de métricas", section, "Sprint 1", "Alta de usuarios", "3 Story Points")
		assertContainsNone(t, "el bloque de métricas", section, "8 Story Points")
		if got := metricsSelectedID(t, body); got != id64(s1) {
			t.Errorf("Sprint elegido = %s; se esperaba %d", got, s1)
		}
	})
}

// T021 (US2; FR-010, SC-002).
func TestMetricsHandler_SprintInvalidoActivoOAjenoSeIgnora(t *testing.T) {
	app := newMetricsHandlerApp(t)
	f := newMetricsProject(t, app.db, "parámetro")
	_, _, s3, active := metricsThreeSprints(t, f, true)
	other := newMetricsProject(t, app.db, "parámetro otro")
	ajeno := other.activeSprint(1, "Sprint privado del otro proyecto", "2026-09-01", "2026-09-14")
	h := other.story(ajeno, "Historia privada", 21, sprint.Done, "2")
	other.effort(h, "9")
	other.close(ajeno)

	casos := map[string]string{
		"no numérico":      "abc",
		"vacío":            "",
		"negativo":         "-1",
		"inexistente":      id64(MaxID),
		"activo":           id64(active),
		"de otro proyecto": id64(ajeno),
		"con texto pegado": id64(s3) + "x",
		"demasiado grande": "99999999999999999999",
	}
	for name, raw := range casos {
		t.Run(name, func(t *testing.T) {
			resp, body, _ := app.open(t, f.ProjectID, pathMetrics+"?sprint="+url.QueryEscape(raw))

			// Entonces se ignora sin error y se muestra el Sprint finalizado más reciente del proyecto.
			assertMetricsPageOK(t, resp, body)
			assertContainsNone(t, "la pantalla", body, MsgNotFound, "Sprint privado del otro proyecto", "21 Story Points")
			assertContainsAll(t, "el bloque de métricas", metricsSection(t, body), "Sprint 3", "8 Story Points")
			assertContainsNone(t, "el bloque de métricas", metricsSection(t, body), "Sprint 4", "13 Story Points")
			if got := metricsSelectedID(t, body); got != id64(s3) {
				t.Errorf("Sprint elegido = %s; se esperaba el más reciente (%d)", got, s3)
			}
		})
	}
}

// T022 (US2, escenario 2; FR-009).
func TestMetricsHandler_SinSprintsFinalizados(t *testing.T) {
	var counter *countingMetricsSource
	app := newMetricsHandlerAppWith(t, nil, func(real MetricsSource) MetricsSource {
		counter = &countingMetricsSource{MetricsSource: real}
		return counter
	})

	t.Run("sin Sprint activo", func(t *testing.T) {
		f := newMetricsProject(t, app.db, "sin Sprints")
		resp, body, _ := app.open(t, f.ProjectID, pathMetrics)
		assertMetricsPageOK(t, resp, body)
		assertContainsAll(t, "la pantalla", body, MsgMetricsNoSprints)
		assertContainsNone(t, "la pantalla", body, `id="metricas-sprint"`, "sigue en curso")
	})

	t.Run("con Sprint activo", func(t *testing.T) {
		f := newMetricsProject(t, app.db, "solo activo")
		active := f.activeSprint(1, "Primer Sprint en curso", "2026-10-01", "2026-10-14")
		f.story(active, "Historia en curso", 5, sprint.Done, "3")
		resp, body, _ := app.open(t, f.ProjectID, pathMetrics)
		assertMetricsPageOK(t, resp, body)
		assertContainsAll(t, "la pantalla", html.UnescapeString(body),
			MsgMetricsNoSprints, fmt.Sprintf(MsgMetricsActiveHint, 1))
		assertContainsNone(t, "la pantalla", body, `id="metricas-sprint"`, "Primer Sprint en curso")
	})

	// Y en ningún caso se le pidió nada a HU-07.
	if n := counter.calls.Load(); n != 0 {
		t.Errorf("se pidieron %d cálculos a HU-07; sin Sprints finalizados no se debía pedir ninguno", n)
	}
}

// T023 (US2, escenario 3; FR-008, SC-002).
func TestMetricsHandler_SprintActivoNoFiguraYSeAvisaQueSeCalculaAlCerrarlo(t *testing.T) {
	var counter *countingMetricsSource
	app := newMetricsHandlerAppWith(t, nil, func(real MetricsSource) MetricsSource {
		counter = &countingMetricsSource{MetricsSource: real}
		return counter
	})
	f := newMetricsProject(t, app.db, "con activo")
	_, _, s3, active := metricsThreeSprints(t, f, true)

	resp, body, _ := app.open(t, f.ProjectID, pathMetrics)

	assertMetricsPageOK(t, resp, body)
	list := html.UnescapeString(metricsList(body))
	if list == "" {
		t.Fatalf("la página no muestra la lista de Sprints finalizados")
	}
	assertContainsNone(t, "la lista", list, "Sprint 4", "Sprint en curso", "?sprint="+id64(active)+`"`)
	assertContainsAll(t, "la pantalla", html.UnescapeString(body), fmt.Sprintf(MsgMetricsActiveHint, 4))
	if got := metricsSelectedID(t, body); got != id64(s3) {
		t.Errorf("Sprint elegido = %s; se esperaba %d", got, s3)
	}
	if n := counter.calls.Load(); n != 1 {
		t.Errorf("se pidieron %d cálculos a HU-07; se esperaba solo el del Sprint elegido", n)
	}
}
