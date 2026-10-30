package project

import "context"

// ProjectStore define el contrato de persistencia para los proyectos.
type ProjectStore interface {
	Create(p *Project) error
	MemberExists(id int) (bool, error)
	// Update guarda la modificación de un proyecto en una sola transacción (HU-11).
	Update(ctx context.Context, u ProjectUpdate) error
}
