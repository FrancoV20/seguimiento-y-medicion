package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLReportCatalog implementa ReportCatalog contra PostgreSQL (HU-09, T032). Solo lee.
type PostgreSQLReportCatalog struct {
	db *sql.DB
}

var _ ReportCatalog = (*PostgreSQLReportCatalog)(nil)

// NewPostgreSQLReportCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLReportCatalog(db *sql.DB) *PostgreSQLReportCatalog {
	return &PostgreSQLReportCatalog{db: db}
}

const reportSprintsQuery = `
	SELECT id, number, goal, status = 'Activo'
	FROM sprints
	WHERE project_id = $1
	ORDER BY (status = 'Activo') DESC, number DESC`

const reportSprintQuery = `
	SELECT id, number, goal, status = 'Activo'
	FROM sprints
	WHERE id = $1 AND project_id = $2`

// ReportSprints devuelve los Sprints del proyecto, el activo primero.
func (c *PostgreSQLReportCatalog) ReportSprints(ctx context.Context, projectID int64) ([]ReportSprintOption, error) {
	rows, err := c.db.QueryContext(ctx, reportSprintsQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("report sprints of project %d: %w", projectID, err)
	}
	defer rows.Close()

	sprints := []ReportSprintOption{}
	for rows.Next() {
		var s ReportSprintOption
		if err := rows.Scan(&s.ID, &s.Number, &s.Goal, &s.Active); err != nil {
			return nil, fmt.Errorf("report sprints of project %d: %w", projectID, err)
		}
		sprints = append(sprints, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("report sprints of project %d: %w", projectID, err)
	}
	return sprints, nil
}

// ReportSprint devuelve el Sprint del proyecto; found = false si no existe o es de otro proyecto.
func (c *PostgreSQLReportCatalog) ReportSprint(ctx context.Context, projectID, sprintID int64) (ReportSprintOption, bool, error) {
	var s ReportSprintOption
	err := c.db.QueryRowContext(ctx, reportSprintQuery, sprintID, projectID).Scan(&s.ID, &s.Number, &s.Goal, &s.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return ReportSprintOption{}, false, nil
	}
	if err != nil {
		return ReportSprintOption{}, false, fmt.Errorf("report sprint %d: %w", sprintID, err)
	}
	return s, true, nil
}
