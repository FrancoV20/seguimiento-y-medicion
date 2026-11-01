package report

import (
	"context"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// MetricsSource es la velocidad y la desviación de HU-07; la cumple *metrics.Service.
type MetricsSource interface {
	CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
}

// QualitySource es la calidad y el avance de HU-10; la cumple *metrics.QualityService.
type QualitySource interface {
	SprintQuality(ctx context.Context, projectID, sprintID int64) (metrics.SprintQuality, error)
}

var (
	_ MetricsSource = (*metrics.Service)(nil)
	_ QualitySource = (*metrics.QualityService)(nil)
)
