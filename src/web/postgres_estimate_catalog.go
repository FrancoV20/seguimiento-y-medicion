package web

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLEstimateCatalog implementa EstimateCatalog con PostgreSQL (HU-17). La consulta se
// implementa en T011.
type PostgreSQLEstimateCatalog struct {
	db *sql.DB
}

var _ EstimateCatalog = (*PostgreSQLEstimateCatalog)(nil)

var errEstimateCatalogNotImplemented = errors.New("catálogo de estimaciones: pendiente de implementar (T011)")

// NewPostgreSQLEstimateCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLEstimateCatalog(db *sql.DB) *PostgreSQLEstimateCatalog {
	return &PostgreSQLEstimateCatalog{db: db}
}

// ListEstimates todavía no está implementado (T011).
func (c *PostgreSQLEstimateCatalog) ListEstimates(ctx context.Context, projectID int64) ([]EstimateRow, error) {
	return nil, errEstimateCatalogNotImplemented
}
