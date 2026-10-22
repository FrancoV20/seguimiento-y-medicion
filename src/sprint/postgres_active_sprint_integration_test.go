package sprint

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
)

// FIX-HU-04 contra PostgreSQL: un solo Sprint activo (T002), creación simultánea (T003)
// y estados de las historias (T004).

func finishPostgreSQLSprint(t *testing.T, store *PostgreSQLSprintStore, sprintID int64) {
	t.Helper()
	if _, err := store.db.Exec(`UPDATE sprints SET status = $1 WHERE id = $2`, SprintFinished, sprintID); err != nil {
		t.Fatalf("no se pudo finalizar el Sprint %d: %v", sprintID, err)
	}
}

func countProjectSprints(t *testing.T, store *PostgreSQLSprintStore, projectID int64) int {
	t.Helper()
	var n int
	if err := store.db.QueryRow(`SELECT count(*) FROM sprints WHERE project_id = $1`, projectID).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar los Sprints: %v", err)
	}
	return n
}

func projectNextSprintNumber(t *testing.T, store *PostgreSQLSprintStore, projectID int64) int64 {
	t.Helper()
	var n int64
	if err := store.db.QueryRow(`SELECT next_sprint_number FROM projects WHERE id = $1`, projectID).Scan(&n); err != nil {
		t.Fatalf("no se pudo leer la secuencia del proyecto: %v", err)
	}
	return n
}

func storyState(t *testing.T, store *PostgreSQLSprintStore, storyID int64) (string, sql.NullInt64) {
	t.Helper()
	var status string
	var sprintID sql.NullInt64
	if err := store.db.QueryRow(`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`, storyID).Scan(&status, &sprintID); err != nil {
		t.Fatalf("no se pudo leer la historia %d: %v", storyID, err)
	}
	return status, sprintID
}

func TestPostgreSQLSprintStore_UnSoloSprintActivoYNumeracionSinHuecos(t *testing.T) {
	store, projectID, storyIDs := setupPostgreSQLSprintTest(t, 2)
	ctx := context.Background()
	first, err := store.CreateSprint(ctx, integrationSprint(projectID, storyIDs[:1]))
	if err != nil {
		t.Fatalf("el primer Sprint debía crearse: %v", err)
	}

	_, err = store.CreateSprint(ctx, integrationSprint(projectID, storyIDs[1:]))

	var active *ActiveSprintExistsError
	if !errors.As(err, &active) || active.Number != first.Number {
		t.Fatalf("se esperaba ErrActiveSprintExists con el Sprint %d, se obtuvo: %v", first.Number, err)
	}
	if n := countProjectSprints(t, store, projectID); n != 1 {
		t.Errorf("no debía crearse otro Sprint, hay %d", n)
	}
	if status, sprintID := storyState(t, store, storyIDs[1]); status != string(Pendiente) || sprintID.Valid {
		t.Errorf("la historia del intento rechazado no debía asignarse: %s %v", status, sprintID)
	}
	if next := projectNextSprintNumber(t, store, projectID); next != 2 {
		t.Errorf("el rechazo no debía consumir un número: next_sprint_number=%d, se esperaba 2", next)
	}

	finishPostgreSQLSprint(t, store, first.ID)
	second, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil || second.DisplayID != "Sprint 2" || second.Status != SprintActive {
		t.Errorf("con el anterior finalizado se esperaba Sprint 2 activo, se obtuvo %+v (error: %v)", second, err)
	}
}

func TestPostgreSQLSprintStore_LaReglaEsPorProyecto(t *testing.T) {
	storeA, projectA, _ := setupPostgreSQLSprintTest(t, 0)
	storeB, projectB, _ := setupPostgreSQLSprintTest(t, 0)
	ctx := context.Background()
	if _, err := storeA.CreateSprint(ctx, integrationSprint(projectA, nil)); err != nil {
		t.Fatal(err)
	}

	created, err := storeB.CreateSprint(ctx, integrationSprint(projectB, nil))

	if err != nil || created.DisplayID != "Sprint 1" {
		t.Errorf("el proyecto B debía crear su Sprint 1, se obtuvo %+v (error: %v)", created, err)
	}
}

func TestPostgreSQLSprintStore_CreacionSimultaneaCreaUnSolo(t *testing.T) {
	store, projectID, _ := setupPostgreSQLSprintTest(t, 0)
	const solicitudes = 4
	start := make(chan struct{})
	errs := make([]error, solicitudes)
	var wg sync.WaitGroup
	for i := 0; i < solicitudes; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = store.CreateSprint(context.Background(), integrationSprint(projectID, nil))
		}(i)
	}
	close(start)
	wg.Wait()

	creados, rechazados := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			creados++
		case errors.Is(err, ErrActiveSprintExists):
			rechazados++
		default:
			t.Errorf("error inesperado en una creación simultánea: %v", err)
		}
	}
	if creados != 1 || rechazados != solicitudes-1 {
		t.Errorf("se esperaba 1 creado y %d rechazados, hubo %d y %d", solicitudes-1, creados, rechazados)
	}
	if n := countProjectSprints(t, store, projectID); n != 1 {
		t.Errorf("debía existir un solo Sprint, hay %d", n)
	}
	if next := projectNextSprintNumber(t, store, projectID); next != 2 {
		t.Errorf("la numeración no debía tener huecos: next_sprint_number=%d, se esperaba 2", next)
	}
}

func TestPostgreSQLSprintStore_EstadosYMensajeDeHistoriaAsignada(t *testing.T) {
	store, projectID, storyIDs := setupPostgreSQLSprintTest(t, 2)
	ctx := context.Background()
	first, err := store.CreateSprint(ctx, integrationSprint(projectID, storyIDs[:1]))
	if err != nil {
		t.Fatal(err)
	}
	if status, sprintID := storyState(t, store, storyIDs[0]); status != string(EnSprint) || sprintID.Int64 != first.ID {
		t.Fatalf("al asignarla debía quedar En Sprint: %s %v", status, sprintID)
	}
	finishPostgreSQLSprint(t, store, first.ID)
	second, err := store.CreateSprint(ctx, integrationSprint(projectID, nil))
	if err != nil {
		t.Fatal(err)
	}

	err = store.AssignStories(ctx, second.ID, storyIDs[:1])
	if !errors.Is(err, ErrActiveSprintConflict) || !strings.Contains(err.Error(), "ya está asignada al Sprint 1") {
		t.Errorf("se esperaba 'ya está asignada al Sprint 1', se obtuvo: %v", err)
	}

	if err := store.AssignStories(ctx, second.ID, storyIDs[1:]); err != nil {
		t.Fatalf("asignar una historia pendiente falló: %v", err)
	}
	if err := store.RemoveStory(ctx, second.ID, storyIDs[1]); err != nil {
		t.Fatalf("quitar la historia falló: %v", err)
	}
	if status, sprintID := storyState(t, store, storyIDs[1]); status != string(Pendiente) || sprintID.Valid {
		t.Fatalf("al quitarla debía volver a Pendiente: %s %v", status, sprintID)
	}
	if err := store.AssignStories(ctx, second.ID, storyIDs[1:]); err != nil {
		t.Fatalf("volver a asignarla debía permitirse: %v", err)
	}
	if status, sprintID := storyState(t, store, storyIDs[1]); status != string(EnSprint) || sprintID.Int64 != second.ID {
		t.Errorf("al volver a asignarla debía quedar En Sprint: %s %v", status, sprintID)
	}
}
