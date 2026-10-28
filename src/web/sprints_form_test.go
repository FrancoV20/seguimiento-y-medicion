package web

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestBuildSprintFormInput_ParsesValidFieldsAndDeduplicatesStories(t *testing.T) {
	values := url.Values{
		FieldGoal:      {"Preparar entrega"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {"12", "12", "8"},
		"project_id":   {"999"},
		"sprint_id":    {"777"},
	}

	form, input := buildSprintFormInput(41, values)

	if input.ProjectID != 41 {
		t.Errorf("ProjectID = %d, se esperaba el ID del contexto 41", input.ProjectID)
	}
	if input.Goal != "Preparar entrega" {
		t.Errorf("Goal = %q, se esperaba %q", input.Goal, "Preparar entrega")
	}
	if want := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC); !input.StartDate.Equal(want) {
		t.Errorf("StartDate = %v, se esperaba %v", input.StartDate, want)
	}
	if want := time.Date(2026, time.November, 14, 0, 0, 0, 0, time.UTC); !input.EndDate.Equal(want) {
		t.Errorf("EndDate = %v, se esperaba %v", input.EndDate, want)
	}
	if !reflect.DeepEqual(input.StoryIDs, []int64{12, 8}) {
		t.Errorf("StoryIDs = %v, se esperaban IDs únicos [12 8]", input.StoryIDs)
	}
	if form.HasErrors() {
		t.Errorf("no se esperaban errores de formulario: %v", form.Errors)
	}
	if _, exists := form.Values["sprint_id"]; exists {
		t.Error("el identificador enviado del Sprint debe ignorarse")
	}
}

func TestBuildSprintFormInput_ReportsMissingDates(t *testing.T) {
	tests := []struct {
		name          string
		values        url.Values
		field         string
		expectedError string
	}{
		{
			name: "missing start date",
			values: url.Values{
				FieldGoal:    {"Preparar entrega"},
				FieldEndDate: {"2026-11-14"},
			},
			field:         FieldStartDate,
			expectedError: MsgMissingStartDate,
		},
		{
			name: "missing end date",
			values: url.Values{
				FieldGoal:      {"Preparar entrega"},
				FieldStartDate: {"2026-11-01"},
			},
			field:         FieldEndDate,
			expectedError: MsgMissingEndDate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form, _ := buildSprintFormInput(41, test.values)
			if got := form.Errors[test.field]; got != test.expectedError {
				t.Errorf("error[%q] = %q, se esperaba %q", test.field, got, test.expectedError)
			}
		})
	}
}

func TestBuildSprintFormInput_ReportsInvalidDateFormatAndPreservesValues(t *testing.T) {
	values := url.Values{
		FieldGoal:      {"Preparar entrega"},
		FieldStartDate: {"01/11/2026"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {"12"},
	}

	form, _ := buildSprintFormInput(41, values)

	if got := form.Errors[FieldStartDate]; got != MsgInvalidDateFormat {
		t.Errorf("error de fecha = %q, se esperaba %q", got, MsgInvalidDateFormat)
	}
	for field, want := range map[string]string{
		FieldGoal:      "Preparar entrega",
		FieldStartDate: "01/11/2026",
		FieldEndDate:   "2026-11-14",
		FieldStory:     "12",
	} {
		if got := form.Values[field]; got != want {
			t.Errorf("value[%q] = %q, se esperaba conservar %q", field, got, want)
		}
	}
}

func TestBuildSprintFormInput_RejectsNonNumericStoryID(t *testing.T) {
	values := url.Values{
		FieldGoal:      {"Preparar entrega"},
		FieldStartDate: {"2026-11-01"},
		FieldEndDate:   {"2026-11-14"},
		FieldStory:     {"no-es-un-id"},
	}

	form, input := buildSprintFormInput(41, values)

	if form.Errors[FieldStory] == "" {
		t.Error("se esperaba un error junto a la lista de historias")
	}
	if len(input.StoryIDs) != 0 {
		t.Errorf("no se esperaba aceptar IDs no numéricos: %v", input.StoryIDs)
	}
	if form.Values[FieldStory] != "no-es-un-id" {
		t.Errorf("se debía conservar el ID enviado, se obtuvo %q", form.Values[FieldStory])
	}
}
