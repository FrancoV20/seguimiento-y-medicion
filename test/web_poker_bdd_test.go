package test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-05a (T037, T049, T060, T069). Usan la API pública de web, el servidor
// real, el servicio real de Planning Poker y PostgreSQL real.

// pokerMundo: proyecto con Ana y Beto y la historia «Login»; proyecto vacío con Ana.
type pokerMundo struct {
	db              *sql.DB
	sello           string
	proyecto, vacio int64
	ana, beto       int64
	login           int64
	tituloLogin     string
}

func dadoUnEquipoQueEstima(t *testing.T) *pokerMundo {
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
	m := &pokerMundo{db: db, sello: fmt.Sprint(time.Now().UnixNano())}
	m.tituloLogin = "Login BDD05a " + m.sello
	t.Cleanup(m.limpiar)

	m.proyecto = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, "Poker BDD "+m.sello)
	m.vacio = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, "Poker BDD vacío "+m.sello)
	m.ana = m.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, "Ana BDD05a "+m.sello)
	m.beto = m.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, "Beto BDD05a "+m.sello)
	for _, pm := range [][2]int64{{m.proyecto, m.ana}, {m.proyecto, m.beto}, {m.vacio, m.ana}} {
		if _, err := db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, pm[0], pm[1]); err != nil {
			t.Fatalf("no se pudo preparar el escenario: %v", err)
		}
	}
	m.login = m.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Alta') RETURNING id`, m.proyecto, m.tituloLogin)
	return m
}

func (m *pokerMundo) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *pokerMundo) limpiar() {
	_, _ = m.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, m.proyecto, m.vacio)
	_, _ = m.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, m.proyecto, m.vacio)
	_, _ = m.db.Exec(`DELETE FROM members WHERE id IN ($1, $2)`, m.ana, m.beto)
}

func (m *pokerMundo) rondaAbierta(t *testing.T) int64 {
	t.Helper()
	return m.id(t, `SELECT id FROM poker_rounds WHERE story_id = $1 AND status = 'Abierta'`, m.login)
}

type pokerBDDApp struct{ servidor *httptest.Server }

func (m *pokerMundo) laAplicacion(t *testing.T) *pokerBDDApp {
	t.Helper()
	srv := web.NewServer(silencioso())
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(m.db))
	if err := web.RegisterPlanningPoker(srv, poker.NewService(poker.NewPostgreSQLStore(m.db)), web.NewPostgreSQLPokerCatalog(m.db)); err != nil {
		t.Fatalf("no se pudo registrar Planning Poker: %v", err)
	}
	app := &pokerBDDApp{servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	return app
}

// navegador sigue las redirecciones, como un navegador real.
func (a *pokerBDDApp) navegador(t *testing.T, proyecto, integrante int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	a.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(proyecto, 10)}})
	if integrante != 0 {
		a.enviar(t, c, "/contexto/integrante", url.Values{"integrante_id": {strconv.FormatInt(integrante, 10)}})
	}
	return c
}

func (a *pokerBDDApp) enviar(t *testing.T, c *http.Client, ruta string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+ruta, v)
	if err != nil {
		t.Fatalf("POST %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *pokerBDDApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func filaDe(cuerpo string, integrante int64) string {
	m := regexp.MustCompile(fmt.Sprintf(`<li data-integrante="%d">(.*?)</li>`, integrante)).FindStringSubmatch(cuerpo)
	if m == nil {
		return ""
	}
	return m[1]
}

func pokerDebeContener(t *testing.T, cuerpo string, partes ...string) {
	t.Helper()
	for _, p := range partes {
		if !strings.Contains(cuerpo, p) {
			t.Errorf("la pantalla debía mostrar %q", p)
		}
	}
}

// Escenario 1: abrir una ronda.
func TestBDD_HU05a_AbrirRonda(t *testing.T) {
	// Dado una historia del proyecto y un integrante identificado
	m := dadoUnEquipoQueEstima(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.proyecto, m.ana)

	// Cuando abro una ronda para la historia
	resp, cuerpo := app.enviar(t, ana, "/planning-poker/abrir", url.Values{"historia": {strconv.FormatInt(m.login, 10)}})

	// Entonces veo el mensaje, la ronda 1 y a todos los integrantes "Sin votar"
	if resp.Request.URL.Path != "/planning-poker" {
		t.Fatalf("se esperaba volver a la pantalla, se obtuvo %s", resp.Request.URL)
	}
	pokerDebeContener(t, cuerpo, "Ronda abierta para la historia «"+m.tituloLogin+"»", "Ronda 1")
	for _, id := range []int64{m.ana, m.beto} {
		if !strings.Contains(filaDe(cuerpo, id), web.LabelPokerNotVoted) {
			t.Errorf("el integrante %d debía figurar «Sin votar»", id)
		}
	}
}

// Escenario 9: historia con ronda abierta.
func TestBDD_HU05a_HistoriaConRondaAbierta(t *testing.T) {
	// Dado una historia que ya tiene una ronda abierta
	m := dadoUnEquipoQueEstima(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.proyecto, m.ana)
	app.enviar(t, ana, "/planning-poker/abrir", url.Values{"historia": {strconv.FormatInt(m.login, 10)}})

	// Cuando intento abrir otra ronda para ella
	resp, cuerpo := app.enviar(t, ana, "/planning-poker/abrir", url.Values{"historia": {strconv.FormatInt(m.login, 10)}})

	// Entonces veo el rechazo y sigue habiendo una sola ronda
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d", resp.StatusCode)
	}
	pokerDebeContener(t, cuerpo, web.MsgPokerRoundAlreadyOpen)
	var n int
	_ = m.db.QueryRow(`SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, m.login).Scan(&n)
	if n != 1 {
		t.Errorf("debía haber una sola ronda, hay %d", n)
	}
}
