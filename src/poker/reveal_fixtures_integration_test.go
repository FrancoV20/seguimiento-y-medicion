package poker

import (
	"database/sql"
	"testing"
	"time"
)

// HU-05b, T008: datos del Issue para las pruebas de cierre contra PostgreSQL.
//
// «Alfa» con Ana, Beto y Caro; ronda 1 abierta por Ana en cada historia:
//
//	«Login» 2/13/5, «Pagos» 3/5/8, «Reportes» 3/13/3, «Perfil» 5/5 (Caro no votó), «Ayuda» sin votos.
//
// «Beta» con Dani: «Tablero» con una ronda abierta (Dani 8) y «Agenda» con una ronda cerrada sin
// votos, para comprobar que una ronda de otro proyecto se trata como inexistente.
type alfaFixture struct {
	db                                    *sql.DB
	alfa, beta                            int64
	ana, beto, caro, dani                 int64
	login, pagos, reportes, perfil, ayuda int64 // historias de «Alfa»
	tablero, agenda                       int64 // historias de «Beta»
	rounds                                map[string]int64
}

// agendaClosedAt es la fecha de cierre explícita de la ronda de «Agenda».
var agendaClosedAt = time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)

func newAlfaFixture(t *testing.T) *alfaFixture {
	t.Helper()
	f := &alfaFixture{db: openTestDB(t), rounds: map[string]int64{}}
	t.Cleanup(f.cleanup)

	f.alfa = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-01-01', '2026-12-31') RETURNING id`)
	f.beta = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-01-01', '2026-12-31') RETURNING id`)
	f.ana = f.member(t, "Ana", f.alfa)
	f.beto = f.member(t, "Beto", f.alfa)
	f.caro = f.member(t, "Caro", f.alfa)
	f.dani = f.member(t, "Dani", f.beta)

	votes := []struct {
		title           string
		story           *int64
		ana, beto, caro int // 0 = no votó
	}{
		{"Login", &f.login, 2, 13, 5},
		{"Pagos", &f.pagos, 3, 5, 8},
		{"Reportes", &f.reportes, 3, 13, 3},
		{"Perfil", &f.perfil, 5, 5, 0},
		{"Ayuda", &f.ayuda, 0, 0, 0},
	}
	for _, v := range votes {
		*v.story = f.story(t, f.alfa, v.title)
		round := f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 1, $3) RETURNING id`,
			f.alfa, *v.story, f.ana)
		f.rounds[v.title] = round
		for member, value := range map[int64]int{f.ana: v.ana, f.beto: v.beto, f.caro: v.caro} {
			if value != 0 {
				f.insertVote(t, round, member, value)
			}
		}
	}

	f.tablero = f.story(t, f.beta, "Tablero")
	f.rounds["Tablero"] = f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 1, $3) RETURNING id`,
		f.beta, f.tablero, f.dani)
	f.insertVote(t, f.rounds["Tablero"], f.dani, 8)
	f.agenda = f.story(t, f.beta, "Agenda")
	f.rounds["Agenda"] = f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at)
		VALUES ($1, $2, 1, 'Cerrada', $3, $4) RETURNING id`, f.beta, f.agenda, f.dani, agendaClosedAt)
	return f
}

func (f *alfaFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *alfaFixture) member(t *testing.T, name string, project int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	if _, err := f.db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, id); err != nil {
		t.Fatalf("no se pudo vincular el integrante: %v", err)
	}
	return id
}

func (f *alfaFixture) story(t *testing.T, project int64, title string) int64 {
	t.Helper()
	return f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Media') RETURNING id`,
		project, title)
}

func (f *alfaFixture) insertVote(t *testing.T, round, member int64, value int) {
	t.Helper()
	if _, err := f.db.Exec(`INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)`, round, member, value); err != nil {
		t.Fatalf("no se pudo preparar el voto: %v", err)
	}
}

// vote devuelve el valor guardado del voto del integrante en la ronda; 0 si no votó.
func (f *alfaFixture) vote(t *testing.T, round, member int64) int {
	t.Helper()
	var value int
	err := f.db.QueryRow(`SELECT value FROM poker_votes WHERE round_id = $1 AND member_id = $2`, round, member).Scan(&value)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		t.Fatalf("no se pudo leer el voto: %v", err)
	}
	return value
}

// status devuelve el estado guardado de la ronda.
func (f *alfaFixture) status(t *testing.T, round int64) string {
	t.Helper()
	var s string
	if err := f.db.QueryRow(`SELECT status FROM poker_rounds WHERE id = $1`, round).Scan(&s); err != nil {
		t.Fatalf("no se pudo leer el estado de la ronda: %v", err)
	}
	return s
}

// closedAt devuelve la fecha de cierre de la ronda (Valid es falso si es NULL).
func (f *alfaFixture) closedAt(t *testing.T, round int64) sql.NullTime {
	t.Helper()
	var at sql.NullTime
	if err := f.db.QueryRow(`SELECT closed_at FROM poker_rounds WHERE id = $1`, round).Scan(&at); err != nil {
		t.Fatalf("no se pudo leer la fecha de cierre: %v", err)
	}
	return at
}

func (f *alfaFixture) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("no se pudo contar: %v", err)
	}
	return n
}

// waitForWaiters espera hasta que n sesiones de la base queden esperando un bloqueo (como
// waitForLock de HU-05a, pero con una cantidad). Plazo: 5 s.
func waitForWaiters(t *testing.T, f *alfaFixture, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if f.count(t, `SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`) >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no quedaron %d sesiones esperando un bloqueo", n)
}

// cleanup: historias (las rondas y los votos caen en cascada) → proyectos → integrantes.
func (f *alfaFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3, $4)`, f.ana, f.beto, f.caro, f.dani)
}
