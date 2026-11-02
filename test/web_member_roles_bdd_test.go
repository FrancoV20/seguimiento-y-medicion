package test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
	"github.com/FrancoV20/seguimiento-y-medicion/src/role"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// HU-28: escenarios BDD de los roles del proyecto, con la aplicación armada como main.go (los
// Register… reales y los adaptadores PostgreSQL) y los datos del Issue:
//   - «Alfa»: Ana "Scrum Master"; Beto y Caro "Integrante". Sprint 1 finalizado y Sprint 2 en curso
//     con «Login», que tiene su ronda 1 abierta (con el voto de Ana, para que se pueda cerrar).
//     «Perfil» en el Backlog, sin Sprint.
//   - «Beta»: Dani "Integrante", sin Scrum Master; «Reportes» en el Backlog.
//   - Contraseñas de HU-27: "ana-clave-2026", "beto-clave-2026", "caro-clave-2026" y "dani-clave-2026".

const claveCaro = "caro-clave-2026"

// relojHU28 es el "hoy" de la aplicación para el registro de esfuerzo (dentro del Sprint 2).
var relojHU28 = func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) }

type hu28Mundo struct {
	db                      *sql.DB
	alfa, beta              int64
	ana, beto, caro, dani   int64
	sprint1, sprint2        int64
	login, perfil, reportes int64
	ronda1                  int64
}

func nuevoMundoHU28(t *testing.T) *hu28Mundo {
	t.Helper()
	raw := os.Getenv(web.EnvDatabaseURL)
	if raw == "" {
		t.Skip("DATABASE_URL no definida: se omite la integración BDD de HU-28")
	}
	db, err := sql.Open("pgx", raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	m := &hu28Mundo{db: db}
	// Se registra antes de insertar: limpia aunque la preparación falle a mitad.
	t.Cleanup(func() {
		for _, p := range []int64{m.alfa, m.beta} {
			if p == 0 {
				continue
			}
			_, _ = db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, p)
			_, _ = db.Exec(`DELETE FROM poker_rounds WHERE project_id = $1`, p)
			_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, p)
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, p) // borra Sprints e integrantes del proyecto
		}
		for _, i := range []int64{m.ana, m.beto, m.caro, m.dani} {
			if i != 0 {
				_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, i)
			}
		}
	})
	id := func(q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(q, args...).Scan(&v); err != nil {
			t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
		}
		return v
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
		}
	}

	m.alfa = id(`INSERT INTO projects (name, start_date, end_date, next_sprint_number)
		VALUES ('Alfa', DATE '2026-01-01', DATE '2026-12-31', 3) RETURNING id`)
	m.beta = id(`INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', DATE '2026-01-01', DATE '2026-12-31') RETURNING id`)
	for _, p := range []struct {
		dst      *int64
		nombre   string
		clave    string
		proyecto int64
	}{{&m.ana, "Ana", claveAna, m.alfa}, {&m.beto, "Beto", claveBeto, m.alfa}, {&m.caro, "Caro", claveCaro, m.alfa}, {&m.dani, "Dani", claveDani, m.beta}} {
		*p.dst = id(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, p.nombre)
		exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, p.proyecto, *p.dst)
	}
	configurarScrumMaster(t, db, m.alfa, m.ana)
	for _, c := range []struct {
		integrante int64
		clave      string
	}{{m.ana, claveAna}, {m.beto, claveBeto}, {m.caro, claveCaro}, {m.dani, claveDani}} {
		configurarClave(t, db, c.integrante, c.clave)
	}

	m.sprint1 = id(`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Primer incremento', DATE '2026-10-01', DATE '2026-10-14', 'Finalizado') RETURNING id`, m.alfa)
	m.sprint2 = id(`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 2, 'Ingreso al sistema', DATE '2026-10-26', DATE '2026-11-08', 'Activo') RETURNING id`, m.alfa)
	historia := func(proyecto int64, titulo, estado string, sprintID any) int64 {
		return id(`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
			VALUES ($1, $2, 'Historia del escenario BDD de HU-28', 'Media', $3, $4) RETURNING id`, proyecto, titulo, estado, sprintID)
	}
	m.login = historia(m.alfa, "Login", "En Sprint", m.sprint2)
	exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, m.sprint2, m.login)
	m.perfil = historia(m.alfa, "Perfil", "Pendiente", nil)
	m.reportes = historia(m.beta, "Reportes", "Pendiente", nil)
	m.ronda1 = id(`INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by)
		VALUES ($1, $2, 1, 'Abierta', $3) RETURNING id`, m.alfa, m.login, m.ana)
	exec(`INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, 5)`, m.ronda1, m.ana)
	return m
}

// falloAlLeerContexto simula una falla de la base al leer el contexto (y el rol) cuando está activo.
type falloAlLeerContexto struct {
	web.ContextStore
	activo *atomic.Bool
}

func (s falloAlLeerContexto) ResolveContext(ctx context.Context, proyecto, integrante int64) (web.ResolvedContext, error) {
	if s.activo.Load() {
		return web.ResolvedContext{}, web.Unavailable(errors.New("pq: rol no disponible en project_members"))
	}
	return s.ContextStore.ResolveContext(ctx, proyecto, integrante)
}

type hu28App struct {
	servidor *httptest.Server
	falla    *atomic.Bool
}

// laAplicacion arma el servidor como main.go: CrossOriginGuard, contexto de HU-20b y las secciones
// de HU-11, HU-14, HU-05a/05b/18, HU-24, HU-23 y HU-15 con sus adaptadores PostgreSQL.
func (m *hu28Mundo) laAplicacion(t *testing.T) *hu28App {
	t.Helper()
	srv := web.NewServer(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	falla := &atomic.Bool{}
	web.RegisterContext(srv, falloAlLeerContexto{ContextStore: web.NewPostgreSQLContextStore(m.db), activo: falla})

	registrar := func(nombre string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("no se pudo registrar %s: %v", nombre, err)
		}
	}
	registrar("Proyectos", web.RegisterProjects(srv, project.NewProjectService(project.NewPostgreSQLProjectStore(m.db)),
		web.NewPostgreSQLProjectCatalog(m.db)))
	registrar("Tablero", web.RegisterBoard(srv, sprint.NewStatusService(sprint.NewPostgreSQLStoryStatusStore(m.db)),
		web.NewPostgreSQLBoardCatalog(m.db)))
	registrar("Planning Poker", web.RegisterPlanningPoker(srv, poker.NewService(poker.NewPostgreSQLStore(m.db)),
		web.NewPostgreSQLPokerCatalog(m.db)))
	esfuerzo := effort.NewService(effort.NewPostgreSQLEffortStore(m.db), relojHU28)
	registrar("Registro de esfuerzo", web.RegisterEffort(srv, web.NewEffortActions(esfuerzo, relojHU28),
		web.NewPostgreSQLEffortCatalog(m.db), relojHU28))
	sprintStore, err := sprint.NewPostgreSQLSprintStore(os.Getenv(web.EnvDatabaseURL))
	if err != nil {
		t.Fatalf("no se pudo abrir SprintStore PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = sprintStore.Close() })
	registrar("Sprints", web.RegisterSprints(srv, web.NewPostgreSQLSprintCatalog(m.db),
		sprintAssignmentBDDAdapter{service: sprint.NewService(sprintStore), store: sprintStore}))
	registrar("Cierre de Sprint", web.RegisterClose(srv,
		sprint.NewCloseSprintService(sprint.NewPostgreSQLSprintCloser(m.db), cierreBDDValidator{}),
		web.NewPostgreSQLBoardCatalog(m.db)))

	app := &hu28App{servidor: httptest.NewServer(srv.Handler()), falla: falla}
	t.Cleanup(app.servidor.Close)
	return app
}

func (a *hu28App) pedir(t *testing.T, c *http.Client, metodo, ruta string, valores url.Values) (*http.Response, string) {
	t.Helper()
	cuerpo := ""
	if valores != nil {
		cuerpo = valores.Encode()
	}
	req, err := http.NewRequest(metodo, a.servidor.URL+ruta, strings.NewReader(cuerpo))
	if err != nil {
		t.Fatal(err)
	}
	if metodo == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", a.servidor.URL)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", metodo, ruta, err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	return resp, b.String()
}

func (a *hu28App) enviar(t *testing.T, c *http.Client, ruta string, valores url.Values) (*http.Response, string) {
	t.Helper()
	return a.pedir(t, c, http.MethodPost, ruta, valores)
}

func (a *hu28App) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	return a.pedir(t, c, http.MethodGet, ruta, nil)
}

// enProyecto devuelve un navegador que eligió el proyecto, sin identificarse.
func (a *hu28App) enProyecto(t *testing.T, proyecto int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if resp, _ := a.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {hu28ID(proyecto)}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: %d", resp.StatusCode)
	}
	return c
}

// identificadoComo devuelve un navegador en el proyecto, identificado con la contraseña (HU-27).
func (a *hu28App) identificadoComo(t *testing.T, proyecto, integrante int64, clave string) *http.Client {
	t.Helper()
	c := a.enProyecto(t, proyecto)
	if resp, _ := a.enviar(t, c, "/contexto/integrante", formIdentificacion(hu28ID(integrante), clave)); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("identificarse: %d", resp.StatusCode)
	}
	return c
}

func hu28ID(id int64) string { return strconv.FormatInt(id, 10) }

// rechazado comprueba el 403 con el texto pedido.
func rechazado(t *testing.T, accion string, resp *http.Response, cuerpo, mensaje string) {
	t.Helper()
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(cuerpo, mensaje) {
		t.Errorf("%s: estado %d; se esperaba 403 con %q", accion, resp.StatusCode, mensaje)
	}
}

func (m *hu28Mundo) texto(t *testing.T, q string, args ...any) string {
	t.Helper()
	var v sql.NullString
	if err := m.db.QueryRow(q, args...).Scan(&v); err != nil {
		t.Fatalf("no se pudo leer el dato: %v", err)
	}
	return v.String
}

func (m *hu28Mundo) numero(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var v sql.NullInt64
	if err := m.db.QueryRow(q, args...).Scan(&v); err != nil {
		t.Fatalf("no se pudo leer el dato: %v", err)
	}
	return v.Int64
}

func (m *hu28Mundo) estadoSprint(t *testing.T, id int64) string {
	t.Helper()
	return m.texto(t, `SELECT status FROM sprints WHERE id = $1`, id)
}

// Escenario 1: Rol visible.
func TestHU28_RolVisible(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que me identifiqué como Ana en «Alfa»
	ana := a.identificadoComo(t, m.alfa, m.ana, claveAna)
	// Cuando navego por la aplicación
	for _, ruta := range []string{"/planning-poker", "/sprints", "/cierre"} {
		_, pagina := a.abrir(t, ana, ruta)
		// Entonces veo "Ana (Scrum Master)" en la estructura común
		if !strings.Contains(pagina, "Integrante: <strong>Ana (Scrum Master)</strong>") {
			t.Errorf("%s: no se ve \"Ana (Scrum Master)\" en la estructura común", ruta)
		}
	}
	_, pagina := a.abrir(t, a.identificadoComo(t, m.alfa, m.beto, claveBeto), "/planning-poker")
	if !strings.Contains(pagina, "Integrante: <strong>Beto (Integrante)</strong>") {
		t.Error("no se ve \"Beto (Integrante)\" en la estructura común")
	}
}

// Escenario 2: El Scrum Master cierra el Sprint.
func TestHU28_ScrumMasterCierraSprint(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Ana
	ana := a.identificadoComo(t, m.alfa, m.ana, claveAna)
	// Cuando cierro el Sprint 2
	resp, _ := a.enviar(t, ana, "/cierre", url.Values{"sprint_id": {hu28ID(m.sprint2)}})
	// Entonces el Sprint se cierra como indica HU-15
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/cierre" {
		t.Fatalf("cerrar: estado %d a %q; se esperaba 303 a /cierre", resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := m.estadoSprint(t, m.sprint2); got != "Finalizado" {
		t.Errorf("Sprint 2 = %q; se esperaba Finalizado", got)
	}
	_, pagina := a.abrir(t, ana, "/cierre")
	if !strings.Contains(pagina, "Sprint 2 cerrado correctamente. Historias devueltas al Backlog: 1") {
		t.Error("no se ve la confirmación de cierre de HU-15")
	}
}

// Escenario 3: Un integrante no puede cerrar el Sprint.
func TestHU28_IntegranteNoCierraSprint(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Beto
	beto := a.identificadoComo(t, m.alfa, m.beto, claveBeto)
	// Cuando intento cerrar el Sprint 2
	resp, cuerpo := a.enviar(t, beto, "/cierre", url.Values{"sprint_id": {hu28ID(m.sprint2)}})
	// Entonces veo el mensaje y el Sprint sigue en curso
	rechazado(t, "cerrar el Sprint", resp, cuerpo, web.MsgScrumMasterOnly)
	if got := m.estadoSprint(t, m.sprint2); got != "Activo" {
		t.Errorf("Sprint 2 = %q; se esperaba que siguiera Activo", got)
	}
	if n := m.numero(t, `SELECT count(*) FROM sprint_story_assignments WHERE sprint_id = $1 AND story_id = $2`, m.sprint2, m.login); n != 1 {
		t.Error("«Login» dejó de estar en el Sprint 2")
	}
}

// Escenario 4: Un integrante no puede crear Sprints ni asignar historias.
func TestHU28_IntegranteNoPlanificaSprint(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Beto
	beto := a.identificadoComo(t, m.alfa, m.beto, claveBeto)
	// Cuando intento crear un Sprint o agregar o quitar historias del Sprint en curso
	resp, cuerpo := a.enviar(t, beto, "/sprints", url.Values{
		web.FieldGoal: {"Perfil de usuario"}, web.FieldStartDate: {"2026-11-09"}, web.FieldEndDate: {"2026-11-22"},
		web.FieldStory: {hu28ID(m.perfil)},
	})
	rechazado(t, "crear un Sprint", resp, cuerpo, web.MsgScrumMasterOnly)
	resp, cuerpo = a.enviar(t, beto, "/sprints/historias", url.Values{web.FieldStory: {hu28ID(m.perfil)}})
	rechazado(t, "agregar «Perfil»", resp, cuerpo, web.MsgScrumMasterOnly)
	resp, cuerpo = a.enviar(t, beto, "/sprints/historias/quitar", url.Values{web.FieldStoryID: {hu28ID(m.login)}})
	rechazado(t, "quitar «Login»", resp, cuerpo, web.MsgScrumMasterOnly)

	// Entonces no cambia nada
	if n := m.numero(t, `SELECT count(*) FROM sprints WHERE project_id = $1`, m.alfa); n != 2 {
		t.Errorf("«Alfa» tiene %d Sprints; se esperaban los 2 de siempre", n)
	}
	if n := m.numero(t, `SELECT count(*) FROM sprint_story_assignments WHERE story_id = $1`, m.perfil); n != 0 ||
		m.numero(t, `SELECT active_sprint_id FROM user_stories WHERE id = $1`, m.perfil) != 0 {
		t.Error("«Perfil» quedó asignada a un Sprint")
	}
	if n := m.numero(t, `SELECT count(*) FROM sprint_story_assignments WHERE sprint_id = $1 AND story_id = $2`, m.sprint2, m.login); n != 1 {
		t.Error("«Login» dejó de estar en el Sprint 2")
	}
	// Y los botones siguen visibles para Beto (FR-012)
	_, pagina := a.abrir(t, beto, "/sprints")
	for _, accion := range []string{`action="/sprints/historias"`, `action="/sprints/historias/quitar"`} {
		if !strings.Contains(pagina, accion) {
			t.Errorf("la pantalla de Sprints de Beto debía seguir mostrando %s", accion)
		}
	}
}

// Escenario 5: Un integrante no puede modificar el proyecto.
func TestHU28_IntegranteNoModificaProyecto(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Caro
	caro := a.identificadoComo(t, m.alfa, m.caro, claveCaro)
	// Cuando intento guardar cambios en «Alfa»
	valores := url.Values{"nombre": {"Alfa renombrado"}, "fecha_inicio": {"2026-01-01"}, "fecha_fin": {"2026-12-31"},
		"integrantes": {hu28ID(m.ana), hu28ID(m.beto), hu28ID(m.caro)}}
	resp, cuerpo := a.enviar(t, caro, fmt.Sprintf("/proyectos/%d/modificar", m.alfa), valores)
	// Entonces veo el mensaje y el proyecto no cambia
	rechazado(t, "modificar «Alfa»", resp, cuerpo, web.MsgScrumMasterOnly)
	if got := m.texto(t, `SELECT name FROM projects WHERE id = $1`, m.alfa); got != "Alfa" {
		t.Errorf("nombre de «Alfa» = %q; no debía cambiar", got)
	}
}

// Escenario 13: Modificar un proyecto que no es el activo.
func TestHU28_ModificarOtroProyecto(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Ana, Scrum Master de «Alfa», con «Alfa» activo
	ana := a.identificadoComo(t, m.alfa, m.ana, claveAna)
	// Cuando intento guardar cambios en «Beta»
	valores := url.Values{"nombre": {"Beta renombrado"}, "fecha_inicio": {"2026-01-01"}, "fecha_fin": {"2026-12-31"},
		"integrantes": {hu28ID(m.dani)}}
	resp, cuerpo := a.enviar(t, ana, fmt.Sprintf("/proyectos/%d/modificar", m.beta), valores)
	// Entonces veo el mensaje y «Beta» no cambia
	rechazado(t, "modificar «Beta»", resp, cuerpo, web.MsgScrumMasterOtherProject)
	if got := m.texto(t, `SELECT name FROM projects WHERE id = $1`, m.beta); got != "Beta" {
		t.Errorf("nombre de «Beta» = %q; no debía cambiar", got)
	}
}

// Escenario 6: Un integrante no puede conducir el Planning Poker.
func TestHU28_IntegranteNoConducePoker(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Beto
	beto := a.identificadoComo(t, m.alfa, m.beto, claveBeto)
	// Cuando intento abrir o cerrar una ronda
	resp, cuerpo := a.enviar(t, beto, "/planning-poker/abrir", url.Values{"historia": {hu28ID(m.perfil)}})
	rechazado(t, "abrir la ronda de «Perfil»", resp, cuerpo, web.MsgScrumMasterOnly)
	resp, cuerpo = a.enviar(t, beto, "/planning-poker/cerrar", url.Values{"ronda": {hu28ID(m.ronda1)}})
	rechazado(t, "cerrar la ronda 1 de «Login»", resp, cuerpo, web.MsgScrumMasterOnly)
	// Entonces no cambia nada: la ronda 1 sigue abierta, sin estimación, y «Perfil» sin rondas
	if got := m.texto(t, `SELECT status FROM poker_rounds WHERE id = $1`, m.ronda1); got != "Abierta" {
		t.Errorf("ronda 1 de «Login» = %q; se esperaba que siguiera Abierta", got)
	}
	if n := m.numero(t, `SELECT count(*) FROM poker_rounds WHERE story_id = $1`, m.perfil); n != 0 {
		t.Error("se abrió una ronda para «Perfil»")
	}

	// Con la ronda 1 cerrada por Ana, Beto intenta abrir una nueva ronda o registrar la estimación acordada
	ana := a.identificadoComo(t, m.alfa, m.ana, claveAna)
	if resp, _ := a.enviar(t, ana, "/planning-poker/cerrar", url.Values{"ronda": {hu28ID(m.ronda1)}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("Ana no pudo cerrar la ronda 1: %d", resp.StatusCode)
	}
	resp, cuerpo = a.enviar(t, beto, "/planning-poker/nueva-ronda", url.Values{"ronda": {hu28ID(m.ronda1)}})
	rechazado(t, "abrir una nueva ronda", resp, cuerpo, web.MsgScrumMasterOnly)
	resp, cuerpo = a.enviar(t, beto, "/planning-poker/estimacion", url.Values{"ronda": {hu28ID(m.ronda1)}, "valor": {"5"}})
	rechazado(t, "registrar la estimación acordada", resp, cuerpo, web.MsgScrumMasterOnly)
	if n := m.numero(t, `SELECT count(*) FROM poker_rounds WHERE story_id = $1`, m.login); n != 1 {
		t.Errorf("«Login» tiene %d rondas; no debía abrirse una nueva", n)
	}
	if puntos := m.numero(t, `SELECT story_points FROM user_stories WHERE id = $1`, m.login); puntos != 0 {
		t.Errorf("«Login» quedó con %d Story Points; no debía registrarse la estimación", puntos)
	}
}

// Escenario 7: Un integrante sí puede votar y trabajar.
func TestHU28_IntegranteVotaYTrabaja(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Beto
	beto := a.identificadoComo(t, m.alfa, m.beto, claveBeto)
	// Cuando voto en la ronda de «Login», cambio el estado de una historia en el Tablero o registro esfuerzo
	resp, _ := a.enviar(t, beto, "/planning-poker/votar", url.Values{"ronda": {hu28ID(m.ronda1)}, "valor": {"8"}})
	if resp.StatusCode == http.StatusForbidden {
		t.Errorf("votar: 403; Beto debía poder votar")
	}
	resp, _ = a.enviar(t, beto, "/tablero/estado", url.Values{
		"historia_id": {hu28ID(m.login)}, "accion": {"comenzar"}, "estado_actual": {"En Sprint"},
	})
	if resp.StatusCode == http.StatusForbidden {
		t.Errorf("Tablero: 403; Beto debía poder cambiar el estado")
	}
	resp, _ = a.enviar(t, beto, "/esfuerzo", url.Values{
		"historia": {hu28ID(m.login)}, "fecha": {"2026-10-30"}, "actividad": {"Programar el ingreso"}, "horas": {"2"},
	})
	if resp.StatusCode == http.StatusForbidden {
		t.Errorf("esfuerzo: 403; Beto debía poder registrar esfuerzo")
	}
	// Entonces la acción se hace normalmente
	if v := m.numero(t, `SELECT value FROM poker_votes WHERE round_id = $1 AND member_id = $2`, m.ronda1, m.beto); v != 8 {
		t.Errorf("voto de Beto = %d; se esperaba 8", v)
	}
	if got := m.texto(t, `SELECT status FROM user_stories WHERE id = $1`, m.login); got != "En progreso" {
		t.Errorf("estado de «Login» = %q; se esperaba En progreso", got)
	}
	if n := m.numero(t, `SELECT count(*) FROM effort_logs WHERE story_id = $1 AND member_id = $2`, m.login, m.beto); n != 1 {
		t.Errorf("cargas de esfuerzo de Beto = %d; se esperaba 1", n)
	}
}

// Escenario 8: Acción de Scrum Master sin identificarse.
func TestHU28_SinIdentificarse(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que no me identifiqué
	c := a.enProyecto(t, m.alfa)
	// Cuando intento cerrar el Sprint 2
	resp, _ := a.enviar(t, c, "/cierre", url.Values{"sprint_id": {hu28ID(m.sprint2)}})
	// Entonces se me pide identificarme y no cambia nada
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/contexto/integrante?volver_a=%2Fcierre" {
		t.Errorf("cerrar sin identificarse = %d a %q; se esperaba 303 a Identificarme con volver_a=/cierre",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := m.estadoSprint(t, m.sprint2); got != "Activo" {
		t.Errorf("Sprint 2 = %q; se esperaba que siguiera Activo", got)
	}
}

// Escenario 9: Proyecto sin Scrum Master.
func TestHU28_ProyectoSinScrumMaster(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que estoy identificado como Dani en «Beta»
	dani := a.identificadoComo(t, m.beta, m.dani, claveDani)
	// Cuando intento crear un Sprint
	resp, cuerpo := a.enviar(t, dani, "/sprints", url.Values{
		web.FieldGoal: {"Primeros reportes"}, web.FieldStartDate: {"2026-11-09"}, web.FieldEndDate: {"2026-11-22"},
		web.FieldStory: {hu28ID(m.reportes)},
	})
	// Entonces veo el mensaje y no cambia nada
	rechazado(t, "crear un Sprint en «Beta»", resp, cuerpo, web.MsgNoScrumMaster)
	if n := m.numero(t, `SELECT count(*) FROM sprints WHERE project_id = $1`, m.beta); n != 0 {
		t.Errorf("«Beta» tiene %d Sprints; no debía crearse ninguno", n)
	}
	if got := m.texto(t, `SELECT status FROM user_stories WHERE id = $1`, m.reportes); got != "Pendiente" {
		t.Errorf("«Reportes» = %q; se esperaba que siguiera Pendiente", got)
	}
}

// Escenario 10: Asignar el rol.
func TestHU28_AsignarRol(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	// Dado que Beto es "Integrante" de «Alfa» (y ya está identificado)
	beto := a.identificadoComo(t, m.alfa, m.beto, claveBeto)
	if resp, _ := a.enviar(t, beto, "/cierre", url.Values{"sprint_id": {hu28ID(m.sprint2)}}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("antes de asignar el rol, Beto debía recibir 403 al cerrar; obtuvo %d", resp.StatusCode)
	}
	// Cuando se le da el rol "Scrum Master" en «Alfa», como lo hace el comando de roles
	nombre, err := role.NewPostgreSQLStore(m.db).SetMemberRole(context.Background(), m.alfa, m.beto, role.RoleScrumMaster)
	if err != nil || nombre != "Beto" {
		t.Fatalf("SetMemberRole = %q, %v; se esperaba Beto", nombre, err)
	}
	// Entonces Beto puede cerrar el Sprint, sin volver a identificarse
	resp, _ := a.enviar(t, beto, "/cierre", url.Values{"sprint_id": {hu28ID(m.sprint2)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("cerrar como Scrum Master: estado %d; se esperaba 303", resp.StatusCode)
	}
	if got := m.estadoSprint(t, m.sprint2); got != "Finalizado" {
		t.Errorf("Sprint 2 = %q; se esperaba Finalizado", got)
	}
}

// Escenario 12: Error al verificar el rol.
func TestHU28_ErrorAlVerificarRol(t *testing.T) {
	m := nuevoMundoHU28(t)
	a := m.laAplicacion(t)
	ana := a.identificadoComo(t, m.alfa, m.ana, claveAna)
	// Dado un fallo al leer el rol del integrante
	a.falla.Store(true)
	// Cuando intento una acción de Scrum Master
	resp, cuerpo := a.enviar(t, ana, "/cierre", url.Values{"sprint_id": {hu28ID(m.sprint2)}})
	a.falla.Store(false)
	// Entonces veo la página de error de la aplicación y no cambia nada
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("estado = %d; se esperaba la página 503 de HU-20a", resp.StatusCode)
	}
	for _, interno := range []string{"pq:", "project_members", "rol no disponible"} {
		if strings.Contains(cuerpo, interno) {
			t.Errorf("la página de error expone un detalle interno (%q)", interno)
		}
	}
	if got := m.estadoSprint(t, m.sprint2); got != "Activo" {
		t.Errorf("Sprint 2 = %q; se esperaba que siguiera Activo", got)
	}
}
