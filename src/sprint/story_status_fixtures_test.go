package sprint

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"
)

// Fixtures de HU-14 (T006 y T007): cada prueba crea sus propios proyectos, Sprints, historias y
// horas en PostgreSQL y los elimina al terminar.

// openStatusTestDB abre la base de DATABASE_URL; si no está definida, omite la prueba.
func openStatusTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL no está definida; se omite la prueba PostgreSQL")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// newStatusServiceForTest arma el servicio real sobre PostgreSQL.
func newStatusServiceForTest(t *testing.T) (StatusService, *sql.DB) {
	t.Helper()
	db := openStatusTestDB(t)
	return NewStatusService(NewPostgreSQLStoryStatusStore(db)), db
}

// newStatusProject crea un proyecto (fin posterior al inicio, regla de FIX-HU-01) y lo elimina al
// terminar junto con sus historias y horas. Sus Sprints caen en cascada.
func newStatusProject(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var projectID int64
	err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, $2, $3, 'Activo')
		RETURNING id`,
		name,
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	).Scan(&projectID)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba: %v", err)
	}
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM projects WHERE id = $1`,
		} {
			if _, err := db.Exec(query, projectID); err != nil {
				t.Errorf("no se pudo limpiar el proyecto de prueba %d: %v", projectID, err)
			}
		}
	})
	return projectID
}

// newStatusSprint crea un Sprint del proyecto con el estado indicado (Activo o Finalizado).
func newStatusSprint(t *testing.T, db *sql.DB, projectID int64, status SprintStatus) int64 {
	t.Helper()
	var number int
	if err := db.QueryRow(
		`SELECT COALESCE(MAX(number), 0) + 1 FROM sprints WHERE project_id = $1`, projectID,
	).Scan(&number); err != nil {
		t.Fatalf("no se pudo calcular el número de Sprint: %v", err)
	}
	var sprintID int64
	err := db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		projectID, number, "Objetivo de HU-14",
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		string(status),
	).Scan(&sprintID)
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint de prueba: %v", err)
	}
	return sprintID
}

// newStatusStory crea una historia del proyecto en el estado indicado. Si sprintID es mayor que
// cero, la asigna a ese Sprint como lo hace HU-04 (asignación y active_sprint_id).
func newStatusStory(t *testing.T, db *sql.DB, projectID int64, status StoryStatus, sprintID int64) int64 {
	t.Helper()
	var storyID int64
	err := db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
		VALUES ($1, 'Historia de HU-14', 'Historia para probar el cambio de estado', 'Media', $2, $3)
		RETURNING id`,
		projectID, string(status), sql.NullInt64{Int64: sprintID, Valid: sprintID > 0},
	).Scan(&storyID)
	if err != nil {
		t.Fatalf("no se pudo crear la historia de prueba: %v", err)
	}
	if sprintID > 0 {
		if _, err := db.Exec(
			`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID,
		); err != nil {
			t.Fatalf("no se pudo asignar la historia al Sprint: %v", err)
		}
	}
	return storyID
}

// newStoryInSprint crea un proyecto con un Sprint en el estado indicado y una historia asignada.
func newStoryInSprint(t *testing.T, db *sql.DB, sprintStatus SprintStatus, status StoryStatus) (projectID, storyID int64) {
	t.Helper()
	projectID = newStatusProject(t, db, "Proyecto de HU-14")
	sprintID := newStatusSprint(t, db, projectID, sprintStatus)
	storyID = newStatusStory(t, db, projectID, status, sprintID)
	return projectID, storyID
}

// addStatusEffort registra horas de un integrante en la historia (HU-03).
func addStatusEffort(t *testing.T, db *sql.DB, storyID int64) {
	t.Helper()
	var memberID int64
	if err := db.QueryRow(
		`INSERT INTO members (display_name) VALUES ('Integrante de HU-14') RETURNING id`,
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, memberID); err != nil {
			t.Errorf("no se pudieron limpiar las horas de prueba: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM members WHERE id = $1`, memberID); err != nil {
			t.Errorf("no se pudo limpiar el integrante de prueba: %v", err)
		}
	})
	if _, err := db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, $3, 'Trabajo de HU-14', 2)`,
		storyID, memberID, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("no se pudieron registrar las horas de prueba: %v", err)
	}
}

// storyStatusInDB lee el estado guardado de la historia.
func storyStatusInDB(t *testing.T, db *sql.DB, storyID int64) StoryStatus {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM user_stories WHERE id = $1`, storyID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado de la historia %d: %v", storyID, err)
	}
	return StoryStatus(status)
}

// wantStoryStatus verifica el estado guardado de la historia.
func wantStoryStatus(t *testing.T, db *sql.DB, storyID int64, want StoryStatus) {
	t.Helper()
	if got := storyStatusInDB(t, db, storyID); got != want {
		t.Errorf("estado guardado de la historia = %q, se esperaba %q", got, want)
	}
}

// wantDomainError verifica que err sea el error del dominio esperado.
func wantDomainError(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, se esperaba %v", err, want)
	}
}
