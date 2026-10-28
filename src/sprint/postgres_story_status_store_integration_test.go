package sprint

import (
	"context"
	"sync"
	"testing"
)

// HU-14 (T007): PostgreSQLStoryStatusStore contra PostgreSQL real con fixtures propias.

// FindStory lee el estado, el Sprint asignado, si está activo y si tiene horas registradas.
func TestPostgreSQLStoryStatusStore_FindStoryLeeEstadoSprintYHoras(t *testing.T) {
	casos := []struct {
		nombre       string
		sprintStatus SprintStatus // vacío: la historia no tiene Sprint
		status       StoryStatus
		conHoras     bool
	}{
		{"En Sprint, Sprint activo, sin horas", SprintActive, EnSprint, false},
		{"En progreso, Sprint activo, con horas", SprintActive, InProgress, true},
		{"Hecho, Sprint finalizado", SprintFinished, Done, false},
		{"Pendiente, sin Sprint", "", Pendiente, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			db := openStatusTestDB(t)
			store := NewPostgreSQLStoryStatusStore(db)
			projectID := newStatusProject(t, db, "Proyecto de HU-14")
			var sprintID int64
			if c.sprintStatus != "" {
				sprintID = newStatusSprint(t, db, projectID, c.sprintStatus)
			}
			storyID := newStatusStory(t, db, projectID, c.status, sprintID)
			if c.conHoras {
				addStatusEffort(t, db, storyID)
			}

			got, err := store.FindStory(context.Background(), projectID, storyID)

			if err != nil {
				t.Fatalf("FindStory() devolvió error: %v", err)
			}
			want := StoryState{
				Status:       c.status,
				SprintID:     sprintID,
				SprintActive: c.sprintStatus == SprintActive,
				HasEffort:    c.conHoras,
			}
			if got != want {
				t.Errorf("FindStory() = %+v, se esperaba %+v", got, want)
			}
		})
	}
}

// FR-014: FindStory no devuelve historias de otro proyecto ni inexistentes.
func TestPostgreSQLStoryStatusStore_FindStoryAislaPorProyecto(t *testing.T) {
	t.Run("de otro proyecto", func(t *testing.T) {
		db := openStatusTestDB(t)
		store := NewPostgreSQLStoryStatusStore(db)
		_, storyID := newStoryInSprint(t, db, SprintActive, EnSprint)
		otroProyecto := newStatusProject(t, db, "Otro proyecto de HU-14")

		_, err := store.FindStory(context.Background(), otroProyecto, storyID)

		wantDomainError(t, err, ErrStoryNotFound)
	})
	t.Run("inexistente", func(t *testing.T) {
		db := openStatusTestDB(t)
		store := NewPostgreSQLStoryStatusStore(db)
		projectID, _ := newStoryInSprint(t, db, SprintActive, EnSprint)

		_, err := store.FindStory(context.Background(), projectID, idInexistente)

		wantDomainError(t, err, ErrStoryNotFound)
	})
}

// FR-011, FR-008: UpdateStatus cambia solo si el estado sigue siendo el esperado y la historia
// pertenece a un Sprint activo; si no, no cambia nada.
func TestPostgreSQLStoryStatusStore_UpdateStatus(t *testing.T) {
	casos := []struct {
		nombre       string
		sprintStatus SprintStatus // vacío: la historia no tiene Sprint
		real         StoryStatus
		from, to     StoryStatus
		quiereCambio bool
		quedaEn      StoryStatus
	}{
		{"cambia con el estado esperado", SprintActive, InProgress, InProgress, Done, true, Done},
		{"no cambia con otro estado esperado", SprintActive, InProgress, EnSprint, Done, false, InProgress},
		{"no cambia si el Sprint está finalizado", SprintFinished, InProgress, InProgress, Done, false, InProgress},
		{"no cambia si no tiene Sprint", "", Pendiente, Pendiente, InProgress, false, Pendiente},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			db := openStatusTestDB(t)
			store := NewPostgreSQLStoryStatusStore(db)
			projectID := newStatusProject(t, db, "Proyecto de HU-14")
			var sprintID int64
			if c.sprintStatus != "" {
				sprintID = newStatusSprint(t, db, projectID, c.sprintStatus)
			}
			storyID := newStatusStory(t, db, projectID, c.real, sprintID)

			cambio, err := store.UpdateStatus(context.Background(), storyID, c.from, c.to)

			if err != nil {
				t.Fatalf("UpdateStatus() devolvió error: %v", err)
			}
			if cambio != c.quiereCambio {
				t.Errorf("UpdateStatus() = %v, se esperaba %v", cambio, c.quiereCambio)
			}
			wantStoryStatus(t, db, storyID, c.quedaEn)
		})
	}
}

// SC-006: ante dos o más cambios simultáneos sobre la misma historia, solo uno se aplica.
func TestPostgreSQLStoryStatusStore_CambiosSimultaneosSoloUnoSeAplica(t *testing.T) {
	db := openStatusTestDB(t)
	store := NewPostgreSQLStoryStatusStore(db)
	_, storyID := newStoryInSprint(t, db, SprintActive, InProgress)

	const simultaneos = 10
	var wg sync.WaitGroup
	largada := make(chan struct{})
	cambios := make(chan bool, simultaneos)
	errores := make(chan error, simultaneos)
	for i := 0; i < simultaneos; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-largada // todos esperan la señal para arrancar a la vez
			cambio, err := store.UpdateStatus(context.Background(), storyID, InProgress, Done)
			if err != nil {
				errores <- err
				return
			}
			cambios <- cambio
		}()
	}
	close(largada)
	wg.Wait()
	close(cambios)
	close(errores)

	var primerError error
	for err := range errores {
		if primerError == nil {
			primerError = err
		}
	}
	if primerError != nil {
		t.Errorf("UpdateStatus() devolvió error: %v", primerError)
	}
	aplicados := 0
	for cambio := range cambios {
		if cambio {
			aplicados++
		}
	}
	if aplicados != 1 {
		t.Errorf("cambios aplicados = %d, se esperaba exactamente 1 de %d", aplicados, simultaneos)
	}
	wantStoryStatus(t, db, storyID, Done)
}
