package web

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

// HU-05a, T019-T020: PostgreSQLPokerCatalog contra PostgreSQL real, con fixtures propios.

// pokerWebFixture: proyecto A con Ana y Beto e historias «Login» (Pendiente) y «Pagos» (Hecho);
// proyecto B con Caro y una historia; proyecto vacío sin historias.
type pokerWebFixture struct {
	db                             *sql.DB
	stamp                          string
	projectA, projectB, empty      int64
	ana, beto, caro                int64
	anaName, betoName              string
	login, pagos, storyB           int64
	loginTitle, pagosTitle, bTitle string
}

func newPokerWebFixture(t *testing.T, db *sql.DB) *pokerWebFixture {
	t.Helper()
	f := &pokerWebFixture{db: db, stamp: fmt.Sprint(time.Now().UnixNano())}
	f.anaName, f.betoName = "Ana Poker "+f.stamp, "Beto Poker "+f.stamp
	f.loginTitle, f.pagosTitle, f.bTitle = "Login Poker "+f.stamp, "Pagos Poker "+f.stamp, "Otra Poker "+f.stamp
	t.Cleanup(f.cleanup)

	project := func(name string) int64 {
		return f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, name+" "+f.stamp)
	}
	f.projectA, f.projectB, f.empty = project("Poker A"), project("Poker B"), project("Poker vacío")
	f.ana = f.member(t, f.anaName, f.projectA)
	f.beto = f.member(t, f.betoName, f.projectA)
	f.caro = f.member(t, "Caro Poker "+f.stamp, f.projectB)
	f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, f.empty, f.ana)
	f.login = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status) VALUES ($1, $2, 'd', 'Alta', 'Pendiente') RETURNING id`, f.projectA, f.loginTitle)
	f.pagos = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, story_points) VALUES ($1, $2, 'd', 'Media', 'Hecho', 5) RETURNING id`, f.projectA, f.pagosTitle)
	f.storyB = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Baja') RETURNING id`, f.projectB, f.bTitle)
	return f
}

func (f *pokerWebFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *pokerWebFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *pokerWebFixture) member(t *testing.T, name string, project int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, id)
	return id
}

// round inserta una ronda directamente (abierta o cerrada).
func (f *pokerWebFixture) round(t *testing.T, project, story int64, open bool) int64 {
	t.Helper()
	if open {
		return f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 1, $3) RETURNING id`, project, story, f.ana)
	}
	return f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at) VALUES ($1, $2, 1, 'Cerrada', $3, now()) RETURNING id`, project, story, f.ana)
}

func (f *pokerWebFixture) vote(t *testing.T, round, member int64, value int) {
	t.Helper()
	f.exec(t, `INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (round_id, member_id) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, round, member, value)
}

func (f *pokerWebFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2, $3)`, f.projectA, f.projectB, f.empty)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2, $3)`, f.projectA, f.projectB, f.empty)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3)`, f.ana, f.beto, f.caro)
}

func TestPostgresPokerCatalog_Historias(t *testing.T) {
	db := openContextTestDB(t)
	f := newPokerWebFixture(t, db)
	board, err := NewPostgreSQLPokerCatalog(db).PokerBoard(context.Background(), f.projectA, f.ana)
	if err != nil {
		t.Fatalf("PokerBoard devolvió error: %v", err)
	}
	want := []PokerStoryOption{{ID: f.login, Title: f.loginTitle, Status: "Pendiente"}, {ID: f.pagos, Title: f.pagosTitle, Status: "Hecho"}}
	if len(board.Stories) != 2 || board.Stories[0] != want[0] || board.Stories[1] != want[1] {
		t.Errorf("se esperaban todas las historias del proyecto en orden, se obtuvo %+v", board.Stories)
	}
}

func TestPostgresPokerCatalog_RondasAbiertas(t *testing.T) {
	db := openContextTestDB(t)
	f := newPokerWebFixture(t, db)
	r1 := f.round(t, f.projectA, f.pagos, true)
	r2 := f.round(t, f.projectA, f.login, true)
	f.round(t, f.projectB, f.storyB, true)
	board, err := NewPostgreSQLPokerCatalog(db).PokerBoard(context.Background(), f.projectA, f.ana)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Rounds) != 2 || board.Rounds[0].ID != r1 || board.Rounds[1].ID != r2 {
		t.Fatalf("se esperaban las dos rondas abiertas del proyecto en orden de apertura, se obtuvo %+v", board.Rounds)
	}
	if board.Rounds[1].StoryID != f.login || board.Rounds[1].StoryTitle != f.loginTitle || board.Rounds[1].Number != 1 {
		t.Errorf("datos de ronda inesperados: %+v", board.Rounds[1])
	}

	// Una ronda cerrada no se muestra.
	f.exec(t, `UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`, r1)
	board, _ = NewPostgreSQLPokerCatalog(db).PokerBoard(context.Background(), f.projectA, f.ana)
	if len(board.Rounds) != 1 || board.Rounds[0].ID != r2 {
		t.Errorf("solo debía quedar la ronda abierta, se obtuvo %+v", board.Rounds)
	}
}

func TestPostgresPokerCatalog_Vacio(t *testing.T) {
	db := openContextTestDB(t)
	f := newPokerWebFixture(t, db)
	board, err := NewPostgreSQLPokerCatalog(db).PokerBoard(context.Background(), f.empty, f.ana)
	if err != nil || len(board.Stories) != 0 || len(board.Rounds) != 0 {
		t.Errorf("se esperaba un tablero vacío, se obtuvo %+v, %v", board, err)
	}
}

func TestPostgresPokerCatalog_Participantes(t *testing.T) {
	db := openContextTestDB(t)
	f := newPokerWebFixture(t, db)
	r := f.round(t, f.projectA, f.login, true)
	f.vote(t, r, f.ana, 5)
	board, err := NewPostgreSQLPokerCatalog(db).PokerBoard(context.Background(), f.projectA, f.beto)
	if err != nil || len(board.Rounds) != 1 {
		t.Fatalf("se esperaba una ronda, se obtuvo %+v, %v", board, err)
	}
	want := []PokerParticipantView{
		{MemberID: f.ana, Name: f.anaName, HasVoted: true, IsMe: false},
		{MemberID: f.beto, Name: f.betoName, HasVoted: false, IsMe: true},
	}
	got := board.Rounds[0].Participants
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("participantes inesperados: %+v", got)
	}
}

func TestPostgresPokerCatalog_VotoPropio(t *testing.T) {
	db := openContextTestDB(t)
	f := newPokerWebFixture(t, db)
	r := f.round(t, f.projectA, f.login, true)
	f.vote(t, r, f.ana, 8)
	catalog := NewPostgreSQLPokerCatalog(db)
	board, _ := catalog.PokerBoard(context.Background(), f.projectA, f.ana)
	if own := board.Rounds[0].OwnVote; own == nil || *own != 8 {
		t.Errorf("se esperaba el voto propio 8, se obtuvo %v", own)
	}
	board, _ = catalog.PokerBoard(context.Background(), f.projectA, f.beto)
	if own := board.Rounds[0].OwnVote; own != nil {
		t.Errorf("Beto no votó: se esperaba nil, se obtuvo %d", *own)
	}
}

func TestPostgresPokerCatalog_SinValoresAjenos(t *testing.T) {
	db := openContextTestDB(t)
	f := newPokerWebFixture(t, db)
	r := f.round(t, f.projectA, f.login, true)
	f.vote(t, r, f.ana, 13)
	f.vote(t, r, f.beto, 3)
	board, err := NewPostgreSQLPokerCatalog(db).PokerBoard(context.Background(), f.projectA, f.beto)
	if err != nil {
		t.Fatal(err)
	}
	if own := board.Rounds[0].OwnVote; own == nil || *own != 3 {
		t.Errorf("se esperaba el voto propio 3, se obtuvo %v", own)
	}
	// Nada del tablero de Beto contiene el 13 de Ana (salvo coincidencias de identificadores).
	dump := fmt.Sprintf("%+v", board.Rounds[0].Participants)
	if strings.Contains(dump, ":13 ") || strings.Contains(dump, ":13}") {
		t.Errorf("el tablero de Beto contiene el valor ajeno: %s", dump)
	}
}
