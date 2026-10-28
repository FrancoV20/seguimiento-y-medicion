package web

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

func openSprintCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv(EnvDatabaseURL)
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("no se pudo conectar a la base: %v", err)
	}
	return db
}

func TestPostgreSQLSprintCatalog_ActiveSprintAndStoryOptions(t *testing.T) {
	db := openSprintCatalogTestDB(t)
	ctx := context.Background()
	stamp := time.Now().UnixNano()

	var projectAID, projectBID int64
	var activeSprintID, finishedSprintID int64
	var memberID int64
	var storyIDs []int64
	t.Cleanup(func() {
		for _, storyID := range storyIDs {
			_, _ = db.ExecContext(ctx, `DELETE FROM effort_logs WHERE story_id = $1`, storyID)
			_, _ = db.ExecContext(ctx, `DELETE FROM user_stories WHERE id = $1`, storyID)
		}
		for _, sprintID := range []int64{activeSprintID, finishedSprintID} {
			if sprintID != 0 {
				_, _ = db.ExecContext(ctx, `DELETE FROM sprints WHERE id = $1`, sprintID)
			}
		}
		for _, projectID := range []int64{projectAID, projectBID} {
			if projectID != 0 {
				_, _ = db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, projectID)
			}
		}
		if memberID != 0 {
			_, _ = db.ExecContext(ctx, `DELETE FROM members WHERE id = $1`, memberID)
		}
	})

	insertProject := func(name string) int64 {
		t.Helper()
		var id int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31')
			RETURNING id`, name).Scan(&id)
		if err != nil {
			t.Fatalf("no se pudo crear el proyecto de prueba: %v", err)
		}
		return id
	}
	insertStory := func(projectID int64, title string, status sprint.StoryStatus, points *int) int64 {
		t.Helper()
		var id int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO user_stories (project_id, title, description, priority, story_points, status)
			VALUES ($1, $2, $3, 'Media', $4, $5)
			RETURNING id`, projectID, title, "Descripción de prueba", points, status).Scan(&id)
		if err != nil {
			t.Fatalf("no se pudo crear la historia de prueba: %v", err)
		}
		storyIDs = append(storyIDs, id)
		return id
	}
	insertSprint := func(projectID int64, number int, status sprint.SprintStatus) int64 {
		t.Helper()
		var id int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, $3, DATE '2026-06-01', DATE '2026-06-14', $4)
			RETURNING id`, projectID, number, fmt.Sprintf("Goal %d", number), status).Scan(&id)
		if err != nil {
			t.Fatalf("no se pudo crear el Sprint de prueba: %v", err)
		}
		return id
	}
	assignStory := func(sprintID, storyID int64) {
		t.Helper()
		if _, err := db.ExecContext(ctx,
			`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID); err != nil {
			t.Fatalf("no se pudo asignar la historia de prueba: %v", err)
		}
		if _, err := db.ExecContext(ctx,
			`UPDATE user_stories SET active_sprint_id = $1 WHERE id = $2`, sprintID, storyID); err != nil {
			t.Fatalf("no se pudo actualizar el Sprint activo de la historia: %v", err)
		}
	}

	projectAID = insertProject(fmt.Sprintf("HU23 catálogo A %d", stamp))
	projectBID = insertProject(fmt.Sprintf("HU23 catálogo B %d", stamp))
	activeSprintID = insertSprint(projectAID, 1, sprint.SprintActive)
	finishedSprintID = insertSprint(projectBID, 1, sprint.SprintFinished)

	storyPoints := 5
	assignedWithEffortID := insertStory(projectAID, fmt.Sprintf("Asignada con horas %d", stamp), sprint.EnSprint, &storyPoints)
	assignStory(activeSprintID, assignedWithEffortID)
	pendingID := insertStory(projectAID, fmt.Sprintf("Pendiente %d", stamp), sprint.Pendiente, nil)
	inProgressID := insertStory(projectAID, fmt.Sprintf("En progreso %d", stamp), sprint.InProgress, nil)
	doneID := insertStory(projectAID, fmt.Sprintf("Hecha %d", stamp), sprint.Done, nil)
	otherProjectStoryID := insertStory(projectBID, fmt.Sprintf("Otro proyecto %d", stamp), sprint.Pendiente, nil)

	if err := db.QueryRowContext(ctx, `
		INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("HU23 miembro %d", stamp)).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, DATE '2026-06-02', 'Trabajo de prueba', 1.5)`, assignedWithEffortID, memberID); err != nil {
		t.Fatalf("no se pudo crear el registro de esfuerzo: %v", err)
	}
	assignStory(finishedSprintID, otherProjectStoryID)

	catalog := NewPostgreSQLSprintCatalog(db)

	active, err := catalog.ActiveSprint(ctx, projectAID)
	if err != nil {
		t.Fatalf("ActiveSprint devolvió error: %v", err)
	}
	if active == nil {
		t.Fatal("se esperaba el Sprint activo del proyecto A")
	}
	if active.ID != activeSprintID || active.Number != 1 || active.Label != "Sprint 1" {
		t.Errorf("Sprint activo inesperado: %+v", active)
	}
	if len(active.Stories) != 1 || active.Stories[0].ID != assignedWithEffortID {
		t.Fatalf("se esperaba solo la historia asignada al Sprint activo, se obtuvo %+v", active.Stories)
	}
	if !active.Stories[0].HasEffort {
		t.Error("se esperaba hasEffort=true para la historia con horas registradas")
	}

	optionsA, err := catalog.ListStoryOptions(ctx, projectAID)
	if err != nil {
		t.Fatalf("ListStoryOptions(A) devolvió error: %v", err)
	}
	optionsByID := make(map[int64]StoryOption, len(optionsA))
	for _, option := range optionsA {
		optionsByID[option.ID] = option
	}
	if len(optionsByID) != 4 {
		t.Fatalf("se esperaban cuatro historias del proyecto A, se obtuvieron %d: %+v", len(optionsA), optionsA)
	}
	if optionsByID[pendingID].Available != true {
		t.Errorf("la historia Pendiente debía estar disponible: %+v", optionsByID[pendingID])
	}
	if optionsByID[assignedWithEffortID].Available || optionsByID[assignedWithEffortID].AssignedSprintNumber != 1 {
		t.Errorf("la historia asignada debía estar no disponible en Sprint 1: %+v", optionsByID[assignedWithEffortID])
	}
	if optionsByID[inProgressID].Available || optionsByID[inProgressID].UnavailableReason != "No está pendiente" {
		t.Errorf("la historia en progreso debía indicar que no está pendiente: %+v", optionsByID[inProgressID])
	}
	if optionsByID[doneID].Available || optionsByID[doneID].UnavailableReason != "No está pendiente" {
		t.Errorf("la historia hecha debía indicar que no está pendiente: %+v", optionsByID[doneID])
	}

	optionsB, err := catalog.ListStoryOptions(ctx, projectBID)
	if err != nil {
		t.Fatalf("ListStoryOptions(B) devolvió error: %v", err)
	}
	if len(optionsB) != 1 || optionsB[0].ID != otherProjectStoryID {
		t.Errorf("el catálogo del proyecto B devolvió historias ajenas o faltantes: %+v", optionsB)
	}
	if active, err := catalog.ActiveSprint(ctx, projectBID); err != nil || active != nil {
		t.Errorf("un proyecto con solo Sprint finalizado debía no tener Sprint activo: %+v (error: %v)", active, err)
	}
}
