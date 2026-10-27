package web

import (
	"context"
	"database/sql"
)

// PostgreSQLProjectCatalog lee proyectos e integrantes para la pantalla de proyectos.
type PostgreSQLProjectCatalog struct {
	db *sql.DB
}

var _ ProjectCatalog = (*PostgreSQLProjectCatalog)(nil)

// NewPostgreSQLProjectCatalog crea un catálogo respaldado por PostgreSQL.
func NewPostgreSQLProjectCatalog(db *sql.DB) *PostgreSQLProjectCatalog {
	return &PostgreSQLProjectCatalog{db: db}
}

// ListProjects devuelve proyectos con su cantidad de integrantes, los más recientes primero.
func (c *PostgreSQLProjectCatalog) ListProjects(ctx context.Context) ([]ProjectRow, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT p.id, p.name, p.start_date, p.end_date, COUNT(pm.member_id)
		FROM projects p
		LEFT JOIN project_members pm ON pm.project_id = p.id
		GROUP BY p.id
		ORDER BY p.id DESC`)
	if err != nil {
		return nil, Unavailable(err)
	}
	defer rows.Close()

	projects := make([]ProjectRow, 0)
	for rows.Next() {
		var project ProjectRow
		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.StartDate,
			&project.EndDate,
			&project.MemberCount,
		); err != nil {
			return nil, Unavailable(err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, Unavailable(err)
	}
	return projects, nil
}

// ListMembers devuelve todos los integrantes, ordenados por nombre y luego por id.
func (c *PostgreSQLProjectCatalog) ListMembers(ctx context.Context) ([]MemberOption, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT id, display_name FROM members ORDER BY display_name, id`)
	if err != nil {
		return nil, Unavailable(err)
	}
	defer rows.Close()

	members := make([]MemberOption, 0)
	for rows.Next() {
		var member MemberOption
		if err := rows.Scan(&member.ID, &member.DisplayName); err != nil {
			return nil, Unavailable(err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, Unavailable(err)
	}
	return members, nil
}
