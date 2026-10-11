package effort

import (
	"context"
	"errors"
	"testing"
	"time"
)

// newTestService arma un servicio con un store en memoria y el reloj fijo en fixedNow.
// La historia S1 está en un Sprint activo del proyecto P1 y el integrante M1 pertenece a P1.
func newTestService(t *testing.T) (*Service, *TestStore) {
	t.Helper()
	store := NewTestStore()
	store.Now = func() time.Time { return fixedNow }
	store.AddStory("S1", "P1", true)
	store.AddMember("M1", "P1")
	service := NewService(store, func() time.Time { return fixedNow })
	return service, store
}

// T013: registrar 4 horas suma 4 horas al esfuerzo acumulado de la historia.
func TestLogEffortRegistersHoursAndAccumulates(t *testing.T) {
	service, store := newTestService(t)

	result := service.LogEffort(context.Background(), validRequest())

	if !result.Succeeded() {
		t.Fatalf("el registro debía tener éxito: %+v", result)
	}
	if result.AccumulatedHours != 400 {
		t.Errorf("AccumulatedHours = %d, se esperaba 400", result.AccumulatedHours)
	}
	if result.SuccessMessage != SuccessMessage {
		t.Errorf("SuccessMessage = %q, se esperaba %q", result.SuccessMessage, SuccessMessage)
	}
	if len(result.ValidationErrors) != 0 || result.StorageError != nil {
		t.Errorf("no se esperaban errores: %+v", result)
	}

	entry := result.Entry
	if entry.StoryID != "S1" || entry.MemberID != "M1" ||
		entry.Activity != "Desarrollo de endpoint en Go" || entry.Hours != 400 {
		t.Errorf("el registro creado no coincide con la solicitud: %+v", entry)
	}

	saved := store.Entries()
	if len(saved) != 1 {
		t.Fatalf("debía quedar 1 registro guardado, hay %d", len(saved))
	}
	if saved[0].ID == "" {
		t.Errorf("el registro guardado debe tener un ID")
	}
}

func TestLogEffortSuccessiveEntriesAccumulate(t *testing.T) {
	service, store := newTestService(t)
	ctx := context.Background()

	first := validRequest() // 4 horas
	second := validRequest()
	second.Hours = "2,5"
	second.Activity = "Revisión de código"

	if r := service.LogEffort(ctx, first); !r.Succeeded() || r.AccumulatedHours != 400 {
		t.Fatalf("el primer registro debía dar 400, resultado: %+v", r)
	}
	r := service.LogEffort(ctx, second)
	if !r.Succeeded() {
		t.Fatalf("el segundo registro debía tener éxito: %+v", r)
	}
	if r.AccumulatedHours != 650 {
		t.Errorf("AccumulatedHours = %d, se esperaba 650 (6,5 h)", r.AccumulatedHours)
	}

	saved := store.Entries()
	if len(saved) != 2 {
		t.Fatalf("debían quedar 2 registros independientes, hay %d", len(saved))
	}
	if saved[0].Hours != 400 || saved[1].Hours != 250 {
		t.Errorf("las horas de cada registro deben conservarse: %d y %d", saved[0].Hours, saved[1].Hours)
	}
	if saved[0].ID == saved[1].ID {
		t.Errorf("cada registro debe tener su propio ID")
	}
}

func TestLogEffortRejectsStoryWithoutActiveSprint(t *testing.T) {
	cases := []struct {
		name    string
		storyID string
	}{
		{"historia sin Sprint activo", "S2"},
		{"historia inexistente", "S999"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, store := newTestService(t)
			store.AddStory("S2", "P1", false)

			req := validRequest()
			req.StoryID = c.storyID
			result := service.LogEffort(context.Background(), req)

			if result.Succeeded() || result.SuccessMessage != "" {
				t.Fatalf("el registro debía ser rechazado, sin mensaje de éxito: %+v", result)
			}
			if !errors.Is(errorFor(result.ValidationErrors, FieldStory), ErrStoryNotInSprint) {
				t.Errorf("se esperaba ErrStoryNotInSprint en %q, se obtuvo %v", FieldStory, result.ValidationErrors)
			}
			if store.SaveCalls != 0 || len(store.Entries()) != 0 {
				t.Errorf("no debía guardarse nada (SaveCalls=%d, registros=%d)", store.SaveCalls, len(store.Entries()))
			}
		})
	}
}