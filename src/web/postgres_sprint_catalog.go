package web

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLSprintCatalog consulta el Sprint activo y las historias de un proyecto.
type PostgreSQLSprintCatalog struct {
	db *sql.DB
}

func NewPostgreSQLSprintCatalog(db *sql.DB) *PostgreSQLSprintCatalog {
	return &PostgreSQLSprintCatalog{db: db}
}

var _ SprintCatalog = (*PostgreSQLSprintCatalog)(nil)

func (c *PostgreSQLSprintCatalog) ActiveSprint(ctx context.Context, projectID int64) (*SprintView, error) {
	var view SprintView
	err := c.db.QueryRowContext(ctx, `
		SELECT id, number, goal, start_date, end_date
		FROM sprints
		WHERE project_id = $1 AND status = 'Activo'
		ORDER BY number DESC
		LIMIT 1`, projectID).Scan(
		&view.ID,
		&view.Number,
		&view.Goal,
		&view.StartDate,
		&view.EndDate,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consultar el Sprint activo: %w", err)
	}
	view.Label = fmt.Sprintf("Sprint %d", view.Number)
	view.Stories = make([]SprintStoryRow, 0)

	rows, err := c.db.QueryContext(ctx, `
		SELECT us.id, us.title, us.story_points, us.status,
		       EXISTS (
		           SELECT 1 FROM effort_logs el WHERE el.story_id = us.id
		       ) AS has_effort
		FROM sprint_story_assignments a
		JOIN user_stories us ON us.id = a.story_id
		WHERE a.sprint_id = $1
		ORDER BY us.id`, view.ID)
	if err != nil {
		return nil, fmt.Errorf("consultar las historias del Sprint: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var story SprintStoryRow
		var storyPoints sql.NullInt64
		var status string
		if err := rows.Scan(&story.ID, &story.Title, &storyPoints, &status, &story.HasEffort); err != nil {
			return nil, fmt.Errorf("leer una historia del Sprint: %w", err)
		}
		story.Status = sprint.StoryStatus(status)
		story.CanRemove, story.RemoveBlockedReason = sprint.CanRemoveStory(story.Status, story.HasEffort)
		if storyPoints.Valid {
			points := int(storyPoints.Int64)
			story.StoryPoints = &points
		}
		view.Stories = append(view.Stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las historias del Sprint: %w", err)
	}
	return &view, nil
}

func (c *PostgreSQLSprintCatalog) ListStoryOptions(ctx context.Context, projectID int64) ([]StoryOption, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT us.id, us.title, us.priority, us.story_points, us.status, s.number
		FROM user_stories us
		LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
		LEFT JOIN sprints s ON s.id = a.sprint_id
		WHERE us.project_id = $1
		ORDER BY us.id DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("consultar las historias del proyecto: %w", err)
	}
	defer rows.Close()

	options := make([]StoryOption, 0)
	for rows.Next() {
		var option StoryOption
		var storyPoints, assignedSprintNumber sql.NullInt64
		var status string
		if err := rows.Scan(
			&option.ID,
			&option.Title,
			&option.Priority,
			&storyPoints,
			&status,
			&assignedSprintNumber,
		); err != nil {
			return nil, fmt.Errorf("leer una opción de historia: %w", err)
		}
		option.Status = sprint.StoryStatus(status)
		if storyPoints.Valid {
			points := int(storyPoints.Int64)
			option.StoryPoints = &points
		}
		if assignedSprintNumber.Valid {
			option.AssignedSprintNumber = assignedSprintNumber.Int64
			option.UnavailableReason = fmt.Sprintf("Ya está asignada al Sprint %d", assignedSprintNumber.Int64)
		} else if option.Status != sprint.Pendiente {
			option.UnavailableReason = "No está pendiente"
		}
		option.Available = option.Status == sprint.Pendiente && !assignedSprintNumber.Valid
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las historias del proyecto: %w", err)
	}
	return options, nil
}
