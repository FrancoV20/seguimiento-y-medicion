package metrics

import (
	"fmt"
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

	metrics, err := calculateSprintMetrics(sprint)
	if err != nil {
		return SprintMetrics{}, fmt.Errorf("calculate metrics for sprint %q: %w", sprint.ID, err)
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
