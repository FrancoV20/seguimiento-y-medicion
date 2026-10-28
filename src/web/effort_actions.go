package web

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// ErrEffortNotSaved indica que HU-03 no pudo guardar la carga. No lleva detalles internos.
var ErrEffortNotSaved = errors.New("el registro de esfuerzo no se pudo guardar")

// EffortActions es lo que la pantalla necesita de HU-03 (HU-24, T012).
type EffortActions interface {
	// Validate devuelve todas las violaciones de las reglas de HU-03 que no necesitan la base
	// (horas, actividad, fecha e integrante), cada una con su campo del formulario.
	Validate(sub EffortSubmission) []EffortViolation

	// Register registra la carga con el servicio de HU-03. Devuelve las violaciones si HU-03 la
	// rechazó, o ErrEffortNotSaved si falló el guardado; nil y nil si quedó registrada.
	Register(ctx context.Context, sub EffortSubmission) ([]EffortViolation, error)
}

// effortServiceActions adapta el servicio de HU-03.
type effortServiceActions struct {
	service *effort.Service
	now     func() time.Time
}

// NewEffortActions crea el adaptador sobre el servicio de HU-03. now es el mismo reloj que usa
// el servicio (nil = time.Now), para que Validate y Register apliquen la misma fecha de hoy.
func NewEffortActions(service *effort.Service, now func() time.Time) EffortActions {
	if now == nil {
		now = time.Now
	}
	return &effortServiceActions{service: service, now: now}
}

func (a *effortServiceActions) Validate(sub EffortSubmission) []EffortViolation {
	_, errs := effort.ValidateInput(toLogEffortRequest(sub), a.now())
	return effortViolations(errs)
}

func (a *effortServiceActions) Register(ctx context.Context, sub EffortSubmission) ([]EffortViolation, error) {
	res := a.service.LogEffort(ctx, toLogEffortRequest(sub))
	switch {
	case len(res.ValidationErrors) > 0:
		return effortViolations(res.ValidationErrors), nil
	case res.StorageError != nil || !res.Succeeded():
		return nil, ErrEffortNotSaved
	default:
		return nil, nil
	}
}

// toLogEffortRequest arma la solicitud de HU-03. Un identificador 0 (ausente) viaja vacío para
// que HU-03 aplique su propia regla de integrante no especificado.
func toLogEffortRequest(sub EffortSubmission) effort.LogEffortRequest {
	return effort.LogEffortRequest{
		StoryID:  idText(sub.StoryID),
		MemberID: idText(sub.MemberID),
		WorkDate: sub.Date,
		Activity: sub.Activity,
		Hours:    sub.Hours,
	}
}

func idText(id int64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}

// effortFieldOf traduce el campo de HU-03 al del formulario; el integrante es un error general
// porque no es un campo de la pantalla.
var effortFieldOf = map[string]string{
	effort.FieldHours:    EffortFieldHours,
	effort.FieldActivity: EffortFieldActivity,
	effort.FieldDate:     EffortFieldDate,
	effort.FieldStory:    EffortFieldStory,
	effort.FieldMember:   "",
}

// effortViolations convierte los errores de HU-03 conservando sus mensajes exactos.
func effortViolations(errs []effort.FieldError) []EffortViolation {
	out := make([]EffortViolation, 0, len(errs))
	for _, e := range errs {
		out = append(out, EffortViolation{Field: effortFieldOf[e.Field], Message: e.Err.Error()})
	}
	return out
}
