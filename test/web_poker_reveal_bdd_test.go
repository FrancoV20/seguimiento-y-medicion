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

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-05b (T019). Usan la API pública de web, el servidor real, el servicio real
// de Planning Poker y PostgreSQL real. Las rondas se cierran desde la pantalla.

// revelacionMundo: proyecto «Alfa» con Ana, Beto y Caro y las rondas abiertas del Issue; proyecto
// «Beta» con Dani, «Tablero» (ronda abierta, Dani 8) y «Agenda» (ronda cerrada sin votos).
type revelacionMundo struct {
	db                    *sql.DB
	alfa, beta            int64
	ana, beto, caro, dani int64
	rondas                map[string]int64 // ronda 1 por título de la historia
}

func dadoElEquipoAlfaConSusVotos(t *testing.T) *revelacionMundo {
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
	m := &revelacionMundo{db: db, rondas: map[string]int64{}}
	t.Cleanup(m.limpiar)

	m.alfa = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-01-01', '2026-12-31') RETURNING id`)
	m.beta = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-01-01', '2026-12-31') RETURNING id`)
	m.ana = m.integrante(t, "Ana", m.alfa)
	m.beto = m.integrante(t, "Beto", m.alfa)
	m.caro = m.integrante(t, "Caro", m.alfa)
	m.dani = m.integrante(t, "Dani", m.beta)

	for _, r := range []struct {
		titulo          string
		ana, beto, caro int // 0 = no votó
	}{
		{"Login", 2, 13, 5},
		{"Pagos", 3, 5, 8},
		{"Reportes", 3, 13, 3},
		{"Perfil", 5, 5, 0},
		{"Ayuda", 0, 0, 0},
	} {
		ronda := m.ronda(t, m.alfa, r.titulo, m.ana)
		for integrante, valor := range map[int64]int{m.ana: r.ana, m.beto: r.beto, m.caro: r.caro} {
			if valor != 0 {
				m.voto(t, ronda, integrante, valor)
			}
		}
	}
	m.voto(t, m.ronda(t, m.beta, "Tablero", m.dani), m.dani, 8)
	agenda := m.ronda(t, m.beta, "Agenda", m.dani)
	if _, err := db.Exec(`UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`, agenda); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return m
}

func (m *revelacionMundo) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *revelacionMundo) integrante(t *testing.T, nombre string, proyecto int64) int64 {
	t.Helper()
	id := m.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, nombre)
	if _, err := m.db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, proyecto, id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *revelacionMundo) ronda(t *testing.T, proyecto int64, titulo string, abre int64) int64 {
	t.Helper()
	historia := m.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Media') RETURNING id`, proyecto, titulo)
	ronda := m.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 1, $3) RETURNING id`, proyecto, historia, abre)
	m.rondas[titulo] = ronda
	return ronda
}

func (m *revelacionMundo) voto(t *testing.T, ronda, integrante int64, valor int) {
	t.Helper()
	if _, err := m.db.Exec(`INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)`, ronda, integrante, valor); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
}

func (m *revelacionMundo) estado(t *testing.T, ronda int64) string {
	t.Helper()
	var s string
	if err := m.db.QueryRow(`SELECT status || '|' || COALESCE(closed_at::text, '') FROM poker_rounds WHERE id = $1`, ronda).Scan(&s); err != nil {
		t.Fatalf("no se pudo leer la ronda: %v", err)
	}
	return s
}

func (m *revelacionMundo) valorDe(t *testing.T, ronda, integrante int64) int {
	t.Helper()
	var v int
	_ = m.db.QueryRow(`SELECT value FROM poker_votes WHERE round_id = $1 AND member_id = $2`, ronda, integrante).Scan(&v)
	return v
}

func (m *revelacionMundo) limpiar() {
	_, _ = m.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, m.alfa, m.beta)
	_, _ = m.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, m.alfa, m.beta)
	_, _ = m.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3, $4)`, m.ana, m.beto, m.caro, m.dani)
}

type revelacionApp struct {
	servidor *httptest.Server
	db       *sql.DB // HU-27: para configurar la contraseña de quien se identifica
}

func (m *revelacionMundo) laAplicacion(t *testing.T) *revelacionApp {
	t.Helper()
	srv := web.NewServer(silencioso())
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(m.db))
	if err := web.RegisterPlanningPoker(srv, poker.NewService(poker.NewPostgreSQLStore(m.db)), web.NewPostgreSQLPokerCatalog(m.db)); err != nil {
		t.Fatalf("no se pudo registrar Planning Poker: %v", err)
	}
	app := &revelacionApp{servidor: httptest.NewServer(srv.Handler()), db: m.db}
	t.Cleanup(app.servidor.Close)
	return app
}

// navegador sigue las redirecciones, como un navegador real.
func (a *revelacionApp) navegador(t *testing.T, proyecto, integrante int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	a.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(proyecto, 10)}})
	if integrante != 0 {
		a.enviar(t, c, "/contexto/integrante", identificacionDePrueba(t, a.db, integrante))
	}
	return c
}

func (a *revelacionApp) enviar(t *testing.T, c *http.Client, ruta string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+ruta, v)
	if err != nil {
		t.Fatalf("POST %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *revelacionApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

// cerrar envía "Cerrar ronda" con el identificador tal como lo manda el formulario.
func (a *revelacionApp) cerrar(t *testing.T, c *http.Client, ronda string) (*http.Response, string) {
	t.Helper()
	return a.enviar(t, c, "/planning-poker/cerrar", url.Values{"ronda": {ronda}})
}

func textoID(n int64) string { return strconv.FormatInt(n, 10) }

// rondaCerrada devuelve la sección de la ronda en "Rondas cerradas" ("" si no está).
func rondaCerrada(cuerpo string, ronda int64) string {
	m := regexp.MustCompile(fmt.Sprintf(`(?s)<section class="poker-closed-round" data-ronda-cerrada="%d"[^>]*>(.*?)</section>`, ronda)).FindStringSubmatch(cuerpo)
	if m == nil {
		return ""
	}
	return m[1]
}

// votoRevelado devuelve la fila del integrante en la ronda cerrada ("" si no está).
func votoRevelado(cuerpo string, ronda, integrante int64) string {
	m := regexp.MustCompile(fmt.Sprintf(`<li data-integrante="%d">(.*?)</li>`, integrante)).FindStringSubmatch(rondaCerrada(cuerpo, ronda))
	if m == nil {
		return ""
	}
	return m[1]
}

func debeMostrar(t *testing.T, donde, texto string, partes ...string) {
	t.Helper()
	for _, p := range partes {
		if !strings.Contains(texto, p) {
			t.Errorf("%s debía mostrar %q", donde, p)
		}
	}
}

// Escenario 1: revelación de votos.
func TestBDD_HU05b_RevelacionDeVotos(t *testing.T) {
	// Dado la ronda abierta de «Login», en la que votaron Ana, Beto y Caro
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.alfa, m.ana)

	// Cuando cierro la ronda
	resp, cuerpo := app.cerrar(t, ana, textoID(m.rondas["Login"]))

	// Entonces veo "Ronda cerrada para la historia «Login»" y los votos revelados: Ana 2, Beto 13 y Caro 5
	if resp.Request.URL.Path != "/planning-poker" {
		t.Fatalf("se esperaba volver a Planning Poker, se obtuvo %s (%d)", resp.Request.URL, resp.StatusCode)
	}
	debeMostrar(t, "la pantalla", cuerpo, "Ronda cerrada para la historia «Login»", "Rondas cerradas")
	for integrante, esperado := range map[int64]string{m.ana: "Ana: 2", m.beto: "Beto: 13", m.caro: "Caro: 5"} {
		if fila := votoRevelado(cuerpo, m.rondas["Login"], integrante); fila != esperado {
			t.Errorf("se esperaba %q, se obtuvo %q", esperado, fila)
		}
	}
}

// cerrarComoAna cierra la ronda desde la pantalla de Ana y devuelve la pantalla resultante.
func (m *revelacionMundo) cerrarComoAna(t *testing.T, app *revelacionApp, titulo string) string {
	t.Helper()
	resp, cuerpo := app.cerrar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas[titulo]))
	if resp.StatusCode != http.StatusOK || resp.Request.URL.Path != "/planning-poker" {
		t.Fatalf("no se pudo cerrar «%s»: %d %s", titulo, resp.StatusCode, resp.Request.URL)
	}
	return cuerpo
}

// Escenario 2: alerta de divergencia.
func TestBDD_HU05b_AlertaDeDivergencia(t *testing.T) {
	// Dado la ronda de «Login» cerrada con votos de 2, 13 y 5
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)

	// Cuando veo sus resultados
	cuerpo := m.cerrarComoAna(t, app, "Login")

	// Entonces veo la alerta "Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación"
	debeMostrar(t, "«Login»", rondaCerrada(cuerpo, m.rondas["Login"]), "alert-warning", "Advertencia:",
		"Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación")
}

// Escenario 3: sin divergencia en el límite.
func TestBDD_HU05b_SinDivergenciaEnElLimite(t *testing.T) {
	// Dado la ronda de «Pagos» cerrada con votos de 3, 5 y 8 (dos niveles entre el menor y el mayor)
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)

	// Cuando veo sus resultados
	seccion := rondaCerrada(m.cerrarComoAna(t, app, "Pagos"), m.rondas["Pagos"])

	// Entonces veo "Sin divergencia: los votos van de 3 a 8" y no veo la alerta
	debeMostrar(t, "«Pagos»", seccion, "Sin divergencia: los votos van de 3 a 8")
	if strings.Contains(seccion, "alert-warning") || strings.Contains(seccion, "Divergencia:") {
		t.Errorf("«Pagos» no debía mostrar la alerta:\n%s", seccion)
	}
}

// Escenario 4: divergencia de tres niveles.
func TestBDD_HU05b_DivergenciaDeTresNiveles(t *testing.T) {
	// Dado la ronda de «Reportes» cerrada con votos de 3, 13 y 3
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)

	// Cuando veo sus resultados
	seccion := rondaCerrada(m.cerrarComoAna(t, app, "Reportes"), m.rondas["Reportes"])

	// Entonces veo la alerta "Divergencia: los votos van de 3 a 13. Debatan antes de acordar la estimación"
	debeMostrar(t, "«Reportes»", seccion, "alert-warning", "Advertencia:",
		"Divergencia: los votos van de 3 a 13. Debatan antes de acordar la estimación")
}

// Escenario 5: cerrar sin que todos hayan votado.
func TestBDD_HU05b_CerrarSinQueTodosVoten(t *testing.T) {
	// Dado la ronda abierta de «Perfil», en la que Caro no votó
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)

	// Cuando cierro la ronda
	cuerpo := m.cerrarComoAna(t, app, "Perfil")

	// Entonces la ronda se cierra, veo Ana 5, Beto 5 y Caro "No votó", y veo "Sin divergencia: los votos van de 5 a 5"
	ronda := m.rondas["Perfil"]
	if e := m.estado(t, ronda); !strings.HasPrefix(e, "Cerrada|") {
		t.Errorf("«Perfil» debía quedar cerrada, quedó %s", e)
	}
	for integrante, esperado := range map[int64]string{m.ana: "Ana: 5", m.beto: "Beto: 5", m.caro: "Caro: No votó"} {
		if fila := votoRevelado(cuerpo, ronda, integrante); fila != esperado {
			t.Errorf("se esperaba %q, se obtuvo %q", esperado, fila)
		}
	}
	debeMostrar(t, "«Perfil»", rondaCerrada(cuerpo, ronda), "Sin divergencia: los votos van de 5 a 5")
}

// Escenario 6: ronda sin votos.
func TestBDD_HU05b_RondaSinVotos(t *testing.T) {
	// Dado la ronda abierta de «Ayuda», sin votos
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	antes := m.estado(t, m.rondas["Ayuda"])

	// Cuando intento cerrarla
	resp, cuerpo := app.cerrar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Ayuda"]))

	// Entonces veo "La ronda no tiene votos" y la ronda sigue abierta
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d", resp.StatusCode)
	}
	debeMostrar(t, "la pantalla", cuerpo, "La ronda no tiene votos")
	if despues := m.estado(t, m.rondas["Ayuda"]); despues != antes {
		t.Errorf("«Ayuda» debía seguir abierta: %s → %s", antes, despues)
	}
}

// Escenario 7: los votos se ven para todos.
func TestBDD_HU05b_VotosParaTodos(t *testing.T) {
	// Dado la ronda de «Login» cerrada por Ana
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	deAna := rondaCerrada(m.cerrarComoAna(t, app, "Login"), m.rondas["Login"])

	// Cuando Beto abre "Planning Poker"
	_, cuerpo := app.abrir(t, app.navegador(t, m.alfa, m.beto), "/planning-poker")

	// Entonces ve los mismos votos revelados y la misma alerta
	deBeto := rondaCerrada(cuerpo, m.rondas["Login"])
	if deBeto == "" || deBeto != deAna {
		t.Errorf("Beto debía ver lo mismo que Ana:\nAna:\n%s\nBeto:\n%s", deAna, deBeto)
	}
	debeMostrar(t, "la pantalla de Beto", deBeto, "Ana: 2", "Beto: 13", "Caro: 5",
		"Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación")
}

// Escenario 8: votar en una ronda cerrada.
func TestBDD_HU05b_VotarEnRondaCerrada(t *testing.T) {
	// Dado la ronda de «Login» cerrada
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	m.cerrarComoAna(t, app, "Login")

	// Cuando Caro intenta votar 8
	resp, cuerpo := app.enviar(t, app.navegador(t, m.alfa, m.caro), "/planning-poker/votar",
		url.Values{"ronda": {textoID(m.rondas["Login"])}, "valor": {"8"}})

	// Entonces ve "La ronda ya está cerrada" y su voto sigue siendo 5
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d", resp.StatusCode)
	}
	debeMostrar(t, "la pantalla", cuerpo, "La ronda ya está cerrada")
	if v := m.valorDe(t, m.rondas["Login"], m.caro); v != 5 {
		t.Errorf("el voto de Caro debía seguir en 5, quedó %d", v)
	}
}

// Escenario 9: cerrar dos veces.
func TestBDD_HU05b_CerrarDosVeces(t *testing.T) {
	// Dado la ronda de «Login» cerrada
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	m.cerrarComoAna(t, app, "Login")
	antes := m.estado(t, m.rondas["Login"])

	// Cuando se envía otra vez la acción de cerrarla
	resp, cuerpo := app.cerrar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Login"]))

	// Entonces veo "La ronda ya está cerrada" y no cambia nada
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("se esperaba 409, se obtuvo %d", resp.StatusCode)
	}
	debeMostrar(t, "la pantalla", cuerpo, "La ronda ya está cerrada")
	if despues := m.estado(t, m.rondas["Login"]); despues != antes {
		t.Errorf("la ronda no debía cambiar: %s → %s", antes, despues)
	}
}

// Escenario 10: ronda inexistente o de otro proyecto.
func TestBDD_HU05b_RondaInexistente(t *testing.T) {
	// Dado una ronda que no existe o que pertenece a otro proyecto
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.alfa, m.ana)
	tablero, agenda := m.estado(t, m.rondas["Tablero"]), m.estado(t, m.rondas["Agenda"])

	for _, ronda := range []string{"2147483647", textoID(m.rondas["Tablero"]), textoID(m.rondas["Agenda"])} {
		// Cuando intento cerrarla
		resp, cuerpo := app.cerrar(t, ana, ronda)

		// Entonces veo "La ronda no existe" y no cambia nada
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("ronda %s: se esperaba 404, se obtuvo %d", ronda, resp.StatusCode)
		}
		debeMostrar(t, "ronda "+ronda, cuerpo, "La ronda no existe")
	}
	if m.estado(t, m.rondas["Tablero"]) != tablero || m.estado(t, m.rondas["Agenda"]) != agenda {
		t.Error("las rondas de «Beta» no debían cambiar")
	}
}

// Escenario 11: sin integrante identificado.
func TestBDD_HU05b_SinIdentificarme(t *testing.T) {
	// Dado que no me identifiqué como integrante
	m := dadoElEquipoAlfaConSusVotos(t)
	app := m.laAplicacion(t)
	c := app.navegador(t, m.alfa, 0)
	antes := m.estado(t, m.rondas["Login"])

	// Cuando intento cerrar una ronda
	resp, cuerpo := app.cerrar(t, c, textoID(m.rondas["Login"]))

	// Entonces se me pide identificarme y no se cierra nada
	if resp.Request.URL.Path != "/contexto/integrante" || resp.Request.URL.Query().Get("volver_a") != "/planning-poker" {
		t.Fatalf("se esperaba la pantalla de identificación con regreso a Planning Poker, se obtuvo %s", resp.Request.URL)
	}
	debeMostrar(t, "la pantalla", cuerpo, web.MsgMemberRequired)
	if despues := m.estado(t, m.rondas["Login"]); despues != antes {
		t.Errorf("«Login» no debía cerrarse: %s → %s", antes, despues)
	}
}
