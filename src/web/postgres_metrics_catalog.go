package web

import (
	"context"
	"database/sql"
	"errors"
)

// errMetricsNotImplemented marca el esqueleto de HU-25 antes del corte GREEN.
var errMetricsNotImplemented = errors.New("métricas del Sprint: no implementado")

// PostgreSQLMetricsCatalog implementa MetricsCatalog contra PostgreSQL (HU-25, T007). Solo lee.
type PostgreSQLMetricsCatalog struct {
	db *sql.DB
}

var _ MetricsCatalog = (*PostgreSQLMetricsCatalog)(nil)

// NewPostgreSQLMetricsCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLMetricsCatalog(db *sql.DB) *PostgreSQLMetricsCatalog {
	return &PostgreSQLMetricsCatalog{db: db}
}

// FinishedSprints devuelve los Sprints finalizados del proyecto, del más reciente al más antiguo.
func (c *PostgreSQLMetricsCatalog) FinishedSprints(ctx context.Context, projectID int64) ([]FinishedSprint, error) {
	return nil, errMetricsNotImplemented
}

// ActiveSprintNumber devuelve el número del Sprint activo del proyecto, si tiene.
func (c *PostgreSQLMetricsCatalog) ActiveSprintNumber(ctx context.Context, projectID int64) (int64, bool, error) {
	return 0, false, errMetricsNotImplemented
}
