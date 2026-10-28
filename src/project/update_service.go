package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// UpdateProject modifica el nombre, las fechas y los integrantes de un proyecto (HU-11). Valida
// con las mismas reglas del alta (NewProject, FIX-HU-01) y delega en el almacenamiento, que
// guarda todo en una sola transacción y aplica las reglas de horas y de Sprints.
func (service *ProjectService) UpdateProject(ctx context.Context, req ProjectUpdateRequest) error {
	if req.ID <= 0 {
		return ErrProjectNotFound
	}
	name := strings.TrimSpace(req.Name)
	memberIDs, err := parseMemberIDs(req.Members)
	if err != nil {
		return err
	}
	if _, err := NewProject(name, req.Members, req.StartDate, req.EndDate); err != nil {
		return err
	}

	err = service.store.Update(ctx, ProjectUpdate{
		ID:        req.ID,
		Name:      name,
		MemberIDs: memberIDs,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	})
	if err == nil || isUpdateDomainError(err) {
		return err
	}
	return fmt.Errorf("persist project update: %w", err)
}

// isUpdateDomainError indica si el error es una regla de negocio que se devuelve tal cual.
func isUpdateDomainError(err error) bool {
	var effort *MemberHasEffortError
	var sprint *SprintOutOfRangeError
	return errors.Is(err, ErrProjectNotFound) || errors.Is(err, ErrMemberNotFound) ||
		errors.Is(err, ErrEmptyName) || errors.Is(err, ErrMissingMembers) ||
		errors.Is(err, ErrInvalidPeriod) || errors.Is(err, ErrDuplicateMember) ||
		errors.As(err, &effort) || errors.As(err, &sprint)
}
