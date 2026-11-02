package auth

import (
	"context"
	"errors"
)

// ErrMemberNotFound indica que el integrante no existe.
var ErrMemberNotFound = errors.New("auth: el integrante no existe")

// PasswordStore es el puerto para configurar la contraseña de un integrante (comando set-password).
type PasswordStore interface {
	// MemberName devuelve el nombre del integrante, o ErrMemberNotFound.
	MemberName(ctx context.Context, memberID int64) (string, error)
	// SetPasswordHash guarda el valor de la contraseña (reemplaza el anterior), o ErrMemberNotFound.
	SetPasswordHash(ctx context.Context, memberID int64, hash string) error
}
