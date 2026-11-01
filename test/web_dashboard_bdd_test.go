package test

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// HU-08 (T023): escenarios BDD del Dashboard contra el servidor web real.

type dashboardBDDApp struct {
	*seleccionFixture
	httpServer *httptest.Server
	cliente    *http.Client
}

func nuevaDashboardBDDApp(t *testing.T) *dashboardBDDApp {
	t.Helper()
	fixture := nuevaSeleccionFixture(t)
	server := web.NewServer(silencioso())
	web.RegisterContext(server, web.NewPostgreSQLContextStore(fixture.db))
	if err := web.RegisterDashboard(server,
		web.NewPostgreSQLDashboardCatalog(fixture.db),
		web.NewPostgreSQLBoardCatalog(fixture.db),
		metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(fixture.db)),
		web.NewHU07MetricsSource(fixture.db),
	); err != nil {
		t.Fatalf("no se pudo registrar la sección Dashboard: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	// Corre antes que la limpieza de la selección, que borra los proyectos y el integrante.
	t.Cleanup(func() {
		const sprints = `SELECT id FROM sprints WHERE project_id = $1`
		const historias = `SELECT id FROM user_stories WHERE project_id = $1`
		for _, consulta := range []string{
			`DELETE FROM defects WHERE project_id = $1`,
			`DELETE FROM sprint_story_results WHERE sprint_id IN (` + sprints + `)`,
			`DELETE FROM sprint_story_assignments WHERE sprint_id IN (` + sprints + `)`,
			`DELETE FROM sprint_metrics WHERE sprint_id IN (` + sprints + `)`,
			`DELETE FROM effort_logs WHERE story_id IN (` + historias + `)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM sprints WHERE project_id = $1`,
		} {
			_, _ = fixture.db.Exec(consulta, fixture.proyectoA)
		}
	})
	return &dashboardBDDApp{seleccionFixture: fixture, httpServer: httpServer, cliente: formNavegador(false)}
}

func (a *dashboardBDDApp) sprint(t *testing.T, numero int, estado string) int64 {
	t.Helper()
	return a.insertar(t, `
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, DATE '2026-10-01', DATE '2026-10-15', $4) RETURNING id`,
		a.proyectoA, numero, fmt.Sprintf("Objetivo BDD HU-08 %d", numero), estado)
}

// historia crea una historia en el estado indicado; con sprint distinto de 0 queda asignada a él.
func (a *dashboardBDDApp) historia(t *testing.T, puntos int, estado string, sprint int64, horasEstimadas string) int64 {
	t.Helper()
	activo := any(nil)
	if sprint != 0 {
		activo = sprint
	}
	estimadas := any(nil)
	if horasEstimadas != "" {
		estimadas = horasEstimadas
	}
	id := a.insertar(t, `
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id, estimated_hours)
		VALUES ($1, 'HU08 BDD historia', 'Historia del escenario BDD de HU-08', 'Media', $2, $3, $4, $5::numeric)
		RETURNING id`, a.proyectoA, puntos, estado, activo, estimadas)
	if sprint != 0 {
		if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprint, id); err != nil {
			t.Fatalf("no se pudo asignar la historia: %v", err)
		}
	}
	return id
}

// historiaCerrada crea una historia "Hecho" de un Sprint finalizado, como la deja el cierre de HU-15.
func (a *dashboardBDDApp) historiaCerrada(t *testing.T, sprint int64, puntos int, horasEstimadas string) int64 {
	t.Helper()
	id := a.historia(t, puntos, "Hecho", sprint, horasEstimadas)
	if _, err := a.db.Exec(`INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points) VALUES ($1, $2, 'Hecho', $3)`,
		sprint, id, puntos); err != nil {
		t.Fatalf("no se pudo registrar el cierre de la historia: %v", err)
	}
	return id
}

func (a *dashboardBDDApp) esfuerzo(t *testing.T, historia int64, horas string) {
	t.Helper()
	if _, err := a.db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, DATE '2026-10-05', 'Trabajo BDD HU-08', $3::numeric)`, historia, a.integranteA, horas); err != nil {
		t.Fatalf("no se pudo registrar el esfuerzo: %v", err)
	}
}

// ingresar elige el proyecto activo y abre el Dashboard.
func (a *dashboardBDDApp) ingresar(t *testing.T) string {
	t.Helper()
	respuesta, err := a.cliente.PostForm(a.httpServer.URL+"/contexto/proyecto", url.Values{
		"proyecto_id": {idTexto(a.proyectoA)},
		"volver_a":    {"/dashboard"},
	})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto: %v", err)
	}
	_ = formCuerpo(t, respuesta)
	respuesta, err = a.cliente.Get(a.httpServer.URL + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	cuerpo := formCuerpo(t, respuesta)
	if respuesta.StatusCode != http.StatusOK {
		t.Fatalf("GET /dashboard: estado %d; se esperaba 200", respuesta.StatusCode)
	}
	return cuerpo
}

var (
	dashboardBDDEtiquetas = regexp.MustCompile(`<[^>]*>`)
	dashboardBDDFila      = regexp.MustCompile(`<th scope="row">([^<]*)</th>\s*<td>(\d+)</td>`)
)

func dashboardBDDTexto(cuerpo string) string {
	texto := html.UnescapeString(dashboardBDDEtiquetas.ReplaceAllString(cuerpo, " "))
	return strings.Join(strings.Fields(texto), " ")
}

// dashboardBDDTabla devuelve las filas "etiqueta valor" de la tabla con ese título.
func dashboardBDDTabla(t *testing.T, cuerpo, titulo string) []string {
	t.Helper()
	inicio := strings.Index(cuerpo, "<caption>"+titulo+"</caption>")
	if inicio < 0 {
		t.Fatalf("no se encontró la tabla %q", titulo)
	}
	tabla := cuerpo[inicio:]
	if fin := strings.Index(tabla, "</table>"); fin >= 0 {
		tabla = tabla[:fin]
	}
	filas := []string{}
	for _, m := range dashboardBDDFila.FindAllStringSubmatch(tabla, -1) {
		filas = append(filas, strings.TrimSpace(m[1])+" "+m[2])
	}
	return filas
}

// Escenario 1: Proyecto sin datos suficientes
//
//	Dado un proyecto recién creado, sin historias ni Sprints
//	Cuando ingreso al Dashboard
//	Entonces veo "Aún no hay datos suficientes para calcular métricas" en lugar de un gráfico vacío o un error
func TestEscenarioHU08_ProyectoSinDatosSuficientes(t *testing.T) {
	app := nuevaDashboardBDDApp(t)

	cuerpo := app.ingresar(t)

	if !strings.Contains(dashboardBDDTexto(cuerpo), "Aún no hay datos suficientes para calcular métricas") {
		t.Errorf("debía verse el mensaje de falta de datos")
	}
	if strings.Contains(cuerpo, "<svg") {
		t.Errorf("no debía verse un gráfico vacío")
	}
}

// Escenario 2: Historias por estado
//
//	Dado un proyecto con 2 historias "Pendiente", 1 "En Sprint", 2 "En progreso" y 3 "Hecho"
//	Cuando ingreso al Dashboard
//	Entonces veo un gráfico de historias por estado con esas cuatro cantidades y una tabla con los mismos valores
func TestEscenarioHU08_HistoriasPorEstado(t *testing.T) {
	app := nuevaDashboardBDDApp(t)
	activo := app.sprint(t, 1, "Activo")
	app.historia(t, 1, "Pendiente", 0, "")
	app.historia(t, 2, "Pendiente", 0, "")
	app.historia(t, 3, "En Sprint", activo, "")
	app.historia(t, 5, "En progreso", activo, "")
	app.historia(t, 8, "En progreso", activo, "")
	app.historia(t, 1, "Hecho", activo, "")
	app.historia(t, 2, "Hecho", activo, "")
	app.historia(t, 3, "Hecho", activo, "")

	cuerpo := app.ingresar(t)

	if !strings.Contains(cuerpo, `aria-label="Historias por estado: Pendiente 2, En Sprint 1, En progreso 2, Hecho 3"`) {
		t.Errorf("debía verse el gráfico de historias por estado con 2, 1, 2 y 3")
	}
	want := []string{"Pendiente 2", "En Sprint 1", "En progreso 2", "Hecho 3"}
	if got := dashboardBDDTabla(t, cuerpo, "Historias por estado"); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("tabla de historias por estado = %v; se esperaba %v", got, want)
	}
}

// Escenario 5: Velocidad del equipo
//
//	Dado los Sprints 1 y 2 finalizados con 8 y 13 Story Points completados
//	Cuando ingreso al Dashboard
//	Entonces veo un gráfico de velocidad con una barra por Sprint, la velocidad acumulada en 21 Story Points
//	y el promedio en 10,50 por Sprint
func TestEscenarioHU08_VelocidadDelEquipo(t *testing.T) {
	app := nuevaDashboardBDDApp(t)
	app.historiaCerrada(t, app.sprint(t, 1, "Finalizado"), 8, "")
	app.historiaCerrada(t, app.sprint(t, 2, "Finalizado"), 13, "")

	cuerpo := app.ingresar(t)

	want := []string{"Sprint 1 8", "Sprint 2 13"}
	if got := dashboardBDDTabla(t, cuerpo, "Velocidad por Sprint"); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("barras de velocidad = %v; se esperaba %v", got, want)
	}
	texto := dashboardBDDTexto(cuerpo)
	for _, esperado := range []string{"Velocidad acumulada: 21 Story Points", "Promedio: 10,50 Story Points por Sprint"} {
		if !strings.Contains(texto, esperado) {
			t.Errorf("debía verse %q", esperado)
		}
	}
}

// Escenario 7: Desviación de esfuerzo del último Sprint
//
//	Dado el último Sprint finalizado con 10 horas estimadas y 12 horas reales en sus historias terminadas
//	Cuando ingreso al Dashboard
//	Entonces veo la desviación de ese Sprint como "+20,00 %"
func TestEscenarioHU08_DesviacionDelUltimoSprint(t *testing.T) {
	app := nuevaDashboardBDDApp(t)
	historia := app.historiaCerrada(t, app.sprint(t, 1, "Finalizado"), 5, "10")
	app.esfuerzo(t, historia, "8")
	app.esfuerzo(t, historia, "4")

	cuerpo := app.ingresar(t)

	if !strings.Contains(dashboardBDDTexto(cuerpo), "Desviación de esfuerzo del Sprint 1: +20,00 %") {
		t.Errorf("debía verse la desviación +20,00 %% del último Sprint finalizado")
	}
}
