package web

import (
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// T008: armado del envío del formulario de esfuerzo.

func effortValues(story, date, activity, hours string) map[string]string {
	return map[string]string{
		EffortFieldStory: story, EffortFieldDate: date, EffortFieldActivity: activity, EffortFieldHours: hours,
	}
}

func TestParseEffortForm_Fecha(t *testing.T) {
	cases := []struct {
		name, raw string
		wantDate  time.Time
		wantErr   string
	}{
		{"vacía pasa a HU-03", "", time.Time{}, ""},
		{"solo espacios pasa a HU-03", "   ", time.Time{}, ""},
		{"válida", "2026-10-20", time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), ""},
		{"formato dd/mm/aaaa", "20/10/2026", time.Time{}, MsgEffortInvalidDate},
		{"día inexistente", "2026-02-30", time.Time{}, MsgEffortInvalidDate},
		{"texto", "hoy", time.Time{}, MsgEffortInvalidDate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sub, errs := parseEffortForm(effortValues("1", c.raw, "x", "1"))
			if errs[EffortFieldDate] != c.wantErr {
				t.Errorf("error de fecha: se esperaba %q, se obtuvo %q", c.wantErr, errs[EffortFieldDate])
			}
			if !sub.Date.Equal(c.wantDate) {
				t.Errorf("fecha: se esperaba %v, se obtuvo %v", c.wantDate, sub.Date)
			}
		})
	}
}

func TestParseEffortForm_Historia(t *testing.T) {
	cases := []struct {
		raw     string
		wantID  int64
		wantErr string
	}{
		{"", 0, MsgEffortSelectStory},
		{"abc", 0, MsgEffortSelectStory},
		{"-3", 0, MsgEffortSelectStory},
		{"0", 0, MsgEffortSelectStory},
		{"007", 0, MsgEffortSelectStory},
		{"99999999999999999999", 0, MsgEffortSelectStory},
		{"42", 42, ""},
		{" 42 ", 42, ""},
	}
	for _, c := range cases {
		sub, errs := parseEffortForm(effortValues(c.raw, "2026-10-20", "x", "1"))
		if sub.StoryID != c.wantID || errs[EffortFieldStory] != c.wantErr {
			t.Errorf("historia %q: se esperaba (%d, %q), se obtuvo (%d, %q)", c.raw, c.wantID, c.wantErr, sub.StoryID, errs[EffortFieldStory])
		}
	}
}

func TestParseEffortForm_HorasYActividadPasanTalCualAHU03(t *testing.T) {
	for _, hours := range []string{"1,5", "1.5", "-2", "0", "", "dos", "1e2", "25"} {
		sub, errs := parseEffortForm(effortValues("1", "2026-10-20", "  Diseño  ", hours))
		if sub.Hours != hours {
			t.Errorf("las horas deben llegar a HU-03 tal como se enviaron: se esperaba %q, se obtuvo %q", hours, sub.Hours)
		}
		if _, ok := errs[EffortFieldHours]; ok {
			t.Errorf("las horas %q no deben validarse en la web (las valida HU-03)", hours)
		}
		if sub.Activity != "  Diseño  " {
			t.Errorf("la actividad debe llegar a HU-03 sin cambios, se obtuvo %q", sub.Activity)
		}
	}
}

// Las horas con coma o con punto, negativas, cero, vacías, con letras o en formato científico
// las interpreta HU-03: la web muestra su mensaje exacto.
func TestEffortActions_Validate_HorasConComaPuntoYInvalidas(t *testing.T) {
	now := time.Date(2026, 10, 20, 15, 0, 0, 0, time.Local)
	actions := NewEffortActions(effort.NewService(effort.NewTestStore(), func() time.Time { return now }), func() time.Time { return now })
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	for hours, valid := range map[string]bool{
		"1,5": true, "1.5": true, "24": true, "0,25": true,
		"-2": false, "0": false, "": false, "dos": false, "1e2": false, "24,01": false, "1,555": false,
	} {
		violations := actions.Validate(EffortSubmission{StoryID: 1, MemberID: 1, Date: day, Activity: "Diseño", Hours: hours})
		got := violationFor(violations, EffortFieldHours)
		if valid && got != "" {
			t.Errorf("horas %q: no se esperaba error, se obtuvo %q", hours, got)
		}
		if !valid && got != effort.ErrInvalidHours.Error() {
			t.Errorf("horas %q: se esperaba %q, se obtuvo %q", hours, effort.ErrInvalidHours.Error(), got)
		}
	}
}

func TestMergeEffortErrors_UnErrorPorCampoYGenerales(t *testing.T) {
	form := NewFormView()
	form.AddError(EffortFieldDate, MsgEffortInvalidDate)
	general := mergeEffortErrors(form, []EffortViolation{
		{Field: EffortFieldDate, Message: effort.ErrDateRequired.Error()},
		{Field: EffortFieldHours, Message: effort.ErrInvalidHours.Error()},
		{Field: EffortFieldHours, Message: "otro mensaje"},
		{Field: "", Message: effort.ErrMemberRequired.Error()},
	})
	if form.Errors[EffortFieldDate] != MsgEffortInvalidDate {
		t.Errorf("el error propio de la fecha no debe pisarse, se obtuvo %q", form.Errors[EffortFieldDate])
	}
	if form.Errors[EffortFieldHours] != effort.ErrInvalidHours.Error() {
		t.Errorf("horas: se esperaba el primer mensaje de HU-03, se obtuvo %q", form.Errors[EffortFieldHours])
	}
	if len(form.Errors) != 2 {
		t.Errorf("se esperaba un error por campo (2), se obtuvieron %v", form.Errors)
	}
	if len(general) != 1 || general[0] != effort.ErrMemberRequired.Error() {
		t.Errorf("errores generales: se obtuvo %v", general)
	}
}

func TestFormatEffortHours(t *testing.T) {
	for h, want := range map[effort.Hours]string{0: "0", 100: "1", 250: "2,5", 675: "6,75", 5: "0,05", 2400: "24"} {
		if got := FormatEffortHours(h); got != want {
			t.Errorf("FormatEffortHours(%d) = %q, se esperaba %q", h, got, want)
		}
	}
}

func violationFor(vs []EffortViolation, field string) string {
	for _, v := range vs {
		if v.Field == field {
			return v.Message
		}
	}
	return ""
}
