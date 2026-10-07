package metrics

import (
	"fmt"
	"math/big"
)

type SprintData struct {
	ID      string
	Status  string
	Stories []CompletedStory
}

type CompletedStory struct {
	ID             string
	Status         string
	StoryPoints    *int
	EstimatedHours *big.Rat
	ActualHours    *big.Rat
}

type SprintMetrics struct {
	SprintID              string
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
	StoryID string
	Field   string
	Message string
	Cause   error
}

func (e CalculationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.StoryID, e.Message, e.Cause)
	}
	if e.StoryID != "" {
		return fmt.Sprintf("%s: %s", e.StoryID, e.Message)
	}
	return e.Message
}

func (e CalculationError) Unwrap() error {
	return e.Cause
}
