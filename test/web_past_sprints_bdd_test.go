package test

import (
	"database/sql"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-16 (T012 y T020). Usan la API pública de web, el servidor real, HU-07 real y
// PostgreSQL real, con datos propios cargados como los deja el cierre de HU-15.

// anterioresMundo: «Alfa» con los Sprints 1 y 2 finalizados y el 3 en curso; «Beta» sin Sprints.
type anterioresMundo struct {
	db         *sql.DB
	alfa, beta int64
	sprints    map[int]int64 // número de Sprint de «Alfa» → id
	historias  map[string]int64
}

func dadoElProyectoAlfaConSprintsCerrados(t *testing.T) *anterioresMundo {
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
	m := &anterioresMundo{db: db, sprints: map[int]int64{}, historias: map[string]int64{}}
	t.Cleanup(m.limpiar)

	m.alfa = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-08-18', '2026-11-30') RETURNING id`)
	m.beta = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-10-01', '2026-12-31') RETURNING id`)
	for _, h := range []struct {
		titulo string
		puntos int
	}{{"Alta", 3}, {"Login", 5}, {"Tablero", 3}, {"Perfil", 2}, {"Reportes", 8}, {"Exportar", 5}} {
		m.historias[h.titulo] = m.id(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points)
			VALUES ($1, $2, 'd', 'Media', $3) RETURNING id`, m.alfa, h.titulo, h.puntos)
	}
	m.sprint(t, 1, "Dar de alta usuarios", "2026-08-18", "2026-08-31", "Finalizado")
	m.cerrada(t, 1, "Alta", "Hecho", 3)
	m.sprint(t, 2, "Gestionar perfiles y reportes", "2026-09-01", "2026-09-14", "Finalizado")
	m.cerrada(t, 2, "Login", "Hecho", 5)
	m.cerrada(t, 2, "Tablero", "Hecho", 3)
	m.cerrada(t, 2, "Perfil", "Hecho", 2)
	m.cerrada(t, 2, "Reportes", "En progreso", 8)
	m.sprint(t, 3, "Exportar e informar", "2026-09-15", "2026-09-28", "Activo")
	m.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, m.sprints[3], m.historias["Exportar"])
	m.exec(t, `UPDATE user_stories SET status = 'En progreso', active_sprint_id = $1 WHERE id = $2`, m.sprints[3], m.historias["Exportar"])
	return m
}

func (m *anterioresMundo) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *anterioresMundo) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := m.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
}

func (m *anterioresMundo) sprint(t *testing.T, numero int, objetivo, inicio, fin, estado string) {
	t.Helper()
	m.sprints[numero] = m.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, m.alfa, numero, objetivo, inicio, fin, estado)
}

// cerrada deja la historia como la deja el cierre de HU-15: su fila en el registro y, si quedó
// "Hecho", asignada al Sprint; si no, "Pendiente" en el Backlog.
func (m *anterioresMundo) cerrada(t *testing.T, numero int, titulo, estado string, puntos int) {
	t.Helper()
	sprint, historia := m.sprints[numero], m.historias[titulo]
	m.exec(t, `INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points) VALUES ($1, $2, $3, $4)`,
		sprint, historia, estado, puntos)
	if estado == "Hecho" {
		m.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprint, historia)
		m.exec(t, `UPDATE user_stories SET status = 'Hecho', active_sprint_id = $1 WHERE id = $2`, sprint, historia)
	}
}

func (m *anterioresMundo) limpiar() {
	for _, p := range []int64{m.alfa, m.beta} {
		_, _ = m.db.Exec(`DELETE FROM sprint_story_results WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`, p)
		_, _ = m.db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`, p)
		_, _ = m.db.Exec(`DELETE FROM sprint_metrics WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`, p)
		_, _ = m.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, p)
		_, _ = m.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, p)
		_, _ = m.db.Exec(`DELETE FROM projects WHERE id = $1`, p)
	}
}

type anterioresApp struct{ servidor *httptest.Server }

func (m *anterioresMundo) laAplicacion(t *testing.T) *anterioresApp {
	t.Helper()
	srv := web.NewServer(silencioso())
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(m.db))
	if err := web.RegisterPastSprints(srv, web.NewPostgreSQLPastSprintCatalog(m.db),
		metrics.NewService(metrics.NewPostgreSQLMetricsStore(m.db))); err != nil {
		t.Fatalf("no se pudo registrar Sprints anteriores: %v", err)
	}
	app := &anterioresApp{servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	return app
}

// navegador elige el proyecto y sigue las redirecciones, como un navegador real.
func (a *anterioresApp) navegador(t *testing.T, proyecto int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	resp, err := c.PostForm(a.servidor.URL+"/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(proyecto, 10)}})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto: %v", err)
	}
	formCuerpo(t, resp)
	return c
}

func (a *anterioresApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func debeVerse(t *testing.T, cuerpo string, partes ...string) {
	t.Helper()
	for _, p := range partes {
		if !strings.Contains(cuerpo, p) {
			t.Errorf("la pantalla debía mostrar %q", p)
		}
	}
}

// Escenarios 1 y 2: listado de Sprints finalizados; el Sprint activo no aparece.
func TestBDD_HU16_ListadoDeSprintsFinalizados(t *testing.T) {
	// Dado el proyecto «Alfa», con los Sprints 1 y 2 finalizados y el 3 en curso
	m := dadoElProyectoAlfaConSprintsCerrados(t)
	app := m.laAplicacion(t)

	// Cuando abro "Sprints anteriores"
	resp, cuerpo := app.abrir(t, app.navegador(t, m.alfa), "/sprints-anteriores")

	// Entonces veo el Sprint 2 y después el Sprint 1, cada uno con su número, su objetivo y sus
	// fechas, y el Sprint 3 no figura
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	debeVerse(t, cuerpo, "Sprint 2", "Gestionar perfiles y reportes", "Del 01/09/2026 al 14/09/2026",
		"Sprint 1", "Dar de alta usuarios", "Del 18/08/2026 al 31/08/2026")
	if strings.Index(cuerpo, "Sprint 2") > strings.Index(cuerpo, "Sprint 1") {
		t.Error("el Sprint 2 debía aparecer antes que el Sprint 1")
	}
	if strings.Contains(cuerpo, "Sprint 3") || strings.Contains(cuerpo, "Exportar e informar") {
		t.Error("el Sprint 3, en curso, no debía figurar")
	}
}

// Escenario 6: sin Sprints finalizados.
func TestBDD_HU16_SinSprintsFinalizados(t *testing.T) {
	// Dado el proyecto «Beta», sin Sprints finalizados
	m := dadoElProyectoAlfaConSprintsCerrados(t)
	app := m.laAplicacion(t)

	// Cuando abro "Sprints anteriores"
	_, cuerpo := app.abrir(t, app.navegador(t, m.beta), "/sprints-anteriores")

	// Entonces veo "Aún no hay Sprints anteriores"
	debeVerse(t, cuerpo, "Aún no hay Sprints anteriores")
}
