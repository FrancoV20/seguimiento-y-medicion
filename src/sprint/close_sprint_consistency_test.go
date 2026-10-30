package sprint

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Ajustes 1 y 2 de la revisión de HU-15: la validación de HU-07 se hace dentro de la transacción
// de cierre, después de bloquear las historias, y sobre el mismo conjunto de historias que se cierra
// (user_stories.active_sprint_id).

// Ajuste 1: una historia que pasa a "Hecho" sin Story Points mientras el cierre espera el bloqueo
// debe bloquear el cierre, aunque antes de esperar el Sprint era válido.
func TestCloseSprintServiceValidatesStoriesChangedWhileWaitingForLock(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	sprintID, storyIDs := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
		{status: Done, storyPoints: intPointer(3)},
		{status: InProgress, storyPoints: intPointer(5)},
	})
	changedStoryID := storyIDs[1]
	ctx := context.Background()

	// Otra transacción toma la historia, la pasa a "Hecho" sin Story Points y todavía no confirma.
	concurrent, err := fixture.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("no se pudo abrir la transacción concurrente: %v", err)
	}
	defer func() { _ = concurrent.Rollback() }()
	if _, err := concurrent.ExecContext(ctx,
		`UPDATE user_stories SET status = $1, story_points = NULL WHERE id = $2`,
		string(Done), changedStoryID,
	); err != nil {
		t.Fatalf("no se pudo modificar la historia en la transacción concurrente: %v", err)
	}

	type closeOutcome struct {
		result CloseResult
		err    error
	}
	done := make(chan closeOutcome, 1)
	go func() {
		result, err := fixture.service().CloseSprint(ctx, fixture.projectA, sprintID)
		done <- closeOutcome{result: result, err: err}
	}()

	// El cierre tiene que quedar esperando el bloqueo de la historia antes de que la otra transacción confirme.
	if !waitForBlockedQuery(t, fixture, "FOR UPDATE", 5*time.Second) {
		_ = concurrent.Rollback()
		outcome := <-done
		t.Fatalf("el cierre no esperó el bloqueo de las historias (resultado=%+v, error=%v)", outcome.result, outcome.err)
	}
	if err := concurrent.Commit(); err != nil {
		t.Fatalf("no se pudo confirmar la transacción concurrente: %v", err)
	}

	var outcome closeOutcome
	select {
	case outcome = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("el cierre no terminó después de liberar el bloqueo")
	}

	if !errors.Is(outcome.err, ErrCloseBlocked) {
		t.Fatalf("error = %v, se esperaba ErrCloseBlocked por la historia %d que quedó Hecho sin Story Points", outcome.err, changedStoryID)
	}
	if !strings.Contains(outcome.err.Error(), fmt.Sprintf("Historia %d", changedStoryID)) {
		t.Errorf("el error de cierre no menciona la historia %d: %v", changedStoryID, outcome.err)
	}
	fixture.assertSprintStatus(t, sprintID, SprintActive)
	fixture.assertStoryState(t, storyIDs[0], Done, sprintID)
	fixture.assertStoryState(t, changedStoryID, Done, sprintID)
	assertNoSprintStoryResults(t, fixture, sprintID)
}

// Ajuste 2: la validación usa la misma fuente que el cierre (active_sprint_id), aunque
// sprint_story_assignments no coincida.
func TestCloseSprintServiceValidatesTheSameStoriesItCloses(t *testing.T) {
	t.Run("una historia del Sprint sin fila de asignación se valida", func(t *testing.T) {
		fixture := newCloseSprintServiceFixture(t)
		sprintID, storyIDs := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
			{status: Done, storyPoints: nil},
			{status: InProgress, storyPoints: intPointer(5)},
		})
		invalidStoryID := storyIDs[0]
		if _, err := fixture.db.Exec(
			`DELETE FROM sprint_story_assignments WHERE story_id = $1`, invalidStoryID,
		); err != nil {
			t.Fatalf("no se pudo quitar la fila de asignación de la historia %d: %v", invalidStoryID, err)
		}

		_, err := fixture.service().CloseSprint(context.Background(), fixture.projectA, sprintID)

		if !errors.Is(err, ErrCloseBlocked) {
			t.Fatalf("error = %v, se esperaba ErrCloseBlocked: la historia %d se cierra con el Sprint y es inválida", err, invalidStoryID)
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("Historia %d", invalidStoryID)) {
			t.Errorf("el error de cierre no menciona la historia %d: %v", invalidStoryID, err)
		}
		fixture.assertSprintStatus(t, sprintID, SprintActive)
		fixture.assertStoryState(t, storyIDs[1], InProgress, sprintID)
		assertNoSprintStoryResults(t, fixture, sprintID)
	})

	t.Run("una fila de asignación sin active_sprint_id no se valida ni se cierra", func(t *testing.T) {
		fixture := newCloseSprintServiceFixture(t)
		sprintID, storyIDs := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
			{status: Done, storyPoints: intPointer(3)},
			{status: Done, storyPoints: nil},
		})
		orphanStoryID := storyIDs[1]
		if _, err := fixture.db.Exec(
			`UPDATE user_stories SET active_sprint_id = NULL WHERE id = $1`, orphanStoryID,
		); err != nil {
			t.Fatalf("no se pudo quitar active_sprint_id de la historia %d: %v", orphanStoryID, err)
		}

		result, err := fixture.service().CloseSprint(context.Background(), fixture.projectA, sprintID)

		if err != nil {
			t.Fatalf("CloseSprint() devolvió error: %v; la historia %d no forma parte del cierre", err, orphanStoryID)
		}
		if result.CompletedCount != 1 || result.ReturnedCount != 0 {
			t.Errorf("CloseSprint() = %+v; se esperaba 1 completada y 0 devueltas", result)
		}
		fixture.assertSprintStatus(t, sprintID, SprintFinished)
		fixture.assertStoryState(t, storyIDs[0], Done, sprintID)
		fixture.assertStoryState(t, orphanStoryID, Done, 0)

		var recorded int
		if err := fixture.db.QueryRow(
			`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1 AND story_id = $2`,
			sprintID, orphanStoryID,
		).Scan(&recorded); err != nil {
			t.Fatalf("no se pudo leer sprint_story_results: %v", err)
		}
		if recorded != 0 {
			t.Errorf("la historia %d sin active_sprint_id quedó registrada en el cierre", orphanStoryID)
		}
	})
}

// waitForBlockedQuery espera a que alguna consulta que contenga fragment quede esperando un bloqueo.
func waitForBlockedQuery(t *testing.T, fixture *closeSprintServiceFixture, fragment string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var waiting int
		if err := fixture.db.QueryRow(`
			SELECT count(*)
			FROM pg_stat_activity
			WHERE datname = current_database()
			  AND wait_event_type = 'Lock'
			  AND query LIKE '%' || $1 || '%'`,
			fragment,
		).Scan(&waiting); err != nil {
			t.Fatalf("no se pudo consultar pg_stat_activity: %v", err)
		}
		if waiting > 0 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func assertNoSprintStoryResults(t *testing.T, fixture *closeSprintServiceFixture, sprintID int64) {
	t.Helper()
	var recorded int
	if err := fixture.db.QueryRow(
		`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`, sprintID,
	).Scan(&recorded); err != nil {
		t.Fatalf("no se pudo leer sprint_story_results: %v", err)
	}
	if recorded != 0 {
		t.Errorf("sprint_story_results tiene %d filas del Sprint %d; un cierre bloqueado no debe guardar nada", recorded, sprintID)
	}
}
