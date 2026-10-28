package backlog

import (
	"context"
	"database/sql"
)

// PostgreSQLEstimateStore implementa EstimateStore con PostgreSQL (HU-17). Las sentencias se
// implementan en T010.
type PostgreSQLEstimateStore struct {
	db *sql.DB
}

var _ EstimateStore = (*PostgreSQLEstimateStore)(nil)

// NewPostgreSQLEstimateStore crea el adaptador sobre una conexión abierta.
func NewPostgreSQLEstimateStore(db *sql.DB) *PostgreSQLEstimateStore {
	return &PostgreSQLEstimateStore{db: db}
}

// FindEstimate todavía no está implementado (T010).
func (s *PostgreSQLEstimateStore) FindEstimate(ctx context.Context, projectID, storyID int64) (EstimateState, error) {
	return EstimateState{}, errEstimateNotImplemented
}

// UpdateEstimate todavía no está implementado (T010).
func (s *PostgreSQLEstimateStore) UpdateEstimate(ctx context.Context, projectID, storyID int64, hours float64) (bool, error) {
	return false, errEstimateNotImplemented
}
