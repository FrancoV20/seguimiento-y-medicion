package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-14, US1: cambiar el estado de una historia (T018).
// Usan la API pública de web y sprint, el servidor real y PostgreSQL real.

// tableroApp es el servidor real con el contexto de trabajo y el Tablero, y un proyecto propio
// con un Sprint activo.
type tableroApp struct {
	*seleccionFixture
	servidor *httptest.Server
	proyecto int64
	sprintID int64
}

func nuevaTableroApp(t *testing.T) *tableroApp {
	t.Helper()
	f := nuevaSeleccionFixture(t)
	srv := web.NewServer(silencioso())
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(f.db))
	servicio := sprint.NewStatusService(sprint.NewPostgreSQLStoryStatusStore(f.db))
	if err := web.RegisterBoard(srv, servicio, web.NewPostgreSQLBoardCatalog(f.db)); err != nil {
		t.Fatalf("no se pudo registrar el Tablero: %v", err)
	}
	app := &tableroApp{seleccionFixture: f, servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)

	// Se registra antes de insertar: las historias no tienen borrado en cascada.
	t.Cleanup(func() {
		for _, consulta := range []string{
			`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM projects WHERE id = $1`,
		} {
			_, _ = f.db.Exec(consulta, app.proyecto)
		}
	})
	app.proyecto = f.insertar(t, `INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, DATE '2026-10-01', DATE '2026-12-31', 'Activo') RETURNING id`,
		fmt.Sprintf("BDD Tablero %d", time.Now().UnixNano()))
	app.sprintID = f.insertar(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Objetivo BDD', DATE '2026-10-01', DATE '2026-10-15', 'Activo') RETURNING id`, app.proyecto)
	return app
}

// historia crea una historia del Sprint activo en el estado indicado.
func (a *tableroApp) historia(t *testing.T, estado string, puntos int) int64 {
	t.Helper()
	id := a.insertar(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
		VALUES ($1, $2, 'Historia BDD', 'Media', $3, $4, $5) RETURNING id`,
		a.proyecto, "Historia BDD "+estado, puntos, estado, a.sprintID)
	if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, a.sprintID, id); err != nil {
		t.Fatalf("no se pudo asignar la historia al Sprint: %v", err)
	}
	return id
}

// conHoras registra horas reales en la historia.
func (a *tableroApp) conHoras(t *testing.T, historia int64) {
	t.Helper()
	miembro := a.insertar(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("BDD Horas %d", time.Now().UnixNano()))
	t.Cleanup(func() {
		_, _ = a.db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, miembro)
		_, _ = a.db.Exec(`DELETE FROM members WHERE id = $1`, miembro)
	})
	if _, err := a.db.Exec(`INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, DATE '2026-10-05', 'Trabajo BDD', 2)`, historia, miembro); err != nil {
		t.Fatalf("no se pudieron registrar las horas: %v", err)
	}
}

// navegador simula un navegador con el proyecto ya elegido.
func (a *tableroApp) navegador(t *testing.T) *http.Client {
	t.Helper()
	c := formNavegador(false)
	resp, err := c.PostForm(a.servidor.URL+"/contexto/proyecto",
		url.Values{"proyecto_id": {idTexto(a.proyecto)}, "volver_a": {"/tablero"}})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto: %v", err)
	}
	_ = formCuerpo(t, resp)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto: se obtuvo %d", resp.StatusCode)
	}
	return c
}

// presionar envía la acción de una historia, como el botón del Tablero.
func (a *tableroApp) presionar(t *testing.T, c *http.Client, historia int64, accion, estadoActual string) *http.Response {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+"/tablero/estado", url.Values{
		"historia_id": {idTexto(historia)}, "accion": {accion}, "estado_actual": {estadoActual},
	})
	if err != nil {
		t.Fatalf("POST /tablero/estado: %v", err)
	}
	_ = formCuerpo(t, resp)
	return resp
}

// abrirTablero devuelve la pantalla del Tablero.
func (a *tableroApp) abrirTablero(t *testing.T, c *http.Client) string {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + "/tablero")
	if err != nil {
		t.Fatalf("GET /tablero: %v", err)
	}
	return formCuerpo(t, resp)
}

func (a *tableroApp) estadoDe(t *testing.T, historia int64) string {
	t.Helper()
	var estado string
	if err := a.db.QueryRow(`SELECT status FROM user_stories WHERE id = $1`, historia).Scan(&estado); err != nil {
		t.Fatalf("no se pudo leer el estado de la historia: %v", err)
	}
	return estado
}

// US1 - Escenario: Comenzar una historia (FR-004, SC-001).
func TestEscenarioTablero_ComenzarUnaHistoria(t *testing.T) {
	// Dado una historia "En Sprint" del Sprint activo
	app := nuevaTableroApp(t)
	historia := app.historia(t, "En Sprint", 5)
	navegador := app.navegador(t)

	// Cuando presiono "Comenzar"
	app.presionar(t, navegador, historia, "comenzar", "En Sprint")

	// Entonces veo "La historia pasó a En progreso" y la historia queda en estado "En progreso"
	if pantalla := app.abrirTablero(t, navegador); !strings.Contains(pantalla, "La historia pasó a En progreso") {
		t.Errorf("debía verse el mensaje \"La historia pasó a En progreso\"")
	}
	if estado := app.estadoDe(t, historia); estado != "En progreso" {
		t.Errorf("estado = %q, se esperaba \"En progreso\"", estado)
	}
}

// US1 - Escenario: Completar una historia (FR-005, FR-017, SC-001, SC-003).
func TestEscenarioTablero_CompletarUnaHistoria(t *testing.T) {
	// Dado una historia "En progreso" del Sprint activo con horas registradas
	app := nuevaTableroApp(t)
	historia := app.historia(t, "En progreso", 8)
	app.conHoras(t, historia)
	navegador := app.navegador(t)

	// Cuando presiono "Completar"
	app.presionar(t, navegador, historia, "completar", "En progreso")

	// Entonces veo "La historia pasó a Hecho", la historia queda "Hecho" y sus Story Points
	// se suman a los completados del Sprint
	pantalla := app.abrirTablero(t, navegador)
	if !strings.Contains(pantalla, "La historia pasó a Hecho") || strings.Contains(pantalla, "no tiene horas reales") {
		t.Errorf("debía verse \"La historia pasó a Hecho\" sin advertencia")
	}
	if estado := app.estadoDe(t, historia); estado != "Hecho" {
		t.Errorf("estado = %q, se esperaba \"Hecho\"", estado)
	}
	if !strings.Contains(pantalla, "8 de 8") {
		t.Errorf("los Story Points completados debían ser 8 de 8")
	}
}

// US1 - Escenario: Completar sin horas registradas (FR-006, SC-005).
func TestEscenarioTablero_CompletarSinHorasRegistradas(t *testing.T) {
	// Dado una historia "En progreso" del Sprint activo sin horas registradas
	app := nuevaTableroApp(t)
	historia := app.historia(t, "En progreso", 3)
	navegador := app.navegador(t)

	// Cuando presiono "Completar"
	app.presionar(t, navegador, historia, "completar", "En progreso")

	// Entonces la historia pasa a "Hecho" y veo el mensaje con la advertencia en un solo mensaje
	pantalla := app.abrirTablero(t, navegador)
	if !strings.Contains(pantalla, "La historia pasó a Hecho. Advertencia: no tiene horas reales registradas") {
		t.Errorf("debía verse el mensaje con la advertencia de horas")
	}
	if n := strings.Count(pantalla, `class="alert `); n != 1 {
		t.Errorf("debía ser un solo mensaje, hay %d", n)
	}
	if estado := app.estadoDe(t, historia); estado != "Hecho" {
		t.Errorf("estado = %q, se esperaba \"Hecho\"", estado)
	}
}

// US1 - Escenario: Reabrir una historia (FR-007, FR-017, SC-003).
func TestEscenarioTablero_ReabrirUnaHistoria(t *testing.T) {
	// Dado una historia "Hecho" del Sprint activo
	app := nuevaTableroApp(t)
	historia := app.historia(t, "Hecho", 8)
	navegador := app.navegador(t)
	if antes := app.abrirTablero(t, navegador); !strings.Contains(antes, "8 de 8") {
		t.Fatalf("precondición: los Story Points completados eran 8 de 8")
	}

	// Cuando presiono "Reabrir"
	app.presionar(t, navegador, historia, "reabrir", "Hecho")

	// Entonces veo "La historia volvió a En progreso", la historia pasa a "En progreso" y sus
	// Story Points dejan de contarse como completados
	pantalla := app.abrirTablero(t, navegador)
	if !strings.Contains(pantalla, "La historia volvió a En progreso") {
		t.Errorf("debía verse el mensaje \"La historia volvió a En progreso\"")
	}
	if estado := app.estadoDe(t, historia); estado != "En progreso" {
		t.Errorf("estado = %q, se esperaba \"En progreso\"", estado)
	}
	if !strings.Contains(pantalla, "0 de 8") {
		t.Errorf("los Story Points completados debían volver a 0 de 8")
	}
}