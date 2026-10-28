package web

import (
	"strings"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// Campos del formulario de registro de esfuerzo (HU-24, T011). No hay campo de integrante:
// sale de la identificación del servidor (FR-008).
const (
	EffortFieldStory    = "historia"
	EffortFieldDate     = "fecha"
	EffortFieldActivity = "actividad"
	EffortFieldHours    = "horas"
)

// effortFields son los campos que se conservan al volver a mostrar el formulario.
var effortFields = []string{EffortFieldStory, EffortFieldDate, EffortFieldActivity, EffortFieldHours}

// Textos propios de HU-24, definidos en un solo lugar. Los de horas, actividad, fecha requerida,
// fecha futura e integrante son los de HU-03 y no se copian.
const (
	TitleEffort               = "Registro de esfuerzo"
	MsgEffortRecorded         = effort.SuccessMessage // "Esfuerzo registrado correctamente" (HU-03)
	MsgEffortSelectStory      = "Seleccione una historia"
	MsgEffortStoryNotInSprint = "La historia no pertenece al Sprint activo"
	MsgEffortInvalidDate      = "La fecha no es válida"
	MsgEffortNoSprint         = "No hay un Sprint activo"
	MsgEffortNoStories        = "El Sprint activo no tiene historias asignadas"
	MsgEffortNoEntries        = "Todavía no hay horas registradas en esta historia"
)

// effortDateLayout es el formato de <input type="date"> (AAAA-MM-DD).
const effortDateLayout = "2006-01-02"

// EffortSubmission es el envío ya leído, listo para HU-03. Las horas viajan como texto: las
// interpreta HU-03 (coma o punto, rango y formato), así la regla vive en un solo lugar.
type EffortSubmission struct {
	StoryID  int64
	MemberID int64
	Date     time.Time // cero si el campo vino vacío
	Activity string
	Hours    string
}

// EffortViolation es un error para mostrar. Field vacío significa error general.
type EffortViolation struct {
	Field   string
	Message string
}

// parseEffortForm arma el envío a partir de los valores del formulario y devuelve los errores
// de lectura propios de la web: historia faltante o no numérica y fecha con formato inválido.
func parseEffortForm(values map[string]string) (EffortSubmission, map[string]string) {
	errs := map[string]string{}
	var sub EffortSubmission

	sub.StoryID = parseID(strings.TrimSpace(values[EffortFieldStory]))
	if sub.StoryID == 0 {
		errs[EffortFieldStory] = MsgEffortSelectStory
	}

	if raw := strings.TrimSpace(values[EffortFieldDate]); raw != "" {
		d, err := time.Parse(effortDateLayout, raw)
		if err != nil {
			errs[EffortFieldDate] = MsgEffortInvalidDate
		} else {
			sub.Date = d
		}
	}

	sub.Activity = values[EffortFieldActivity]
	sub.Hours = values[EffortFieldHours]
	return sub, errs
}

// mergeEffortErrors suma al formulario las violaciones de HU-03 sin pisar los errores que ya
// tiene: queda un solo error por campo. Devuelve los errores generales (sin campo).
func mergeEffortErrors(form *FormView, violations []EffortViolation) []string {
	var general []string
	for _, v := range violations {
		if v.Field == "" {
			general = append(general, v.Message)
			continue
		}
		if _, exists := form.Errors[v.Field]; !exists {
			form.AddError(v.Field, v.Message)
		}
	}
	return general
}
