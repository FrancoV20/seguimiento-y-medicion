package metrics

type TestMetricsStore struct {
	SprintData   SprintData
	LoadError    error
	SaveError    error
	SavedMetrics []SprintMetrics
}

func (s *TestMetricsStore) LoadSprintData(sprintID string) (SprintData, error) {
	if s.LoadError != nil {
		return SprintData{}, s.LoadError
	}
	return s.SprintData, nil
}

func (s *TestMetricsStore) SaveMetrics(metrics SprintMetrics) error {
	if s.SaveError != nil {
		return s.SaveError
	}
	s.SavedMetrics = append(s.SavedMetrics, metrics)
	return nil
}
