package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// PostgreSQLBoardCatalog implementa BoardCatalog contra PostgreSQL (HU-14, T013).
// Solo lee; no modifica Sprints ni historias.
type PostgreSQLBoardCatalog struct {
	db *sql.DB
}

var _ BoardCatalog = (*PostgreSQLBoardCatalog)(nil)

// NewPostgreSQLBoardCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLBoardCatalog(db *sql.DB) *PostgreSQLBoardCatalog {
	return &PostgreSQLBoardCatalog{db: db}
}

const activeSprintQuery = `
	SELECT id, number, goal, start_date, end_date
	FROM sprints
	WHERE project_id = $1 AND status = 'Activo'
	ORDER BY number DESC
	LIMIT 1`

const sprintStoriesQuery = `
	SELECT us.id, us.title, us.story_points, us.status,
	       EXISTS (SELECT 1 FROM effort_logs el WHERE el.story_id = us.id)
	FROM sprint_story_assignments a
	JOIN user_stories us ON us.id = a.story_id
	WHERE a.sprint_id = $1
	ORDER BY us.id`

// ActiveBoard devuelve el Sprint activo del proyecto (el de mayor número, si por datos anteriores
// a FIX-HU-04 hubiera más de uno) con sus historias agrupadas por estado. Que no haya Sprint
// activo no es un error: devuelve found=false. Solo una falla de lectura devuelve error.
func (c *PostgreSQLBoardCatalog) ActiveBoard(ctx context.Context, projectID int64) (BoardView, bool, error) {
	var view BoardView
	err := c.db.QueryRowContext(ctx, activeSprintQuery, projectID).
		Scan(&view.ID, &view.Number, &view.Goal, &view.StartDate, &view.EndDate)
	if errors.Is(err, sql.ErrNoRows) {
		return BoardView{}, false, nil
	}
	if err != nil {
		return BoardView{}, false, err
	}
	view.Label = fmt.Sprintf("Sprint %d", view.Number)

	rows, err := c.db.QueryContext(ctx, sprintStoriesQuery, view.ID)
	if err != nil {
		return BoardView{}, false, err
	}
	defer rows.Close()

	view.Groups = []BoardGroup{
		{Status: sprint.EnSprint},
		{Status: sprint.InProgress},
		{Status: sprint.Done},
	}
	groupOf := map[sprint.StoryStatus]int{sprint.EnSprint: 0, sprint.InProgress: 1, sprint.Done: 2}
	var all []BoardStory
	for rows.Next() {
		var (
			story  BoardStory
			points sql.NullInt64
			status string
		)
		if err := rows.Scan(&story.ID, &story.Title, &points, &status, &story.HasEffort); err != nil {
			return BoardView{}, false, err
		}
		story.Status = sprint.StoryStatus(status)
		if points.Valid {
			value := int(points.Int64)
			story.StoryPoints = &value
		}
		if action, ok := sprint.NextAction(story.Status); ok {
			story.Action = action
			story.ActionLabel = actionLabels[action]
		}
		all = append(all, story)
		if index, ok := groupOf[story.Status]; ok {
			view.Groups[index].Stories = append(view.Groups[index].Stories, story)
		}
	}
	if err := rows.Err(); err != nil {
		return BoardView{}, false, err
	}
	view.PlannedPoints, view.CompletedPoints = BoardPoints(all)
	return view, true, nil
}
