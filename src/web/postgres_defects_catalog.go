package web

import (
	"context"
	"database/sql"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
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
func (c *PostgreSQLDefectCatalog) Defects(ctx context.Context, projectID int64) ([]DefectRow, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT d.id, d.description, d.severity, d.status, us.id, us.title,
		        ds.number, rs.number
		 FROM defects d
		 JOIN user_stories us ON us.id = d.story_id
		 JOIN sprints ds ON ds.id = d.detection_sprint_id
		 LEFT JOIN sprints rs ON rs.id = d.resolution_sprint_id
		 WHERE d.project_id = $1
		 ORDER BY d.id DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defects := make([]DefectRow, 0)
	for rows.Next() {
		var row DefectRow
		var severity, status string
		var resolutionSprint sql.NullInt64
		if err := rows.Scan(
			&row.ID,
			&row.Description,
			&severity,
			&status,
			&row.StoryID,
			&row.StoryTitle,
			&row.DetectionSprint,
			&resolutionSprint,
		); err != nil {
			return nil, err
		}
		row.Severity = defect.Severity(severity)
		row.Status = defect.Status(status)
		if resolutionSprint.Valid {
			number := resolutionSprint.Int64
			row.ResolutionSprint = &number
		}
		defects = append(defects, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defects, nil
}

// StoryOptions devuelve las historias elegibles de un proyecto.
func (c *PostgreSQLDefectCatalog) StoryOptions(ctx context.Context, projectID int64) ([]DefectStoryOption, error) {
	rows, err := c.db.QueryContext(ctx,
		`SELECT id, title, status
		 FROM user_stories
		 WHERE project_id = $1
		 ORDER BY id DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := make([]DefectStoryOption, 0)
	for rows.Next() {
		var option DefectStoryOption
		if err := rows.Scan(&option.ID, &option.Title, &option.Status); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return options, nil
}

// ActiveSprintNumber devuelve el número de Sprint activo si existe.
func (c *PostgreSQLDefectCatalog) ActiveSprintNumber(ctx context.Context, projectID int64) (int64, bool, error) {
	var number int64
	err := c.db.QueryRowContext(ctx,
		`SELECT number
		 FROM sprints
		 WHERE project_id = $1 AND status = 'Activo'
		 ORDER BY number DESC
		 LIMIT 1`,
		projectID,
	).Scan(&number)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return number, true, nil
}

var _ DefectCatalog = (*PostgreSQLDefectCatalog)(nil)
