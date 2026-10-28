package defect

import (
	"context"
	"errors"
)

const (
	FieldDescription = "description"
	FieldSeverity    = "severity"
	FieldStory       = "story"
)

// FieldError asocia una violación con el campo que la originó.
type FieldError struct {
	Field string
	Err   error
}

// ValidationErrors contiene todas las violaciones de una solicitud.
type ValidationErrors []FieldError

func (ValidationErrors) Error() string {
	return "la solicitud de defecto no es válida"
}

// Service coordina la validación y el registro de defectos.
type Service struct {
	store Store
}

// NewService crea un servicio de registro.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Validate valida y normaliza una solicitud de registro.
func Validate(input RegisterInput) (RegisterInput, []FieldError) {
	return input, nil
}

// Register registra un defecto validado.
func (s *Service) Register(_ context.Context, _ RegisterInput) (Registered, error) {
	return Registered{}, errors.New("implementación pendiente")
}
