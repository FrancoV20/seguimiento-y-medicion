package sprint

import (
	"errors"
	"testing"
)

func TestDomainErrorsAreDistinctAndCanBeWrapped(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "proyecto inexistente", err: ErrProjectNotFound},
		{name: "goal obligatorio", err: ErrGoalRequired},
		{name: "período inválido", err: ErrInvalidSprintPeriod},
		{name: "historia no disponible", err: ErrStoryNotAvailable},
		{name: "conflicto de Sprint activo", err: ErrActiveSprintConflict},
		{name: "ya hay un Sprint activo", err: ErrActiveSprintExists},
		{name: "persistencia", err: ErrPersistence},
	}
	seen := make(map[error]bool, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("el error de dominio no debe ser nil")
			}
			if seen[tt.err] {
				t.Fatalf("el error %q reutiliza otro valor", tt.name)
			}
			seen[tt.err] = true
			wrapped := errors.Join(errors.New("contexto"), tt.err)
			if !errors.Is(wrapped, tt.err) {
				t.Fatalf("errors.Is no reconoció el error envuelto %q", tt.name)
			}
		})
	}
}

func TestPersistenceErrorIncludesOperation(t *testing.T) {
	err := persistenceError("guardar resultados del Sprint")

	if !errors.Is(err, ErrPersistence) {
		t.Errorf("persistenceError() = %v; se esperaba que envolviera ErrPersistence", err)
	}
	if got, want := err.Error(), ErrPersistence.Error()+": guardar resultados del Sprint"; got != want {
		t.Errorf("persistenceError().Error() = %q; se esperaba %q", got, want)
	}
}

func TestActiveSprintExistsErrorMessage(t *testing.T) {
	err := &ActiveSprintExistsError{Number: 7}
	if got, want := err.Error(), "ya hay un Sprint activo (Sprint 7)"; got != want {
		t.Errorf("Error() = %q; se esperaba %q", got, want)
	}
	if !errors.Is(err, ErrActiveSprintExists) {
		t.Errorf("errors.Is(%T, ErrActiveSprintExists) = false", err)
	}
}

func TestStoryAlreadyAssignedErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		err         StoryAlreadyAssignedError
		wantMessage string
	}{
		{
			name:        "Sprint conocido",
			err:         StoryAlreadyAssignedError{StoryID: 42, SprintNumber: 3},
			wantMessage: "la historia 42 ya está asignada al Sprint 3",
		},
		{
			name:        "Sprint desconocido",
			err:         StoryAlreadyAssignedError{StoryID: 43},
			wantMessage: "la historia 43 ya está asignada a un Sprint",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMessage {
				t.Errorf("Error() = %q; se esperaba %q", got, tt.wantMessage)
			}
			if !errors.Is(&tt.err, ErrActiveSprintConflict) {
				t.Errorf("errors.Is(%T, ErrActiveSprintConflict) = false", &tt.err)
			}
		})
	}
}

func TestStoryRemovalErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		err         StoryRemovalError
		wantError   string
		wantMessage string
	}{
		{
			name:        "incluye motivo",
			err:         StoryRemovalError{StoryID: 42, Reason: "Tiene horas registradas"},
			wantError:   "la historia 42 no se puede quitar del Sprint: Tiene horas registradas",
			wantMessage: "Tiene horas registradas",
		},
		{
			name:        "sin motivo usa error base",
			err:         StoryRemovalError{StoryID: 43},
			wantError:   ErrStoryNotAvailable.Error() + ": historia 43",
			wantMessage: ErrStoryNotAvailable.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantError {
				t.Errorf("Error() = %q; se esperaba %q", got, tt.wantError)
			}
			if got := tt.err.Message(); got != tt.wantMessage {
				t.Errorf("Message() = %q; se esperaba %q", got, tt.wantMessage)
			}
			if !errors.Is(&tt.err, ErrStoryNotAvailable) {
				t.Errorf("errors.Is(%T, ErrStoryNotAvailable) = false", &tt.err)
			}
		})
	}
}
