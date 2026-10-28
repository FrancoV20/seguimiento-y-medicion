package web

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLDefectCatalog consulta la información de la pantalla de defectos.
type PostgreSQLDefectCatalog struct {
	db *sql.DB
}

// NewPostgreSQLDefectCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLDefectCatalog(db *sql.DB) *PostgreSQLDefectCatalog {
	return &PostgreSQLDefectCatalog{db: db}
}

// Defects devuelve los defectos de un proyecto.
func (c *PostgreSQLDefectCatalog) Defects(_ context.Context, _ int64) ([]DefectRow, error) {
	return nil, errors.New("implementación pendiente")
}

// StoryOptions devuelve las historias elegibles de un proyecto.
func (c *PostgreSQLDefectCatalog) StoryOptions(_ context.Context, _ int64) ([]DefectStoryOption, error) {
	return nil, errors.New("implementación pendiente")
}

// ActiveSprintNumber devuelve el número de Sprint activo si existe.
func (c *PostgreSQLDefectCatalog) ActiveSprintNumber(_ context.Context, _ int64) (int64, bool, error) {
	return 0, false, errors.New("implementación pendiente")
}
