package web

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// HU-18, T017: datos del Issue para las pruebas de la pantalla (los mismos de src/poker).
//
// «Alfa» con Ana, Beto y Caro:
//
//	«Login»    sin Story Points; ronda 1 cerrada (Ana 2, Beto 13, Caro 5)
//	«Pagos»    sin Story Points; ronda 1 cerrada (Ana 3, Beto 5, Caro 8)
//	«Reportes» 3 Story Points; ronda 1 cerrada (Ana 3, Beto 13, Caro 3) y ronda 2 abierta (Ana 5)
//	«Alta»     3 Story Points, "Hecho", del Sprint 1 "Finalizado"; ronda 1 cerrada (5, 5, 5)
//
// «Beta» con Dani: «Tablero» con su ronda 1 cerrada (Dani 8).
type consensusWebFixture struct {
	db                    *sql.DB
	alfa, beta            int64
	ana, beto, caro, dani int64
	sprint1, sprint2      int64
	stories               map[string]int64 // historia por título
	rounds                map[string]int64 // ronda 1 por título; "Reportes 2" es la ronda 2 abierta
	extraMembers          []int64
}

// consensusBase es la hora de referencia para las fechas de cierre explícitas.
var consensusBase = time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)

func newConsensusWebFixture(t *testing.T, db *sql.DB) *consensusWebFixture {
	t.Helper()
	f := &consensusWebFixture{db: db, stories: map[string]int64{}, rounds: map[string]int64{}}
	t.Cleanup(f.cleanup)

	f.alfa = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-01-01', '2026-12-31') RETURNING id`)
	f.beta = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-01-01', '2026-12-31') RETURNING id`)
	f.ana = f.member(t, "Ana", f.alfa)
	f.beto = f.member(t, "Beto", f.alfa)
	f.caro = f.member(t, "Caro", f.alfa)
	f.dani = f.member(t, "Dani", f.beta)

	f.story(t, f.alfa, "Login", "Pendiente", 0)
	f.story(t, f.alfa, "Pagos", "Pendiente", 0)
	f.story(t, f.alfa, "Reportes", "Pendiente", 3)
	f.story(t, f.alfa, "Alta", "Hecho", 3)
	f.story(t, f.beta, "Tablero", "Pendiente", 0)
	f.rounds["Login"] = f.closedRound(t, f.alfa, "Login", 1, consensusBase, map[int64]int{f.ana: 2, f.beto: 13, f.caro: 5})
	f.rounds["Pagos"] = f.closedRound(t, f.alfa, "Pagos", 1, consensusBase.Add(time.Minute), map[int64]int{f.ana: 3, f.beto: 5, f.caro: 8})
	f.rounds["Reportes"] = f.closedRound(t, f.alfa, "Reportes", 1, consensusBase.Add(2*time.Minute), map[int64]int{f.ana: 3, f.beto: 13, f.caro: 3})
	f.rounds["Reportes 2"] = f.openRound(t, f.alfa, "Reportes", 2, f.ana)
	f.setVote(t, f.rounds["Reportes 2"], f.ana, 5)
	f.rounds["Alta"] = f.closedRound(t, f.alfa, "Alta", 1, consensusBase.Add(3*time.Minute), map[int64]int{f.ana: 5, f.beto: 5, f.caro: 5})
	f.rounds["Tablero"] = f.closedRound(t, f.beta, "Tablero", 1, consensusBase, map[int64]int{f.dani: 8})

	// Sprint 1 "Finalizado" con «Alta» "Hecho", como lo deja el cierre de HU-15.
	f.sprint1 = f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Alta de usuarios', '2026-09-01', '2026-09-14', 'Finalizado') RETURNING id`, f.alfa)
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, f.sprint1, f.stories["Alta"])
	f.exec(t, `UPDATE user_stories SET active_sprint_id = $1 WHERE id = $2`, f.sprint1, f.stories["Alta"])
	return f
}

func (f *consensusWebFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *consensusWebFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *consensusWebFixture) member(t *testing.T, name string, project int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, id)
	return id
}

// addMember suma un integrante a «Alfa» y lo borra al final.
func (f *consensusWebFixture) addMember(t *testing.T, name string) int64 {
	t.Helper()
	id := f.member(t, name, f.alfa)
	f.extraMembers = append(f.extraMembers, id)
	return id
}

// story crea la historia; points 0 = sin Story Points.
func (f *consensusWebFixture) story(t *testing.T, project int64, title, status string, points int) int64 {
	t.Helper()
	var sp any
	if points != 0 {
		sp = points
	}
	id := f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, story_points)
		VALUES ($1, $2, 'd', 'Media', $3, $4) RETURNING id`, project, title, status, sp)
	f.stories[title] = id
	return id
}

// openRound crea una ronda abierta de la historia; devuelve la ronda.
func (f *consensusWebFixture) openRound(t *testing.T, project int64, title string, number int, openedBy int64) int64 {
	t.Helper()
	return f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, $3, $4) RETURNING id`,
		project, f.stories[title], number, openedBy)
}

// closedRound carga una ronda cerrada con fecha explícita y sus votos.
func (f *consensusWebFixture) closedRound(t *testing.T, project int64, title string, number int, at time.Time, votes map[int64]int) int64 {
	t.Helper()
	var opener int64
	for member := range votes {
		opener = member
		break
	}
	round := f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at)
		VALUES ($1, $2, $3, 'Cerrada', $4, $5) RETURNING id`, project, f.stories[title], number, opener, at)
	for member, value := range votes {
		f.setVote(t, round, member, value)
	}
	return round
}

func (f *consensusWebFixture) setVote(t *testing.T, round, member int64, value int) {
	t.Helper()
	f.exec(t, `INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (round_id, member_id) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, round, member, value)
}

// assignToActiveSprint crea el Sprint 2 "Activo" de «Alfa» (si no existe) y le asigna la historia.
func (f *consensusWebFixture) assignToActiveSprint(t *testing.T, title, status string) {
	t.Helper()
	if f.sprint2 == 0 {
		f.sprint2 = f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, 2, 'Pagos online', '2026-09-15', '2026-09-30', 'Activo') RETURNING id`, f.alfa)
	}
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, f.sprint2, f.stories[title])
	f.exec(t, `UPDATE user_stories SET active_sprint_id = $1, status = $2 WHERE id = $3`, f.sprint2, status, f.stories[title])
}

// storyPoints devuelve los Story Points guardados; 0 = sin Story Points.
func (f *consensusWebFixture) storyPoints(t *testing.T, title string) int {
	t.Helper()
	var sp sql.NullInt64
	if err := f.db.QueryRow(`SELECT story_points FROM user_stories WHERE id = $1`, f.stories[title]).Scan(&sp); err != nil {
		t.Fatalf("no se pudieron leer los Story Points: %v", err)
	}
	return int(sp.Int64)
}

// roundCount cuenta las rondas de la historia.
func (f *consensusWebFixture) roundCount(t *testing.T, title string) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, f.stories[title]).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar las rondas: %v", err)
	}
	return n
}

// roundNumbered devuelve la ronda de la historia con ese número.
func (f *consensusWebFixture) roundNumbered(t *testing.T, title string, number int) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(`SELECT id FROM poker_rounds WHERE story_id = $1 AND round_number = $2`, f.stories[title], number).Scan(&id); err != nil {
		t.Fatalf("«%s» no tiene la ronda %d: %v", title, number, err)
	}
	return id
}

func (f *consensusWebFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM sprint_story_assignments WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2))`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM sprints WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.alfa, f.beta)
	for _, m := range append([]int64{f.ana, f.beto, f.caro, f.dani}, f.extraMembers...) {
		_, _ = f.db.Exec(`DELETE FROM members WHERE id = $1`, m)
	}
}

// newConsensusApp es el servidor de newPokerApp con el Backlog registrado, para comprobar los
// Story Points; acepta la opción de failingPokerStore.
func newConsensusApp(t *testing.T, opts ...pokerOption) (*pokerApp, *consensusWebFixture) {
	t.Helper()
	db := openContextTestDB(t)
	var store poker.Store = poker.NewPostgreSQLStore(db)
	var catalog PokerCatalog = NewPostgreSQLPokerCatalog(db)
	for _, o := range opts {
		if o.store != nil {
			store = o.store(store)
		}
		if o.catalog != nil {
			catalog = o.catalog(catalog)
		}
	}
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.Use(CrossOriginGuard(srv.Renderer()))
	RegisterContext(srv, NewPostgreSQLContextStore(db))
	if err := RegisterPlanningPoker(srv, poker.NewService(store), catalog); err != nil {
		t.Fatalf("no se pudo registrar Planning Poker: %v", err)
	}
	if err := RegisterBacklog(srv, backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(db)), NewPostgreSQLStoryCatalog(db)); err != nil {
		t.Fatalf("no se pudo registrar el Backlog: %v", err)
	}
	a := &pokerApp{ts: httptest.NewServer(srv.Handler()), fx: &pokerWebFixture{db: db}}
	t.Cleanup(a.ts.Close)
	return a, newConsensusWebFixture(t, db)
}

// consensusPage devuelve la pantalla de Planning Poker del integrante, sin mensajes pendientes.
func (a *pokerApp) consensusPage(t *testing.T, c *http.Client) string {
	t.Helper()
	a.get(t, c, "/planning-poker")
	resp, body := a.get(t, c, "/planning-poker")
	pokerAssertStatus(t, resp, http.StatusOK)
	return body
}

func (a *pokerApp) nextRound(t *testing.T, c *http.Client, round string, extra ...string) (*http.Response, string) {
	t.Helper()
	v := url.Values{"ronda": {round}}
	for i := 0; i+1 < len(extra); i += 2 {
		v.Set(extra[i], extra[i+1])
	}
	return a.post(t, c, "/planning-poker/nueva-ronda", v)
}

func (a *pokerApp) agree(t *testing.T, c *http.Client, round, value string) (*http.Response, string) {
	t.Helper()
	return a.post(t, c, "/planning-poker/estimacion", url.Values{"ronda": {round}, "valor": {value}})
}

// backlogPoints devuelve la celda de Story Points de la historia en el Backlog ("" si no está).
func backlogPoints(body, title string) string {
	m := regexp.MustCompile(fmt.Sprintf(`<td>%s</td>\s*<td>[^<]*</td>\s*<td>([^<]*)</td>`, regexp.QuoteMeta(title))).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}
