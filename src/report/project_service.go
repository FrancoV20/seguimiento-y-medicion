package report

import (
	"context"
	"errors"
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

// BuildProject arma el reporte del proyecto (HU-09, T044, research.md §18): estado y avance con las
// reglas de HU-12, una fila por Sprint con HU-10 y HU-07 para los finalizados, y la velocidad
// acumulada y el promedio con HU-08. Un Sprint en curso deja el reporte parcial. Devuelve
// ErrProjectNotFound si el proyecto no existe.
func (s *Service) BuildProject(ctx context.Context, projectID int64) (ProjectReport, error) {
	if err := s.ready(); err != nil {
		return ProjectReport{}, err
	}
	if projectID <= 0 {
		return ProjectReport{}, ErrProjectNotFound
	}
	data, err := s.store.LoadProjectReport(ctx, projectID)
	if errors.Is(err, ErrProjectNotFound) {
		return ProjectReport{}, ErrProjectNotFound
	}
	if err != nil {
		return ProjectReport{}, fmt.Errorf("load project report %d: %w", projectID, err)
	}

	now := s.now()
	r := ProjectReport{
		Data:     data,
		Status:   string(project.StatusOn(data.StartDate, data.EndDate, now)),
		Progress: project.ProgressPercentage(data.Stories.Done, data.Stories.Total()),
		Sprints:  make([]ProjectSprintRow, 0, len(data.Sprints)),
	}
	var velocities []int
	for _, sp := range data.Sprints {
		row := ProjectSprintRow{Number: sp.Number}
		if sp.Status != sprintFinished {
			row.Active = true
			r.Partial = true
			r.Sprints = append(r.Sprints, row)
			continue
		}
		q, err := s.quality.SprintQuality(ctx, projectID, sp.ID)
		if err != nil {
			return ProjectReport{}, fmt.Errorf("sprint quality %d: %w", sp.ID, err)
		}
		m, err := s.metrics.CalculateMetrics(sp.ID)
		if err != nil {
			return ProjectReport{}, fmt.Errorf("sprint metrics %d: %w", sp.ID, err)
		}
		row.Quality, row.Metrics = &q, &m
		r.Sprints = append(r.Sprints, row)
		velocities = append(velocities, m.Velocity)
	}
	r.Velocity = metrics.SummarizeVelocity(velocities)
	r.GeneratedAt = now
	return r, nil
}

// ProjectPDF arma el reporte del proyecto y lo convierte en PDF. Ante cualquier error no devuelve
// bytes.
func (s *Service) ProjectPDF(ctx context.Context, projectID int64) ([]byte, error) {
	r, err := s.BuildProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	b, err := RenderProject(r)
	if err != nil {
		return nil, fmt.Errorf("render project report %d: %w", projectID, err)
	}
	return b, nil
}
