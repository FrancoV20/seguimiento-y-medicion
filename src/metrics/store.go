package metrics

type MetricsStore interface {
	LoadSprintData(sprintID int64) (SprintData, error)
	SaveMetrics(metrics SprintMetrics) error
}
