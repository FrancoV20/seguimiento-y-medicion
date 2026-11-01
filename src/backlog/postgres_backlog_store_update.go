package backlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// errAssignmentChanged indica que el Sprint de la historia cambió dos veces seguidas mientras se
// guardaba la edición. No es un error de dominio: el servicio lo informa como ErrPersistence.
var errAssignmentChanged = errors.New("el Sprint de la historia cambió durante el guardado")

// storyAssignmentQuery lee el Sprint de la historia del proyecto (NULL si no tiene).
const storyAssignmentQuery = `
	SELECT a.sprint_id
	FROM user_stories us
	LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
	WHERE us.id = $1 AND us.project_id = $2`

// UpdateStory guarda la edición de HU-13 en una sola transacción (data-model.md, "Transacción de
// guardado"): lee el Sprint de la historia, lo bloquea con FOR SHARE (mismo orden que HU-04, HU-23 y
// HU-15: Sprint antes que historias), bloquea la historia y confirma que la asignación no cambió;
// recién entonces decide si está en un Sprint finalizado, actualiza los datos y reemplaza los
// criterios. Si la asignación cambió, repite una sola vez.
func (s *PostgreSQLBacklogStore) UpdateStory(ctx context.Context, projectID int64, story UserStory) error {
	err := s.updateStoryOnce(ctx, projectID, story)
	if errors.Is(err, errAssignmentChanged) {
		err = s.updateStoryOnce(ctx, projectID, story)
	}
	return err
}

func (s *PostgreSQLBacklogStore) updateStoryOnce(ctx context.Context, projectID int64, story UserStory) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("editar la historia %d: %w", story.ID, err)
	}
	defer func() { _ = tx.Rollback() }() // sin efecto después del Commit

	// 1. Sprint de la historia, sin bloquear.
	var sprintID sql.NullInt64
	err = tx.QueryRowContext(ctx, storyAssignmentQuery, story.ID, projectID).Scan(&sprintID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrStoryNotFound
	}
	if err != nil {
		return fmt.Errorf("editar la historia %d: leer su Sprint: %w", story.ID, err)
	}

	// 2. El Sprint primero, con FOR SHARE: un cierre en curso (FOR UPDATE) se espera.
	finished := false
	if sprintID.Valid {
		var status string
		err = tx.QueryRowContext(ctx, `SELECT status FROM sprints WHERE id = $1 FOR SHARE`, sprintID.Int64).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return errAssignmentChanged // el Sprint se borró y con él la asignación
		}
		if err != nil {
			return fmt.Errorf("editar la historia %d: bloquear su Sprint: %w", story.ID, err)
		}
		finished = status == "Finalizado"
	}
	if s.beforeStoryLock != nil {
		s.beforeStoryLock()
	}

	// 3. La historia, y la confirmación de que su Sprint es el mismo.
	var lockedSprintID sql.NullInt64
	err = tx.QueryRowContext(ctx, storyAssignmentQuery+` FOR NO KEY UPDATE OF us`, story.ID, projectID).Scan(&lockedSprintID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrStoryNotFound
	}
	if err != nil {
		return fmt.Errorf("editar la historia %d: bloquearla: %w", story.ID, err)
	}
	if lockedSprintID != sprintID {
		return errAssignmentChanged
	}
	if finished {
		return ErrStoryInFinishedSprint
	}

	// 4. Datos de la historia. El estado, las horas, el Sprint y la fecha de creación no se tocan.
	var storyPoints sql.NullInt64
	if story.StoryPoints != nil {
		storyPoints = sql.NullInt64{Int64: int64(*story.StoryPoints), Valid: true}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE user_stories
		SET title = $2, description = $3, priority = $4, story_points = $5
		WHERE id = $1`,
		story.ID, story.Title, story.Description, string(story.Priority), storyPoints,
	); err != nil {
		return fmt.Errorf("editar la historia %d: guardar sus datos: %w", story.ID, err)
	}

	// 5. Criterios completos, en el orden recibido, posiciones 1..n.
	if _, err := tx.ExecContext(ctx, `DELETE FROM acceptance_criteria WHERE story_id = $1`, story.ID); err != nil {
		return fmt.Errorf("editar la historia %d: borrar sus criterios: %w", story.ID, err)
	}
	for i, c := range story.AcceptanceCriteria {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, $2, $3)`,
			story.ID, i+1, c.Content,
		); err != nil {
			return fmt.Errorf("editar la historia %d: guardar sus criterios: %w", story.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("editar la historia %d: confirmar: %w", story.ID, err)
	}
	return nil
}
