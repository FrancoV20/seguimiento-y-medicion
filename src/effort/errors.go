package effort

import "errors"

// Mensajes exactos definidos en los criterios de aceptación de HU-03 (Issue #4).
var (
	ErrInvalidHours     = errors.New("El valor de horas no es válido")
	ErrActivityRequired = errors.New("La descripción de la actividad es obligatoria")
	ErrDateRequired     = errors.New("La fecha es requerida")
	ErrDateInFuture     = errors.New("La fecha no puede ser posterior a la fecha actual")
	ErrMemberRequired   = errors.New("Se debe especificar el integrante responsable")
	ErrStoryNotInSprint = errors.New("La historia no está asignada a un Sprint activo")
	ErrStorage          = errors.New("La operación no pudo completarse")
)

// errMemberNotInProject lo devuelve el almacenamiento cuando, al guardar, el integrante ya no
// pertenece al proyecto de la historia (por ejemplo, HU-11 lo quitó mientras cargaba horas).
// El servicio lo informa como ErrMemberRequired.
var errMemberNotInProject = errors.New("el integrante ya no pertenece al proyecto de la historia")
