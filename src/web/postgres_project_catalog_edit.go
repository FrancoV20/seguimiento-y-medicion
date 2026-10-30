package web

import (
	"context"
	"database/sql"
	"fmt"
)

// ProjectForEdit lee el nombre, las fechas y los integrantes actuales del proyecto (HU-11).
func (c *PostgreSQLProjectCatalog) ProjectForEdit(ctx context.Context, id int64) (ProjectEdit, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT p.name, p.start_date, p.end_date, pm.member_id
		FROM projects p
		LEFT JOIN project_members pm ON pm.project_id = p.id
		WHERE p.id = $1
		ORDER BY pm.member_id`, id)
	if err != nil {
		return ProjectEdit{}, Unavailable(err)
	}
	defer rows.Close()

	edit := ProjectEdit{ID: id, MemberIDs: []int64{}}
	found := false
	for rows.Next() {
		var member sql.NullInt64
		if err := rows.Scan(&edit.Name, &edit.StartDate, &edit.EndDate, &member); err != nil {
			return ProjectEdit{}, Unavailable(err)
		}
		found = true
		if member.Valid {
			edit.MemberIDs = append(edit.MemberIDs, member.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return ProjectEdit{}, Unavailable(err)
	}
	if !found {
		return ProjectEdit{}, NotFound(fmt.Errorf("el proyecto %d no existe", id))
	}
	return edit, nil
}
