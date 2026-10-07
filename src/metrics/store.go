package metrics

type MetricsStore interface {
	LoadSprintData(sprintID string) (SprintData, error)
	SaveMetrics(metrics SprintMetrics) error
}
