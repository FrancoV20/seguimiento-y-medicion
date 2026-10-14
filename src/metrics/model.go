package metrics

import (
	"fmt"
	"math/big"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

type SprintData struct {
	ID      int64
	Status  string
	Stories []CompletedStory
}

type CompletedStory struct {
	ID             int64
	Status         backlog.StoryStatus
	StoryPoints    *int
	EstimatedHours *big.Rat
	ActualHours    *big.Rat
}

type SprintMetrics struct {
	SprintID              int64
	Velocity              int
	EstimatedHoursTotal   *big.Rat
	ActualHoursTotal      *big.Rat
	DeviationPercentage   string
	CalculationUsedCount  int
	CalculationTotalCount int
	IsPartial             bool
	Warning               string
}

type CalculationError struct {
	StoryID int64
	Field   string
	Message string
	Cause   error
}

func (e CalculationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%d: %s: %v", e.StoryID, e.Message, e.Cause)
	}
	if e.StoryID != 0 {
		return fmt.Sprintf("%d: %s", e.StoryID, e.Message)
	}
	return e.Message
}

func (e CalculationError) Unwrap() error {
	return e.Cause
}
