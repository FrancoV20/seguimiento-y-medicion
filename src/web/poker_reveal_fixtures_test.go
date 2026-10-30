package web

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

// HU-05b, T015: datos del Issue para las pruebas de la pantalla (los mismos de src/poker).
//
// «Alfa» con Ana, Beto y Caro; ronda 1 abierta por Ana en cada historia:
//
//	«Login» 2/13/5, «Pagos» 3/5/8, «Reportes» 3/13/3, «Perfil» 5/5 (Caro no votó), «Ayuda» sin votos.
//
// «Beta» con Dani: «Tablero» con una ronda abierta (Dani 8) y «Agenda» con una ronda cerrada sin
// votos.
type alfaWebFixture struct {
	db                    *sql.DB
	alfa, beta            int64
	ana, beto, caro, dani int64
	stories               map[string]int64 // historia por título
	rounds                map[string]int64 // ronda 1 por título de la historia
	extraMembers          []int64          // integrantes agregados por una prueba
}

// alfaBase es la hora de referencia para las fechas de cierre explícitas.
var alfaBase = time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)

func newAlfaWebFixture(t *testing.T, db *sql.DB) *alfaWebFixture {
	t.Helper()
	f := &alfaWebFixture{db: db, stories: map[string]int64{}, rounds: map[string]int64{}}
	t.Cleanup(f.cleanup)

	f.alfa = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', '2026-01-01', '2026-12-31') RETURNING id`)
	f.beta = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', '2026-01-01', '2026-12-31') RETURNING id`)
	f.ana = f.member(t, "Ana", f.alfa)
	f.beto = f.member(t, "Beto", f.alfa)
	f.caro = f.member(t, "Caro", f.alfa)
	f.dani = f.member(t, "Dani", f.beta)

	for _, r := range []struct {
		title           string
		ana, beto, caro int // 0 = no votó
	}{
		{"Login", 2, 13, 5},
		{"Pagos", 3, 5, 8},
		{"Reportes", 3, 13, 3},
		{"Perfil", 5, 5, 0},
		{"Ayuda", 0, 0, 0},
	} {
		round := f.openRound(t, f.alfa, r.title, f.ana)
		for member, value := range map[int64]int{f.ana: r.ana, f.beto: r.beto, f.caro: r.caro} {
			if value != 0 {
				f.setVote(t, round, member, value)
			}
		}
	}
	f.setVote(t, f.openRound(t, f.beta, "Tablero", f.dani), f.dani, 8)
	f.closeAt(t, f.openRound(t, f.beta, "Agenda", f.dani), alfaBase.Add(-24*time.Hour))
	return f
}

func (f *alfaWebFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *alfaWebFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *alfaWebFixture) member(t *testing.T, name string, project int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, id)
	return id
}

// addMember suma un integrante a «Alfa» y lo borra al final.
func (f *alfaWebFixture) addMember(t *testing.T, name string) int64 {
	t.Helper()
	id := f.member(t, name, f.alfa)
	f.extraMembers = append(f.extraMembers, id)
	return id
}

// openRound crea la historia y su ronda 1 abierta; devuelve la ronda.
func (f *alfaWebFixture) openRound(t *testing.T, project int64, title string, openedBy int64) int64 {
	t.Helper()
	story := f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Media') RETURNING id`,
		project, title)
	round := f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 1, $3) RETURNING id`,
		project, story, openedBy)
	f.stories[title], f.rounds[title] = story, round
	return round
}

// setVote registra o reemplaza un voto directamente en la base.
func (f *alfaWebFixture) setVote(t *testing.T, round, member int64, value int) {
	t.Helper()
	f.exec(t, `INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (round_id, member_id) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`, round, member, value)
}

// closeAt cierra la ronda con una fecha explícita, para que el orden sea determinista.
func (f *alfaWebFixture) closeAt(t *testing.T, round int64, ts time.Time) {
	t.Helper()
	f.exec(t, `UPDATE poker_rounds SET status = 'Cerrada', closed_at = $2 WHERE id = $1`, round, ts)
}

// removeMember quita al integrante de «Alfa» (sigue existiendo en members, como sus votos).
func (f *alfaWebFixture) removeMember(t *testing.T, member int64) {
	t.Helper()
	f.exec(t, `DELETE FROM project_members WHERE project_id = $1 AND member_id = $2`, f.alfa, member)
}

func (f *alfaWebFixture) status(t *testing.T, round int64) string {
	t.Helper()
	var s string
	if err := f.db.QueryRow(`SELECT status FROM poker_rounds WHERE id = $1`, round).Scan(&s); err != nil {
		t.Fatalf("no se pudo leer el estado de la ronda: %v", err)
	}
	return s
}

func (f *alfaWebFixture) closedAt(t *testing.T, round int64) sql.NullTime {
	t.Helper()
	var at sql.NullTime
	if err := f.db.QueryRow(`SELECT closed_at FROM poker_rounds WHERE id = $1`, round).Scan(&at); err != nil {
		t.Fatalf("no se pudo leer la fecha de cierre: %v", err)
	}
	return at
}

// vote devuelve el valor guardado del voto; 0 si no votó.
func (f *alfaWebFixture) vote(t *testing.T, round, member int64) int {
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

func (f *alfaWebFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.alfa, f.beta)
	members := append([]int64{f.ana, f.beto, f.caro, f.dani}, f.extraMembers...)
	for _, m := range members {
		_, _ = f.db.Exec(`DELETE FROM members WHERE id = $1`, m)
	}
}

// roundSection devuelve el contenido de la sección de una ronda abierta ("" si no está).
func roundSection(body string, round int64) string {
	m := regexp.MustCompile(fmt.Sprintf(`(?s)<section class="poker-round" data-ronda="%d"[^>]*>(.*?)</section>`, round)).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// closedSection devuelve el contenido de la sección de una ronda cerrada ("" si no está).
func closedSection(body string, round int64) string {
	m := regexp.MustCompile(fmt.Sprintf(`(?s)<section class="poker-closed-round" data-ronda-cerrada="%d"[^>]*>(.*?)</section>`, round)).FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// closedRoundsSection devuelve la sección "Rondas cerradas" completa, hasta el final del contenido.
func closedRoundsSection(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<section aria-labelledby="poker-closed-title">`)
	end := strings.Index(body, `</main>`)
	if start < 0 || end < start {
		t.Fatal("no se encontró la sección «Rondas cerradas»")
	}
	return body[start:end]
}

// memberRow devuelve el texto de la fila data-integrante dentro de una sección.
func memberRow(t *testing.T, section, where string, member int64) string {
	t.Helper()
	m := regexp.MustCompile(fmt.Sprintf(`<li data-integrante="%d">(.*?)</li>`, member)).FindStringSubmatch(section)
	if m == nil {
		t.Fatalf("no se encontró la fila del integrante %d en %s", member, where)
	}
	return m[1]
}

// roundRow devuelve la fila del integrante en la ronda abierta (el atributo data-integrante se
// repite entre secciones, por eso se busca dentro de la ronda).
func roundRow(t *testing.T, body string, round, member int64) string {
	t.Helper()
	return memberRow(t, roundSection(body, round), fmt.Sprintf("la ronda abierta %d", round), member)
}

// closedRow devuelve la fila del integrante en la ronda cerrada.
func closedRow(t *testing.T, body string, round, member int64) string {
	t.Helper()
	return memberRow(t, closedSection(body, round), fmt.Sprintf("la ronda cerrada %d", round), member)
}
