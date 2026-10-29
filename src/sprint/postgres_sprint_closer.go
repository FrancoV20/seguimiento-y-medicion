package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLSprintCloser conecta el puerto de cierre a PostgreSQL.
type PostgreSQLSprintCloser struct {
	db *sql.DB
}

// CloseResult contiene el resultado mínimo del cierre requerido por las pruebas RED.
type CloseResult struct {
	ReturnedCount int
}

var (
	ErrSprintNotFound      = errors.New("sprint not found")
	ErrSprintAlreadyClosed = errors.New("sprint already closed")
)

// NewPostgreSQLSprintCloser crea un closer sobre una conexión ya abierta.
func NewPostgreSQLSprintCloser(db *sql.DB) *PostgreSQLSprintCloser {
	return &PostgreSQLSprintCloser{db: db}
}

// CloseSprint cierra el Sprint y devuelve las historias incompletas al Backlog en una transacción.
func (s *PostgreSQLSprintCloser) CloseSprint(ctx context.Context, sprintID int64) (CloseResult, error) {
	if s == nil || s.db == nil {
		return CloseResult{}, errors.New("PostgreSQL database connection is required")
	}
	if sprintID <= 0 {
		return CloseResult{}, ErrSprintNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CloseResult{}, fmt.Errorf("begin sprint close transaction: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM sprints WHERE id = $1 FOR UPDATE`,
		sprintID,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return CloseResult{}, ErrSprintNotFound
	}
	if err != nil {
		return CloseResult{}, fmt.Errorf("lock sprint %d for close: %w", sprintID, err)
	}
	if status == string(SprintFinished) {
		return CloseResult{}, ErrSprintAlreadyClosed
	}
	if status != string(SprintActive) {
		return CloseResult{}, fmt.Errorf("sprint %d has unsupported status %q", sprintID, status)
	}

	storyRows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM user_stories
		WHERE active_sprint_id = $1
		ORDER BY id
		FOR UPDATE`,
		sprintID,
	)
	if err != nil {
		return CloseResult{}, fmt.Errorf("lock stories for sprint %d: %w", sprintID, err)
	}
	for storyRows.Next() {
		var storyID int64
		if err := storyRows.Scan(&storyID); err != nil {
			_ = storyRows.Close()
			return CloseResult{}, fmt.Errorf("scan story locked for sprint %d: %w", sprintID, err)
		}
	}
	if err := storyRows.Err(); err != nil {
		_ = storyRows.Close()
		return CloseResult{}, fmt.Errorf("iterate stories locked for sprint %d: %w", sprintID, err)
	}
	if err := storyRows.Close(); err != nil {
		return CloseResult{}, fmt.Errorf("close story rows for sprint %d: %w", sprintID, err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		SELECT $1::integer, id, status, story_points
		FROM user_stories
		WHERE active_sprint_id = $2`,
		sprintID,
		sprintID,
	); err != nil {
		return CloseResult{}, fmt.Errorf("record sprint %d story results: %w", sprintID, err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM sprint_story_assignments AS assignments
		USING user_stories AS stories
		WHERE assignments.sprint_id = $1
		  AND assignments.story_id = stories.id
		  AND stories.active_sprint_id = $1
		  AND stories.status <> $2`,
		sprintID,
		string(Done),
	); err != nil {
		return CloseResult{}, fmt.Errorf("remove incomplete story assignments for sprint %d: %w", sprintID, err)
	}

	updateResult, err := tx.ExecContext(ctx, `
		UPDATE user_stories
		SET status = $1, active_sprint_id = NULL
		WHERE active_sprint_id = $2
		  AND status <> $3`,
		string(Pendiente),
		sprintID,
		string(Done),
	)
	if err != nil {
		return CloseResult{}, fmt.Errorf("return incomplete stories from sprint %d: %w", sprintID, err)
	}
	returnedCount, err := updateResult.RowsAffected()
	if err != nil {
		return CloseResult{}, fmt.Errorf("count returned stories from sprint %d: %w", sprintID, err)
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE sprints SET status = $1 WHERE id = $2`,
		string(SprintFinished),
		sprintID,
	)
	if err != nil {
		return CloseResult{}, fmt.Errorf("finish sprint %d: %w", sprintID, err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return CloseResult{}, fmt.Errorf("verify sprint %d was finished: %w", sprintID, err)
	}
	if updated != 1 {
		return CloseResult{}, fmt.Errorf("finish sprint %d: expected one row, updated %d", sprintID, updated)
	}

	if err := tx.Commit(); err != nil {
		return CloseResult{}, fmt.Errorf("commit sprint %d close: %w", sprintID, err)
	}
	return CloseResult{ReturnedCount: int(returnedCount)}, nil
}
