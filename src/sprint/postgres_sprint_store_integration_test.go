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

	finishPostgreSQLSprint(t, store, first.ID) // un solo Sprint activo por proyecto (FIX-HU-04)
	second, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatalf("la creación del segundo Sprint devolvió error: %v", err)
	}
	if second.DisplayID != "Sprint 2" {
		t.Errorf("segundo DisplayID = %q, se esperaba %q", second.DisplayID, "Sprint 2")
	}
}

func TestPostgreSQLSprintStoreCreatesSprintWithoutStoryAssignments(t *testing.T) {
	store, projectID, _ := setupPostgreSQLSprintTest(t, 0)
	ctx := context.Background()

	created, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatalf("la creación de Sprint sin historias devolvió error: %v", err)
	}
	if created.Status != SprintActive {
		t.Errorf("estado = %q, se esperaba %q", created.Status, SprintActive)
	}
	if len(created.StoryIDs) != 0 {
		t.Errorf("historias devueltas = %v, se esperaba una lista vacía", created.StoryIDs)
	}

	var sprintCount, assignmentCount int
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sprints WHERE id = $1 AND project_id = $2`,
		created.ID,
		projectID,
	).Scan(&sprintCount); err != nil {
		t.Fatalf("no se pudo verificar la persistencia del Sprint: %v", err)
	}
	if err := store.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sprint_story_assignments WHERE sprint_id = $1`,
		created.ID,
	).Scan(&assignmentCount); err != nil {
		t.Fatalf("no se pudieron verificar las asociaciones: %v", err)
	}
	if sprintCount != 1 || assignmentCount != 0 {
		t.Errorf("Sprints persistidos=%d, asociaciones=%d; se esperaba 1 y 0", sprintCount, assignmentCount)
	}
}

func TestPostgreSQLSprintStoreAssignmentConflictRollsBack(t *testing.T) {
	store, projectID, storyIDs := setupPostgreSQLSprintTest(t, 2)
	ctx := context.Background()

	first, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AssignStories(ctx, first.ID, []int64{storyIDs[1]}); err != nil {
		t.Fatalf("primera asignación falló: %v", err)
	}
	// Un solo Sprint activo (FIX-HU-04): la historia sigue asignada al Sprint 1 ya finalizado.
	finishPostgreSQLSprint(t, store, first.ID)

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
	if sprintCount != 1 {
		t.Errorf("cantidad de Sprints = %d, se esperaba 1 tras el rechazo", sprintCount)
	}
	second, err := store.CreateSprint(ctx, integrationSprint(projectID, storyIDs[:1]))
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint posterior al rechazo: %v", err)
	}
	if second.DisplayID != "Sprint 2" {
		t.Errorf("DisplayID posterior al rechazo = %q, se esperaba Sprint 2", second.DisplayID)
	}

	if err := store.RemoveStory(ctx, second.ID, storyIDs[0]); err != nil {
		t.Fatalf("RemoveStory() falló: %v", err)
	}
	if err := store.AssignStories(ctx, second.ID, []int64{storyIDs[0]}); err != nil {
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

func TestPostgreSQLSprintStore_RemoveStoryRejectsStatusAndRecordedEffort(t *testing.T) {
	store, projectID, storyIDs := setupPostgreSQLSprintTest(t, 3)
	ctx := context.Background()
	created, err := store.CreateSprint(ctx, integrationSprint(projectID, storyIDs))
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint de prueba: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE user_stories SET status = $1 WHERE id = $2`, InProgress, storyIDs[0]); err != nil {
		t.Fatalf("no se pudo poner la historia en progreso: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE user_stories SET status = $1 WHERE id = $2`, Done, storyIDs[1]); err != nil {
		t.Fatalf("no se pudo marcar la historia como hecha: %v", err)
	}

	var memberID int64
	if err := store.db.QueryRowContext(ctx,
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		"HU23 RemoveStory effort test",
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante para el esfuerzo: %v", err)
	}
	t.Cleanup(func() {
		if _, err := store.db.ExecContext(ctx, `DELETE FROM effort_logs WHERE member_id = $1`, memberID); err != nil {
			t.Errorf("no se pudieron borrar los esfuerzos de prueba: %v", err)
		}
		if _, err := store.db.ExecContext(ctx, `DELETE FROM members WHERE id = $1`, memberID); err != nil {
			t.Errorf("no se pudo borrar el integrante de prueba: %v", err)
		}
	})
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, CURRENT_DATE, 'Trabajo registrado', 1.5)`, storyIDs[2], memberID); err != nil {
		t.Fatalf("no se pudo registrar esfuerzo para la historia: %v", err)
	}

	cases := []struct {
		name       string
		storyID    int64
		wantStatus StoryStatus
		wantReason string
	}{
		{name: "en progreso", storyID: storyIDs[0], wantStatus: InProgress, wantReason: "Ya está en progreso"},
		{name: "hecha", storyID: storyIDs[1], wantStatus: Done, wantReason: "Ya está hecha"},
		{name: "con horas", storyID: storyIDs[2], wantStatus: EnSprint, wantReason: "Tiene horas registradas"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := store.RemoveStory(ctx, created.ID, test.storyID)
			var removalError *StoryRemovalError
			if !errors.As(err, &removalError) {
				t.Fatalf("se esperaba *StoryRemovalError, se obtuvo %T: %v", err, err)
			}
			if !errors.Is(err, ErrStoryNotAvailable) || removalError.StoryID != test.storyID || removalError.Reason != test.wantReason {
				t.Errorf("error tipado inesperado: %+v (error: %v)", removalError, err)
			}

			status, activeSprintID := storyState(t, store, test.storyID)
			var assignmentCount int
			if err := store.db.QueryRowContext(ctx,
				`SELECT count(*) FROM sprint_story_assignments WHERE sprint_id = $1 AND story_id = $2`,
				created.ID, test.storyID).Scan(&assignmentCount); err != nil {
				t.Fatalf("no se pudo consultar la asignación: %v", err)
			}
			if status != string(test.wantStatus) || !activeSprintID.Valid || activeSprintID.Int64 != created.ID || assignmentCount != 1 {
				t.Errorf("el rechazo debía conservar estado y asignación: status=%q activeSprintID=%v assignmentCount=%d",
					status, activeSprintID, assignmentCount)
			}
		})
	}
}

func TestPostgreSQLSprintStore_RemoveStoryRejectsOtherSprintAndMissingStory(t *testing.T) {
	store, projectAID, storyIDs := setupPostgreSQLSprintTest(t, 1)
	ctx := context.Background()
	sprintA, err := store.CreateSprint(ctx, integrationSprint(projectAID, storyIDs))
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint A: %v", err)
	}

	var projectBID, foreignStoryID int64
	if err := store.db.QueryRowContext(ctx, `
		INSERT INTO projects (name, start_date, end_date, status)
		VALUES ('HU23 RemoveStory foreign project', DATE '2026-01-01', DATE '2026-12-31', 'Activo')
		RETURNING id`).Scan(&projectBID); err != nil {
		t.Fatalf("no se pudo crear el proyecto B: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `
		INSERT INTO user_stories (project_id, title, description, priority, status)
		VALUES ($1, 'Historia de otro Sprint', 'Historia extranjera', 'Media', $2)
		RETURNING id`, projectBID, Pendiente).Scan(&foreignStoryID); err != nil {
		t.Fatalf("no se pudo crear la historia del proyecto B: %v", err)
	}
	t.Cleanup(func() {
		if _, err := store.db.ExecContext(ctx, `DELETE FROM user_stories WHERE id = $1`, foreignStoryID); err != nil {
			t.Errorf("no se pudo borrar la historia extranjera: %v", err)
		}
		if _, err := store.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, projectBID); err != nil {
			t.Errorf("no se pudo borrar el proyecto B: %v", err)
		}
	})
	sprintB, err := store.CreateSprint(ctx, integrationSprint(projectBID, []int64{foreignStoryID}))
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint B: %v", err)
	}

	for _, test := range []struct {
		name    string
		storyID int64
	}{
		{name: "historia de otro Sprint", storyID: foreignStoryID},
		{name: "historia inexistente", storyID: 9223372036854775807},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := store.RemoveStory(ctx, sprintA.ID, test.storyID)
			var removalError *StoryRemovalError
			if !errors.As(err, &removalError) {
				t.Fatalf("se esperaba *StoryRemovalError, se obtuvo %T: %v", err, err)
			}
			if !errors.Is(err, ErrStoryNotAvailable) || removalError.StoryID != test.storyID {
				t.Errorf("error tipado inesperado: %+v (error: %v)", removalError, err)
			}
		})
	}
	status, activeSprintID := storyState(t, store, foreignStoryID)
	if status != string(EnSprint) || !activeSprintID.Valid || activeSprintID.Int64 != sprintB.ID {
		t.Errorf("el rechazo no debía alterar la historia del Sprint B: status=%q activeSprintID=%v", status, activeSprintID)
	}
}
