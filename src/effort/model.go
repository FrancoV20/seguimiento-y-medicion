package effort

import "time"

// Nombres de campo usados en los errores de validación.
const (
	FieldHours    = "hours"
	FieldActivity = "activity"
	FieldDate     = "date"
	FieldMember   = "member"
	FieldStory    = "story"
)

// SuccessMessage se muestra solo después de que el registro quedó persistido.
const SuccessMessage = "Esfuerzo registrado correctamente"

// EffortEntry es un registro de esfuerzo ya guardado sobre una historia.
type EffortEntry struct {
	ID        string
	StoryID   string
	MemberID  string
	WorkDate  time.Time // fecha de calendario del trabajo realizado
	Activity  string
	Hours     Hours
	CreatedAt time.Time
}

// LogEffortRequest son los datos tal como los ingresa el Product Builder, sin validar.
// El ID y CreatedAt no forman parte de la solicitud: los asigna el sistema.
type LogEffortRequest struct {
	StoryID  string
	MemberID string
	WorkDate time.Time // el valor cero (IsZero) significa "fecha omitida"
	Activity string
	Hours    string // texto ingresado, por ejemplo "4" o "1,5"; se interpreta con ParseHours
}

// FieldError asocia un error de validación con el campo que lo causó.
type FieldError struct {
	Field string
	Err   error
}

// LogEffortResult es el resultado de registrar un esfuerzo.
type LogEffortResult struct {
	Entry            *EffortEntry // registro creado; nil si la operación falló
	AccumulatedHours Hours        // esfuerzo real acumulado de la historia tras el registro
	SuccessMessage   string       // vacío salvo que el registro se haya persistido
	ValidationErrors []FieldError // uno por campo inválido
	StorageError     error        // ErrStorage si falló el almacenamiento, sin detalles internos
}

// Succeeded indica que el registro se persistió sin errores.
func (r LogEffortResult) Succeeded() bool {
	return r.Entry != nil && r.StorageError == nil && len(r.ValidationErrors) == 0
}

// AccumulatedEffort es el esfuerzo real acumulado de una historia.
type AccumulatedEffort struct {
	StoryID    string
	TotalHours Hours // suma de las horas de todos sus registros; 0 si no tiene
}