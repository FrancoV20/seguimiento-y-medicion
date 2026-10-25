package sprint

import (
	"context"
	"errors"
	"fmt"
)

// Action es una acción que se puede hacer sobre una historia del Sprint activo
// desde el Tablero (HU-14). El valor es el que viaja en el campo "accion".
type Action string

const (
	ActionStart    Action = "comenzar"
	ActionComplete Action = "completar"
	ActionReopen   Action = "reabrir"
)

// storyStatusStep es un paso permitido del ciclo de vida de una historia del Sprint activo.
type storyStatusStep struct{ from, to StoryStatus }

// storyStatusSteps son los tres pasos permitidos (HU-14): En Sprint → En progreso → Hecho →
// En progreso. Es la única fuente de la regla: las demás funciones salen de acá.
var storyStatusSteps = map[Action]storyStatusStep{
	ActionStart:    {from: EnSprint, to: InProgress},
	ActionComplete: {from: InProgress, to: Done},
	ActionReopen:   {from: Done, to: InProgress},
}

// CanTransition indica si una historia puede pasar de un estado a otro (HU-14, T010).

func CanTransition(from, to StoryStatus) bool {
	for _, step := range storyStatusSteps {
		if step.from == from && step.to == to {
			return true
		}
	}
	return false
}

// NextAction devuelve la única acción válida para un estado; ok es false si no hay (HU-14, T010).

func NextAction(status StoryStatus) (action Action, ok bool) {
	for action, step := range storyStatusSteps {
		if step.from == status {
			return action, true
		}
	}
	return "", false
}

// ActionTransition devuelve el estado de origen y el de destino de una acción; ok es false
// si la acción no existe (HU-14, T010).

func ActionTransition(action Action) (from, to StoryStatus, ok bool) {
	step, ok := storyStatusSteps[action]
	return step.from, step.to, ok
}

// Errores del dominio del cambio de estado (HU-14). Cualquier otro error es una falla de
// almacenamiento. Los textos que ve la persona se definen en la pantalla (board_form.go).
var (
	ErrStoryNotFound        = errors.New("la historia no existe")
	ErrNotInActiveSprint    = errors.New("la historia no está asignada a un Sprint activo")
	ErrAlreadyInStatus      = errors.New("la historia ya está en ese estado")
	ErrStatusChanged        = errors.New("el estado de la historia cambió")
	ErrTransitionNotAllowed = errors.New("la transición de estado no está permitida")
)

// AlreadyInStatusError indica que la historia ya está en el estado pedido e informa cuál.
// errors.Is(err, ErrAlreadyInStatus) lo reconoce.
type AlreadyInStatusError struct {
	Status StoryStatus
}

func (e *AlreadyInStatusError) Error() string {
	return fmt.Sprintf("la historia ya está en %s", e.Status)
}

// Is permite usar errors.Is(err, ErrAlreadyInStatus).
func (e *AlreadyInStatusError) Is(target error) bool { return target == ErrAlreadyInStatus }

// TransitionNotAllowedError indica que el paso de un estado a otro no está permitido e
// informa el origen y el destino. errors.Is(err, ErrTransitionNotAllowed) lo reconoce.
type TransitionNotAllowedError struct {
	From, To StoryStatus
}

func (e *TransitionNotAllowedError) Error() string {
	return fmt.Sprintf("no se puede pasar de %s a %s", e.From, e.To)
}

// Is permite usar errors.Is(err, ErrTransitionNotAllowed).
func (e *TransitionNotAllowedError) Is(target error) bool { return target == ErrTransitionNotAllowed }

// StoryState es lo que se lee de una historia para decidir el cambio.
type StoryState struct {
	Status       StoryStatus
	SprintID     int64 // 0 si la historia no está asignada a ningún Sprint
	SprintActive bool  // verdadero si el Sprint asignado está activo
	HasEffort    bool  // verdadero si tiene horas registradas
}

// StatusResult es el resultado de un cambio aplicado.
type StatusResult struct {
	From, To       StoryStatus
	NoHoursWarning bool // verdadero al completar una historia sin horas registradas
}

// StoryStatusStore es el puerto de almacenamiento del cambio de estado.
type StoryStatusStore interface {
	// FindStory lee la historia dentro del proyecto; devuelve ErrStoryNotFound si no existe
	// o es de otro proyecto.
	FindStory(ctx context.Context, projectID, storyID int64) (StoryState, error)
	// UpdateStatus cambia el estado solo si sigue siendo from y la historia pertenece a un
	// Sprint activo, en una sola operación atómica. Devuelve si se cambió una fila.
	UpdateStatus(ctx context.Context, storyID int64, from, to StoryStatus) (changed bool, err error)
}

// StatusService aplica la regla de cambio de estado sobre un almacenamiento.
type StatusService struct {
	store StoryStatusStore
}

// NewStatusService crea el servicio respaldado por el almacenamiento indicado.
func NewStatusService(store StoryStatusStore) StatusService {
	return StatusService{store: store}
}

var errNotImplemented = errors.New("HU-14: todavía no implementado")

// ChangeStatus cambia el estado de una historia del Sprint activo. from y to son el origen y el
// destino de la acción pedida; expected es el estado que la persona veía ("" si no se envió).
// RED: esqueleto sin implementar.
func (s StatusService) ChangeStatus(ctx context.Context, projectID, storyID int64, from, to, expected StoryStatus) (StatusResult, error) {
	return StatusResult{}, errNotImplemented
}
