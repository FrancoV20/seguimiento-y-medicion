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
