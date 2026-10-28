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
	ErrActiveSprintConflict = errors.New("la historia ya está asignada a un Sprint")
	ErrInvalidSprintNumber  = errors.New("el número de Sprint debe ser positivo")
	ErrPersistence          = errors.New("no se pudo completar la operación de persistencia")
	ErrActiveSprintExists   = errors.New("ya hay un Sprint activo en el proyecto")
)

// ActiveSprintExistsError indica que el proyecto ya tiene un Sprint activo e informa
// su número. errors.Is(err, ErrActiveSprintExists) lo reconoce (FIX-HU-04).
type ActiveSprintExistsError struct {
	Number int64
}

func (e *ActiveSprintExistsError) Error() string {
	return fmt.Sprintf("ya hay un Sprint activo (Sprint %d)", e.Number)
}

// Is permite usar errors.Is(err, ErrActiveSprintExists).
func (e *ActiveSprintExistsError) Is(target error) bool { return target == ErrActiveSprintExists }

// StoryAlreadyAssignedError indica que la historia ya está asignada a un Sprint e informa
// cuál. errors.Is(err, ErrActiveSprintConflict) lo reconoce (FIX-HU-04).
type StoryAlreadyAssignedError struct {
	StoryID      int64
	SprintNumber int64
}

func (e *StoryAlreadyAssignedError) Error() string {
	if e.SprintNumber > 0 {
		return fmt.Sprintf("la historia %d ya está asignada al Sprint %d", e.StoryID, e.SprintNumber)
	}
	return fmt.Sprintf("la historia %d ya está asignada a un Sprint", e.StoryID)
}

// Is permite usar errors.Is(err, ErrActiveSprintConflict).
func (e *StoryAlreadyAssignedError) Is(target error) bool { return target == ErrActiveSprintConflict }

func persistenceError(operation string) error {
	return fmt.Errorf("%w: %s", ErrPersistence, operation)
}

// StoryRemovalError indica que una historia no se puede quitar del Sprint. Reason contiene
// el motivo visible de la regla de quitar ("Ya está en progreso", "Ya está hecha" o "Tiene
// horas registradas"); vacío cuando la historia no existe o no pertenece al Sprint.
// errors.Is(err, ErrStoryNotAvailable) lo reconoce (T047).
type StoryRemovalError struct {
	StoryID int64
	Reason  string
}

func (e *StoryRemovalError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("la historia %d no se puede quitar del Sprint: %s", e.StoryID, e.Reason)
	}
	return fmt.Sprintf("%s: historia %d", ErrStoryNotAvailable.Error(), e.StoryID)
}

// Message devuelve el texto que se muestra al usuario: el motivo de la regla de quitar o,
// si no lo hay, el mensaje base de historia no disponible.
func (e *StoryRemovalError) Message() string {
	if e.Reason != "" {
		return e.Reason
	}
	return ErrStoryNotAvailable.Error()
}

// Is permite usar errors.Is(err, ErrStoryNotAvailable).
func (e *StoryRemovalError) Is(target error) bool { return target == ErrStoryNotAvailable }
