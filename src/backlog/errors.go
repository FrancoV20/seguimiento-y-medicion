package backlog

import "errors"

// Errores de validación y persistencia del alta de historias (T007).
var (
	ErrProjectNotActive           = errors.New("el proyecto no existe o no está activo")
	ErrTitleRequired              = errors.New("el título es obligatorio")
	ErrDescriptionRequired        = errors.New("la descripción es obligatoria")
	ErrInvalidPriority            = errors.New("la prioridad debe ser Alta, Media o Baja")
	ErrAcceptanceCriteriaRequired = errors.New("agregá al menos un criterio de aceptación")
	ErrInvalidStoryPoints         = errors.New("los Story Points deben ser 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89")
	ErrPersistence                = errors.New("no se pudo registrar la historia")
)
