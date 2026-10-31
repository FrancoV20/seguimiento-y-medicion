package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLMetricsCatalog implementa MetricsCatalog contra PostgreSQL (HU-25, T007). Solo lee.
type PostgreSQLMetricsCatalog struct {
	db *sql.DB
}

var _ MetricsCatalog = (*PostgreSQLMetricsCatalog)(nil)

// NewPostgreSQLMetricsCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLMetricsCatalog(db *sql.DB) *PostgreSQLMetricsCatalog {
	return &PostgreSQLMetricsCatalog{db: db}
}

const metricsFinishedSprintsQuery = `
	SELECT id, number, goal, start_date, end_date
	FROM sprints
	WHERE project_id = $1 AND status = 'Finalizado'
	ORDER BY number DESC`

const metricsActiveSprintQuery = `
	SELECT number
	FROM sprints
	WHERE project_id = $1 AND status = 'Activo'
	ORDER BY number DESC
	LIMIT 1`

// FinishedSprints devuelve los Sprints finalizados del proyecto, del más reciente al más antiguo.
// Sin Sprints finalizados devuelve una lista vacía.
func (c *PostgreSQLMetricsCatalog) FinishedSprints(ctx context.Context, projectID int64) ([]FinishedSprint, error) {
	rows, err := c.db.QueryContext(ctx, metricsFinishedSprintsQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("finished sprints of project %d: %w", projectID, err)
	}
	defer rows.Close()

	sprints := []FinishedSprint{}
	for rows.Next() {
		var s FinishedSprint
		if err := rows.Scan(&s.ID, &s.Number, &s.Goal, &s.StartDate, &s.EndDate); err != nil {
			return nil, fmt.Errorf("scan finished sprint of project %d: %w", projectID, err)
		}
		sprints = append(sprints, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finished sprints of project %d: %w", projectID, err)
	}
	return sprints, nil
}

// ActiveSprintNumber devuelve el número del Sprint activo del proyecto; found es false si no tiene.
func (c *PostgreSQLMetricsCatalog) ActiveSprintNumber(ctx context.Context, projectID int64) (int64, bool, error) {
	var number int64
	err := c.db.QueryRowContext(ctx, metricsActiveSprintQuery, projectID).Scan(&number)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("active sprint of project %d: %w", projectID, err)
	}
	return number, true, nil
}
