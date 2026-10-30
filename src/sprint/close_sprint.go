package sprint

import (
	"context"
	"errors"
	"fmt"
	"math/big"
)

// CloseResult informa el resultado del cierre confirmado.
type CloseResult struct {
	SprintNumber   int64
	CompletedCount int
	ReturnedCount  int
}

// ClosingStory es una historia que forma parte del cierre, leída dentro de la transacción de
// cierre después de bloquearla. Las historias del cierre son las que tienen active_sprint_id igual
// al Sprint que se cierra.
type ClosingStory struct {
	ID          int64
	Status      StoryStatus
	StoryPoints *int
	ActualHours *big.Rat
}

// ClosureCheck valida las historias bloqueadas antes de aplicar el cierre. Si devuelve error, el
// cierre se revierte sin cambios.
type ClosureCheck func(ctx context.Context, stories []ClosingStory) error

// SprintCloseStore consulta la pertenencia del Sprint y delega el cierre atómico. CloseSprint debe
// ejecutar check dentro de la transacción, después de bloquear el Sprint y sus historias.
type SprintCloseStore interface {
	FindSprint(ctx context.Context, projectID, sprintID int64) (Sprint, error)
	CloseSprint(ctx context.Context, sprintID int64, check ClosureCheck) (CloseResult, error)
}

// SprintClosureValidator aplica las reglas de cierre definidas por HU-07 sobre las historias que
// se cierran.
type SprintClosureValidator interface {
	ValidateSprintClosure(ctx context.Context, stories []ClosingStory) error
}

// CloseSprintService orquesta las validaciones del caso de uso y el cierre persistente.
type CloseSprintService struct {
	store     SprintCloseStore
	validator SprintClosureValidator
}

// NewCloseSprintService crea el servicio con sus puertos de salida.
func NewCloseSprintService(store SprintCloseStore, validator SprintClosureValidator) *CloseSprintService {
	return &CloseSprintService{store: store, validator: validator}
}

// CloseSprint valida la pertenencia y el estado activo, y cierra el Sprint aplicando las reglas de
// HU-07 dentro de la transacción de cierre.
func (s *CloseSprintService) CloseSprint(
	ctx context.Context,
	projectID int64,
	sprintID int64,
) (CloseResult, error) {
	if s == nil || s.store == nil || s.validator == nil {
		return CloseResult{}, fmt.Errorf("%w: servicio de cierre incompleto", ErrPersistence)
	}

	sprint, err := s.store.FindSprint(ctx, projectID, sprintID)
	if err != nil {
		if errors.Is(err, ErrSprintNotFound) {
			return CloseResult{}, ErrSprintNotFound
		}
		return CloseResult{}, fmt.Errorf("%w: consultar Sprint para cierre", ErrPersistence)
	}
	if sprint.ProjectID != projectID {
		return CloseResult{}, ErrSprintNotFound
	}
	if sprint.Status != SprintActive {
		return CloseResult{}, ErrSprintAlreadyClosed
	}

	check := func(ctx context.Context, stories []ClosingStory) error {
		if err := s.validator.ValidateSprintClosure(ctx, stories); err != nil {
			return fmt.Errorf("%w: %w", ErrCloseBlocked, err)
		}
		return nil
	}

	result, err := s.store.CloseSprint(ctx, sprintID, check)
	if err != nil {
		switch {
		case errors.Is(err, ErrCloseBlocked):
			return CloseResult{}, err
		case errors.Is(err, ErrSprintNotFound):
			return CloseResult{}, ErrSprintNotFound
		case errors.Is(err, ErrSprintAlreadyClosed):
			return CloseResult{}, ErrSprintAlreadyClosed
		default:
			return CloseResult{}, fmt.Errorf("%w: cerrar Sprint", ErrPersistence)
		}
	}
	result.SprintNumber = sprint.Number
	return result, nil
}
