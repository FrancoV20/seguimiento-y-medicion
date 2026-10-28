package sprint

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLSprintCloser conecta el puerto de cierre a PostgreSQL.
type PostgreSQLSprintCloser struct {
	db *sql.DB
}

// CloseResult contiene el resultado mínimo del cierre requerido por las pruebas RED.
type CloseResult struct {
	ReturnedCount int
}

var ErrSprintAlreadyClosed = errors.New("sprint already closed")

// NewPostgreSQLSprintCloser crea un closer sobre una conexión ya abierta.
func NewPostgreSQLSprintCloser(db *sql.DB) *PostgreSQLSprintCloser {
	return &PostgreSQLSprintCloser{db: db}
}

// CloseSprint todavía no está implementado; esta firma mantiene compilables las pruebas de la fase RED.
func (*PostgreSQLSprintCloser) CloseSprint(context.Context, int64) (CloseResult, error) {
	return CloseResult{}, errors.New("not implemented")
}
