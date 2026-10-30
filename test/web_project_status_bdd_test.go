package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-12: consulta del estado de un proyecto. Usan la API pública de web, el
// servidor real, el reloj fijo en 28/10/2026 10:30 y PostgreSQL real con datos propios.

type estadoApp struct {
	*seleccionFixture
	servidor *httptest.Server
}

func nuevaEstadoApp(t *testing.T) *estadoApp {
	t.Helper()
	f := nuevaSeleccionFixture(t)
	srv := web.NewServer(silencioso())
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(f.db))
	hoy := func() time.Time { return time.Date(2026, time.October, 28, 10, 30, 0, 0, time.Local) }
	if err := web.RegisterProjectStatus(srv, web.NewPostgreSQLProjectStatusCatalog(f.db),
		web.NewPostgreSQLBoardCatalog(f.db), hoy); err != nil {
		t.Fatalf("no se pudo registrar Estado del proyecto: %v", err)
	}
	app := &estadoApp{seleccionFixture: f, servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	return app
}

// proyecto crea un proyecto con fechas aaaa-mm-dd y, al terminar, lo borra con sus datos.
func (a *estadoApp) proyecto(t *testing.T, nombre, inicio, fin string) int64 {
	t.Helper()
	id := a.insertar(t, `INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, $2::date, $3::date, 'Activo') RETURNING id`,
		fmt.Sprintf("BDD Estado %s %d", nombre, time.Now().UnixNano()), inicio, fin)
	t.Cleanup(func() {
		for _, consulta := range []string{
			`DELETE FROM sprint_story_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM sprints WHERE project_id = $1`,
			`DELETE FROM projects WHERE id = $1`,
		} {
			_, _ = a.db.Exec(consulta, id)
		}
	})
	return id
}

// sprint crea un Sprint del proyecto.
func (a *estadoApp) sprint(t *testing.T, proyecto int64, numero int, objetivo, inicio, fin, estado string) int64 {
	t.Helper()
	return a.insertar(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4::date, $5::date, $6) RETURNING id`, proyecto, numero, objetivo, inicio, fin, estado)
}

// historias crea cantidad historias en el estado indicado; si sprintID es mayor que cero, las asigna.
func (a *estadoApp) historias(t *testing.T, proyecto, sprintID int64, estado string, cantidad int) {
	t.Helper()
	for i := 1; i <= cantidad; i++ {
		titulo := fmt.Sprintf("Historia BDD %s %d.%d", estado, sprintID, i)
		if sprintID == 0 {
			a.insertar(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, status)
				VALUES ($1, $2, 'Historia BDD', 'Media', 3, $3) RETURNING id`, proyecto, titulo, estado)
			continue
		}
		id := a.insertar(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
			VALUES ($1, $2, 'Historia BDD', 'Media', 3, $3, $4) RETURNING id`, proyecto, titulo, estado, sprintID)
		if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id); err != nil {
			t.Fatalf("no se pudo asignar la historia al Sprint: %v", err)
		}
	}
}

// navegador simula un navegador con el proyecto ya elegido.
func (a *estadoApp) navegador(t *testing.T, proyecto int64) *http.Client {
	t.Helper()
	c := formNavegador(false)
	resp, err := c.PostForm(a.servidor.URL+"/contexto/proyecto",
		url.Values{"proyecto_id": {idTexto(proyecto)}, "volver_a": {"/estado"}})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto: %v", err)
	}
	_ = formCuerpo(t, resp)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto: se obtuvo %d", resp.StatusCode)
	}
	return c
}

// abrirEstado devuelve la pantalla "Estado del proyecto".
func (a *estadoApp) abrirEstado(t *testing.T, c *http.Client) string {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + "/estado")
	if err != nil {
		t.Fatalf("GET /estado: %v", err)
	}
	return formCuerpo(t, resp)
}

// Escenario: Proyecto en curso (FR-002, FR-003, FR-007, FR-008).
func TestEscenarioEstado_ProyectoEnCurso(t *testing.T) {
	// Dado un proyecto del 01/10/2026 al 30/11/2026 con 8 historias (4 "Hecho") y un Sprint activo
	app := nuevaEstadoApp(t)
	proyecto := app.proyecto(t, "en curso", "2026-10-01", "2026-11-30")
	sprintID := app.sprint(t, proyecto, 1, "Pagos online", "2026-10-19", "2026-11-01", "Activo")
	app.historias(t, proyecto, sprintID, "En Sprint", 1)
	app.historias(t, proyecto, sprintID, "En progreso", 1)
	app.historias(t, proyecto, sprintID, "Hecho", 4)
	app.historias(t, proyecto, 0, "Pendiente", 2)
	navegador := app.navegador(t, proyecto)

	// Cuando abro "Estado del proyecto" el 28/10/2026
	pantalla := app.abrirEstado(t, navegador)

	// Entonces veo "En curso", el avance 4 de 8 (50,00 %) y el Sprint en curso con su objetivo
	for _, esperado := range []string{
		"Estado: En curso", "Del 01/10/2026 al 30/11/2026",
		"4 de 8 historias completadas (50,00 %)",
		"Sprint 1 en curso", "Objetivo: Pagos online",
	} {
		if !strings.Contains(pantalla, esperado) {
			t.Errorf("falta %q en la pantalla", esperado)
		}
	}
}

// Escenario: Proyecto no iniciado (FR-003, FR-006, FR-009).
func TestEscenarioEstado_ProyectoNoIniciado(t *testing.T) {
	// Dado un proyecto que empieza el 01/12/2026, sin historias
	app := nuevaEstadoApp(t)
	proyecto := app.proyecto(t, "no iniciado", "2026-12-01", "2027-01-31")
	navegador := app.navegador(t, proyecto)

	// Cuando abro "Estado del proyecto" el 28/10/2026
	pantalla := app.abrirEstado(t, navegador)

	// Entonces veo "No iniciado", "0 de 0 historias completadas (0,00 %)" y que no hay Sprint en curso
	for _, esperado := range []string{
		"Estado: No iniciado", "0 de 0 historias completadas (0,00 %)", "No hay un Sprint en curso",
	} {
		if !strings.Contains(pantalla, esperado) {
			t.Errorf("falta %q en la pantalla", esperado)
		}
	}
}

// Escenario: Proyecto finalizado (FR-003, FR-008).
func TestEscenarioEstado_ProyectoFinalizado(t *testing.T) {
	// Dado un proyecto que terminó el 30/09/2026, con 3 historias "Hecho" y sin Sprint activo
	app := nuevaEstadoApp(t)
	proyecto := app.proyecto(t, "finalizado", "2026-08-01", "2026-09-30")
	sprintID := app.sprint(t, proyecto, 1, "Primer Sprint", "2026-08-01", "2026-08-14", "Finalizado")
	app.historias(t, proyecto, sprintID, "Hecho", 3)
	navegador := app.navegador(t, proyecto)

	// Cuando abro "Estado del proyecto" el 28/10/2026
	pantalla := app.abrirEstado(t, navegador)

	// Entonces veo "Finalizado" y "3 de 3 historias completadas (100,00 %)"
	for _, esperado := range []string{
		"Estado: Finalizado", "3 de 3 historias completadas (100,00 %)", "No hay un Sprint en curso",
	} {
		if !strings.Contains(pantalla, esperado) {
			t.Errorf("falta %q en la pantalla", esperado)
		}
	}
}

// Escenario: Sin proyecto seleccionado (FR-001).
func TestEscenarioEstado_SinProyectoSeleccionado(t *testing.T) {
	// Dado que no elegí ningún proyecto
	app := nuevaEstadoApp(t)
	navegador := formNavegador(false)

	// Cuando abro "Estado del proyecto"
	resp, err := navegador.Get(app.servidor.URL + "/estado")
	if err != nil {
		t.Fatalf("GET /estado: %v", err)
	}
	_ = formCuerpo(t, resp)

	// Entonces me lleva a elegir un proyecto y vuelvo a "Estado del proyecto" después
	want := "/contexto/proyecto?volver_a=%2Festado"
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
		t.Errorf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, resp.Header.Get("Location"))
	}
}
