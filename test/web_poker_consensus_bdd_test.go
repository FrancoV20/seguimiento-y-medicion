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

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-18 (T020). Usan la API pública de web, el servidor real, el servicio real de
// Planning Poker, el Backlog real y PostgreSQL real.

// consensoMundo: «Alfa» con Ana, Beto y Caro y las historias del Issue («Login», «Pagos»,
// «Reportes» y «Alta», esta en el Sprint 1 finalizado); «Beta» con Dani y «Tablero».
type consensoMundo struct {
	db                    *sql.DB
	alfa, beta            int64
	ana, beto, caro, dani int64
	historias             map[string]int64
	rondas                map[string]int64 // ronda 1 por título; "Reportes 2" es la ronda 2 abierta
}

func dadoElEquipoAlfaDespuesDeVotar(t *testing.T) *consensoMundo {
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
	m := &consensoMundo{db: db, historias: map[string]int64{}, rondas: map[string]int64{}}
	t.Cleanup(m.limpiar)

	m.alfa = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-01-01', '2026-12-31') RETURNING id`)
	m.beta = m.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-01-01', '2026-12-31') RETURNING id`)
	m.ana = m.integrante(t, "Ana", m.alfa)
	m.beto = m.integrante(t, "Beto", m.alfa)
	m.caro = m.integrante(t, "Caro", m.alfa)
	m.dani = m.integrante(t, "Dani", m.beta)

	cierre := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	for i, h := range []struct {
		titulo, estado  string
		puntos          int // 0 = sin Story Points
		ana, beto, caro int
	}{
		{"Login", "Pendiente", 0, 2, 13, 5},
		{"Pagos", "Pendiente", 0, 3, 5, 8},
		{"Reportes", "Pendiente", 3, 3, 13, 3},
		{"Alta", "Hecho", 3, 5, 5, 5},
	} {
		m.historia(t, m.alfa, h.titulo, h.estado, h.puntos)
		m.rondas[h.titulo] = m.rondaCerrada(t, m.alfa, h.titulo, cierre.Add(time.Duration(i)*time.Minute),
			map[int64]int{m.ana: h.ana, m.beto: h.beto, m.caro: h.caro})
	}
	m.rondas["Reportes 2"] = m.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 2, $3) RETURNING id`,
		m.alfa, m.historias["Reportes"], m.ana)
	m.voto(t, m.rondas["Reportes 2"], m.ana, 5)
	sprint := m.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Alta de usuarios', '2026-09-01', '2026-09-14', 'Finalizado') RETURNING id`, m.alfa)
	m.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprint, m.historias["Alta"])
	m.exec(t, `UPDATE user_stories SET active_sprint_id = $1 WHERE id = $2`, sprint, m.historias["Alta"])

	m.historia(t, m.beta, "Tablero", "Pendiente", 0)
	m.rondas["Tablero"] = m.rondaCerrada(t, m.beta, "Tablero", cierre, map[int64]int{m.dani: 8})
	return m
}

func (m *consensoMundo) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
	return id
}

func (m *consensoMundo) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := m.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v", err)
	}
}

func (m *consensoMundo) integrante(t *testing.T, nombre string, proyecto int64) int64 {
	t.Helper()
	id := m.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, nombre)
	m.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, proyecto, id)
	return id
}

func (m *consensoMundo) historia(t *testing.T, proyecto int64, titulo, estado string, puntos int) {
	t.Helper()
	var sp any
	if puntos != 0 {
		sp = puntos
	}
	m.historias[titulo] = m.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, story_points)
		VALUES ($1, $2, 'd', 'Media', $3, $4) RETURNING id`, proyecto, titulo, estado, sp)
}

func (m *consensoMundo) rondaCerrada(t *testing.T, proyecto int64, titulo string, cierre time.Time, votos map[int64]int) int64 {
	t.Helper()
	var abre int64
	for integrante := range votos {
		abre = integrante
		break
	}
	ronda := m.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at)
		VALUES ($1, $2, 1, 'Cerrada', $3, $4) RETURNING id`, proyecto, m.historias[titulo], abre, cierre)
	for integrante, valor := range votos {
		m.voto(t, ronda, integrante, valor)
	}
	return ronda
}

func (m *consensoMundo) voto(t *testing.T, ronda, integrante int64, valor int) {
	t.Helper()
	m.exec(t, `INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)`, ronda, integrante, valor)
}

// puntos devuelve los Story Points guardados de la historia; 0 = sin Story Points.
func (m *consensoMundo) puntos(t *testing.T, titulo string) int {
	t.Helper()
	var sp sql.NullInt64
	if err := m.db.QueryRow(`SELECT story_points FROM user_stories WHERE id = $1`, m.historias[titulo]).Scan(&sp); err != nil {
		t.Fatalf("no se pudieron leer los Story Points: %v", err)
	}
	return int(sp.Int64)
}

// rondas cuenta las rondas de la historia.
func (m *consensoMundo) cantidadDeRondas(t *testing.T, titulo string) int {
	t.Helper()
	var n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, m.historias[titulo]).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar las rondas: %v", err)
	}
	return n
}

// rondaNumero devuelve la ronda de la historia con ese número.
func (m *consensoMundo) rondaNumero(t *testing.T, titulo string, numero int) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow(`SELECT id FROM poker_rounds WHERE story_id = $1 AND round_number = $2`, m.historias[titulo], numero).Scan(&id); err != nil {
		t.Fatalf("«%s» no tiene la ronda %d: %v", titulo, numero, err)
	}
	return id
}

func (m *consensoMundo) limpiar() {
	_, _ = m.db.Exec(`DELETE FROM sprint_story_assignments WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2))`, m.alfa, m.beta)
	_, _ = m.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, m.alfa, m.beta)
	_, _ = m.db.Exec(`DELETE FROM sprints WHERE project_id IN ($1, $2)`, m.alfa, m.beta)
	_, _ = m.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, m.alfa, m.beta)
	_, _ = m.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3, $4)`, m.ana, m.beto, m.caro, m.dani)
}

type consensoApp struct{ servidor *httptest.Server }

func (m *consensoMundo) laAplicacion(t *testing.T) *consensoApp {
	t.Helper()
	srv := web.NewServer(silencioso())
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(m.db))
	if err := web.RegisterPlanningPoker(srv, poker.NewService(poker.NewPostgreSQLStore(m.db)), web.NewPostgreSQLPokerCatalog(m.db)); err != nil {
		t.Fatalf("no se pudo registrar Planning Poker: %v", err)
	}
	if err := web.RegisterBacklog(srv, backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(m.db)), web.NewPostgreSQLStoryCatalog(m.db)); err != nil {
		t.Fatalf("no se pudo registrar el Backlog: %v", err)
	}
	app := &consensoApp{servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	return app
}

// navegador sigue las redirecciones, como un navegador real.
func (a *consensoApp) navegador(t *testing.T, proyecto, integrante int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	a.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(proyecto, 10)}})
	if integrante != 0 {
		a.enviar(t, c, "/contexto/integrante", url.Values{"integrante_id": {strconv.FormatInt(integrante, 10)}})
	}
	return c
}

func (a *consensoApp) enviar(t *testing.T, c *http.Client, ruta string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+ruta, v)
	if err != nil {
		t.Fatalf("POST %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *consensoApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *consensoApp) nuevaRonda(t *testing.T, c *http.Client, ronda string) (*http.Response, string) {
	t.Helper()
	return a.enviar(t, c, "/planning-poker/nueva-ronda", url.Values{"ronda": {ronda}})
}

func (a *consensoApp) estimar(t *testing.T, c *http.Client, ronda, valor string) (*http.Response, string) {
	t.Helper()
	return a.enviar(t, c, "/planning-poker/estimacion", url.Values{"ronda": {ronda}, "valor": {valor}})
}

// filaEnRonda devuelve la fila del integrante en la ronda abierta ("" si no está).
func filaEnRonda(cuerpo string, ronda, integrante int64) string {
	seccion := regexp.MustCompile(fmt.Sprintf(`(?s)<section class="poker-round" data-ronda="%d"[^>]*>(.*?)</section>`, ronda)).FindStringSubmatch(cuerpo)
	if seccion == nil {
		return ""
	}
	fila := regexp.MustCompile(fmt.Sprintf(`<li data-integrante="%d">(.*?)</li>`, integrante)).FindStringSubmatch(seccion[1])
	if fila == nil {
		return ""
	}
	return fila[1]
}

// puntosEnElBacklog devuelve la celda de Story Points de la historia en el Backlog.
func puntosEnElBacklog(cuerpo, titulo string) string {
	m := regexp.MustCompile(fmt.Sprintf(`<td>%s</td>\s*<td>[^<]*</td>\s*<td>([^<]*)</td>`, regexp.QuoteMeta(titulo))).FindStringSubmatch(cuerpo)
	if m == nil {
		return ""
	}
	return m[1]
}

func debeResponder(t *testing.T, resp *http.Response, codigo int) {
	t.Helper()
	if resp.StatusCode != codigo {
		t.Fatalf("se esperaba %d, se obtuvo %d (%s)", codigo, resp.StatusCode, resp.Request.URL)
	}
}

// Escenario 1: nueva ronda tras divergencia.
func TestBDD_HU18_NuevaRondaTrasDivergencia(t *testing.T) {
	// Dado la ronda 1 de «Login» cerrada con divergencia
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando presiono "Nueva ronda"
	_, cuerpo := app.nuevaRonda(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Login"]))

	// Entonces veo "Ronda 2 abierta para la historia «Login»", la ronda 2 aparece abierta sin votos y
	// la ronda 1 sigue en "Rondas cerradas" con Ana 2, Beto 13 y Caro 5
	debeMostrar(t, "la pantalla", cuerpo, "Ronda 2 abierta para la historia «Login»")
	segunda := m.rondaNumero(t, "Login", 2)
	for integrante, nombre := range map[int64]string{m.ana: "Ana", m.beto: "Beto", m.caro: "Caro"} {
		if fila := filaEnRonda(cuerpo, segunda, integrante); fila != nombre+": Sin votar" {
			t.Errorf("en la ronda 2 se esperaba «%s: Sin votar», se obtuvo %q", nombre, fila)
		}
	}
	for integrante, esperado := range map[int64]string{m.ana: "Ana: 2", m.beto: "Beto: 13", m.caro: "Caro: 5"} {
		if fila := votoRevelado(cuerpo, m.rondas["Login"], integrante); fila != esperado {
			t.Errorf("en la ronda 1 se esperaba %q, se obtuvo %q", esperado, fila)
		}
	}
}

// Escenario 2: votos ocultos en la nueva ronda.
func TestBDD_HU18_VotosOcultosEnLaNuevaRonda(t *testing.T) {
	// Dado la ronda 2 de «Login» abierta
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.alfa, m.ana)
	app.nuevaRonda(t, ana, textoID(m.rondas["Login"]))
	segunda := m.rondaNumero(t, "Login", 2)

	// Cuando Ana vota 8
	app.enviar(t, ana, "/planning-poker/votar", url.Values{"ronda": {textoID(segunda)}, "valor": {"8"}})

	// Entonces Beto ve que Ana votó, sin ver el valor, igual que en la ronda 1
	_, cuerpo := app.abrir(t, app.navegador(t, m.alfa, m.beto), "/planning-poker")
	if fila := filaEnRonda(cuerpo, segunda, m.ana); fila != "Ana: Votó" {
		t.Errorf("Beto debía ver «Ana: Votó» sin el valor, vio %q", fila)
	}
}

// Escenario 3: registrar la estimación acordada.
func TestBDD_HU18_RegistrarEstimacionAcordada(t *testing.T) {
	// Dado la ronda 2 de «Login» cerrada con Ana 8, Beto 8 y Caro 5 (sin divergencia), abierta,
	// votada y cerrada desde la pantalla
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.alfa, m.ana)
	app.nuevaRonda(t, ana, textoID(m.rondas["Login"]))
	segunda := textoID(m.rondaNumero(t, "Login", 2))
	for integrante, valor := range map[int64]string{m.ana: "8", m.beto: "8", m.caro: "5"} {
		app.enviar(t, app.navegador(t, m.alfa, integrante), "/planning-poker/votar", url.Values{"ronda": {segunda}, "valor": {valor}})
	}
	app.enviar(t, ana, "/planning-poker/cerrar", url.Values{"ronda": {segunda}})

	// Cuando registro 8 como estimación acordada
	_, cuerpo := app.estimar(t, ana, segunda, "8")

	// Entonces veo "Estimación registrada: «Login» queda con 8 Story Points" y el Backlog muestra
	// «Login» con 8 Story Points
	debeMostrar(t, "la pantalla", cuerpo, "Estimación registrada: «Login» queda con 8 Story Points")
	_, lista := app.abrir(t, ana, "/backlog")
	if got := puntosEnElBacklog(lista, "Login"); got != "8" {
		t.Errorf("el Backlog debía mostrar «Login» con 8 Story Points, muestra %q", got)
	}
}

// Escenario 4: consenso directo.
func TestBDD_HU18_ConsensoDirecto(t *testing.T) {
	// Dado la ronda 1 de «Pagos» cerrada sin divergencia
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando registro 5 como estimación acordada
	_, cuerpo := app.estimar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Pagos"]), "5")

	// Entonces se registra sin pedir otra ronda y «Pagos» queda con 5 Story Points
	debeMostrar(t, "la pantalla", cuerpo, "Estimación registrada: «Pagos» queda con 5 Story Points")
	if sp, n := m.puntos(t, "Pagos"), m.cantidadDeRondas(t, "Pagos"); sp != 5 || n != 1 {
		t.Errorf("«Pagos» debía quedar con 5 Story Points y una sola ronda: %d SP, %d rondas", sp, n)
	}
}

// Escenario 5: registrar después de debatir una divergencia.
func TestBDD_HU18_RegistrarDespuesDeDebatir(t *testing.T) {
	// Dado la ronda 1 de «Login» cerrada con divergencia y sin una ronda nueva
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.alfa, m.ana)

	// Cuando abro el formulario de estimación acordada
	_, cuerpo := app.abrir(t, ana, "/planning-poker")

	// Entonces veo el recordatorio y, si registro 5, «Login» queda con 5 Story Points
	debeMostrar(t, "«Login»", rondaCerrada(cuerpo, m.rondas["Login"]), "Advertencia:",
		"La última ronda tuvo divergencia: registrá el valor que acordaron después de debatir")
	app.estimar(t, ana, textoID(m.rondas["Login"]), "5")
	if sp := m.puntos(t, "Login"); sp != 5 {
		t.Errorf("«Login» debía quedar con 5 Story Points, tiene %d", sp)
	}
}

// Escenario 6: valor fuera de la serie.
func TestBDD_HU18_ValorFueraDeLaSerie(t *testing.T) {
	// Dado la ronda 1 de «Pagos» cerrada
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando registro 4 como estimación acordada
	resp, cuerpo := app.estimar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Pagos"]), "4")

	// Entonces veo "El valor no pertenece a la serie de Fibonacci" y «Pagos» no cambia
	debeResponder(t, resp, http.StatusUnprocessableEntity)
	debeMostrar(t, "la pantalla", cuerpo, "El valor no pertenece a la serie de Fibonacci")
	if sp := m.puntos(t, "Pagos"); sp != 0 {
		t.Errorf("«Pagos» no debía cambiar, tiene %d Story Points", sp)
	}
}

// Escenario 7: ronda sin cerrar.
func TestBDD_HU18_RondaSinCerrar(t *testing.T) {
	// Dado la ronda 2 de «Reportes» abierta
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando intento registrar la estimación acordada
	resp, cuerpo := app.estimar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Reportes 2"]), "5")

	// Entonces veo "La ronda sigue abierta: cerrala antes de registrar la estimación" y «Reportes»
	// sigue con 3 Story Points
	debeResponder(t, resp, http.StatusConflict)
	debeMostrar(t, "la pantalla", cuerpo, "La ronda sigue abierta: cerrala antes de registrar la estimación")
	if sp := m.puntos(t, "Reportes"); sp != 3 {
		t.Errorf("«Reportes» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

// Escenario 8: nueva ronda con una ronda abierta.
func TestBDD_HU18_NuevaRondaConRondaAbierta(t *testing.T) {
	// Dado «Reportes», que ya tiene la ronda 2 abierta
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando se envía otra vez la acción "Nueva ronda" desde su ronda 1
	resp, cuerpo := app.nuevaRonda(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Reportes"]))

	// Entonces veo "Ya hay una ronda abierta para esta historia" y no se abre otra
	debeResponder(t, resp, http.StatusConflict)
	debeMostrar(t, "la pantalla", cuerpo, "Ya hay una ronda abierta para esta historia")
	if n := m.cantidadDeRondas(t, "Reportes"); n != 2 {
		t.Errorf("«Reportes» debía seguir con dos rondas, tiene %d", n)
	}
}

// Escenario 9: ronda que ya no es la última.
func TestBDD_HU18_RondaQueYaNoEsLaUltima(t *testing.T) {
	// Dado «Reportes», con la ronda 2 abierta
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando se envía la estimación acordada desde su ronda 1
	resp, cuerpo := app.estimar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Reportes"]), "5")

	// Entonces veo "Hay una ronda más reciente de esta historia" y «Reportes» no cambia
	debeResponder(t, resp, http.StatusConflict)
	debeMostrar(t, "la pantalla", cuerpo, "Hay una ronda más reciente de esta historia")
	if sp := m.puntos(t, "Reportes"); sp != 3 {
		t.Errorf("«Reportes» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

// Escenario 10: historia de un Sprint finalizado.
func TestBDD_HU18_HistoriaDeUnSprintFinalizado(t *testing.T) {
	// Dado «Alta», del Sprint 1 finalizado, con su ronda 1 cerrada
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)

	// Cuando registro 5 como estimación acordada
	resp, cuerpo := app.estimar(t, app.navegador(t, m.alfa, m.ana), textoID(m.rondas["Alta"]), "5")

	// Entonces veo "La historia pertenece a un Sprint finalizado; sus Story Points no se pueden
	// modificar" y «Alta» sigue con 3 Story Points
	debeResponder(t, resp, http.StatusConflict)
	debeMostrar(t, "la pantalla", cuerpo, "La historia pertenece a un Sprint finalizado; sus Story Points no se pueden modificar")
	if sp := m.puntos(t, "Alta"); sp != 3 {
		t.Errorf("«Alta» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

// Escenario 11: ronda inexistente o de otro proyecto.
func TestBDD_HU18_RondaInexistente(t *testing.T) {
	// Dado una ronda que no existe o que pertenece a otro proyecto
	m := dadoElEquipoAlfaDespuesDeVotar(t)
	app := m.laAplicacion(t)
	ana := app.navegador(t, m.alfa, m.ana)

	// Cuando intento abrir una nueva ronda o registrar la estimación desde ella
	for _, ronda := range []string{"999999999", textoID(m.rondas["Tablero"])} {
		for accion, enviar := range map[string]func() (*http.Response, string){
			"nueva ronda": func() (*http.Response, string) { return app.nuevaRonda(t, ana, ronda) },
			"estimación":  func() (*http.Response, string) { return app.estimar(t, ana, ronda, "5") },
		} {
			// Entonces veo "La ronda no existe" y no cambia nada
			resp, cuerpo := enviar()
			debeResponder(t, resp, http.StatusNotFound)
			if !strings.Contains(cuerpo, "La ronda no existe") {
				t.Errorf("%s desde %s: debía mostrar «La ronda no existe»", accion, ronda)
			}
		}
	}
	if sp, n := m.puntos(t, "Tablero"), m.cantidadDeRondas(t, "Tablero"); sp != 0 || n != 1 {
		t.Errorf("«Tablero» no debía cambiar: %d SP, %d rondas", sp, n)
	}
}
