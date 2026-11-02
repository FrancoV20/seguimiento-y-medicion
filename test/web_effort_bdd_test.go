package test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-24 (Issue: Interfaz web de registro de esfuerzo), T018, T028 y T036.
// Usan la API pública de web, el servidor real, el servicio real de HU-03 y PostgreSQL real.
// Solo la falla de almacenamiento usa un store que falla al guardar.

var esfHoy = time.Date(2026, time.October, 20, 11, 0, 0, 0, time.Local)

func esfReloj() time.Time { return esfHoy }

// esfMundo: proyecto con Sprint activo (Ana y Beto), proyecto sin Sprint activo (Ana) y una
// historia de otro proyecto.
type esfMundo struct {
	db                          *sql.DB
	sello                       string
	conSprint, sinSprint, otro  int64
	ana, beto                   int64
	nombreAna, nombreBeto       string
	historia, conHoras, ajena   int64
	pendiente                   int64
	tituloHistoria, tituloAjena string
}

func dadoUnMundoDeEsfuerzo(t *testing.T) *esfMundo {
	t.Helper()
	dsn := os.Getenv(web.EnvDatabaseURL)
	if dsn == "" {
		t.Skip("DATABASE_URL no definida: se omite el escenario con PostgreSQL")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	m := &esfMundo{db: db, sello: fmt.Sprint(time.Now().UnixNano())}
	m.nombreAna, m.nombreBeto = "Ana BDD24 "+m.sello, "Beto BDD24 "+m.sello
	m.tituloHistoria, m.tituloAjena = "BDD24 historia "+m.sello, "BDD24 ajena "+m.sello
	t.Cleanup(m.limpiar)

	proyecto := func(nombre string) int64 {
		return m.id(t, `INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, nombre+" "+m.sello)
	}
	integrante := func(nombre string, proyectos ...int64) int64 {
		id := m.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, nombre)
		for _, p := range proyectos {
			m.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, p, id)
		}
		return id
	}
	sprint := func(p int64) int64 {
		return m.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, 1, 'Objetivo BDD24', DATE '2026-10-01', DATE '2026-10-31', 'Activo') RETURNING id`, p)
	}
	historia := func(p int64, titulo, estado string, s int64) int64 {
		var activo any
		if s != 0 {
			activo = s
		}
		id := m.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
			VALUES ($1, $2, 'Descripción BDD24', 'Alta', $3, $4) RETURNING id`, p, titulo, estado, activo)
		if s != 0 {
			m.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, s, id)
		}
		return id
	}

	m.conSprint, m.sinSprint, m.otro = proyecto("BDD24 con Sprint"), proyecto("BDD24 sin Sprint"), proyecto("BDD24 otro")
	m.ana = integrante(m.nombreAna, m.conSprint, m.sinSprint)
	m.beto = integrante(m.nombreBeto, m.conSprint)
	s1, s2 := sprint(m.conSprint), sprint(m.otro)
	m.historia = historia(m.conSprint, m.tituloHistoria, "En Sprint", s1)
	m.conHoras = historia(m.conSprint, "BDD24 con horas "+m.sello, "En progreso", s1)
	m.pendiente = historia(m.conSprint, "BDD24 pendiente "+m.sello, "Pendiente", 0)
	m.ajena = historia(m.otro, m.tituloAjena, "En progreso", s2)
	m.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours) VALUES
		($1, $2, DATE '2026-10-15', 'Análisis BDD24', 2.5),
		($1, $3, DATE '2026-10-16', 'Desarrollo BDD24', 4)`, m.conHoras, m.ana, m.beto)
	return m
}

func (m *esfMundo) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *esfMundo) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := m.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
}

func (m *esfMundo) limpiar() {
	ps := []any{m.conSprint, m.sinSprint, m.otro}
	_, _ = m.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2, $3))`, ps...)
	_, _ = m.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2, $3)`, ps...)
	_, _ = m.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2, $3)`, ps...)
	_, _ = m.db.Exec(`DELETE FROM members WHERE id IN ($1, $2)`, m.ana, m.beto)
}

func (m *esfMundo) cargas(t *testing.T) int {
	t.Helper()
	var n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM effort_logs WHERE story_id IN
		(SELECT id FROM user_stories WHERE project_id IN ($1, $2, $3))`, m.conSprint, m.sinSprint, m.otro).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar las cargas: %v", err)
	}
	return n
}

// esfStoreQueFalla es el store real salvo que no puede guardar.
type esfStoreQueFalla struct{ effort.EffortStore }

func (esfStoreQueFalla) SaveEntry(context.Context, effort.EffortEntry) (effort.EffortEntry, effort.Hours, error) {
	return effort.EffortEntry{}, 0, errors.New("disco lleno")
}

type esfApp struct {
	servidor *httptest.Server
	mundo    *esfMundo
}

func (m *esfMundo) laAplicacion(t *testing.T, guardadoFalla bool) *esfApp {
	t.Helper()
	srv := web.NewServer(silencioso())
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(m.db))
	var store effort.EffortStore = effort.NewPostgreSQLEffortStore(m.db)
	if guardadoFalla {
		store = esfStoreQueFalla{store}
	}
	acciones := web.NewEffortActions(effort.NewService(store, esfReloj), esfReloj)
	if err := web.RegisterEffort(srv, acciones, web.NewPostgreSQLEffortCatalog(m.db), esfReloj); err != nil {
		t.Fatalf("no se pudo registrar la sección: %v", err)
	}
	app := &esfApp{servidor: httptest.NewServer(srv.Handler()), mundo: m}
	t.Cleanup(app.servidor.Close)
	return app
}

// navegador sigue las redirecciones, como un navegador real.
func (a *esfApp) navegador() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func (a *esfApp) enviar(t *testing.T, c *http.Client, ruta string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+ruta, v)
	if err != nil {
		t.Fatalf("POST %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *esfApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

// dadoQueTrabajoEn elige el proyecto y, si integrante no es 0, se identifica (HU-20b).
func (a *esfApp) dadoQueTrabajoEn(t *testing.T, proyecto, integrante int64) *http.Client {
	t.Helper()
	c := a.navegador()
	a.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(proyecto, 10)}})
	if integrante != 0 {
		a.enviar(t, c, "/contexto/integrante", identificacionDePrueba(t, a.mundo.db, integrante))
	}
	return c
}

func esfCarga(historia, fecha, actividad, horas string) url.Values {
	return url.Values{"historia": {historia}, "fecha": {fecha}, "actividad": {actividad}, "horas": {horas}}
}

func esfID(id int64) string { return strconv.FormatInt(id, 10) }

func esfDebeContener(t *testing.T, cuerpo string, partes ...string) {
	t.Helper()
	for _, p := range partes {
		if !strings.Contains(cuerpo, p) {
			t.Errorf("la pantalla debía mostrar %q", p)
		}
	}
}

func esfNoDebeContener(t *testing.T, cuerpo string, partes ...string) {
	t.Helper()
	for _, p := range partes {
		if strings.Contains(cuerpo, p) {
			t.Errorf("la pantalla no debía mostrar %q", p)
		}
	}
}

const esfFormulario = `<form method="post" action="/esfuerzo">`

// Escenario 1: carga exitosa.
func TestEscenarioEsfuerzo_CargaExitosa(t *testing.T) {
	// Given una historia del Sprint activo y un integrante identificado
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)

	// When abro la pantalla, completo fecha (por defecto hoy), actividad y horas y guardo
	_, cuerpo := app.abrir(t, c, "/esfuerzo")
	esfDebeContener(t, cuerpo, `name="fecha" value="2026-10-20"`)
	resp, cuerpo := app.enviar(t, c, "/esfuerzo", esfCarga(esfID(m.historia), "2026-10-20", "Escribir pruebas", "3,5"))

	// Then veo el mensaje, el acumulado actualizado y la carga a mi nombre
	if resp.StatusCode != http.StatusOK || resp.Request.URL.RequestURI() != "/esfuerzo?historia="+esfID(m.historia) {
		t.Fatalf("se esperaba volver a la pantalla de la historia, se obtuvo %d %s", resp.StatusCode, resp.Request.URL)
	}
	esfDebeContener(t, cuerpo, web.MsgEffortRecorded, "<td>3,5</td>", "Total: <strong>3,5</strong>", m.nombreAna, "Escribir pruebas")
	var integrante int64
	if err := m.db.QueryRow(`SELECT member_id FROM effort_logs WHERE story_id = $1`, m.historia).Scan(&integrante); err != nil || integrante != m.ana {
		t.Errorf("la carga debía quedar a nombre de Ana, se obtuvo %d (%v)", integrante, err)
	}
}

// Escenario 5: historias disponibles.
func TestEscenarioEsfuerzo_SoloHistoriasDelSprintActivo(t *testing.T) {
	// Given historias en distintos estados de Sprint
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)

	// When abro el selector
	_, cuerpo := app.abrir(t, c, "/esfuerzo")

	// Then solo aparecen las del Sprint activo del proyecto
	esfDebeContener(t, cuerpo, `<option value="`+esfID(m.historia)+`"`, `<option value="`+esfID(m.conHoras)+`"`)
	esfNoDebeContener(t, cuerpo, `<option value="`+esfID(m.pendiente)+`"`, `<option value="`+esfID(m.ajena)+`"`, m.tituloAjena)
}

// Escenario 6: sin Sprint activo.
func TestEscenarioEsfuerzo_SinSprintActivo(t *testing.T) {
	// Given un proyecto sin Sprint activo
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.sinSprint, m.ana)

	// When abro "Registro de esfuerzo"
	_, cuerpo := app.abrir(t, c, "/esfuerzo")

	// Then veo "No hay un Sprint activo" y no veo el formulario
	esfDebeContener(t, cuerpo, web.MsgEffortNoSprint)
	esfNoDebeContener(t, cuerpo, esfFormulario)
}

// Escenario 7: sin integrante identificado.
func TestEscenarioEsfuerzo_SinIntegranteIdentificado(t *testing.T) {
	// Given que no me identifiqué como integrante
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, 0)
	antes := m.cargas(t)

	// When entro a "Registro de esfuerzo"
	resp, cuerpo := app.abrir(t, c, "/esfuerzo")

	// Then se me pide identificarme y no se registra nada
	if resp.Request.URL.Path != "/contexto/integrante" {
		t.Fatalf("se esperaba la pantalla de identificación, se obtuvo %s", resp.Request.URL)
	}
	esfDebeContener(t, cuerpo, web.MsgMemberRequired)

	// And al identificarme (con mi contraseña, HU-27) vuelvo a la pantalla
	configurarClave(t, m.db, m.ana, claveDePrueba)
	resp, cuerpo = app.enviar(t, c, "/contexto/integrante", url.Values{
		"integrante_id": {esfID(m.ana)}, "contrasena": {claveDePrueba}, "volver_a": {resp.Request.URL.Query().Get("volver_a")},
	})
	if resp.Request.URL.Path != "/esfuerzo" {
		t.Fatalf("después de identificarme se esperaba volver a /esfuerzo, se obtuvo %s", resp.Request.URL)
	}
	esfDebeContener(t, cuerpo, esfFormulario)
	if despues := m.cargas(t); despues != antes {
		t.Errorf("no debía registrarse nada: antes %d, después %d", antes, despues)
	}
}

// Escenario 2: errores juntos.
func TestEscenarioEsfuerzo_ErroresJuntos(t *testing.T) {
	// Given un formulario con varios campos inválidos
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)
	antes := m.cargas(t)

	// When guardo
	resp, cuerpo := app.enviar(t, c, "/esfuerzo", esfCarga(esfID(m.historia), "", "", "-2"))

	// Then veo todos los mensajes de HU-03 junto a su campo, con los datos conservados
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("se esperaba 422, se obtuvo %d", resp.StatusCode)
	}
	esfDebeContener(t, cuerpo,
		`id="horas-error"><span class="field-error-label">Error:</span> `+effort.ErrInvalidHours.Error(),
		`id="actividad-error"><span class="field-error-label">Error:</span> `+effort.ErrActivityRequired.Error(),
		`id="fecha-error"><span class="field-error-label">Error:</span> `+effort.ErrDateRequired.Error(),
		`name="horas" value="-2"`, `<option value="`+esfID(m.historia)+`" selected>`)
	if despues := m.cargas(t); despues != antes {
		t.Errorf("no debía registrarse nada: antes %d, después %d", antes, despues)
	}
}

// Escenario 8: historia no disponible.
func TestEscenarioEsfuerzo_HistoriaNoDisponible(t *testing.T) {
	// Given una historia que no pertenece al Sprint activo del proyecto
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)
	antes := m.cargas(t)

	for _, historia := range []int64{m.ajena, m.pendiente} {
		// When se envía una carga sobre ella
		resp, cuerpo := app.enviar(t, c, "/esfuerzo", esfCarga(esfID(historia), "2026-10-20", "Revisión", "1"))

		// Then veo el error junto a la historia, con los datos conservados, y no se registra nada
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("se esperaba 422, se obtuvo %d", resp.StatusCode)
		}
		esfDebeContener(t, cuerpo,
			`id="historia-error"><span class="field-error-label">Error:</span> `+web.MsgEffortStoryNotInSprint,
			`name="actividad" value="Revisión"`, `name="horas" value="1"`)
	}
	if despues := m.cargas(t); despues != antes {
		t.Errorf("no debía registrarse nada: antes %d, después %d", antes, despues)
	}
}

// Escenario 9: falla de almacenamiento.
func TestEscenarioEsfuerzo_FallaDeAlmacenamiento(t *testing.T) {
	// Given una carga válida y un almacenamiento que no puede guardar
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, true)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)
	antes := m.cargas(t)

	// When guardo
	resp, cuerpo := app.enviar(t, c, "/esfuerzo", esfCarga(esfID(m.historia), "2026-10-20", "Integración", "2"))

	// Then veo "La operación no pudo completarse", los datos se conservan y no se muestra éxito
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("se esperaba 500, se obtuvo %d", resp.StatusCode)
	}
	esfDebeContener(t, cuerpo, web.MsgOperationFailed, `name="actividad" value="Integración"`, `name="horas" value="2"`)
	esfNoDebeContener(t, cuerpo, web.MsgEffortRecorded, "disco lleno")
	if despues := m.cargas(t); despues != antes {
		t.Errorf("no debía registrarse nada: antes %d, después %d", antes, despues)
	}
}

// Escenario 3: historial de la historia.
func TestEscenarioEsfuerzo_HistorialDeLaHistoria(t *testing.T) {
	// Given una historia con registros de dos integrantes
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)

	// When la abro
	_, cuerpo := app.abrir(t, c, "/esfuerzo?historia="+esfID(m.conHoras))

	// Then veo las cargas, las más recientes primero, y el total
	esfDebeContener(t, cuerpo, m.nombreAna, m.nombreBeto, "<td>16/10/2026</td>", "<td>15/10/2026</td>", "Total: <strong>6,5</strong>")
	if strings.Index(cuerpo, "Desarrollo BDD24") > strings.Index(cuerpo, "Análisis BDD24") {
		t.Error("la carga más reciente (Desarrollo, 16/10) debía verse primero")
	}
}

// Escenario 4: historia sin registros.
func TestEscenarioEsfuerzo_HistoriaSinRegistros(t *testing.T) {
	// Given una historia del Sprint activo sin horas cargadas
	m := dadoUnMundoDeEsfuerzo(t)
	app := m.laAplicacion(t, false)
	c := app.dadoQueTrabajoEn(t, m.conSprint, m.ana)

	// When la abro
	_, cuerpo := app.abrir(t, c, "/esfuerzo?historia="+esfID(m.historia))

	// Then veo el mensaje y el total en cero
	esfDebeContener(t, cuerpo, web.MsgEffortNoEntries, "Total: <strong>0</strong>")
}
