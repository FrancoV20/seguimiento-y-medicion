package metrics

import (
	"errors"
	"strings"
)

type ClosureValidationError struct {
	Reasons []string
}

func (e ClosureValidationError) Error() string {
	return strings.Join(e.Reasons, "\n")
}

var (
	ErrSprintNotFinished   = errors.New("sprint is not finished")
	ErrInvalidStoryPoints  = errors.New("Story Points inválidos")
	ErrNegativeActualHours = errors.New("horas reales negativas")
	ErrEffortUnavailable   = errors.New("estimated effort is unavailable")
	ErrMetricsPersistence  = errors.New("could not persist sprint metrics")
	ErrNotImplemented      = errors.New("metrics operation is not implemented")
)
