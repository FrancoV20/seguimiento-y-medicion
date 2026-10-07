package metrics

import (
	"fmt"
	"math/big"
)

type Service struct {
	store MetricsStore
}

func NewService(store MetricsStore) *Service {
	return &Service{store: store}
}

func (s *Service) CalculateMetrics(sprintID string) (SprintMetrics, error) {
	if s.store == nil {
		return SprintMetrics{}, fmt.Errorf("metrics store is required")
	}

	sprint, err := s.store.LoadSprintData(sprintID)
	if err != nil {
		return SprintMetrics{}, fmt.Errorf("load sprint %q: %w", sprintID, err)
	}
	if sprint.Status != "Finalizado" {
		return SprintMetrics{}, fmt.Errorf("%w: %s", ErrSprintNotFinished, sprint.ID)
	}

	metrics := SprintMetrics{
		SprintID:            sprint.ID,
		EstimatedHoursTotal: new(big.Rat),
		ActualHoursTotal:    new(big.Rat),
		DeviationPercentage: "No disponible",
	}

	for _, story := range sprint.Stories {
		if story.Status != "Terminada" {
			continue
		}

		metrics.CalculationTotalCount++
		if story.StoryPoints == nil {
			return SprintMetrics{}, CalculationError{
				StoryID: story.ID,
				Field:   "storyPoints",
				Message: ErrInvalidStoryPoints.Error(),
				Cause:   ErrInvalidStoryPoints,
			}
		}
		metrics.Velocity += *story.StoryPoints

		if story.EstimatedHours == nil || story.ActualHours == nil {
			continue
		}

		metrics.CalculationUsedCount++
		metrics.EstimatedHoursTotal.Add(metrics.EstimatedHoursTotal, story.EstimatedHours)
		metrics.ActualHoursTotal.Add(metrics.ActualHoursTotal, story.ActualHours)
	}

	deviation, err := CalculateDeviation(metrics.EstimatedHoursTotal, metrics.ActualHoursTotal)
	if err != nil {
		return SprintMetrics{}, fmt.Errorf("calculate effort deviation for sprint %q: %w", sprint.ID, err)
	}
	metrics.DeviationPercentage = deviation

	if err := s.store.SaveMetrics(metrics); err != nil {
		return SprintMetrics{}, fmt.Errorf("%w: %w", ErrMetricsPersistence, err)
	}

	return metrics, nil
}
