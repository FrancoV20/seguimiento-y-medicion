package web

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgreSQLDashboardCatalog implementa DashboardCatalog contra PostgreSQL (HU-08). Solo lee.
type PostgreSQLDashboardCatalog struct {
	db *sql.DB
}

var _ DashboardCatalog = (*PostgreSQLDashboardCatalog)(nil)

// NewPostgreSQLDashboardCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLDashboardCatalog(db *sql.DB) *PostgreSQLDashboardCatalog {
	return &PostgreSQLDashboardCatalog{db: db}
}

const dashboardStoryCountsQuery = `
	SELECT COUNT(*) FILTER (WHERE status = 'Pendiente'),
	       COUNT(*) FILTER (WHERE status = 'En Sprint'),
	       COUNT(*) FILTER (WHERE status = 'En progreso'),
	       COUNT(*) FILTER (WHERE status = 'Hecho')
	FROM user_stories
	WHERE project_id = $1`

const dashboardOpenDefectsQuery = `
	SELECT COUNT(*) FROM defects WHERE project_id = $1 AND status = 'Abierto'`

const dashboardFinishedSprintsQuery = `
	SELECT id, number
	FROM sprints
	WHERE project_id = $1 AND status = 'Finalizado'
	ORDER BY number`

// DashboardSummary lee las historias por estado, los defectos abiertos y los Sprints finalizados del
// proyecto en una transacción de solo lectura con REPEATABLE READ, para que los tres datos sean
// coherentes entre sí. Ante cualquier error no devuelve datos parciales.
func (c *PostgreSQLDashboardCatalog) DashboardSummary(ctx context.Context, projectID int64) (DashboardSummary, error) {
	summary, err := c.readSummary(ctx, projectID)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("dashboard summary %d: %w", projectID, err)
	}
	return summary, nil
}

func (c *PostgreSQLDashboardCatalog) readSummary(ctx context.Context, projectID int64) (DashboardSummary, error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return DashboardSummary{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var summary DashboardSummary
	if err := tx.QueryRowContext(ctx, dashboardStoryCountsQuery, projectID).Scan(
		&summary.Stories.Pending,
		&summary.Stories.InSprint,
		&summary.Stories.InProgress,
		&summary.Stories.Done,
	); err != nil {
		return DashboardSummary{}, err
	}
	if err := tx.QueryRowContext(ctx, dashboardOpenDefectsQuery, projectID).Scan(&summary.OpenDefects); err != nil {
		return DashboardSummary{}, err
	}

	rows, err := tx.QueryContext(ctx, dashboardFinishedSprintsQuery, projectID)
	if err != nil {
		return DashboardSummary{}, err
	}
	defer rows.Close()
	summary.Finished = make([]DashboardSprint, 0)
	for rows.Next() {
		var sprint DashboardSprint
		if err := rows.Scan(&sprint.ID, &sprint.Number); err != nil {
			return DashboardSummary{}, err
		}
		summary.Finished = append(summary.Finished, sprint)
	}
	if err := rows.Err(); err != nil {
		return DashboardSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return DashboardSummary{}, err
	}
	return summary, nil
}
