package web

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-12 (T009): datos de prueba de "Estado del proyecto". Cada prueba arma sus propios
// proyectos, con nombre único, y los borra al terminar.

// statusFixture es un proyecto de prueba de HU-12.
type statusFixture struct {
	ProjectID int64
	Name      string
}

// newStatusProject crea un proyecto con las fechas dadas (aaaa-mm-dd) y, al terminar, borra
// asignaciones, historias, Sprints y el proyecto, en ese orden.
func newStatusProject(t *testing.T, db *sql.DB, label, start, end string) statusFixture {
	t.Helper()
	name := fmt.Sprintf("%s %d", label, time.Now().UnixNano())
	var id int64
	err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, $2::date, $3::date, 'Activo')
		RETURNING id`, name, start, end).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba %q: %v", label, err)
	}
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM sprint_story_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM sprints WHERE project_id = $1`,
			`DELETE FROM projects WHERE id = $1`,
		} {
			if _, err := db.Exec(query, id); err != nil {
				t.Errorf("no se pudo limpiar el proyecto de prueba %d: %v", id, err)
			}
		}
	})
	return statusFixture{ProjectID: id, Name: name}
}

// newStatusSprint crea un Sprint con número, objetivo, fechas (aaaa-mm-dd) y estado dados.
func newStatusSprint(t *testing.T, db *sql.DB, projectID int64, number int, goal, start, end, status string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4::date, $5::date, $6)
		RETURNING id`, projectID, number, goal, start, end, status).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint %d de prueba: %v", number, err)
	}
	return id
}

// addStatusStories crea count historias con el estado dado; si sprintID es mayor que cero,
// las asigna a ese Sprint.
func addStatusStories(t *testing.T, db *sql.DB, projectID, sprintID int64, status sprint.StoryStatus, count int) {
	t.Helper()
	for i := 1; i <= count; i++ {
		title := fmt.Sprintf("Historia %s %d.%d", status, sprintID, i)
		newCatStory(t, db, projectID, sprintID, title, nil, status)
	}
}

// seedStatusAlfa: 01/10/2026 al 30/11/2026; 2 Pendiente, 1 En Sprint, 1 En progreso y 4 Hecho (8).
// El Sprint 1 está finalizado (3 Hecho) y el Sprint 2 "Pagos online" está activo.
func seedStatusAlfa(t *testing.T, db *sql.DB) statusFixture {
	t.Helper()
	p := newStatusProject(t, db, "Alfa", "2026-10-01", "2026-11-30")
	s1 := newStatusSprint(t, db, p.ProjectID, 1, "Dar de alta usuarios", "2026-10-05", "2026-10-18", "Finalizado")
	addStatusStories(t, db, p.ProjectID, s1, catDone, 3)
	s2 := newStatusSprint(t, db, p.ProjectID, 2, "Pagos online", "2026-10-19", "2026-11-01", "Activo")
	addStatusStories(t, db, p.ProjectID, s2, catDone, 1)
	addStatusStories(t, db, p.ProjectID, s2, catEnSprint, 1)
	addStatusStories(t, db, p.ProjectID, s2, catInProgress, 1)
	addStatusStories(t, db, p.ProjectID, 0, catPending, 2)
	return p
}

// seedStatusBeta: 01/12/2026 al 31/01/2027, sin historias ni Sprints.
func seedStatusBeta(t *testing.T, db *sql.DB) statusFixture {
	t.Helper()
	return newStatusProject(t, db, "Beta", "2026-12-01", "2027-01-31")
}

// seedStatusGamma: 01/08/2026 al 30/09/2026; 3 historias Hecho de un Sprint 1 finalizado, sin Sprint activo.
func seedStatusGamma(t *testing.T, db *sql.DB) statusFixture {
	t.Helper()
	p := newStatusProject(t, db, "Gamma", "2026-08-01", "2026-09-30")
	s1 := newStatusSprint(t, db, p.ProjectID, 1, "Primer Sprint", "2026-08-01", "2026-08-14", "Finalizado")
	addStatusStories(t, db, p.ProjectID, s1, catDone, 3)
	return p
}

// seedStatusDelta: empieza el 28/10/2026 (hoy en las pruebas), sin historias ni Sprints.
func seedStatusDelta(t *testing.T, db *sql.DB) statusFixture {
	t.Helper()
	return newStatusProject(t, db, "Delta", "2026-10-28", "2026-12-15")
}

// seedStatusEpsilon: termina el 28/10/2026 (hoy en las pruebas), sin historias ni Sprints.
func seedStatusEpsilon(t *testing.T, db *sql.DB) statusFixture {
	t.Helper()
	return newStatusProject(t, db, "Épsilon", "2026-09-01", "2026-10-28")
}
