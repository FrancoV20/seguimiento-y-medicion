package web

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func openDefectsCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no está definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}
	return db
}

type defectsCatalogFixture struct {
	projectA, projectB, emptyProject     int64
	storyA, storyDone, storyB            int64
	activeSprintA, finishedSprintA       int64
	oldDefect, newestDefect, otherDefect int64
}

func newDefectsCatalogFixture(t *testing.T, db *sql.DB) defectsCatalogFixture {
	t.Helper()
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	fixture := defectsCatalogFixture{}
	insertID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear un dato de catálogo de prueba: %v", err)
		}
		return id
	}
	insertProject := func(name string) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
			fmt.Sprintf("%s %d", name, stamp),
		)
	}
	insertSprint := func(projectID int64, number int, status string) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			 VALUES ($1, $2, 'goal', '2026-01-01', '2026-01-14', $3) RETURNING id`,
			projectID, number, status,
		)
	}
	insertStory := func(projectID int64, title, status string, sprintID *int64) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
			 VALUES ($1, $2, 'descripción', 'Media', $3, $4) RETURNING id`,
			projectID, fmt.Sprintf("%s %d", title, stamp), status, sprintID,
		)
	}
	insertDefect := func(projectID, storyID, detectionSprintID int64, description string, status string, resolutionSprintID *int64) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
			 VALUES ($1, $2, $3, 'Alta', $4, $5, $6) RETURNING id`,
			projectID, storyID, description, status, detectionSprintID, resolutionSprintID,
		)
	}

	fixture.projectA = insertProject("HU06 Catalog A")
	fixture.projectB = insertProject("HU06 Catalog B")
	fixture.emptyProject = insertProject("HU06 Catalog vacío")
	fixture.activeSprintA = insertSprint(fixture.projectA, 1, "Activo")
	fixture.finishedSprintA = insertSprint(fixture.projectA, 2, "Finalizado")
	activeSprintB := insertSprint(fixture.projectB, 3, "Activo")
	fixture.storyA = insertStory(fixture.projectA, "Historia activa", "En progreso", &fixture.activeSprintA)
	fixture.storyDone = insertStory(fixture.projectA, "Historia finalizada", "Hecho", &fixture.finishedSprintA)
	fixture.storyB = insertStory(fixture.projectB, "Historia ajena", "Pendiente", &activeSprintB)
	fixture.oldDefect = insertDefect(fixture.projectA, fixture.storyA, fixture.activeSprintA, "Defecto anterior", "Abierto", nil)
	fixture.newestDefect = insertDefect(fixture.projectA, fixture.storyDone, fixture.activeSprintA, "Defecto resuelto", "Resuelto", &fixture.finishedSprintA)
	fixture.otherDefect = insertDefect(fixture.projectB, fixture.storyB, activeSprintB, "Defecto de otro proyecto", "Abierto", nil)

	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM user_stories WHERE project_id IN ($1, $2, $3)`,
			fixture.projectA, fixture.projectB, fixture.emptyProject); err != nil {
			t.Errorf("no se pudieron borrar las historias de prueba: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM sprints WHERE project_id IN ($1, $2, $3)`,
			fixture.projectA, fixture.projectB, fixture.emptyProject); err != nil {
			t.Errorf("no se pudieron borrar los Sprints de prueba: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM projects WHERE id IN ($1, $2, $3)`,
			fixture.projectA, fixture.projectB, fixture.emptyProject); err != nil {
			t.Errorf("no se pudieron borrar los proyectos de prueba: %v", err)
		}
	})
	return fixture
}

func TestPostgreSQLDefectCatalog_DefectsAreProjectScopedAndNewestFirst(t *testing.T) {
	db := openDefectsCatalogTestDB(t)
	fixture := newDefectsCatalogFixture(t, db)

	rows, err := NewPostgreSQLDefectCatalog(db).Defects(context.Background(), fixture.projectA)
	if err != nil {
		t.Fatalf("Defects devolvió error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("se esperaban los dos defectos del proyecto A, se obtuvieron %+v", rows)
	}
	if rows[0].ID != fixture.newestDefect || rows[1].ID != fixture.oldDefect {
		t.Errorf("los defectos no están del más reciente al más antiguo: %+v", rows)
	}
	if rows[0].StoryID != fixture.storyDone || rows[0].StoryTitle == "" ||
		rows[0].Severity != defect.SeverityHigh || rows[0].Status != defect.StatusResolved ||
		rows[0].DetectionSprint != 1 || rows[0].ResolutionSprint == nil || *rows[0].ResolutionSprint != 2 {
		t.Errorf("trazabilidad del defecto resuelto incorrecta: %+v", rows[0])
	}
	if rows[1].StoryID != fixture.storyA || rows[1].Status != defect.StatusOpen ||
		rows[1].DetectionSprint != 1 || rows[1].ResolutionSprint != nil {
		t.Errorf("el Sprint de resolución ausente o la historia no se representó correctamente: %+v", rows[1])
	}
	for _, row := range rows {
		if row.ID == fixture.otherDefect || row.Description == "Defecto de otro proyecto" {
			t.Errorf("se filtró un defecto de otro proyecto: %+v", row)
		}
	}
}

func TestPostgreSQLDefectCatalog_StoryOptionsAndActiveSprint(t *testing.T) {
	db := openDefectsCatalogTestDB(t)
	fixture := newDefectsCatalogFixture(t, db)
	catalog := NewPostgreSQLDefectCatalog(db)
	ctx := context.Background()

	options, err := catalog.StoryOptions(ctx, fixture.projectA)
	if err != nil {
		t.Fatalf("StoryOptions devolvió error: %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("se esperaban solo las historias del proyecto A, se obtuvieron %+v", options)
	}
	if options[0].ID != fixture.storyDone || options[0].Status != "Hecho" ||
		options[1].ID != fixture.storyA || options[1].Status != "En progreso" {
		t.Errorf("opciones de historia incorrectas u ordenadas incorrectamente: %+v", options)
	}
	for _, option := range options {
		if option.ID == fixture.storyB {
			t.Errorf("se filtró una historia de otro proyecto: %+v", option)
		}
	}

	number, active, err := catalog.ActiveSprintNumber(ctx, fixture.projectA)
	if err != nil {
		t.Fatalf("ActiveSprintNumber devolvió error: %v", err)
	}
	if !active || number != 1 {
		t.Errorf("Sprint activo del proyecto A: número=%d activo=%v", number, active)
	}
	number, active, err = catalog.ActiveSprintNumber(ctx, fixture.emptyProject)
	if err != nil {
		t.Fatalf("ActiveSprintNumber para proyecto sin Sprint devolvió error: %v", err)
	}
	if active || number != 0 {
		t.Errorf("proyecto sin Sprint activo: número=%d activo=%v", number, active)
	}

	rows, err := catalog.Defects(ctx, fixture.emptyProject)
	if err != nil {
		t.Fatalf("Defects para un proyecto vacío devolvió error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("se esperaba una lista vacía, se obtuvo %+v", rows)
	}
}
