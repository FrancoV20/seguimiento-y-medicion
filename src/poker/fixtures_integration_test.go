package poker

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// HU-05a, T010: datos de prueba propios para las pruebas contra PostgreSQL.

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// pokerFixture: proyecto A con Ana y Beto e historias «Login» (Pendiente, sin Story Points) y
// «Pagos» (Hecho, 5 SP); proyecto B con Caro y una historia.
type pokerFixture struct {
	db                 *sql.DB
	stamp              string
	projectA, projectB int64
	ana, beto, caro    int64
	login, pagos       int64
	storyB             int64
	loginTitle         string
}

func newPokerFixture(t *testing.T) *pokerFixture {
	t.Helper()
	db := openTestDB(t)
	f := &pokerFixture{db: db, stamp: fmt.Sprint(time.Now().UnixNano())}
	f.loginTitle = "Login HU05a " + f.stamp
	t.Cleanup(f.cleanup)

	f.projectA = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, "Poker A "+f.stamp)
	f.projectB = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, "Poker B "+f.stamp)
	f.ana = f.member(t, "Ana HU05a "+f.stamp, f.projectA)
	f.beto = f.member(t, "Beto HU05a "+f.stamp, f.projectA)
	f.caro = f.member(t, "Caro HU05a "+f.stamp, f.projectB)
	f.login = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status) VALUES ($1, $2, 'd', 'Alta', 'Pendiente') RETURNING id`, f.projectA, f.loginTitle)
	f.pagos = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, story_points) VALUES ($1, $2, 'd', 'Media', 'Hecho', 5) RETURNING id`, f.projectA, "Pagos HU05a "+f.stamp)
	f.storyB = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Baja') RETURNING id`, f.projectB, "Otra HU05a "+f.stamp)
	return f
}

func (f *pokerFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *pokerFixture) member(t *testing.T, name string, project int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	if _, err := f.db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, id); err != nil {
		t.Fatalf("no se pudo vincular el integrante: %v", err)
	}
	return id
}

// insertClosedRound prepara una ronda ya cerrada (como la dejará HU-05b).
func (f *pokerFixture) insertClosedRound(t *testing.T, story, project int64) int64 {
	t.Helper()
	return f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at)
		VALUES ($1, $2, 1, 'Cerrada', $3, now()) RETURNING id`, project, story, f.ana)
}

func (f *pokerFixture) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("no se pudo contar: %v", err)
	}
	return n
}

// cleanup: historias (las rondas y los votos caen en cascada) → proyectos → integrantes.
func (f *pokerFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.projectA, f.projectB)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.projectA, f.projectB)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3)`, f.ana, f.beto, f.caro)
}
