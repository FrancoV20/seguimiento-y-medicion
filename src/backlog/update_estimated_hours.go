package backlog

import (
	"context"
	"errors"
)

// Horas estimadas por historia (HU-17). Este archivo solo declara el contrato: la lógica del
// servicio (T009) todavía no está implementada.

var (
	// ErrStoryNotFound indica que la historia no existe o pertenece a otro proyecto.
	ErrStoryNotFound = errors.New("la historia no existe")
	// ErrStoryInFinishedSprint indica que la historia está en un Sprint finalizado.
	ErrStoryInFinishedSprint = errors.New("la historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar")
	// ErrEstimatedHoursTooLarge indica un valor mayor que 9999,99. errors.Is lo reconoce también
	// como ErrInvalidEstimatedHours (regla de HU-22).
	ErrEstimatedHoursTooLarge error = estimatedHoursTooLargeError{}

	errEstimateNotImplemented = errors.New("horas estimadas: pendiente de implementar (T009-T010)")
)

type estimatedHoursTooLargeError struct{}

func (estimatedHoursTooLargeError) Error() string {
	return "las horas estimadas no pueden superar 9999,99"
}

func (estimatedHoursTooLargeError) Is(target error) bool { return target == ErrInvalidEstimatedHours }

// EstimateState es la lectura de una historia para decidir el cambio.
type EstimateState struct {
	Hours            *float64 // nil si no tiene horas estimadas
	InFinishedSprint bool
}

// EstimateResult es el resultado de un guardado.
type EstimateResult struct {
	Previous *float64 // nil si no tenía horas: "Horas estimadas registradas"
	Current  float64
}

// EstimateStore es el puerto de almacenamiento de las horas estimadas.
type EstimateStore interface {
	// FindEstimate devuelve el estado de la historia del proyecto o ErrStoryNotFound.
	FindEstimate(ctx context.Context, projectID, storyID int64) (EstimateState, error)
	// UpdateEstimate cambia solo estimated_hours, de forma atómica y excluyendo las historias de un
	// Sprint finalizado; devuelve si se cambió una fila.
	UpdateEstimate(ctx context.Context, projectID, storyID int64, hours float64) (bool, error)
}

// EstimateService registra y modifica las horas estimadas de una historia.
type EstimateService struct {
	store EstimateStore
}

// NewEstimateService crea el servicio con el store indicado.
func NewEstimateService(store EstimateStore) *EstimateService {
	return &EstimateService{store: store}
}

// UpdateEstimatedHours valida el texto con la regla de horas, lee la historia y guarda el valor.
func (s *EstimateService) UpdateEstimatedHours(ctx context.Context, projectID, storyID int64, input string) (EstimateResult, error) {
	return EstimateResult{}, errEstimateNotImplemented
}
