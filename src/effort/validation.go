package effort

import (
	"strings"
	"time"
)

// ValidatedInput son los datos ya interpretados y normalizados de una solicitud válida.
type ValidatedInput struct {
	Hours    Hours
	Activity string
}

// ValidateInput valida los campos que no necesitan consultar el store: horas, actividad,
// fecha e integrante omitido. Acumula todos los errores en el orden horas, actividad,
// fecha, integrante, para informarlos juntos (FR-010). now se inyecta para que las
// pruebas de fecha sean deterministas.
func ValidateInput(req LogEffortRequest, now time.Time) (ValidatedInput, []FieldError) {
	var errs []FieldError
	var in ValidatedInput

	hours, err := ParseHours(req.Hours)
	if err != nil {
		errs = append(errs, FieldError{Field: FieldHours, Err: ErrInvalidHours})
	} else {
		in.Hours = hours
	}

	// Normaliza espacios: recorta los extremos y junta los espacios internos en uno.
	activity := strings.Join(strings.Fields(req.Activity), " ")
	if activity == "" {
		errs = append(errs, FieldError{Field: FieldActivity, Err: ErrActivityRequired})
	} else {
		in.Activity = activity
	}

	if req.WorkDate.IsZero() {
		errs = append(errs, FieldError{Field: FieldDate, Err: ErrDateRequired})
	} else if afterCalendarDay(req.WorkDate, now) {
		errs = append(errs, FieldError{Field: FieldDate, Err: ErrDateInFuture})
	}

	if strings.TrimSpace(req.MemberID) == "" {
		errs = append(errs, FieldError{Field: FieldMember, Err: ErrMemberRequired})
	}

	return in, errs
}

// afterCalendarDay indica si date cae en un día de calendario posterior al de now,
// ignorando la hora: trabajar "hoy a las 23:59" no es una fecha futura.
func afterCalendarDay(date, now time.Time) bool {
	dy, dm, dd := date.Date()
	ny, nm, nd := now.Date()
	return time.Date(dy, dm, dd, 0, 0, 0, 0, time.UTC).After(time.Date(ny, nm, nd, 0, 0, 0, 0, time.UTC))
}