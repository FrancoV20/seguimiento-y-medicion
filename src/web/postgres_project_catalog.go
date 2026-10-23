package web

import (
	"context"
	"database/sql"
)

// PostgreSQLProjectCatalog es un stub provisional hasta implementar T007.
type PostgreSQLProjectCatalog struct {
	db *sql.DB
}

var _ ProjectCatalog = (*PostgreSQLProjectCatalog)(nil)

// NewPostgreSQLProjectCatalog crea un catálogo respaldado por PostgreSQL.
func NewPostgreSQLProjectCatalog(db *sql.DB) *PostgreSQLProjectCatalog {
	return &PostgreSQLProjectCatalog{db: db}
}

func (c *PostgreSQLProjectCatalog) ListProjects(context.Context) ([]ProjectRow, error) {
	return nil, nil
}

func (c *PostgreSQLProjectCatalog) ListMembers(context.Context) ([]MemberOption, error) {
	return nil, nil
}
