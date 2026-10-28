package web

import (
	"net/url"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

func TestParseDefectFormMissingFieldsAreEmptyAndSubmittedValuesArePreserved(t *testing.T) {
	tests := []struct {
		name        string
		values      url.Values
		wantForm    DefectForm
		wantError   string
		wantProject int64
	}{
		{
			name:        "campos ausentes",
			values:      url.Values{},
			wantForm:    DefectForm{},
			wantError:   "Seleccione la historia relacionada",
			wantProject: 42,
		},
		{
			name: "valores tal como llegaron",
			values: url.Values{
				DefectDescriptionField: {"  descripción enviada  "},
				DefectSeverityField:    {"Crítica"},
				DefectStoryField:       {"123"},
			},
			wantForm: DefectForm{
				Description: "  descripción enviada  ",
				Severity:    "Crítica",
				StoryID:     "123",
			},
			wantProject: 42,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form, input, fieldErrors := parseDefectForm(test.values, test.wantProject)
			if form != test.wantForm {
				t.Errorf("formulario = %+v; se esperaba %+v", form, test.wantForm)
			}
			if input.ProjectID != test.wantProject {
				t.Errorf("el proyecto debe provenir del contexto servidor: %d", input.ProjectID)
			}
			if test.wantError == "" && len(fieldErrors) != 0 {
				t.Errorf("no se esperaban errores al armar el formulario: %v", fieldErrors)
			}
			if test.wantError != "" && fieldErrors[DefectStoryField] != test.wantError {
				t.Errorf("error de historia = %q; se esperaba %q", fieldErrors[DefectStoryField], test.wantError)
			}
		})
	}
}

func TestParseDefectFormRejectsMissingAndNonNumericStory(t *testing.T) {
	tests := []struct {
		name      string
		story     string
		wantError string
	}{
		{name: "historia vacía", story: "", wantError: "Seleccione la historia relacionada"},
		{name: "historia no numérica", story: "no-es-un-id", wantError: "La historia no pertenece al proyecto"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := url.Values{
				DefectDescriptionField: {"texto conservado"},
				DefectSeverityField:    {"Alta"},
			}
			if test.story != "" {
				values.Set(DefectStoryField, test.story)
			}
			form, _, fieldErrors := parseDefectForm(values, 42)
			if form.StoryID != test.story {
				t.Errorf("se debía conservar la historia enviada %q, se obtuvo %q", test.story, form.StoryID)
			}
			if fieldErrors[DefectStoryField] != test.wantError {
				t.Errorf("error de historia = %q; se esperaba %q", fieldErrors[DefectStoryField], test.wantError)
			}
		})
	}
}

func TestDefectFieldErrorMapsEachDomainErrorToItsFieldAndMessage(t *testing.T) {
	tests := []struct {
		err     error
		field   string
		message string
	}{
		{defect.ErrDescriptionRequired, DefectDescriptionField, "La descripción es obligatoria"},
		{defect.ErrDescriptionTooLong, DefectDescriptionField, "La descripción no puede superar 1000 caracteres"},
		{defect.ErrSeverityRequired, DefectSeverityField, "Seleccione la severidad"},
		{defect.ErrSeverityInvalid, DefectSeverityField, "La severidad no es válida"},
		{defect.ErrStoryRequired, DefectStoryField, "Seleccione la historia relacionada"},
		{defect.ErrStoryNotFound, DefectStoryField, "La historia no pertenece al proyecto"},
	}
	for _, test := range tests {
		field, message, ok := defectFieldError(test.err)
		if !ok || field != test.field || message != test.message {
			t.Errorf("defectFieldError(%v) = (%q, %q, %v); se esperaba (%q, %q, true)",
				test.err, field, message, ok, test.field, test.message)
		}
	}

	if field, message, ok := defectFieldError(defect.ErrNoActiveSprint); ok || field != "" || message != "" {
		t.Errorf("ErrNoActiveSprint no es un error de campo: (%q, %q, %v)", field, message, ok)
	}
}
