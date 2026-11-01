package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// StoryForEdit lee la historia del proyecto con sus criterios en orden y si pertenece a un Sprint
// finalizado (HU-13). Sin filas devuelve NotFound; una falla de la base, Unavailable.
func (c *PostgreSQLStoryCatalog) StoryForEdit(ctx context.Context, projectID, storyID int64) (StoryEdit, error) {
	if c == nil || c.db == nil {
		return StoryEdit{}, Unavailable(errors.New("PostgreSQL story catalog database is required"))
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT us.title, us.description, us.priority, us.story_points,
		       COALESCE(s.status = 'Finalizado', false), ac.content
		FROM user_stories us
		LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
		LEFT JOIN sprints s ON s.id = a.sprint_id
		LEFT JOIN acceptance_criteria ac ON ac.story_id = us.id
		WHERE us.id = $1 AND us.project_id = $2
		ORDER BY ac.position`, storyID, projectID)
	if err != nil {
		return StoryEdit{}, Unavailable(fmt.Errorf("read story %d for edit: %w", storyID, err))
	}
	defer rows.Close()

	story := StoryEdit{ID: storyID, Criteria: make([]string, 0)}
	found := false
	for rows.Next() {
		var (
			storyPoints sql.NullInt64
			criterion   sql.NullString
		)
		if err := rows.Scan(&story.Title, &story.Description, &story.Priority, &storyPoints,
			&story.InFinishedSprint, &criterion); err != nil {
			return StoryEdit{}, Unavailable(fmt.Errorf("scan story %d for edit: %w", storyID, err))
		}
		found = true
		if storyPoints.Valid {
			value := int(storyPoints.Int64)
			story.StoryPoints = &value
		}
		if criterion.Valid {
			story.Criteria = append(story.Criteria, criterion.String)
		}
	}
	if err := rows.Err(); err != nil {
		return StoryEdit{}, Unavailable(fmt.Errorf("iterate story %d for edit: %w", storyID, err))
	}
	if !found {
		return StoryEdit{}, NotFound(fmt.Errorf("story %d for edit: %w", storyID, sql.ErrNoRows))
	}
	return story, nil
}
