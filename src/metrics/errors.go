package metrics

import "errors"

var (
	ErrSprintNotFinished   = errors.New("sprint is not finished")
	ErrInvalidStoryPoints  = errors.New("invalid story points")
	ErrNegativeActualHours = errors.New("actual hours cannot be negative")
	ErrEffortUnavailable   = errors.New("estimated effort is unavailable")
	ErrMetricsPersistence  = errors.New("could not persist sprint metrics")
	ErrNotImplemented      = errors.New("metrics operation is not implemented")
)
