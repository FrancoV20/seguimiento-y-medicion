package web

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestParseProjectForm_ValidDatesAndPreservedValues(t *testing.T) {
	values := url.Values{
		ProjectNameField:      {"  Proyecto validado  "},
		ProjectStartDateField: {"2026-10-23"},
		ProjectEndDateField:   {"2026-10-24"},
		ProjectMembersField:   {"7"},
	}

	form, input, errors := parseProjectForm(values)

	if len(errors) != 0 {
		t.Fatalf("no se esperaban errores, se obtuvo %v", errors)
	}
	if form.Name != "  Proyecto validado  " || form.StartDate != "2026-10-23" ||
		form.EndDate != "2026-10-24" || !reflect.DeepEqual(form.MemberIDs, []string{"7"}) {
		t.Errorf("se debían conservar los valores originales, se obtuvo %+v", form)
	}
	if input.Name != "Proyecto validado" || !sameCalendarDate(input.StartDate, time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)) ||
		!sameCalendarDate(input.EndDate, time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)) ||
		!reflect.DeepEqual(input.MemberIDs, []string{"7"}) {
		t.Errorf("entrada parseada incorrectamente: %+v", input)
	}
}

func TestParseProjectForm_InvalidDateFormat(t *testing.T) {
	values := url.Values{
		ProjectNameField:      {"Proyecto"},
		ProjectStartDateField: {"23/10/2026"},
		ProjectEndDateField:   {"2026-10-24-extra"},
		ProjectMembersField:   {"7"},
	}

	form, _, errors := parseProjectForm(values)

	if errors[ProjectStartDateField] != MsgInvalidDateFormat ||
		errors[ProjectEndDateField] != MsgInvalidDateFormat {
		t.Errorf("se esperaba el error de formato junto a ambas fechas, se obtuvo %v", errors)
	}
	if form.StartDate != values.Get(ProjectStartDateField) || form.EndDate != values.Get(ProjectEndDateField) {
		t.Errorf("se debían conservar las fechas inválidas, se obtuvo %+v", form)
	}
}

func TestParseProjectForm_MissingDates(t *testing.T) {
	tests := []struct {
		name        string
		values      url.Values
		field, want string
	}{
		{
			name: "fecha de inicio ausente",
			values: url.Values{
				ProjectEndDateField: {"2026-10-24"},
				ProjectMembersField: {"7"},
			},
			field: ProjectStartDateField,
			want:  MsgMissingStartDate,
		},
		{
			name: "fecha de fin ausente",
			values: url.Values{
				ProjectStartDateField: {"2026-10-23"},
				ProjectMembersField:   {"7"},
			},
			field: ProjectEndDateField,
			want:  MsgMissingEndDate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			form, _, errors := parseProjectForm(tc.values)
			if errors[tc.field] != tc.want {
				t.Errorf("para %s se esperaba %q, se obtuvo %q", tc.field, tc.want, errors[tc.field])
			}
			if form.StartDate != tc.values.Get(ProjectStartDateField) ||
				form.EndDate != tc.values.Get(ProjectEndDateField) {
				t.Errorf("se debían conservar las fechas enviadas, se obtuvo %+v", form)
			}
		})
	}
}

func TestParseProjectForm_DeduplicatesMemberIDsAndPreservesSubmittedValues(t *testing.T) {
	values := url.Values{
		ProjectNameField:      {"Proyecto"},
		ProjectStartDateField: {"2026-10-23"},
		ProjectEndDateField:   {"2026-10-24"},
		ProjectMembersField:   {"7", "9", "7"},
	}

	form, input, errors := parseProjectForm(values)

	if len(errors) != 0 {
		t.Fatalf("no se esperaban errores, se obtuvo %v", errors)
	}
	if !reflect.DeepEqual(form.MemberIDs, []string{"7", "9", "7"}) {
		t.Errorf("se debían conservar los identificadores enviados, se obtuvo %v", form.MemberIDs)
	}
	if !reflect.DeepEqual(input.MemberIDs, []string{"7", "9"}) {
		t.Errorf("se esperaban identificadores únicos en la entrada, se obtuvo %v", input.MemberIDs)
	}
}
