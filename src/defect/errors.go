package defect

import "errors"

var (
	ErrDescriptionRequired = errors.New("falta la descripción")
	ErrDescriptionTooLong  = errors.New("la descripción excede el límite")
	ErrDescriptionInvalid  = errors.New("la descripción contiene caracteres no válidos")
	ErrSeverityRequired    = errors.New("falta la severidad")
	ErrSeverityInvalid     = errors.New("la severidad no está permitida")
	ErrStoryRequired       = errors.New("falta la historia")
	ErrStoryNotFound       = errors.New("la historia no pertenece al proyecto")
	ErrNoActiveSprint      = errors.New("no existe un Sprint activo")
)
