package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLStoryCatalog reads backlog stories from PostgreSQL.
type PostgreSQLStoryCatalog struct {
	db *sql.DB
}

var _ StoryCatalog = (*PostgreSQLStoryCatalog)(nil)

// NewPostgreSQLStoryCatalog creates a catalog backed by an open database pool.
func NewPostgreSQLStoryCatalog(db *sql.DB) *PostgreSQLStoryCatalog {
	return &PostgreSQLStoryCatalog{db: db}
}

// ListStories returns a project's stories ordered by priority (Alta, Media, Baja), then oldest first
// (created_at, id), and marks the ones assigned to a finished Sprint (HU-13). Missing estimates remain nil.
func (c *PostgreSQLStoryCatalog) ListStories(ctx context.Context, projectID int64) ([]StoryRow, error) {
	if c == nil || c.db == nil {
		return nil, Unavailable(errors.New("PostgreSQL story catalog database is required"))
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT us.id, us.title, us.priority, us.story_points, us.estimated_hours, us.status,
		       COALESCE(s.status = 'Finalizado', false) AS in_finished_sprint
		FROM user_stories us
		LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
		LEFT JOIN sprints s ON s.id = a.sprint_id
		WHERE us.project_id = $1
		ORDER BY CASE us.priority WHEN 'Alta' THEN 1 WHEN 'Media' THEN 2 ELSE 3 END,
		         us.created_at, us.id`, projectID)
	if err != nil {
		return nil, Unavailable(fmt.Errorf("list stories for project %d: %w", projectID, err))
	}
	defer rows.Close()

	stories := make([]StoryRow, 0)
	for rows.Next() {
		var (
			story          StoryRow
			storyPoints    sql.NullInt64
			estimatedHours sql.NullFloat64
		)
		if err := rows.Scan(
			&story.ID,
			&story.Title,
			&story.Priority,
			&storyPoints,
			&estimatedHours,
			&story.Status,
			&story.InFinishedSprint,
		); err != nil {
			return nil, Unavailable(fmt.Errorf("scan story for project %d: %w", projectID, err))
		}
		if storyPoints.Valid {
			value := int(storyPoints.Int64)
			story.StoryPoints = &value
		}
		if estimatedHours.Valid {
			story.EstimatedHours = &estimatedHours.Float64
		}
		stories = append(stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, Unavailable(fmt.Errorf("iterate stories for project %d: %w", projectID, err))
	}

	return stories, nil
}
