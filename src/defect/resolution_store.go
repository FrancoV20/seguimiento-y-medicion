package defect

import "context"

// ResolutionStore resuelve y reabre defectos de forma persistente. Cada operación es atómica y
// aplica CheckResolve o CheckReopen sobre los datos bloqueados.
type ResolutionStore interface {
	// Resolve registra el Sprint activo del proyecto como Sprint de resolución.
	Resolve(ctx context.Context, projectID, defectID int64) (Resolved, error)
	// Reopen vuelve el defecto a "Abierto" y borra su Sprint de resolución.
	Reopen(ctx context.Context, projectID, defectID int64) error
}
