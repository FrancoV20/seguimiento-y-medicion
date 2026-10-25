package web

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLBoardCatalog implementa BoardCatalog contra PostgreSQL (HU-14, T013).
type PostgreSQLBoardCatalog struct {
	db *sql.DB
}

var _ BoardCatalog = (*PostgreSQLBoardCatalog)(nil)

// NewPostgreSQLBoardCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLBoardCatalog(db *sql.DB) *PostgreSQLBoardCatalog {
	return &PostgreSQLBoardCatalog{db: db}
}

var errBoardNotImplemented = errors.New("HU-14: todavía no implementado")

// ActiveBoard devuelve el Sprint activo del proyecto con sus historias.
// RED (T009): esqueleto sin implementar; se implementa en T013.
func (c *PostgreSQLBoardCatalog) ActiveBoard(ctx context.Context, projectID int64) (BoardView, bool, error) {
	return BoardView{}, false, errBoardNotImplemented
}
