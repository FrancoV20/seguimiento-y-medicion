package defect

import "errors"

// Errores de HU-19 al resolver o reabrir un defecto. Sin Sprint activo se reutiliza
// ErrNoActiveSprint de HU-06.
var (
	ErrDefectNotFound            = errors.New("el defecto no existe en el proyecto")
	ErrResolutionBeforeDetection = errors.New("el Sprint de resolución es anterior al Sprint de detección")
	ErrAlreadyResolved           = errors.New("el defecto ya está resuelto")
	ErrAlreadyOpen               = errors.New("el defecto ya está abierto")
	errUnknownDefectStatus       = errors.New("el defecto tiene un estado desconocido")
)
