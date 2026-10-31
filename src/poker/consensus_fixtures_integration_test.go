package poker

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// HU-18, T007: datos del Issue para las pruebas de "Nueva ronda" y de la estimación acordada.
//
// «Alfa» con Ana, Beto y Caro:
//
//	«Login»    sin Story Points, "Pendiente", sin Sprint; ronda 1 cerrada (Ana 2, Beto 13, Caro 5)
//	«Pagos»    sin Story Points, "Pendiente", sin Sprint; ronda 1 cerrada (Ana 3, Beto 5, Caro 8)
//	«Reportes» 3 Story Points, "Pendiente", sin Sprint; ronda 1 cerrada (Ana 3, Beto 13, Caro 3) y
//	           ronda 2 abierta con el voto de Ana (5)
//	«Alta»     3 Story Points, "Hecho", en el Sprint 1 "Finalizado"; ronda 1 cerrada (5, 5, 5)
//
// «Beta» con Dani: «Tablero» con su ronda 1 cerrada (Dani 8), para el escenario 11.
type consensusFixture struct {
	db                           *sql.DB
	alfa, beta                   int64
	ana, beto, caro, dani        int64
	login, pagos, reportes, alta int64 // historias de «Alfa»
	tablero                      int64 // historia de «Beta»
	sprint1                      int64 // Sprint 1 de «Alfa», "Finalizado"
	sprint2                      int64 // Sprint 2 de «Alfa», "Activo" (solo con assignToActiveSprint)
	rounds                       map[string]int64
}

// consensusClosedAt es la fecha de cierre de las rondas cargadas en la base.
var consensusClosedAt = time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)

func newConsensusFixture(t *testing.T) *consensusFixture {
	t.Helper()
	f := &consensusFixture{db: openTestDB(t), rounds: map[string]int64{}}
	t.Cleanup(f.cleanup)

	f.alfa = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-01-01', '2026-12-31') RETURNING id`)
	f.beta = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-01-01', '2026-12-31') RETURNING id`)
	f.ana = f.member(t, "Ana", f.alfa)
	f.beto = f.member(t, "Beto", f.alfa)
	f.caro = f.member(t, "Caro", f.alfa)
	f.dani = f.member(t, "Dani", f.beta)

	f.login = f.story(t, f.alfa, "Login", "Pendiente", 0)
	f.pagos = f.story(t, f.alfa, "Pagos", "Pendiente", 0)
	f.reportes = f.story(t, f.alfa, "Reportes", "Pendiente", 3)
	f.alta = f.story(t, f.alfa, "Alta", "Hecho", 3)
	f.tablero = f.story(t, f.beta, "Tablero", "Pendiente", 0)

	f.rounds["Login"] = f.closedRound(t, f.alfa, f.login, 1, map[int64]int{f.ana: 2, f.beto: 13, f.caro: 5})
	f.rounds["Pagos"] = f.closedRound(t, f.alfa, f.pagos, 1, map[int64]int{f.ana: 3, f.beto: 5, f.caro: 8})
	f.rounds["Reportes"] = f.closedRound(t, f.alfa, f.reportes, 1, map[int64]int{f.ana: 3, f.beto: 13, f.caro: 3})
	f.rounds["Reportes 2"] = f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 2, $3) RETURNING id`,
		f.alfa, f.reportes, f.ana)
	f.insertVote(t, f.rounds["Reportes 2"], f.ana, 5)
	f.rounds["Alta"] = f.closedRound(t, f.alfa, f.alta, 1, map[int64]int{f.ana: 5, f.beto: 5, f.caro: 5})
	f.rounds["Tablero"] = f.closedRound(t, f.beta, f.tablero, 1, map[int64]int{f.dani: 8})

	// Sprint 1 "Finalizado" con «Alta» "Hecho", como lo deja el cierre de HU-15.
	f.sprint1 = f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Alta de usuarios', '2026-09-01', '2026-09-14', 'Finalizado') RETURNING id`, f.alfa)
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, f.sprint1, f.alta)
	f.exec(t, `UPDATE user_stories SET active_sprint_id = $1 WHERE id = $2`, f.sprint1, f.alta)
	return f
}

func (f *consensusFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *consensusFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *consensusFixture) member(t *testing.T, name string, project int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, id)
	return id
}

// story crea una historia; points 0 = sin Story Points.
func (f *consensusFixture) story(t *testing.T, project int64, title, status string, points int) int64 {
	t.Helper()
	var sp any
	if points != 0 {
		sp = points
	}
	return f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, story_points)
		VALUES ($1, $2, 'd', 'Media', $3, $4) RETURNING id`, project, title, status, sp)
}

func (f *consensusFixture) insertVote(t *testing.T, round, member int64, value int) {
	t.Helper()
	f.exec(t, `INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)`, round, member, value)
}

// closedRound carga una ronda ya cerrada, con closed_at explícito, y sus votos.
func (f *consensusFixture) closedRound(t *testing.T, project, story int64, number int, votes map[int64]int) int64 {
	t.Helper()
	var opener int64
	for member := range votes {
		opener = member
		break
	}
	round := f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at)
		VALUES ($1, $2, $3, 'Cerrada', $4, $5) RETURNING id`, project, story, number, opener, consensusClosedAt)
	for member, value := range votes {
		f.insertVote(t, round, member, value)
	}
	return round
}

// storyRound es una ronda guardada de la historia.
type storyRound struct {
	ID       int64
	Number   int
	Status   string
	OpenedBy int64
	Votes    int
}

// rounds devuelve las rondas guardadas de la historia, por número.
func (f *consensusFixture) storyRounds(t *testing.T, story int64) []storyRound {
	t.Helper()
	rows, err := f.db.Query(`SELECT r.id, r.round_number, r.status, r.opened_by,
		(SELECT COUNT(*) FROM poker_votes v WHERE v.round_id = r.id)
		FROM poker_rounds r WHERE r.story_id = $1 ORDER BY r.round_number`, story)
	if err != nil {
		t.Fatalf("no se pudieron leer las rondas: %v", err)
	}
	defer rows.Close()
	var out []storyRound
	for rows.Next() {
		var r storyRound
		if err := rows.Scan(&r.ID, &r.Number, &r.Status, &r.OpenedBy, &r.Votes); err != nil {
			t.Fatalf("no se pudo leer una ronda: %v", err)
		}
		out = append(out, r)
	}
	return out
}

// storyPoints devuelve los Story Points guardados de la historia; 0 = sin Story Points.
func (f *consensusFixture) storyPoints(t *testing.T, story int64) int {
	t.Helper()
	var sp sql.NullInt64
	if err := f.db.QueryRow(`SELECT story_points FROM user_stories WHERE id = $1`, story).Scan(&sp); err != nil {
		t.Fatalf("no se pudieron leer los Story Points: %v", err)
	}
	return int(sp.Int64)
}

// storyState es lo que la estimación no debe tocar.
type storyState struct {
	Status       string
	ActiveSprint sql.NullInt64
	Rounds       int
}

func (f *consensusFixture) storyState(t *testing.T, story int64) storyState {
	t.Helper()
	var s storyState
	if err := f.db.QueryRow(`SELECT status, active_sprint_id, (SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1)
		FROM user_stories WHERE id = $1`, story).Scan(&s.Status, &s.ActiveSprint, &s.Rounds); err != nil {
		t.Fatalf("no se pudo leer la historia: %v", err)
	}
	return s
}

// vote devuelve el valor guardado del voto del integrante en la ronda; 0 si no votó.
func (f *consensusFixture) vote(t *testing.T, round, member int64) int {
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

// closeRound cierra en la base una ronda abierta (como CloseRound de HU-05b).
func (f *consensusFixture) closeRound(t *testing.T, round int64) {
	t.Helper()
	f.exec(t, `UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`, round)
}

// assignToActiveSprint crea el Sprint 2 "Activo" de «Alfa» (si no existe) y le asigna la historia
// con ese estado. Solo para las pruebas de concurrencia con el cierre de Sprint.
func (f *consensusFixture) assignToActiveSprint(t *testing.T, story int64, status string) {
	t.Helper()
	if f.sprint2 == 0 {
		f.sprint2 = f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, 2, 'Pagos online', '2026-09-15', '2026-09-30', 'Activo') RETURNING id`, f.alfa)
	}
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, f.sprint2, story)
	f.exec(t, `UPDATE user_stories SET active_sprint_id = $1, status = $2 WHERE id = $3`, f.sprint2, status, story)
}

// waitForWaiters espera hasta que n sesiones queden esperando un bloqueo (ayudante de HU-05b).
func (f *consensusFixture) waitForWaiters(t *testing.T, n int) {
	t.Helper()
	waitForWaiters(t, &alfaFixture{db: f.db}, n)
}

// closeSprintLikeHU15 hace, dentro de tx y sin confirmar, los pasos del cierre de Sprint de HU-15
// (sprint.PostgreSQLSprintCloser) que tocan estas tablas: Sprint FOR UPDATE, historias FOR UPDATE,
// resultados, las no "Hecho" vuelven al Backlog sin asignación y el Sprint queda "Finalizado".
func closeSprintLikeHU15(t *testing.T, ctx context.Context, tx *sql.Tx, sprintID int64) {
	t.Helper()
	steps := []string{
		`SELECT status FROM sprints WHERE id = $1 FOR UPDATE`,
		`SELECT id FROM user_stories WHERE active_sprint_id = $1 ORDER BY id FOR UPDATE`,
		`INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		 SELECT $1::bigint, id, status, story_points FROM user_stories WHERE active_sprint_id = $1`,
		`DELETE FROM sprint_story_assignments AS a USING user_stories AS us
		  WHERE a.sprint_id = $1 AND a.story_id = us.id AND us.active_sprint_id = $1 AND us.status <> 'Hecho'`,
		`UPDATE user_stories SET status = 'Pendiente', active_sprint_id = NULL WHERE active_sprint_id = $1 AND status <> 'Hecho'`,
		`UPDATE sprints SET status = 'Finalizado' WHERE id = $1`,
	}
	for _, q := range steps {
		if _, err := tx.ExecContext(ctx, q, sprintID); err != nil {
			t.Errorf("cierre de Sprint de prueba: %v", err)
			return
		}
	}
}

// cleanup: asignaciones → historias (rondas y votos en cascada) → Sprints → proyectos → integrantes.
func (f *consensusFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM sprint_story_assignments WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2))`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM sprints WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3, $4)`, f.ana, f.beto, f.caro, f.dani)
}
