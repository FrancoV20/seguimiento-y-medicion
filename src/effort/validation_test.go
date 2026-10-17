package effort

import (
	"errors"
	"testing"
	"time"
)

// now fijo para que las pruebas de fecha sean deterministas.
var fixedNow = time.Date(2026, time.October, 10, 15, 0, 0, 0, time.UTC)

// validRequest devuelve una solicitud correcta; cada prueba rompe solo el campo que le interesa.
func validRequest() LogEffortRequest {
	return LogEffortRequest{
		StoryID:  "S1",
		MemberID: "M1",
		WorkDate: time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC),
		Activity: "Desarrollo de endpoint en Go",
		Hours:    "4",
	}
}

// errorFor busca el error asociado a un campo; devuelve nil si no hay.
func errorFor(errs []FieldError, field string) error {
	for _, e := range errs {
		if e.Field == field {
			return e.Err
		}
	}
	return nil
}

func TestValidateInputValid(t *testing.T) {
	input, errs := ValidateInput(validRequest(), fixedNow)
	if len(errs) != 0 {
		t.Fatalf("no se esperaban errores, se obtuvo %v", errs)
	}
	if input.Hours != 400 {
		t.Errorf("Hours = %d, se esperaba 400", input.Hours)
	}
	if input.Activity != "Desarrollo de endpoint en Go" {
		t.Errorf("Activity = %q", input.Activity)
	}
}

func TestValidateInputHours(t *testing.T) {
	cases := []string{"", "abc", "0", "-1", "24,01", "25", "1,234"}
	for _, h := range cases {
		t.Run("horas "+h, func(t *testing.T) {
			req := validRequest()
			req.Hours = h
			_, errs := ValidateInput(req, fixedNow)
			if len(errs) != 1 {
				t.Fatalf("se esperaba 1 error, se obtuvo %v", errs)
			}
			if !errors.Is(errorFor(errs, FieldHours), ErrInvalidHours) {
				t.Errorf("se esperaba ErrInvalidHours en %q, se obtuvo %v", FieldHours, errs)
			}
		})
	}
}

func TestValidateInputActivity(t *testing.T) {
	for _, a := range []string{"", "   ", "\t\n "} {
		t.Run("actividad vacía", func(t *testing.T) {
			req := validRequest()
			req.Activity = a
			_, errs := ValidateInput(req, fixedNow)
			if len(errs) != 1 {
				t.Fatalf("se esperaba 1 error, se obtuvo %v", errs)
			}
			if !errors.Is(errorFor(errs, FieldActivity), ErrActivityRequired) {
				t.Errorf("se esperaba ErrActivityRequired, se obtuvo %v", errs)
			}
		})
	}
}

func TestValidateInputActivityIsNormalized(t *testing.T) {
	req := validRequest()
	req.Activity = "  Desarrollo   de\n endpoint \t"
	input, errs := ValidateInput(req, fixedNow)
	if len(errs) != 0 {
		t.Fatalf("no se esperaban errores, se obtuvo %v", errs)
	}
	if input.Activity != "Desarrollo de endpoint" {
		t.Errorf("Activity = %q, se esperaba %q", input.Activity, "Desarrollo de endpoint")
	}
}

func TestValidateInputDate(t *testing.T) {
	cases := []struct {
		name    string
		date    time.Time
		wantErr error
	}{
		{"omitida", time.Time{}, ErrDateRequired},
		{"mañana", time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC), ErrDateInFuture},
		{"hoy", time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC), nil},
		{"hoy más tarde que ahora", time.Date(2026, time.October, 10, 23, 59, 0, 0, time.UTC), nil},
		{"ayer", time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := validRequest()
			req.WorkDate = c.date
			_, errs := ValidateInput(req, fixedNow)
			got := errorFor(errs, FieldDate)
			if !errors.Is(got, c.wantErr) {
				t.Errorf("error de fecha = %v, se esperaba %v", got, c.wantErr)
			}
			if c.wantErr == nil && len(errs) != 0 {
				t.Errorf("no se esperaban errores, se obtuvo %v", errs)
			}
		})
	}
}

func TestValidateInputMember(t *testing.T) {
	for _, m := range []string{"", "   "} {
		t.Run("integrante omitido", func(t *testing.T) {
			req := validRequest()
			req.MemberID = m
			_, errs := ValidateInput(req, fixedNow)
			if len(errs) != 1 {
				t.Fatalf("se esperaba 1 error, se obtuvo %v", errs)
			}
			if !errors.Is(errorFor(errs, FieldMember), ErrMemberRequired) {
				t.Errorf("se esperaba ErrMemberRequired, se obtuvo %v", errs)
			}
		})
	}
}

func TestValidateInputAccumulatesAllErrors(t *testing.T) {
	req := LogEffortRequest{StoryID: "S1", Hours: "0"} // todo mal: horas, actividad, fecha, integrante
	_, errs := ValidateInput(req, fixedNow)

	want := []struct {
		field string
		err   error
	}{
		{FieldHours, ErrInvalidHours},
		{FieldActivity, ErrActivityRequired},
		{FieldDate, ErrDateRequired},
		{FieldMember, ErrMemberRequired},
	}
	if len(errs) != len(want) {
		t.Fatalf("se esperaban %d errores juntos, se obtuvo %v", len(want), errs)
	}
	for i, w := range want {
		if errs[i].Field != w.field || !errors.Is(errs[i].Err, w.err) {
			t.Errorf("error %d = {%s, %v}, se esperaba {%s, %v}", i, errs[i].Field, errs[i].Err, w.field, w.err)
		}
	}
}
