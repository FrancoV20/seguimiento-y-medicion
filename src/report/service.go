package report

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Service arma los reportes (HU-09, research.md §10). No guarda nada: la única escritura es la de
// HU-07 en sprint_metrics.
type Service struct {
	store   Store
	metrics MetricsSource
	quality QualitySource
	now     func() time.Time
}

// NewService crea el servicio. Con now nil usa time.Now.
func NewService(store Store, metricsSrc MetricsSource, quality QualitySource, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, metrics: metricsSrc, quality: quality, now: now}
}

func (s *Service) ready() error {
	if s == nil || s.store == nil || s.metrics == nil || s.quality == nil {
		return errors.New("report: el servicio necesita el almacén y las fuentes de HU-07 y HU-10")
	}
	return nil
}

// Build arma el reporte del Sprint del proyecto. Un Sprint finalizado trae las métricas de HU-07 y
// HU-10; uno en curso queda parcial y no las pide. Devuelve ErrSprintNotFound si el Sprint no existe
// en el proyecto y ErrInconsistentReport si sus historias o defectos no coinciden con HU-10.
func (s *Service) Build(ctx context.Context, projectID, sprintID int64) (SprintReport, error) {
	if err := s.ready(); err != nil {
		return SprintReport{}, err
	}
	if projectID <= 0 || sprintID <= 0 {
		return SprintReport{}, ErrSprintNotFound
	}
	data, err := s.store.LoadSprintReport(ctx, projectID, sprintID)
	if errors.Is(err, ErrSprintNotFound) {
		return SprintReport{}, ErrSprintNotFound
	}
	if err != nil {
		return SprintReport{}, fmt.Errorf("load sprint report %d: %w", sprintID, err)
	}

	r := SprintReport{Data: data}
	if data.Sprint.Status == sprintFinished {
		m, err := s.metrics.CalculateMetrics(sprintID)
		if err != nil {
			return SprintReport{}, fmt.Errorf("sprint metrics %d: %w", sprintID, err)
		}
		q, err := s.quality.SprintQuality(ctx, projectID, sprintID)
		if err != nil {
			return SprintReport{}, fmt.Errorf("sprint quality %d: %w", sprintID, err)
		}
		if len(data.Stories) != q.StoriesPlanned || len(data.Defects) != q.DefectsDetected {
			return SprintReport{}, fmt.Errorf("%w: sprint %d", ErrInconsistentReport, sprintID)
		}
		r.Metrics, r.Quality = &m, &q
	} else {
		r.Partial = true
	}
	r.GeneratedAt = s.now()
	return r, nil
}

// SprintPDF arma el reporte y lo convierte en PDF. Ante cualquier error no devuelve bytes.
func (s *Service) SprintPDF(ctx context.Context, projectID, sprintID int64) ([]byte, error) {
	r, err := s.Build(ctx, projectID, sprintID)
	if err != nil {
		return nil, err
	}
	b, err := Render(r)
	if err != nil {
		return nil, fmt.Errorf("render sprint report %d: %w", sprintID, err)
	}
	return b, nil
}
