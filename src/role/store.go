package role

import (
	"context"
	"errors"
)

var (
	// ErrProjectNotFound indica que el proyecto no existe.
	ErrProjectNotFound = errors.New("role: el proyecto no existe")
	// ErrMemberNotInProject indica que el integrante no existe o no pertenece al proyecto.
	ErrMemberNotInProject = errors.New("role: el integrante no pertenece al proyecto")
)

// Store es el puerto para asignar roles (comando set-role).
type Store interface {
	// ProjectName devuelve el nombre del proyecto, o ErrProjectNotFound.
	ProjectName(ctx context.Context, projectID int64) (string, error)
	// SetMemberRole guarda el rol del integrante en el proyecto y devuelve su nombre, o
	// ErrMemberNotInProject.
	SetMemberRole(ctx context.Context, projectID, memberID int64, r Role) (memberName string, err error)
}
