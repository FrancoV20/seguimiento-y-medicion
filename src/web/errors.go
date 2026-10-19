package web

import "errors"

// ErrorKind clasifica un fallo para elegir la respuesta HTTP (HU-20a, T006).
type ErrorKind int

const (
	// KindInternal corresponde a un fallo inesperado (500).
	KindInternal ErrorKind = iota
	// KindNotFound corresponde a un recurso o dirección inexistente (404).
	KindNotFound
	// KindUnavailable corresponde a la base de datos no disponible (503).
	KindUnavailable
)

// AppError envuelve un error con su clasificación. El detalle técnico (Err)
// solo se registra en el servidor; nunca se muestra al usuario.
type AppError struct {
	Kind ErrorKind
	Err  error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return "error de aplicación"
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error { return e.Err }

// NotFound clasifica err como "no encontrado".
func NotFound(err error) error { return &AppError{Kind: KindNotFound, Err: err} }

// Unavailable clasifica err como "servicio no disponible".
func Unavailable(err error) error { return &AppError{Kind: KindUnavailable, Err: err} }

// Internal clasifica err como fallo inesperado.
func Internal(err error) error { return &AppError{Kind: KindInternal, Err: err} }

// KindOf devuelve la clasificación de err; un error sin clasificar es interno.
func KindOf(err error) ErrorKind {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Kind
	}
	return KindInternal
}
