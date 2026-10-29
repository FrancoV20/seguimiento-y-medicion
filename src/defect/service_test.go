package defect

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type registerTestStore struct {
	calls      int
	input      RegisterInput
	registered Registered
	err        error
}

func (s *registerTestStore) Create(_ context.Context, input RegisterInput) (Registered, error) {
	s.calls++
	s.input = input
	return s.registered, s.err
}

func validRegisterInput() RegisterInput {
	return RegisterInput{
		ProjectID:   10,
		StoryID:     20,
		Description: "El botón no guarda los cambios",
		Severity:    SeverityHigh,
	}
}

func TestValidateDescriptionBoundariesAndTrim(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantError   error
		wantTrimmed string
	}{
		{name: "vacía", description: "", wantError: ErrDescriptionRequired},
		{name: "solo espacios", description: " \t\n ", wantError: ErrDescriptionRequired},
		{name: "1000 caracteres unicode", description: strings.Repeat("á", 1000), wantTrimmed: strings.Repeat("á", 1000)},
		{name: "1001 caracteres unicode", description: strings.Repeat("界", 1001), wantError: ErrDescriptionTooLong},
		{name: "recortada", description: "  defecto  ", wantTrimmed: "defecto"},
		{name: "carácter NUL", description: "falla\x00al guardar", wantError: ErrDescriptionInvalid},
		{name: "UTF-8 inválido", description: "falla \xff al guardar", wantError: ErrDescriptionInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validRegisterInput()
			input.Description = test.description
			got, violations := Validate(input)

			if test.wantError != nil {
				if !hasViolation(violations, FieldDescription, test.wantError) {
					t.Errorf("se esperaba %v en el campo %q, se obtuvo %+v", test.wantError, FieldDescription, violations)
				}
				return
			}
			if len(violations) != 0 {
				t.Fatalf("no se esperaban errores, se obtuvo %+v", violations)
			}
			if got.Description != test.wantTrimmed {
				t.Errorf("descripción normalizada = %q; se esperaba %q", got.Description, test.wantTrimmed)
			}
		})
	}
}

func TestValidateSeverityAndStory(t *testing.T) {
	tests := []struct {
		name      string
		severity  Severity
		storyID   int64
		wantField string
		wantError error
	}{
		{name: "severidad vacía", severity: "", storyID: 20, wantField: FieldSeverity, wantError: ErrSeverityRequired},
		{name: "severidad no permitida", severity: "Urgente", storyID: 20, wantField: FieldSeverity, wantError: ErrSeverityInvalid},
		{name: "historia vacía", severity: SeverityLow, storyID: 0, wantField: FieldStory, wantError: ErrStoryRequired},
		{name: "historia negativa", severity: SeverityLow, storyID: -1, wantField: FieldStory, wantError: ErrStoryRequired},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validRegisterInput()
			input.Severity = test.severity
			input.StoryID = test.storyID
			_, violations := Validate(input)
			if !hasViolation(violations, test.wantField, test.wantError) {
				t.Errorf("se esperaba %v en %q, se obtuvo %+v", test.wantError, test.wantField, violations)
			}
		})
	}
}

func TestValidateCollectsAllViolationsTogether(t *testing.T) {
	input := RegisterInput{ProjectID: 10, Description: " ", Severity: "Urgente", StoryID: 0}
	_, violations := Validate(input)

	for _, expected := range []FieldError{
		{Field: FieldDescription, Err: ErrDescriptionRequired},
		{Field: FieldSeverity, Err: ErrSeverityInvalid},
		{Field: FieldStory, Err: ErrStoryRequired},
	} {
		if !hasViolation(violations, expected.Field, expected.Err) {
			t.Errorf("falta la violación %+v en %+v", expected, violations)
		}
	}
	if len(violations) != 3 {
		t.Errorf("se esperaban las tres violaciones juntas, se obtuvo %+v", violations)
	}
}

func TestServiceRegisterValidatesBeforeStoreAndTrimsDescription(t *testing.T) {
	store := &registerTestStore{registered: Registered{ID: 31, SprintNumber: 4}}
	input := validRegisterInput()
	input.Description = "  botón bloqueado  "

	got, err := NewService(store).Register(context.Background(), input)

	if err != nil {
		t.Fatalf("Register devolvió error: %v", err)
	}
	if got != store.registered || store.calls != 1 {
		t.Errorf("resultado o cantidad de guardados inesperados: got=%+v calls=%d", got, store.calls)
	}
	if store.input.Description != "botón bloqueado" {
		t.Errorf("la descripción enviada al store no está recortada: %q", store.input.Description)
	}

	invalidStore := &registerTestStore{}
	input.Severity = "Urgente"
	_, err = NewService(invalidStore).Register(context.Background(), input)
	var violations ValidationErrors
	if !errors.As(err, &violations) {
		t.Errorf("se esperaba ValidationErrors, se obtuvo %T: %v", err, err)
	}
	if invalidStore.calls != 0 {
		t.Errorf("el store no debe invocarse ante errores de validación; llamadas=%d", invalidStore.calls)
	}
}

func hasViolation(violations []FieldError, field string, want error) bool {
	for _, violation := range violations {
		if violation.Field == field && errors.Is(violation.Err, want) {
			return true
		}
	}
	return false
}
