package defect

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLStore persiste defectos en PostgreSQL.
type PostgreSQLStore struct {
	db *sql.DB
}

// NewPostgreSQLStore crea un store sobre una conexión abierta.
func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// Create registra un defecto en PostgreSQL.
func (s *PostgreSQLStore) Create(_ context.Context, _ RegisterInput) (Registered, error) {
	return Registered{}, errors.New("implementación pendiente")
}
