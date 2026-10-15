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

type metricsStoreError struct {
	operation string
	cause     error
}

func (e metricsStoreError) Error() string {
	if e.operation == storeOperationLoad {
		return ErrSprintLoad.Error()
	}
	return ErrMetricsPersistence.Error()
}

func (e metricsStoreError) Unwrap() error {
	return e.cause
}

func (e metricsStoreError) Is(target error) bool {
	if e.operation == storeOperationLoad {
		return target == ErrSprintLoad
	}
	return target == ErrMetricsPersistence
}

const (
	storeOperationLoad = "load"
	storeOperationSave = "save"
)

var (
	ErrSprintLoad          = errors.New("could not load sprint data")
	ErrSprintNotFinished   = errors.New("sprint is not finished")
	ErrInvalidStoryPoints  = errors.New("Story Points inválidos")
	ErrNegativeActualHours = errors.New("horas reales negativas")
	ErrEffortUnavailable   = errors.New("estimated effort is unavailable")
	ErrMetricsPersistence  = errors.New("could not persist sprint metrics")
	ErrNotImplemented      = errors.New("metrics operation is not implemented")
)
