package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestPostgreSQLSprintCloserCloseSprintPersistsSnapshotAndReturnsIncompleteStories(t *testing.T) {
	db, projectID, sprintID, stories := newSprintCloserIntegrationFixture(t,
		Done,
		InProgress,
		EnSprint,
	)
	closer := NewPostgreSQLSprintCloser(db)

	result, err := closer.CloseSprint(context.Background(), sprintID)
	if err != nil {
		t.Fatalf("CloseSprint() devolvió error: %v", err)
	}
	if result.ReturnedCount != 2 {
		t.Errorf("ReturnedCount = %d, se esperaban 2 historias devueltas", result.ReturnedCount)
	}

	var sprintStatus string
	if err := db.QueryRow(
		`SELECT status FROM sprints WHERE id = $1 AND project_id = $2`,
		sprintID,
		projectID,
	).Scan(&sprintStatus); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
	}
	if sprintStatus != string(SprintFinished) {
		t.Errorf("estado del Sprint = %q, se esperaba %q", sprintStatus, SprintFinished)
	}

	for _, story := range stories {
		var (
			status       string
			activeSprint sql.NullInt64
			assignments  int
		)
		if err := db.QueryRow(`
			SELECT us.status, us.active_sprint_id,
			       (SELECT count(*) FROM sprint_story_assignments a WHERE a.story_id = us.id)
			FROM user_stories us
			WHERE us.id = $1`,
			story.id,
		).Scan(&status, &activeSprint, &assignments); err != nil {
			t.Fatalf("no se pudo inspeccionar la historia %d: %v", story.id, err)
		}

		if story.status == Done {
			if status != string(Done) || !activeSprint.Valid || activeSprint.Int64 != sprintID || assignments != 1 {
				t.Errorf("la historia Hecho %d perdió su asignación: estado=%q sprint=%v asociaciones=%d",
					story.id, status, activeSprint, assignments)
			}
		} else if status != string(Pendiente) || activeSprint.Valid || assignments != 0 {
			t.Errorf("la historia incompleta %d no volvió al Backlog: estado=%q sprint=%v asociaciones=%d",
				story.id, status, activeSprint, assignments)
		}

		assertSprintStoryResult(t, db, sprintID, story)
	}

	var resultCount int
	if err := db.QueryRow(
		`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`,
		sprintID,
	).Scan(&resultCount); err != nil {
		t.Fatalf("no se pudieron contar los registros del Sprint: %v", err)
	}
	if resultCount != len(stories) {
		t.Errorf("registros guardados = %d, se esperaban %d", resultCount, len(stories))
	}
}

func TestPostgreSQLSprintCloserSnapshotDoesNotChangeAfterReassignmentOrStoryPointEdit(t *testing.T) {
	db, projectID, sprintID, stories := newSprintCloserIntegrationFixture(t, InProgress)
	closer := NewPostgreSQLSprintCloser(db)

	if _, err := closer.CloseSprint(context.Background(), sprintID); err != nil {
		t.Fatalf("CloseSprint() devolvió error: %v", err)
	}
	story := stories[0]

	nextSprintID := newStatusSprint(t, db, projectID, SprintActive)
	sprintStore, err := NewPostgreSQLSprintStore(testDatabaseURL(t))
	if err != nil {
		t.Fatalf("NewPostgreSQLSprintStore() devolvió error: %v", err)
	}
	t.Cleanup(func() {
		if err := sprintStore.Close(); err != nil {
			t.Errorf("no se pudo cerrar el store PostgreSQL: %v", err)
		}
	})
	if err := sprintStore.AssignStories(context.Background(), nextSprintID, []int64{story.id}); err != nil {
		t.Fatalf("no se pudo reasignar la historia devuelta: %v", err)
	}
	const editedStoryPoints = 8
	if _, err := db.Exec(
		`UPDATE user_stories SET story_points = $1 WHERE id = $2`,
		editedStoryPoints,
		story.id,
	); err != nil {
		t.Fatalf("no se pudieron editar los Story Points: %v", err)
	}

	var (
		finalState  string
		storyPoints sql.NullInt64
	)
	if err := db.QueryRow(`
		SELECT final_status, story_points
		FROM sprint_story_results
		WHERE sprint_id = $1 AND story_id = $2`,
		sprintID, story.id,
	).Scan(&finalState, &storyPoints); err != nil {
		t.Fatalf("no se pudo volver a leer el registro histórico: %v", err)
	}
	if finalState != string(story.status) {
		t.Errorf("estado registrado = %q, se esperaba el estado al cierre %q", finalState, story.status)
	}
	if !storyPoints.Valid || storyPoints.Int64 != int64(story.storyPoints) {
		t.Errorf("Story Points registrados = %v, se esperaban los originales %d",
			storyPoints, story.storyPoints)
	}

	var currentStatus string
	var currentActiveSprint sql.NullInt64
	if err := db.QueryRow(
		`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`,
		story.id,
	).Scan(&currentStatus, &currentActiveSprint); err != nil {
		t.Fatalf("no se pudo comprobar la historia reasignada: %v", err)
	}
	if currentStatus != string(EnSprint) || !currentActiveSprint.Valid || currentActiveSprint.Int64 != nextSprintID {
		t.Errorf("la historia debía quedar en el nuevo Sprint: estado=%q sprint=%v",
			currentStatus, currentActiveSprint)
	}
}

func TestPostgreSQLSprintCloserRollsBackWhenFinalSprintUpdateFails(t *testing.T) {
	db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, InProgress)
	triggerName := fmt.Sprintf("hu15_fail_sprint_close_%d", sprintID)
	functionName := fmt.Sprintf("hu15_fail_sprint_close_fn_%d", sprintID)
	if _, err := db.Exec(fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger
		LANGUAGE plpgsql
		AS $body$
		BEGIN
			RAISE EXCEPTION 'injected HU-15 close failure';
		END;
		$body$`,
		functionName,
	)); err != nil {
		t.Fatalf("no se pudo crear la función de fallo controlado: %v", err)
	}
	if _, err := db.Exec(fmt.Sprintf(`
		CREATE TRIGGER %s
		BEFORE UPDATE OF status ON sprints
		FOR EACH ROW
		WHEN (OLD.id = %d AND NEW.status = 'Finalizado')
		EXECUTE FUNCTION %s()`,
		triggerName,
		sprintID,
		functionName,
	)); err != nil {
		_, _ = db.Exec(fmt.Sprintf(`DROP FUNCTION %s()`, functionName))
		t.Fatalf("no se pudo crear el trigger de fallo controlado: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON sprints`, triggerName)); err != nil {
			t.Errorf("no se pudo quitar el trigger de fallo controlado: %v", err)
		}
		if _, err := db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName)); err != nil {
			t.Errorf("no se pudo quitar la función de fallo controlado: %v", err)
		}
	})

	_, closeErr := NewPostgreSQLSprintCloser(db).CloseSprint(context.Background(), sprintID)
	if closeErr == nil {
		t.Fatal("CloseSprint() no devolvió error ante el fallo inyectado")
	}
	if !strings.Contains(closeErr.Error(), "injected HU-15 close failure") {
		t.Fatalf("CloseSprint() falló antes de alcanzar el error inyectado del último paso: %v", closeErr)
	}

	var sprintStatus string
	if err := db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&sprintStatus); err != nil {
		t.Fatalf("no se pudo comprobar el estado del Sprint tras el rollback: %v", err)
	}
	if sprintStatus != string(SprintActive) {
		t.Errorf("estado del Sprint tras el rollback = %q, se esperaba %q", sprintStatus, SprintActive)
	}

	var resultCount int
	if err := db.QueryRow(
		`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`,
		sprintID,
	).Scan(&resultCount); err != nil {
		t.Fatalf("no se pudo comprobar el registro tras el rollback: %v", err)
	}
	if resultCount != 0 {
		t.Errorf("quedaron %d registros después del rollback, se esperaba 0", resultCount)
	}

	for _, story := range stories {
		var (
			status       string
			activeSprint sql.NullInt64
			assignments  int
		)
		if err := db.QueryRow(`
			SELECT us.status, us.active_sprint_id,
			       (SELECT count(*) FROM sprint_story_assignments a WHERE a.story_id = us.id)
			FROM user_stories us
			WHERE us.id = $1`,
			story.id,
		).Scan(&status, &activeSprint, &assignments); err != nil {
			t.Fatalf("no se pudo comprobar la historia %d tras el rollback: %v", story.id, err)
		}
		if status != string(story.status) || !activeSprint.Valid || activeSprint.Int64 != sprintID || assignments != 1 {
			t.Errorf("la historia %d cambió pese al rollback: estado=%q sprint=%v asociaciones=%d",
				story.id, status, activeSprint, assignments)
		}
	}
}

func TestPostgreSQLSprintCloserConcurrentCloseAppliesOnlyOnce(t *testing.T) {
	db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, EnSprint)
	closer := NewPostgreSQLSprintCloser(db)
	const attempts = 2
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := closer.CloseSprint(context.Background(), sprintID)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes, rejected := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrSprintAlreadyClosed) {
			rejected++
		} else {
			t.Errorf("el segundo cierre devolvió un error inesperado: %v", err)
		}
	}
	if successes != 1 || rejected != 1 {
		t.Fatalf("cierres exitosos=%d rechazados=%d; se esperaba uno de cada", successes, rejected)
	}

	var resultCount int
	if err := db.QueryRow(
		`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`,
		sprintID,
	).Scan(&resultCount); err != nil {
		t.Fatalf("no se pudieron contar los registros después de los cierres concurrentes: %v", err)
	}
	if resultCount != len(stories) {
		t.Errorf("registros después del cierre concurrente = %d, se esperaban %d",
			resultCount, len(stories))
	}
	var finishedCount int
	if err := db.QueryRow(
		`SELECT count(*) FROM sprints WHERE id = $1 AND status = 'Finalizado'`,
		sprintID,
	).Scan(&finishedCount); err != nil {
		t.Fatalf("no se pudo verificar el estado final del Sprint: %v", err)
	}
	if finishedCount != 1 {
		t.Errorf("Sprints finalizados = %d, se esperaba 1", finishedCount)
	}
}

type sprintCloserTestStory struct {
	id          int64
	status      StoryStatus
	storyPoints int
}

func newSprintCloserIntegrationFixture(
	t *testing.T,
	states ...StoryStatus,
) (*sql.DB, int64, int64, []sprintCloserTestStory) {
	t.Helper()
	db := openStatusTestDB(t)
	projectID := newStatusProject(t, db, "Proyecto de integración HU-15")
	sprintID := newStatusSprint(t, db, projectID, SprintActive)
	stories := make([]sprintCloserTestStory, 0, len(states))
	points := []int{3, 5, 8, 13, 21}

	for i, status := range states {
		storyID := newStatusStory(t, db, projectID, status, sprintID)
		storyPoints := points[i%len(points)]
		if _, err := db.Exec(
			`UPDATE user_stories SET story_points = $1 WHERE id = $2`,
			storyPoints, storyID,
		); err != nil {
			t.Fatalf("no se pudieron guardar Story Points de la historia %d: %v", storyID, err)
		}
		stories = append(stories, sprintCloserTestStory{
			id:          storyID,
			status:      status,
			storyPoints: storyPoints,
		})
	}
	return db, projectID, sprintID, stories
}

func assertSprintStoryResult(t *testing.T, db *sql.DB, sprintID int64, story sprintCloserTestStory) {
	t.Helper()
	var (
		finalStatus string
		storyPoints sql.NullInt64
	)
	err := db.QueryRow(`
		SELECT final_status, story_points
		FROM sprint_story_results
		WHERE sprint_id = $1 AND story_id = $2`,
		sprintID, story.id,
	).Scan(&finalStatus, &storyPoints)
	if err != nil {
		t.Fatalf("no se encontró el registro del Sprint para la historia %d: %v", story.id, err)
	}
	if finalStatus != string(story.status) {
		t.Errorf("estado registrado para historia %d = %q, se esperaba %q",
			story.id, finalStatus, story.status)
	}
	if !storyPoints.Valid || storyPoints.Int64 != int64(story.storyPoints) {
		t.Errorf("Story Points registrados para historia %d = %v, se esperaban %d",
			story.id, storyPoints, story.storyPoints)
	}
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL no está definida; se omite la prueba PostgreSQL")
	}
	return dsn
}
