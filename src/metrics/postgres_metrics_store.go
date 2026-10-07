package metrics

import "database/sql"

type PostgreSQLMetricsStore struct {
	db *sql.DB
}

func NewPostgreSQLMetricsStore(db *sql.DB) *PostgreSQLMetricsStore {
	return &PostgreSQLMetricsStore{db: db}
}

func (s *PostgreSQLMetricsStore) LoadSprintData(sprintID string) (SprintData, error) {
	return SprintData{}, ErrNotImplemented
}

func (s *PostgreSQLMetricsStore) SaveMetrics(metrics SprintMetrics) error {
	return ErrNotImplemented
}
