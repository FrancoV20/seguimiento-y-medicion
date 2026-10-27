package test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
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
	return nuevaTableroAppConAlmacen(t, nil)
}

// nuevaTableroAppConAlmacen permite reemplazar el almacenamiento (para simular fallas de guardado).
func nuevaTableroAppConAlmacen(t *testing.T, envolver func(sprint.StoryStatusStore) sprint.StoryStatusStore) *tableroApp {
	t.Helper()
	f := nuevaSeleccionFixture(t)
	srv := web.NewServer(silencioso())
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(f.db))
	var almacen sprint.StoryStatusStore = sprint.NewPostgreSQLStoryStatusStore(f.db)
	if envolver != nil {
		almacen = envolver(almacen)
	}
	servicio := sprint.NewStatusService(almacen)
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
	if !strings.Contains(pantalla, "alert-warning") || strings.Contains(pantalla, "alert-success") {
		t.Errorf("completar sin horas debe verse como advertencia, no como éxito")
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

// presionarConCuerpo es como presionar pero devuelve también la pantalla de la respuesta.
func (a *tableroApp) presionarConCuerpo(t *testing.T, c *http.Client, historia int64, accion, estadoActual string) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+"/tablero/estado", url.Values{
		"historia_id": {idTexto(historia)}, "accion": {accion}, "estado_actual": {estadoActual},
	})
	if err != nil {
		t.Fatalf("POST /tablero/estado: %v", err)
	}
	return resp, formCuerpo(t, resp)
}

// historiaSinSprint crea una historia Pendiente del proyecto que no está en ningún Sprint.
func (a *tableroApp) historiaSinSprint(t *testing.T) int64 {
	t.Helper()
	return a.insertar(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, status)
		VALUES ($1, 'Historia BDD sin Sprint', 'Historia BDD', 'Media', 2, 'Pendiente') RETURNING id`, a.proyecto)
}

// almacenQueFalla lee con el almacenamiento real pero falla al guardar.
type almacenQueFalla struct{ sprint.StoryStatusStore }

func (almacenQueFalla) UpdateStatus(context.Context, int64, sprint.StoryStatus, sprint.StoryStatus) (bool, error) {
	return false, errors.New("falla simulada de guardado")
}

// accionDeTablero encuentra, en cada formulario de acción, la historia y la acción que ofrece.
var accionDeTablero = regexp.MustCompile(`name="historia_id" value="(\d+)">\s*<input type="hidden" name="accion" value="(\w+)">`)

// US2 - Escenario: Historia fuera de un Sprint activo (FR-008, SC-002).
func TestEscenarioTablero_FueraDeUnSprintActivo(t *testing.T) {
	// Dado una historia sin Sprint
	app := nuevaTableroApp(t)
	historia := app.historiaSinSprint(t)
	navegador := app.navegador(t)

	// Cuando se intenta cambiar su estado
	resp, pantalla := app.presionarConCuerpo(t, navegador, historia, "comenzar", "")

	// Entonces se rechaza con "La historia no está asignada a un Sprint activo" y el estado no cambia
	if resp.StatusCode != http.StatusConflict || !strings.Contains(pantalla, "La historia no está asignada a un Sprint activo") {
		t.Errorf("se esperaba 409 con el mensaje de fuera de un Sprint activo, se obtuvo %d", resp.StatusCode)
	}
	if estado := app.estadoDe(t, historia); estado != "Pendiente" {
		t.Errorf("estado = %q, se esperaba \"Pendiente\"", estado)
	}
}

// US2 - Escenario: Marcar dos veces (FR-009, SC-003).
func TestEscenarioTablero_MarcarDosVeces(t *testing.T) {
	// Dado una historia que ya está en "Hecho"
	app := nuevaTableroApp(t)
	historia := app.historia(t, "Hecho", 8)
	navegador := app.navegador(t)

	// Cuando se envía otra vez la acción de completarla
	app.presionar(t, navegador, historia, "completar", "En progreso")

	// Entonces veo "La historia ya está en Hecho", el estado no cambia y los Story Points
	// completados no se duplican
	pantalla := app.abrirTablero(t, navegador)
	if !strings.Contains(pantalla, "La historia ya está en Hecho") {
		t.Errorf("debía verse \"La historia ya está en Hecho\"")
	}
	if estado := app.estadoDe(t, historia); estado != "Hecho" {
		t.Errorf("estado = %q, se esperaba \"Hecho\"", estado)
	}
	if !strings.Contains(pantalla, "8 de 8") {
		t.Errorf("los Story Points completados no debían duplicarse (8 de 8)")
	}
}

// US2 - Escenario: Transición no permitida (FR-010, SC-002).
func TestEscenarioTablero_TransicionNoPermitida(t *testing.T) {
	// Dado una historia "En Sprint"
	app := nuevaTableroApp(t)
	historia := app.historia(t, "En Sprint", 3)
	navegador := app.navegador(t)

	// Cuando se intenta pasarla directamente a "Hecho"
	resp, pantalla := app.presionarConCuerpo(t, navegador, historia, "completar", "En Sprint")

	// Entonces se rechaza con "No se puede pasar de En Sprint a Hecho" y el estado no cambia
	if resp.StatusCode != http.StatusConflict || !strings.Contains(pantalla, "No se puede pasar de En Sprint a Hecho") {
		t.Errorf("se esperaba 409 con el mensaje de transición no permitida, se obtuvo %d", resp.StatusCode)
	}
	if estado := app.estadoDe(t, historia); estado != "En Sprint" {
		t.Errorf("estado = %q, se esperaba \"En Sprint\"", estado)
	}
}

// US2 - Escenario: Estado desactualizado (FR-011, SC-002, SC-006).
func TestEscenarioTablero_EstadoDesactualizado(t *testing.T) {
	// Dado que otra persona cambió el estado de la historia mientras yo tenía el tablero abierto
	app := nuevaTableroApp(t)
	historia := app.historia(t, "En Sprint", 3)
	navegador := app.navegador(t)
	if _, err := app.db.Exec(`UPDATE user_stories SET status = 'Hecho' WHERE id = $1`, historia); err != nil {
		t.Fatalf("no se pudo simular el cambio de otra persona: %v", err)
	}

	// Cuando presiono una acción
	resp, pantalla := app.presionarConCuerpo(t, navegador, historia, "comenzar", "En Sprint")

	// Entonces no se cambia nada y veo el mensaje de estado desactualizado
	if resp.StatusCode != http.StatusConflict ||
		!strings.Contains(pantalla, "El estado de la historia cambió; revise el estado actual y vuelva a intentarlo") {
		t.Errorf("se esperaba 409 con el mensaje de estado desactualizado, se obtuvo %d", resp.StatusCode)
	}
	if estado := app.estadoDe(t, historia); estado != "Hecho" {
		t.Errorf("estado = %q, se esperaba \"Hecho\" (el de la otra persona)", estado)
	}
}

// US2 - Escenario: Error al guardar (FR-012, FR-018, SC-004).
func TestEscenarioTablero_ErrorAlGuardar(t *testing.T) {
	// Dado una transición válida
	app := nuevaTableroAppConAlmacen(t, func(real sprint.StoryStatusStore) sprint.StoryStatusStore {
		return almacenQueFalla{StoryStatusStore: real}
	})
	historia := app.historia(t, "En progreso", 3)
	navegador := app.navegador(t)

	// Cuando falla el guardado
	resp, pantalla := app.presionarConCuerpo(t, navegador, historia, "completar", "En progreso")

	// Entonces veo "La operación no pudo completarse", el estado no cambia y no veo mensaje de éxito
	if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(pantalla, "La operación no pudo completarse") {
		t.Errorf("se esperaba 500 con \"La operación no pudo completarse\", se obtuvo %d", resp.StatusCode)
	}
	if strings.Contains(pantalla, "alert-success") || strings.Contains(pantalla, "falla simulada") {
		t.Errorf("no debe verse éxito ni el detalle técnico")
	}
	if estado := app.estadoDe(t, historia); estado != "En progreso" {
		t.Errorf("estado = %q, se esperaba \"En progreso\"", estado)
	}
}

// US3 - Escenario: Tablero del Sprint activo (FR-002, SC-007).
func TestEscenarioTablero_TableroDelSprintActivo(t *testing.T) {
	// Dado un Sprint activo con historias en distintos estados
	app := nuevaTableroApp(t)
	app.historia(t, "En Sprint", 3)
	app.historia(t, "En progreso", 5)
	app.historia(t, "Hecho", 8)
	navegador := app.navegador(t)

	// Cuando abro "Tablero"
	pantalla := app.abrirTablero(t, navegador)

	// Entonces veo el identificador del Sprint, las historias agrupadas por estado y los Story
	// Points completados sobre los planificados
	for _, esperado := range []string{
		"Sprint 1", "Objetivo BDD",
		`En Sprint <span class="board-count">(1)</span>`,
		`En progreso <span class="board-count">(1)</span>`,
		`Hecho <span class="board-count">(1)</span>`,
		"Historia BDD En Sprint", "Historia BDD En progreso", "Historia BDD Hecho",
		"8 de 16",
	} {
		if !strings.Contains(pantalla, esperado) {
			t.Errorf("falta %q en el Tablero", esperado)
		}
	}
}

// US3 - Escenario: Acciones según el estado (FR-003).
func TestEscenarioTablero_AccionesSegunElEstado(t *testing.T) {
	// Dado historias en distintos estados
	app := nuevaTableroApp(t)
	enSprint := app.historia(t, "En Sprint", 3)
	enProgreso := app.historia(t, "En progreso", 5)
	hecha := app.historia(t, "Hecho", 8)
	navegador := app.navegador(t)

	// Cuando abro "Tablero"
	pantalla := app.abrirTablero(t, navegador)

	// Entonces las "En Sprint" ofrecen solo "Comenzar", las "En progreso" solo "Completar" y las
	// "Hecho" solo "Reabrir"
	obtenidas := map[string]string{}
	formularios := accionDeTablero.FindAllStringSubmatch(pantalla, -1)
	for _, f := range formularios {
		obtenidas[f[1]] = f[2]
	}
	esperadas := map[string]string{
		idTexto(enSprint): "comenzar", idTexto(enProgreso): "completar", idTexto(hecha): "reabrir",
	}
	if len(formularios) != 3 || !maps.Equal(obtenidas, esperadas) {
		t.Errorf("acciones ofrecidas = %v (%d formularios), se esperaba %v", obtenidas, len(formularios), esperadas)
	}
}

// US3 - Escenario: Sin Sprint activo (FR-013).
func TestEscenarioTablero_SinSprintActivo(t *testing.T) {
	// Dado un proyecto sin Sprint activo
	app := nuevaTableroApp(t)
	app.historia(t, "Hecho", 8)
	navegador := app.navegador(t)
	if _, err := app.db.Exec(`UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, app.sprintID); err != nil {
		t.Fatalf("no se pudo finalizar el Sprint: %v", err)
	}

	// Cuando abro "Tablero"
	pantalla := app.abrirTablero(t, navegador)

	// Entonces veo "No hay un Sprint activo" y ninguna acción
	if !strings.Contains(pantalla, "No hay un Sprint activo") {
		t.Errorf("debía verse \"No hay un Sprint activo\"")
	}
	if strings.Contains(pantalla, `name="accion"`) {
		t.Errorf("sin Sprint activo no debe haber acciones")
	}
}

// US3 - Escenario: Sprint sin historias (FR-013).
func TestEscenarioTablero_SprintSinHistorias(t *testing.T) {
	// Dado un Sprint activo sin historias asignadas
	app := nuevaTableroApp(t)
	navegador := app.navegador(t)

	// Cuando abro "Tablero"
	pantalla := app.abrirTablero(t, navegador)

	// Entonces veo el Sprint y "El Sprint no tiene historias asignadas"
	if !strings.Contains(pantalla, "Sprint 1") || !strings.Contains(pantalla, "El Sprint no tiene historias asignadas") {
		t.Errorf("debía verse el Sprint con el mensaje de que no tiene historias")
	}
	if strings.Contains(pantalla, `name="accion"`) {
		t.Errorf("un Sprint sin historias no ofrece acciones")
	}
}
