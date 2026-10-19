package web

import (
	"context"
	"fmt"
)

// SchemaStatus es la versión de esquema aplicada y si quedó en estado "dirty" (HU-20a, T019).
type SchemaStatus struct {
	Version uint
	Dirty   bool
}

// SchemaVersionReader lee el estado del esquema de la base de datos.
type SchemaVersionReader interface {
	ReadSchemaStatus(ctx context.Context) (SchemaStatus, error)
}

// SchemaError explica por qué el esquema impide arrancar. Err guarda el detalle
// técnico para el registro; Error() solo devuelve el mensaje para el usuario.
type SchemaError struct {
	Message string
	Err     error
}

func (e *SchemaError) Error() string { return e.Message }
func (e *SchemaError) Unwrap() error { return e.Err }

// CheckSchema compara el esquema aplicado con la versión esperada.
// Devuelve una advertencia (no bloqueante) si la base es más nueva que el código.
func CheckSchema(ctx context.Context, reader SchemaVersionReader, expected uint) (warning string, err error) {
	status, err := reader.ReadSchemaStatus(ctx)
	if err != nil {
		return "", &SchemaError{Message: "no se pudo leer la versión del esquema de la base de datos", Err: err}
	}
	switch {
	case status.Dirty:
		return "", &SchemaError{Message: fmt.Sprintf(
			"el esquema de la base de datos quedó en estado \"dirty\" en la versión %d: avisá al equipo antes de forzar la versión con migrate force",
			status.Version)}
	case status.Version < expected:
		return "", &SchemaError{Message: fmt.Sprintf(
			"el esquema de la base de datos está desactualizado (versión %d, se esperaba %d): aplicá las migraciones con migrate -path db/migrations -database \"$env:DATABASE_URL\" up",
			status.Version, expected)}
	case status.Version > expected:
		return fmt.Sprintf("la base de datos tiene la versión de esquema %d, más nueva que la esperada %d; el servidor inicia igual",
			status.Version, expected), nil
	}
	return "", nil
}
