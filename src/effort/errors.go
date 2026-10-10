package effort

import "errors"

// Mensajes exactos definidos en los criterios de aceptación de HU-03 (Issue #4).
var (
	ErrInvalidHours     = errors.New("El valor de horas no es válido")
	ErrActivityRequired = errors.New("La descripción de la actividad es obligatoria")
	ErrDateRequired     = errors.New("La fecha es requerida")
	ErrMemberRequired   = errors.New("Se debe especificar el integrante responsable")
	ErrStoryNotInSprint = errors.New("La historia no está asignada a un Sprint activo")
	ErrStorage          = errors.New("La operación no pudo completarse")
)
