package defect

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// HU-19 (T011): fixtures de integración para resolver y reabrir defectos. Cada prueba crea dos
// proyectos con una historia cada uno; los Sprints y defectos se agregan según el escenario. Al
// terminar se borran las historias (los defectos caen en cascada), los Sprints y los proyectos.

type resolutionFixture struct {
	db                 *sql.DB
	projectA, projectB int64
	storyA, storyB     int64
	stamp              int64
}

// defectSnapshot es el estado completo de un defecto, para comprobar que no cambió.
type defectSnapshot struct {
	Status           string
	ResolutionSprint sql.NullInt64
	DetectionSprint  int64
	Description      string
	Severity         string
	StoryID          int64
	ProjectID        int64
}

func newResolutionFixture(t *testing.T) *resolutionFixture {
	t.Helper()
	db := openDefectTestDB(t)
	f := &resolutionFixture{db: db, stamp: time.Now().UnixNano()}
	f.projectA = f.insertID(t,
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU19 Resolución A %d", f.stamp),
	)
	f.projectB = f.insertID(t,
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU19 Resolución B %d", f.stamp),
	)
	t.Cleanup(func() {
		ctx := context.Background()
		for _, query := range []string{
			`DELETE FROM user_stories WHERE project_id IN ($1, $2)`,
			`DELETE FROM sprints WHERE project_id IN ($1, $2)`,
			`DELETE FROM projects WHERE id IN ($1, $2)`,
		} {
			if _, err := db.ExecContext(ctx, query, f.projectA, f.projectB); err != nil {
				t.Errorf("no se pudieron borrar los datos de prueba de HU-19: %v", err)
			}
		}
	})
	f.storyA = f.story(t, f.projectA)
	f.storyB = f.story(t, f.projectB)
	return f
}

func (f *resolutionFixture) insertID(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear un dato de prueba de HU-19: %v", err)
	}
	return id
}

// story crea una historia "Pendiente" en el proyecto.
func (f *resolutionFixture) story(t *testing.T, projectID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO user_stories (project_id, title, description, priority, status)
		 VALUES ($1, $2, 'Historia con defectos', 'Media', 'Pendiente') RETURNING id`,
		projectID, fmt.Sprintf("HU19 historia %d %d", projectID, f.stamp),
	)
}

// sprint crea un Sprint con el número y el estado ("Activo" o "Finalizado") indicados.
func (f *resolutionFixture) sprint(t *testing.T, projectID int64, number int, status string) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, $2, $3, '2026-01-01', '2026-01-14', $4) RETURNING id`,
		projectID, number, fmt.Sprintf("Objetivo HU19 %d", number), status,
	)
}

// openDefect crea un defecto "Abierto" detectado en el Sprint indicado.
func (f *resolutionFixture) openDefect(t *testing.T, projectID, storyID, detectionSprintID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		 VALUES ($1, $2, 'Defecto de HU-19', 'Alta', 'Abierto', $3) RETURNING id`,
		projectID, storyID, detectionSprintID,
	)
}

// resolvedDefect crea un defecto "Resuelto" con su Sprint de detección y de resolución.
func (f *resolutionFixture) resolvedDefect(t *testing.T, projectID, storyID, detectionSprintID, resolutionSprintID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		 VALUES ($1, $2, 'Defecto resuelto de HU-19', 'Media', 'Resuelto', $3, $4) RETURNING id`,
		projectID, storyID, detectionSprintID, resolutionSprintID,
	)
}

// snapshot lee el estado completo del defecto.
func (f *resolutionFixture) snapshot(t *testing.T, defectID int64) defectSnapshot {
	t.Helper()
	var s defectSnapshot
	if err := f.db.QueryRowContext(context.Background(),
		`SELECT status, resolution_sprint_id, detection_sprint_id, description, severity, story_id, project_id
		 FROM defects WHERE id = $1`,
		defectID,
	).Scan(&s.Status, &s.ResolutionSprint, &s.DetectionSprint, &s.Description, &s.Severity, &s.StoryID, &s.ProjectID); err != nil {
		t.Fatalf("no se pudo leer el defecto %d: %v", defectID, err)
	}
	return s
}

// assertUnchanged comprueba que el defecto conserva exactamente el estado anterior.
func (f *resolutionFixture) assertUnchanged(t *testing.T, defectID int64, before defectSnapshot) {
	t.Helper()
	if after := f.snapshot(t, defectID); after != before {
		t.Errorf("el defecto %d cambió: antes %+v, después %+v", defectID, before, after)
	}
}
