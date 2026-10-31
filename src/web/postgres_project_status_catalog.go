package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLProjectStatusCatalog implementa ProjectStatusCatalog contra PostgreSQL (HU-12).
// Solo lee; no modifica proyectos ni historias.
type PostgreSQLProjectStatusCatalog struct {
	db *sql.DB
}

var _ ProjectStatusCatalog = (*PostgreSQLProjectStatusCatalog)(nil)

// NewPostgreSQLProjectStatusCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLProjectStatusCatalog(db *sql.DB) *PostgreSQLProjectStatusCatalog {
	return &PostgreSQLProjectStatusCatalog{db: db}
}

const projectStatusProjectQuery = `
	SELECT name, start_date, end_date
	FROM projects
	WHERE id = $1`

const projectStatusCountsQuery = `
	SELECT COUNT(*) FILTER (WHERE status = 'Pendiente'),
	       COUNT(*) FILTER (WHERE status = 'En Sprint'),
	       COUNT(*) FILTER (WHERE status = 'En progreso'),
	       COUNT(*) FILTER (WHERE status = 'Hecho')
	FROM user_stories
	WHERE project_id = $1`

// StatusSummary lee el proyecto y sus cantidades por estado en una sola lectura consistente
// (transacción de solo lectura con aislamiento REPEATABLE READ), para que el nombre, las
// fechas y las cantidades salgan del mismo instante. Un proyecto inexistente devuelve NotFound;
// cualquier otra falla se devuelve envuelta, sin datos parciales.
func (c *PostgreSQLProjectStatusCatalog) StatusSummary(ctx context.Context, projectID int64) (ProjectStatusSummary, error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return ProjectStatusSummary{}, fmt.Errorf("project status %d: %w", projectID, err)
	}
	defer tx.Rollback()

	var summary ProjectStatusSummary
	err = tx.QueryRowContext(ctx, projectStatusProjectQuery, projectID).
		Scan(&summary.Name, &summary.StartDate, &summary.EndDate)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectStatusSummary{}, NotFound(fmt.Errorf("el proyecto %d no existe", projectID))
	}
	if err != nil {
		return ProjectStatusSummary{}, fmt.Errorf("project status %d: %w", projectID, err)
	}

	counts := &summary.Counts
	err = tx.QueryRowContext(ctx, projectStatusCountsQuery, projectID).
		Scan(&counts.Pending, &counts.InSprint, &counts.InProgress, &counts.Done)
	if err != nil {
		return ProjectStatusSummary{}, fmt.Errorf("project status %d: %w", projectID, err)
	}

	if err := tx.Commit(); err != nil {
		return ProjectStatusSummary{}, fmt.Errorf("project status %d: %w", projectID, err)
	}
	return summary, nil
}
