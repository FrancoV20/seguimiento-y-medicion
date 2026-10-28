package backlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

// PostgreSQLEstimateStore implementa EstimateStore con PostgreSQL (HU-17, T010). Solo lee y
// actualiza user_stories.estimated_hours.
type PostgreSQLEstimateStore struct {
	db *sql.DB
}

var _ EstimateStore = (*PostgreSQLEstimateStore)(nil)

// NewPostgreSQLEstimateStore crea el adaptador sobre una conexión abierta.
func NewPostgreSQLEstimateStore(db *sql.DB) *PostgreSQLEstimateStore {
	return &PostgreSQLEstimateStore{db: db}
}

// inFinishedSprintCondition es verdadera si la historia está asignada a un Sprint finalizado.
const inFinishedSprintCondition = `EXISTS (
	SELECT 1 FROM sprint_story_assignments a
	JOIN sprints s ON s.id = a.sprint_id
	WHERE a.story_id = us.id AND s.status = 'Finalizado')`

// FindEstimate devuelve las horas estimadas y el bloqueo de la historia del proyecto, o
// ErrStoryNotFound si no existe en ese proyecto.
func (s *PostgreSQLEstimateStore) FindEstimate(ctx context.Context, projectID, storyID int64) (EstimateState, error) {
	var hours sql.NullFloat64
	var state EstimateState
	err := s.db.QueryRowContext(ctx, `
		SELECT us.estimated_hours::float8, `+inFinishedSprintCondition+`
		FROM user_stories us
		WHERE us.id = $1 AND us.project_id = $2`,
		storyID, projectID,
	).Scan(&hours, &state.InFinishedSprint)
	if errors.Is(err, sql.ErrNoRows) {
		return EstimateState{}, ErrStoryNotFound
	}
	if err != nil {
		return EstimateState{}, fmt.Errorf("leer las horas estimadas: %w", err)
	}
	if hours.Valid {
		value := hours.Float64
		state.Hours = &value
	}
	return state, nil
}

// UpdateEstimate guarda las horas en una única sentencia que excluye las historias de un Sprint
// finalizado; devuelve si se cambió una fila. El valor viaja como texto con dos decimales para que
// NUMERIC(6,2) guarde exactamente lo validado.
func (s *PostgreSQLEstimateStore) UpdateEstimate(ctx context.Context, projectID, storyID int64, hours float64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE user_stories us
		SET estimated_hours = $3::numeric
		WHERE us.id = $1 AND us.project_id = $2
		  AND NOT `+inFinishedSprintCondition,
		storyID, projectID, strconv.FormatFloat(hours, 'f', 2, 64),
	)
	if err != nil {
		return false, fmt.Errorf("guardar las horas estimadas: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar el guardado de las horas estimadas: %w", err)
	}
	return affected == 1, nil
}
