package web

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLSchemaVersionReader lee la tabla schema_migrations de golang-migrate (HU-20a, T020).
type PostgreSQLSchemaVersionReader struct {
	db *sql.DB
}

// NewPostgreSQLSchemaVersionReader crea el lector sobre una conexión abierta.
func NewPostgreSQLSchemaVersionReader(db *sql.DB) *PostgreSQLSchemaVersionReader {
	return &PostgreSQLSchemaVersionReader{db: db}
}

// ReadSchemaStatus devuelve la versión y el estado dirty. Si la tabla no existe
// o está vacía, no hay migraciones aplicadas (versión 0).
func (r *PostgreSQLSchemaVersionReader) ReadSchemaStatus(ctx context.Context) (SchemaStatus, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return SchemaStatus{}, err
	}
	if !exists {
		return SchemaStatus{}, nil
	}
	var version int64
	var dirty bool
	err := r.db.QueryRowContext(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return SchemaStatus{}, nil
	}
	if err != nil {
		return SchemaStatus{}, err
	}
	if version < 0 {
		version = 0
	}
	return SchemaStatus{Version: uint(version), Dirty: dirty}, nil
}
