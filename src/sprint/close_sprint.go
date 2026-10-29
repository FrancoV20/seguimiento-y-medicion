package sprint

import (
	"context"
	"errors"
	"fmt"
)

// CloseResult informa el resultado del cierre confirmado.
type CloseResult struct {
	SprintNumber   int64
	CompletedCount int
	ReturnedCount  int
}

// SprintCloseStore consulta la pertenencia del Sprint y delega el cierre atómico.
type SprintCloseStore interface {
	FindSprint(ctx context.Context, projectID, sprintID int64) (Sprint, error)
	CloseSprint(ctx context.Context, sprintID int64) (CloseResult, error)
}

// SprintClosureValidator aplica las reglas de cierre definidas por HU-07.
type SprintClosureValidator interface {
	ValidateSprintClosure(ctx context.Context, sprintID int64) error
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

// CloseSprint valida la pertenencia, el estado activo y las reglas HU-07 antes de cerrar.
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

	if err := s.validator.ValidateSprintClosure(ctx, sprintID); err != nil {
		return CloseResult{}, fmt.Errorf("%w: %w", ErrCloseBlocked, err)
	}

	result, err := s.store.CloseSprint(ctx, sprintID)
	if err != nil {
		if errors.Is(err, ErrSprintNotFound) {
			return CloseResult{}, ErrSprintNotFound
		}
		if errors.Is(err, ErrSprintAlreadyClosed) {
			return CloseResult{}, ErrSprintAlreadyClosed
		}
		return CloseResult{}, fmt.Errorf("%w: cerrar Sprint", ErrPersistence)
	}
	result.SprintNumber = sprint.Number
	return result, nil
}
