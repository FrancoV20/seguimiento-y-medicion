package sprint

import (
	"errors"
	"fmt"
)

var (
	ErrProjectNotFound      = errors.New("el proyecto no existe")
	ErrGoalRequired         = errors.New("el Sprint Goal es obligatorio")
	ErrInvalidSprintPeriod  = errors.New("el período del Sprint no es válido")
	ErrStoryNotAvailable    = errors.New("la historia no está disponible para asignación")
	ErrActiveSprintConflict = errors.New("la historia ya pertenece a otro Sprint activo")
	ErrInvalidSprintNumber  = errors.New("el número de Sprint debe ser positivo")
	ErrPersistence          = errors.New("no se pudo completar la operación de persistencia")
)

func persistenceError(operation string) error {
	return fmt.Errorf("%w: %s", ErrPersistence, operation)
}
