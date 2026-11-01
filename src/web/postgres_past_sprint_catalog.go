package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLPastSprintCatalog implementa PastSprintCatalog contra PostgreSQL (HU-16, T008). Solo lee.
type PostgreSQLPastSprintCatalog struct {
	db *sql.DB
}

var _ PastSprintCatalog = (*PostgreSQLPastSprintCatalog)(nil)

// NewPostgreSQLPastSprintCatalog crea el catálogo sobre una conexión abierta.
func NewPostgreSQLPastSprintCatalog(db *sql.DB) *PostgreSQLPastSprintCatalog {
	return &PostgreSQLPastSprintCatalog{db: db}
}

const pastSprintsQuery = `
	SELECT id, number, goal, start_date, end_date
	FROM sprints
	WHERE project_id = $1 AND status = 'Finalizado'
	ORDER BY number DESC`

const pastSprintQuery = `
	SELECT id, number, goal, start_date, end_date
	FROM sprints
	WHERE id = $1 AND project_id = $2 AND status = 'Finalizado'`

// Registro de cierre de HU-15 con el título actual de cada historia, en orden de creación.
const pastSprintStoriesQuery = `
	SELECT r.story_id, us.title, r.story_points, r.final_status
	FROM sprint_story_results r
	JOIN user_stories us ON us.id = r.story_id
	WHERE r.sprint_id = $1
	ORDER BY r.story_id`

// FinishedSprints devuelve los Sprints finalizados del proyecto, del más reciente al más antiguo.
// Sin Sprints finalizados devuelve una lista vacía.
func (c *PostgreSQLPastSprintCatalog) FinishedSprints(ctx context.Context, projectID int64) ([]PastSprintSummary, error) {
	rows, err := c.db.QueryContext(ctx, pastSprintsQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("past sprints of project %d: %w", projectID, err)
	}
	defer rows.Close()

	sprints := []PastSprintSummary{}
	for rows.Next() {
		var s PastSprintSummary
		if err := rows.Scan(&s.ID, &s.Number, &s.Goal, &s.StartDate, &s.EndDate); err != nil {
			return nil, fmt.Errorf("past sprints of project %d: %w", projectID, err)
		}
		sprints = append(sprints, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("past sprints of project %d: %w", projectID, err)
	}
	return sprints, nil
}

// SprintDetail lee el Sprint finalizado del proyecto y su registro de cierre en una sola lectura
// consistente (transacción de solo lectura, REPEATABLE READ). Un Sprint inexistente, de otro
// proyecto o que no está 'Finalizado' devuelve NotFound; cualquier otra falla se devuelve envuelta.
func (c *PostgreSQLPastSprintCatalog) SprintDetail(ctx context.Context, projectID, sprintID int64) (PastSprintDetail, error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return PastSprintDetail{}, fmt.Errorf("past sprint %d: %w", sprintID, err)
	}
	defer tx.Rollback()

	detail := PastSprintDetail{Stories: []PastSprintStory{}}
	s := &detail.PastSprintSummary
	err = tx.QueryRowContext(ctx, pastSprintQuery, sprintID, projectID).Scan(&s.ID, &s.Number, &s.Goal, &s.StartDate, &s.EndDate)
	if errors.Is(err, sql.ErrNoRows) {
		return PastSprintDetail{}, NotFound(fmt.Errorf("past sprint %d: %w", sprintID, sql.ErrNoRows))
	}
	if err != nil {
		return PastSprintDetail{}, fmt.Errorf("past sprint %d: %w", sprintID, err)
	}

	rows, err := tx.QueryContext(ctx, pastSprintStoriesQuery, sprintID)
	if err != nil {
		return PastSprintDetail{}, fmt.Errorf("past sprint %d stories: %w", sprintID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var story PastSprintStory
		var points sql.NullInt64
		if err := rows.Scan(&story.StoryID, &story.Title, &points, &story.FinalStatus); err != nil {
			return PastSprintDetail{}, fmt.Errorf("past sprint %d stories: %w", sprintID, err)
		}
		if points.Valid {
			p := int(points.Int64)
			story.StoryPoints = &p
		}
		detail.Stories = append(detail.Stories, story)
	}
	if err := rows.Err(); err != nil {
		return PastSprintDetail{}, fmt.Errorf("past sprint %d stories: %w", sprintID, err)
	}
	if err := tx.Commit(); err != nil {
		return PastSprintDetail{}, fmt.Errorf("past sprint %d: %w", sprintID, err)
	}
	return detail, nil
}
