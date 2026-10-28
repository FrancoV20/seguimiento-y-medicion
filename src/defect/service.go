package defect

import (
	"context"
	"strings"
	"unicode/utf8"
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
	var violations []FieldError
	input.Description = strings.TrimSpace(input.Description)
	switch {
	case input.Description == "":
		violations = append(violations, FieldError{Field: FieldDescription, Err: ErrDescriptionRequired})
	case utf8.RuneCountInString(input.Description) > 1000:
		violations = append(violations, FieldError{Field: FieldDescription, Err: ErrDescriptionTooLong})
	}

	switch {
	case input.Severity == "":
		violations = append(violations, FieldError{Field: FieldSeverity, Err: ErrSeverityRequired})
	case !input.Severity.IsValid():
		violations = append(violations, FieldError{Field: FieldSeverity, Err: ErrSeverityInvalid})
	}

	if input.StoryID <= 0 {
		violations = append(violations, FieldError{Field: FieldStory, Err: ErrStoryRequired})
	}
	return input, violations
}

// Register registra un defecto validado.
func (s *Service) Register(ctx context.Context, input RegisterInput) (Registered, error) {
	input, violations := Validate(input)
	if len(violations) > 0 {
		return Registered{}, ValidationErrors(violations)
	}
	return s.store.Create(ctx, input)
}
