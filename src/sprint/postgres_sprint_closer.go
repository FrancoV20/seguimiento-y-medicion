package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
)

// PostgreSQLSprintCloser conecta el puerto de cierre a PostgreSQL.
type PostgreSQLSprintCloser struct {
	db *sql.DB
}

var _ SprintCloseStore = (*PostgreSQLSprintCloser)(nil)

// NewPostgreSQLSprintCloser crea un closer sobre una conexión ya abierta.
func NewPostgreSQLSprintCloser(db *sql.DB) *PostgreSQLSprintCloser {
	return &PostgreSQLSprintCloser{db: db}
}

// FindSprint devuelve un Sprint si pertenece al proyecto indicado.
func (s *PostgreSQLSprintCloser) FindSprint(ctx context.Context, projectID, sprintID int64) (Sprint, error) {
	if s == nil || s.db == nil {
		return Sprint{}, fmt.Errorf("%w: PostgreSQL database connection is required", ErrPersistence)
	}

	var found Sprint
	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, status
		FROM sprints
		WHERE project_id = $1 AND id = $2`,
		projectID,
		sprintID,
	).Scan(&found.ID, &found.ProjectID, &found.Number, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return Sprint{}, ErrSprintNotFound
	}
	if err != nil {
		return Sprint{}, fmt.Errorf("%w: find sprint %d for project %d: %w", ErrPersistence, sprintID, projectID, err)
	}
	found.Status = SprintStatus(status)
	return found, nil
}

// CloseSprint cierra el Sprint y devuelve las historias incompletas al Backlog en una transacción.
// Las historias del cierre son las de user_stories.active_sprint_id; check las valida después de
// bloquearlas, con los datos leídos en la misma transacción.
func (s *PostgreSQLSprintCloser) CloseSprint(ctx context.Context, sprintID int64, check ClosureCheck) (CloseResult, error) {
	if s == nil || s.db == nil {
		return CloseResult{}, errors.New("PostgreSQL database connection is required")
	}
	if check == nil {
		return CloseResult{}, errors.New("closure check is required")
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
	var sprintNumber int64
	err = tx.QueryRowContext(ctx,
		`SELECT status, number FROM sprints WHERE id = $1 FOR UPDATE`,
		sprintID,
	).Scan(&status, &sprintNumber)
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

	// FOR UPDATE espera a las transacciones que modifican estas historias y devuelve su versión
	// confirmada: la validación ve el estado con el que se va a cerrar.
	storyRows, err := tx.QueryContext(ctx, `
		SELECT id, status, story_points
		FROM user_stories
		WHERE active_sprint_id = $1
		ORDER BY id
		FOR UPDATE`,
		sprintID,
	)
	if err != nil {
		return CloseResult{}, fmt.Errorf("lock stories for sprint %d: %w", sprintID, err)
	}
	var stories []ClosingStory
	for storyRows.Next() {
		var (
			story       ClosingStory
			status      string
			storyPoints sql.NullInt64
		)
		if err := storyRows.Scan(&story.ID, &status, &storyPoints); err != nil {
			_ = storyRows.Close()
			return CloseResult{}, fmt.Errorf("scan story locked for sprint %d: %w", sprintID, err)
		}
		story.Status = StoryStatus(status)
		if storyPoints.Valid {
			points := int(storyPoints.Int64)
			story.StoryPoints = &points
		}
		stories = append(stories, story)
	}
	if err := storyRows.Err(); err != nil {
		_ = storyRows.Close()
		return CloseResult{}, fmt.Errorf("iterate stories locked for sprint %d: %w", sprintID, err)
	}
	if err := storyRows.Close(); err != nil {
		return CloseResult{}, fmt.Errorf("close story rows for sprint %d: %w", sprintID, err)
	}

	if err := loadClosingStoryHours(ctx, tx, sprintID, stories); err != nil {
		return CloseResult{}, err
	}
	if err := check(ctx, stories); err != nil {
		return CloseResult{}, err
	}

	var completedCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT count(*)
		FROM user_stories
		WHERE active_sprint_id = $1 AND status = $2`,
		sprintID,
		string(Done),
	).Scan(&completedCount); err != nil {
		return CloseResult{}, fmt.Errorf("count completed stories for sprint %d: %w", sprintID, err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		SELECT $1::bigint, id, status, story_points
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
	return CloseResult{
		SprintNumber:   sprintNumber,
		CompletedCount: completedCount,
		ReturnedCount:  int(returnedCount),
	}, nil
}

// loadClosingStoryHours completa las horas registradas de las historias bloqueadas, leídas con la
// misma transacción del cierre.
func loadClosingStoryHours(ctx context.Context, tx *sql.Tx, sprintID int64, stories []ClosingStory) error {
	if len(stories) == 0 {
		return nil
	}
	index := make(map[int64]int, len(stories))
	for i, story := range stories {
		index[story.ID] = i
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT el.story_id, SUM(el.hours)::text
		FROM effort_logs AS el
		JOIN user_stories AS us ON us.id = el.story_id
		WHERE us.active_sprint_id = $1
		GROUP BY el.story_id`,
		sprintID,
	)
	if err != nil {
		return fmt.Errorf("load effort of stories for sprint %d: %w", sprintID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			storyID int64
			total   string
		)
		if err := rows.Scan(&storyID, &total); err != nil {
			return fmt.Errorf("scan effort of story for sprint %d: %w", sprintID, err)
		}
		i, ok := index[storyID]
		if !ok {
			continue
		}
		hours, ok := new(big.Rat).SetString(total)
		if !ok {
			return fmt.Errorf("parse effort of story %d: invalid decimal %q", storyID, total)
		}
		stories[i].ActualHours = hours
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate effort of stories for sprint %d: %w", sprintID, err)
	}
	return nil
}
