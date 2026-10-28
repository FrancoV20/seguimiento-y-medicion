package web

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// PostgreSQLEstimateCatalog implementa EstimateCatalog con PostgreSQL (HU-17, T011). Solo lee.
type PostgreSQLEstimateCatalog struct {
	db *sql.DB
}

var _ EstimateCatalog = (*PostgreSQLEstimateCatalog)(nil)

// NewPostgreSQLEstimateCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLEstimateCatalog(db *sql.DB) *PostgreSQLEstimateCatalog {
	return &PostgreSQLEstimateCatalog{db: db}
}

// ListEstimates devuelve las historias del proyecto por id descendente, con sus horas estimadas,
// la suma de horas reales (0 sin registros) y si están bloqueadas por un Sprint finalizado.
func (c *PostgreSQLEstimateCatalog) ListEstimates(ctx context.Context, projectID int64) ([]EstimateRow, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT us.id, us.title, us.status, us.story_points, us.estimated_hours::float8,
		       COALESCE((SELECT SUM(el.hours) FROM effort_logs el WHERE el.story_id = us.id), 0)::float8,
		       EXISTS (SELECT 1 FROM sprint_story_assignments a
		               JOIN sprints s ON s.id = a.sprint_id
		               WHERE a.story_id = us.id AND s.status = 'Finalizado')
		FROM user_stories us
		WHERE us.project_id = $1
		ORDER BY us.id DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("consultar las estimaciones: %w", err)
	}
	defer rows.Close()

	estimates := make([]EstimateRow, 0)
	for rows.Next() {
		var row EstimateRow
		var status string
		var points sql.NullInt64
		var hours sql.NullFloat64
		if err := rows.Scan(&row.ID, &row.Title, &status, &points, &hours, &row.ActualHours, &row.Locked); err != nil {
			return nil, fmt.Errorf("leer una estimación: %w", err)
		}
		row.Status = backlog.StoryStatus(status)
		if points.Valid {
			value := int(points.Int64)
			row.StoryPoints = &value
		}
		if hours.Valid {
			value := hours.Float64
			row.EstimatedHours = &value
		}
		if row.Locked {
			row.LockedReason = EstimateLockedReason
		}
		estimates = append(estimates, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las estimaciones: %w", err)
	}
	return estimates, nil
}
