package backlog

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Horas estimadas por historia (HU-17, T009): interpretación del texto, servicio y puerto de
// almacenamiento. La regla numérica es la de HU-22 (validEstimatedHours, en service.go).

var (
	// ErrStoryNotFound indica que la historia no existe o pertenece a otro proyecto.
	ErrStoryNotFound = errors.New("la historia no existe")
	// ErrStoryInFinishedSprint indica que la historia está en un Sprint finalizado.
	ErrStoryInFinishedSprint = errors.New("la historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar")
	// ErrEstimatedHoursTooLarge indica un valor mayor que 9999,99. errors.Is lo reconoce también
	// como ErrInvalidEstimatedHours (regla de HU-22).
	ErrEstimatedHoursTooLarge error = estimatedHoursTooLargeError{}
)

// estimatedHoursPattern admite solo dígitos con una coma o un punto decimal: sin signo, exponente,
// hexadecimal, espacios internos ni separador de miles.
var estimatedHoursPattern = regexp.MustCompile(`^[0-9]+(?:[.,]([0-9]+))?$`)

type estimatedHoursTooLargeError struct{}

func (estimatedHoursTooLargeError) Error() string {
	return "las horas estimadas no pueden superar 9999,99"
}

func (estimatedHoursTooLargeError) Is(target error) bool { return target == ErrInvalidEstimatedHours }

// ParseEstimatedHours interpreta el texto de horas estimadas con la regla de HU-22: mayor que 0,
// hasta dos decimales, coma o punto y máximo 9999,99, sin redondear. Más de dos decimales es
// inválido aunque el valor supere el máximo; un valor por encima del máximo devuelve
// ErrEstimatedHoursTooLarge.
func ParseEstimatedHours(text string) (float64, error) {
	trimmed := strings.TrimSpace(text)
	match := estimatedHoursPattern.FindStringSubmatch(trimmed)
	if match == nil || len(match[1]) > 2 {
		return 0, ErrInvalidEstimatedHours
	}
	value, err := strconv.ParseFloat(strings.Replace(trimmed, ",", ".", 1), 64)
	if err != nil {
		return 0, ErrInvalidEstimatedHours
	}
	if value > maxEstimatedHours {
		return 0, ErrEstimatedHoursTooLarge
	}
	if !validEstimatedHours(&value) {
		return 0, ErrInvalidEstimatedHours
	}
	return value, nil
}

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

// UpdateEstimatedHours valida el texto con la regla de horas, lee la historia y guarda el valor de
// forma atómica. Devuelve los errores de la regla de horas, ErrStoryNotFound,
// ErrStoryInFinishedSprint o, ante cualquier otra falla, el error de almacenamiento.
func (s *EstimateService) UpdateEstimatedHours(ctx context.Context, projectID, storyID int64, input string) (EstimateResult, error) {
	hours, err := ParseEstimatedHours(input)
	if err != nil {
		return EstimateResult{}, err
	}
	if projectID <= 0 || storyID <= 0 {
		return EstimateResult{}, ErrStoryNotFound
	}
	state, err := s.store.FindEstimate(ctx, projectID, storyID)
	if err != nil {
		return EstimateResult{}, err
	}
	if state.InFinishedSprint {
		return EstimateResult{}, ErrStoryInFinishedSprint
	}
	changed, err := s.store.UpdateEstimate(ctx, projectID, storyID, hours)
	if err != nil {
		return EstimateResult{}, err
	}
	if !changed {
		// La historia cambió entre la lectura y el guardado: se relee para informar el motivo.
		current, err := s.store.FindEstimate(ctx, projectID, storyID)
		if err != nil {
			return EstimateResult{}, err
		}
		if current.InFinishedSprint {
			return EstimateResult{}, ErrStoryInFinishedSprint
		}
		return EstimateResult{}, ErrStoryNotFound
	}
	return EstimateResult{Previous: state.Hours, Current: hours}, nil
}
