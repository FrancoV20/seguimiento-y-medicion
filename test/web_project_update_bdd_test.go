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
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-11 (T044, T059, T067 y T075). Usan la API pública de web, el servidor
// real, los servicios reales de HU-01 y HU-03 y PostgreSQL real.

// modMundo: «Alfa» (2026-03-01 a 2026-06-30) con Ana y Beto y el Sprint 2 activo
// (2026-06-01 a 2026-06-14) con una historia; Carla sin proyecto.
type modMundo struct {
	db               *sql.DB
	sello            string
	alfa             int64
	nombreAlfa       string
	ana, beto, carla int64
	nombreBeto       string
	nombreCarla      string
	historia         int64
}

func dadoUnProyectoParaModificar(t *testing.T) *modMundo {
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

	m := &modMundo{db: db, sello: fmt.Sprint(time.Now().UnixNano())}
	m.nombreAlfa = "Alfa BDD11 " + m.sello
	m.nombreBeto, m.nombreCarla = "Beto BDD11 "+m.sello, "Carla BDD11 "+m.sello
	t.Cleanup(m.limpiar)

	m.alfa = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-03-01', '2026-06-30') RETURNING id`, m.nombreAlfa)
	integrante := func(nombre string, enAlfa bool) int64 {
		id := m.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, nombre)
		if enAlfa {
			m.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, m.alfa, id)
		}
		return id
	}
	m.ana = integrante("Ana BDD11 "+m.sello, true)
	m.beto = integrante(m.nombreBeto, true)
	m.carla = integrante(m.nombreCarla, false)
	sprint := m.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date) VALUES ($1, 2, 'Objetivo BDD11', '2026-06-01', '2026-06-14') RETURNING id`, m.alfa)
	m.historia = m.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id) VALUES ($1, 'BDD11 historia', 'd', 'Alta', 'En Sprint', $2) RETURNING id`, m.alfa, sprint)
	m.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprint, m.historia)
	return m
}

func (m *modMundo) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *modMundo) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := m.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
}

func (m *modMundo) limpiar() {
	_, _ = m.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, m.alfa)
	_, _ = m.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, m.alfa)
	_, _ = m.db.Exec(`DELETE FROM projects WHERE id = $1`, m.alfa)
	_, _ = m.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3)`, m.ana, m.beto, m.carla)
}

func (m *modMundo) estado(t *testing.T) string {
	t.Helper()
	var s string
	if err := m.db.QueryRow(`SELECT name || '|' || start_date || '|' || end_date || '|' ||
		COALESCE((SELECT string_agg(member_id::text, ',' ORDER BY member_id) FROM project_members WHERE project_id = p.id), '')
		FROM projects p WHERE id = $1`, m.alfa).Scan(&s); err != nil {
		t.Fatalf("no se pudo leer «Alfa»: %v", err)
	}
	return s
}

// modStoreQueFalla es el almacenamiento real salvo que la modificación falla.
type modStoreQueFalla struct{ project.ProjectStore }

func (modStoreQueFalla) Update(context.Context, project.ProjectUpdate) error {
	return errors.New("disco lleno")
}

type modApp struct {
	servidor *httptest.Server
	m        *modMundo
}

func (m *modMundo) laAplicacion(t *testing.T, guardadoFalla bool) *modApp {
	t.Helper()
	srv := web.NewServer(silencioso())
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(m.db))
	var store project.ProjectStore = project.NewPostgreSQLProjectStore(m.db)
	if guardadoFalla {
		store = modStoreQueFalla{store}
	}
	if err := web.RegisterProjects(srv, project.NewProjectService(store), web.NewPostgreSQLProjectCatalog(m.db)); err != nil {
		t.Fatalf("no se pudo registrar Proyectos: %v", err)
	}
	app := &modApp{servidor: httptest.NewServer(srv.Handler()), m: m}
	t.Cleanup(app.servidor.Close)
	return app
}

// navegador sigue las redirecciones, como un navegador real.
func (a *modApp) navegador() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func (a *modApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *modApp) enviar(t *testing.T, c *http.Client, ruta string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+ruta, v)
	if err != nil {
		t.Fatalf("POST %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *modApp) modificar(t *testing.T, c *http.Client, nombre, inicio, fin string, integrantes ...int64) (*http.Response, string) {
	t.Helper()
	v := url.Values{"nombre": {nombre}, "fecha_inicio": {inicio}, "fecha_fin": {fin}}
	for _, id := range integrantes {
		v.Add("integrantes", strconv.FormatInt(id, 10))
	}
	return a.enviar(t, c, fmt.Sprintf("/proyectos/%d/modificar", a.m.alfa), v)
}

func modContiene(t *testing.T, cuerpo string, partes ...string) {
	t.Helper()
	for _, p := range partes {
		if !strings.Contains(cuerpo, p) {
			t.Errorf("la pantalla debía mostrar %q", p)
		}
	}
}

// Escenario: Modificación exitosa.
func TestEscenarioModificarProyecto_ModificacionExitosa(t *testing.T) {
	// Dado un proyecto existente
	m := dadoUnProyectoParaModificar(t)
	app := m.laAplicacion(t, false)
	c := app.navegador()

	// Cuando cambio el nombre o las fechas con valores válidos y guardo
	nuevo := "Alfa renombrado " + m.sello
	resp, cuerpo := app.modificar(t, c, nuevo, "2026-02-01", "2026-07-31", m.ana, m.beto)

	// Entonces veo el mensaje, vuelvo a la lista y el proyecto muestra los valores nuevos
	if resp.Request.URL.Path != "/proyectos" {
		t.Fatalf("se esperaba volver a la lista, se obtuvo %s", resp.Request.URL)
	}
	modContiene(t, cuerpo, web.MsgProjectUpdated, nuevo, "<td>01/02/2026</td>", "<td>31/07/2026</td>")
}

// Escenario: Nombre visible en la estructura común.
func TestEscenarioModificarProyecto_NombreVisibleEnEstructuraComun(t *testing.T) {
	// Dado un proyecto que es mi contexto de trabajo
	m := dadoUnProyectoParaModificar(t)
	app := m.laAplicacion(t, false)
	c := app.navegador()
	app.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(m.alfa, 10)}})

	// Cuando cambio su nombre y guardo
	nuevo := "Alfa visible " + m.sello
	app.modificar(t, c, nuevo, "2026-03-01", "2026-06-30", m.ana, m.beto)

	// Entonces el nombre nuevo se ve en la estructura común sin elegir el proyecto otra vez
	_, cuerpo := app.abrir(t, c, "/")
	modContiene(t, cuerpo, "Proyecto: <strong>"+nuevo+"</strong>")
}

// Escenario: Agregar integrante.
func TestEscenarioModificarProyecto_AgregarIntegrante(t *testing.T) {
	// Dado un proyecto existente y un integrante del sistema que no pertenece a él
	m := dadoUnProyectoParaModificar(t)
	app := m.laAplicacion(t, false)
	c := app.navegador()

	// Cuando lo marco y guardo
	app.modificar(t, c, m.nombreAlfa, "2026-03-01", "2026-06-30", m.ana, m.beto, m.carla)

	// Entonces aparece en el selector de identificación y puede cargar esfuerzo
	app.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(m.alfa, 10)}})
	_, cuerpo := app.abrir(t, c, "/contexto/integrante")
	modContiene(t, cuerpo, m.nombreCarla)
	hoy := time.Now()
	res := effort.NewService(effort.NewPostgreSQLEffortStore(m.db), func() time.Time { return hoy }).LogEffort(context.Background(), effort.LogEffortRequest{
		StoryID: strconv.FormatInt(m.historia, 10), MemberID: strconv.FormatInt(m.carla, 10),
		WorkDate: hoy, Activity: "Primera carga", Hours: "1",
	})
	if !res.Succeeded() {
		t.Errorf("Carla debía poder cargar esfuerzo: %+v", res)
	}
}

// Escenario: Quitar un integrante con horas registradas.
func TestEscenarioModificarProyecto_QuitarIntegranteConHoras(t *testing.T) {
	// Dado un integrante con horas registradas en historias de este proyecto (cargadas con HU-03)
	m := dadoUnProyectoParaModificar(t)
	hoy := time.Now()
	res := effort.NewService(effort.NewPostgreSQLEffortStore(m.db), func() time.Time { return hoy }).LogEffort(context.Background(), effort.LogEffortRequest{
		StoryID: strconv.FormatInt(m.historia, 10), MemberID: strconv.FormatInt(m.beto, 10),
		WorkDate: hoy, Activity: "Desarrollo", Hours: "3",
	})
	if !res.Succeeded() {
		t.Fatalf("no se pudieron cargar las horas de Beto: %+v", res)
	}
	app := m.laAplicacion(t, false)
	antes := m.estado(t)

	// Cuando lo desmarco y guardo
	resp, cuerpo := app.modificar(t, app.navegador(), "Otro nombre", "2026-03-01", "2026-06-30", m.ana, m.carla)

	// Entonces veo el mensaje junto a los integrantes y no se guarda ningún cambio
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d", resp.StatusCode)
	}
	modContiene(t, cuerpo, `id="integrantes-error"><span class="field-error-label">Error:</span> `+
		fmt.Sprintf(web.MsgMemberHasEffort, m.nombreBeto))
	if despues := m.estado(t); despues != antes {
		t.Errorf("no debía cambiar nada: antes %s, después %s", antes, despues)
	}
}

// Escenario: Fechas que dejan Sprints fuera.
func TestEscenarioModificarProyecto_FechasQueDejanSprintsFuera(t *testing.T) {
	// Dado un proyecto con Sprints creados
	m := dadoUnProyectoParaModificar(t)
	app := m.laAplicacion(t, false)
	antes := m.estado(t)

	// Cuando acorto sus fechas de modo que un Sprint queda fuera y guardo
	resp, cuerpo := app.modificar(t, app.navegador(), m.nombreAlfa, "2026-03-01", "2026-06-10", m.ana, m.beto)

	// Entonces veo el mensaje junto a la fecha que lo causa, los datos se conservan y no cambia nada
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d", resp.StatusCode)
	}
	modContiene(t, cuerpo, `id="fecha_fin-error"><span class="field-error-label">Error:</span> `+
		fmt.Sprintf(web.MsgSprintOutOfRange, 2), `name="fecha_fin" value="2026-06-10"`)
	if despues := m.estado(t); despues != antes {
		t.Errorf("no debía cambiar nada: antes %s, después %s", antes, despues)
	}
}

// Escenario: Error al guardar.
func TestEscenarioModificarProyecto_ErrorAlGuardar(t *testing.T) {
	// Dado que completé los datos correctamente
	m := dadoUnProyectoParaModificar(t)
	app := m.laAplicacion(t, true)
	antes := m.estado(t)

	// Cuando falla el guardado
	resp, cuerpo := app.modificar(t, app.navegador(), "Nombre nuevo", "2026-03-01", "2026-06-30", m.ana, m.carla)

	// Entonces veo el error, los datos se conservan, no hay éxito y no cambia nada
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("se esperaba 500, se obtuvo %d", resp.StatusCode)
	}
	modContiene(t, cuerpo, web.MsgOperationFailed, `name="nombre" value="Nombre nuevo"`)
	if strings.Contains(cuerpo, web.MsgProjectUpdated) || strings.Contains(cuerpo, "disco lleno") {
		t.Error("no debía mostrarse éxito ni el detalle interno")
	}
	if despues := m.estado(t); despues != antes {
		t.Errorf("no debía cambiar nada: antes %s, después %s", antes, despues)
	}
}
