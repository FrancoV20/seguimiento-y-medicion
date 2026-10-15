package sprint

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"
)

func TestPostgreSQLSprintStoreCreateSprintAndSequence(t *testing.T) {
	store, projectID, storyIDs := setupPostgreSQLSprintTest(t, 3)
	ctx := context.Background()

	first, err := store.CreateSprint(ctx, integrationSprint(projectID, storyIDs))
	if err != nil {
		t.Fatalf("CreateSprint() devolvió error: %v", err)
	}
	if first.DisplayID != "Sprint 1" || first.Status != SprintActive || len(first.StoryIDs) != 3 {
		t.Fatalf("Sprint creado con valores incorrectos: %+v", first)
	}

	var assignments, inSprint int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sprint_story_assignments WHERE sprint_id = $1`,
		first.ID,
	).Scan(&assignments); err != nil {
		t.Fatalf("no se pudieron contar asociaciones: %v", err)
	}
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM user_stories WHERE active_sprint_id = $1 AND status = $2`,
		first.ID,
		EnSprint,
	).Scan(&inSprint); err != nil {
		t.Fatalf("no se pudieron verificar estados de historias: %v", err)
	}
	if assignments != 3 || inSprint != 3 {
		t.Fatalf("asociaciones=%d e historias En Sprint=%d; se esperaban 3 de cada", assignments, inSprint)
	}

	second, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatalf("la creación del segundo Sprint devolvió error: %v", err)
	}
	if second.DisplayID != "Sprint 2" {
		t.Errorf("segundo DisplayID = %q, se esperaba %q", second.DisplayID, "Sprint 2")
	}
}

func TestPostgreSQLSprintStoreAssignmentConflictRollsBack(t *testing.T) {
	store, projectID, storyIDs := setupPostgreSQLSprintTest(t, 2)
	ctx := context.Background()

	first, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AssignStories(ctx, first.ID, []int64{storyIDs[1]}); err != nil {
		t.Fatalf("primera asignación falló: %v", err)
	}

	_, err = store.CreateSprint(ctx, integrationSprint(projectID, storyIDs))
	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("CreateSprint() error = %v, se esperaba conflicto de Sprint activo", err)
	}

	var status string
	var activeSprintID sql.NullInt64
	if err := store.db.QueryRowContext(ctx,
		`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`,
		storyIDs[0],
	).Scan(&status, &activeSprintID); err != nil {
		t.Fatalf("no se pudo inspeccionar la historia tras rollback: %v", err)
	}
	if status != string(Pendiente) || activeSprintID.Valid {
		t.Fatalf("la primera historia quedó parcialmente asignada: estado=%q sprint=%v", status, activeSprintID)
	}

	var sprintCount int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sprints WHERE project_id = $1`,
		projectID,
	).Scan(&sprintCount); err != nil {
		t.Fatalf("no se pudo comprobar el rollback del Sprint: %v", err)
	}
	if sprintCount != 2 {
		t.Errorf("cantidad de Sprints = %d, se esperaban 2 tras el rechazo", sprintCount)
	}
	third, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint posterior al rechazo: %v", err)
	}
	if third.DisplayID != "Sprint 3" {
		t.Errorf("DisplayID posterior al rechazo = %q, se esperaba Sprint 3", third.DisplayID)
	}

	if err := store.RemoveStory(ctx, first.ID, storyIDs[1]); err != nil {
		t.Fatalf("RemoveStory() falló: %v", err)
	}
	if err := store.AssignStories(ctx, second.ID, []int64{storyIDs[1]}); err != nil {
		t.Fatalf("reasignación tras quitar la historia falló: %v", err)
	}
}

func setupPostgreSQLSprintTest(t *testing.T, storyCount int) (*PostgreSQLSprintStore, int64, []int64) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL no está definida; se omite la prueba PostgreSQL")
	}
	store, err := NewPostgreSQLSprintStore(dsn)
	if err != nil {
		t.Fatalf("NewPostgreSQLSprintStore() devolvió error: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("no se pudo cerrar la conexión PostgreSQL: %v", err)
		}
	})

	ctx := context.Background()
	var projectID int64
	err = store.db.QueryRowContext(ctx, `
		INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		"Proyecto de integración HU-04",
		time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		"Activo",
	).Scan(&projectID)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba: %v", err)
	}
	t.Cleanup(func() {
		if _, err := store.db.ExecContext(ctx, `DELETE FROM user_stories WHERE project_id = $1`, projectID); err != nil {
			t.Errorf("no se pudieron limpiar historias de prueba: %v", err)
		}
		if _, err := store.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, projectID); err != nil {
			t.Errorf("no se pudo limpiar el proyecto de prueba: %v", err)
		}
	})

	storyIDs := make([]int64, 0, storyCount)
	for i := 0; i < storyCount; i++ {
		var storyID int64
		err := store.db.QueryRowContext(ctx, `
			INSERT INTO user_stories (project_id, title, description, priority, status)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			projectID,
			"Historia de prueba",
			"Historia para comprobar la persistencia del Sprint",
			"Media",
			Pendiente,
		).Scan(&storyID)
		if err != nil {
			t.Fatalf("no se pudo crear historia de prueba: %v", err)
		}
		storyIDs = append(storyIDs, storyID)
	}
	return store, projectID, storyIDs
}

func integrationSprint(projectID int64, storyIDs []int64) Sprint {
	return Sprint{
		ProjectID: projectID,
		Goal:      "Objetivo de integración",
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		StoryIDs:  storyIDs,
	}
}
