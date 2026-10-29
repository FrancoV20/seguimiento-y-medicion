package defect

import "context"

// Store registra defectos de forma persistente.
type Store interface {
	Create(ctx context.Context, input RegisterInput) (Registered, error)
}
