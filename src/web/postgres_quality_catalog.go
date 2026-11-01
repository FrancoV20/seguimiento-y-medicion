package web

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgreSQLQualityCatalog implementa QualityCatalog con PostgreSQL (HU-10, T026). Solo lee.
type PostgreSQLQualityCatalog struct {
	db *sql.DB
}

var _ QualityCatalog = (*PostgreSQLQualityCatalog)(nil)

// NewPostgreSQLQualityCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLQualityCatalog(db *sql.DB) *PostgreSQLQualityCatalog {
	return &PostgreSQLQualityCatalog{db: db}
}

// QualitySprints devuelve los Sprints finalizados del proyecto, del número mayor al menor, y el
// número del Sprint activo (como mucho uno, FIX-HU-04). Un error de la base devuelve Unavailable.
func (c *PostgreSQLQualityCatalog) QualitySprints(ctx context.Context, projectID int64) (QualitySprintList, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT id, number, status
		FROM sprints
		WHERE project_id = $1 AND status IN ('Activo', 'Finalizado')
		ORDER BY number DESC`, projectID)
	if err != nil {
		return QualitySprintList{}, Unavailable(fmt.Errorf("consultar los Sprints de calidad: %w", err))
	}
	defer rows.Close()

	list := QualitySprintList{Finished: make([]QualitySprintOption, 0)}
	for rows.Next() {
		var (
			option QualitySprintOption
			status string
		)
		if err := rows.Scan(&option.ID, &option.Number, &status); err != nil {
			return QualitySprintList{}, Unavailable(fmt.Errorf("leer los Sprints de calidad: %w", err))
		}
		if status == "Finalizado" {
			list.Finished = append(list.Finished, option)
		} else {
			list.ActiveNumber = option.Number
		}
	}
	if err := rows.Err(); err != nil {
		return QualitySprintList{}, Unavailable(fmt.Errorf("recorrer los Sprints de calidad: %w", err))
	}
	return list, nil
}
