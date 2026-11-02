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
	// MemberCredential lee el nombre y la contraseña guardada de un integrante del proyecto (HU-27).
	// Que no pertenezca no es un error; solo lo es la base no disponible (Unavailable).
	MemberCredential(ctx context.Context, projectID, memberID int64) (MemberCredential, error)
}

// MemberCredential es lo que la pantalla "Identificarme" necesita de un integrante (HU-27).
type MemberCredential struct {
	Belongs      bool   // pertenece al proyecto
	Name         string // display_name
	PasswordHash string // "" si password_hash es NULL (sin contraseña configurada)
}
