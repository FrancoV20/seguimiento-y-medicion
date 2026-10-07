package metrics

type Service struct {
	store MetricsStore
}

func NewService(store MetricsStore) *Service {
	return &Service{store: store}
}

func (s *Service) CalculateMetrics(sprintID string) (SprintMetrics, error) {
	return SprintMetrics{}, ErrNotImplemented
}
