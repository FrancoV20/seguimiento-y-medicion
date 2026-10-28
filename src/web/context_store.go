package web

import "context"

// ContextStore lee proyectos e integrantes para el contexto de trabajo (solo lectura).
type ContextStore interface {
	// ListProjects devuelve los proyectos, los más recientes primero.
	ListProjects(ctx context.Context) ([]ProjectSummary, error)
	// ListMembers devuelve los integrantes del proyecto; vacío si no hay o no existe.
	ListMembers(ctx context.Context, projectID int64) ([]MemberSummary, error)
	// ResolveContext valida proyecto e integrante en una consulta. Que no existan no es un
	// error; solo lo es la base no disponible (error clasificado con Unavailable).
	ResolveContext(ctx context.Context, projectID, memberID int64) (ResolvedContext, error)
}
