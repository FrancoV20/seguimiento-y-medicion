package sprint

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLStoryStatusStore implementa StoryStatusStore mediante PostgreSQL (HU-14, T012).
type PostgreSQLStoryStatusStore struct {
	db *sql.DB
}

var _ StoryStatusStore = (*PostgreSQLStoryStatusStore)(nil)

// NewPostgreSQLStoryStatusStore crea el almacenamiento sobre una conexión ya abierta.
func NewPostgreSQLStoryStatusStore(db *sql.DB) *PostgreSQLStoryStatusStore {
	return &PostgreSQLStoryStatusStore{db: db}
}

const findStoryQuery = `
	SELECT us.status, s.id, (s.status = 'Activo'),
	       EXISTS (SELECT 1 FROM effort_logs el WHERE el.story_id = us.id)
	FROM user_stories us
	LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
	LEFT JOIN sprints s ON s.id = a.sprint_id
	WHERE us.id = $1 AND us.project_id = $2`

// FindStory lee la historia dentro del proyecto: su estado, el Sprint asignado (si lo hay y si
// está activo) y si tiene horas registradas. Una historia inexistente o de otro proyecto
// devuelve ErrStoryNotFound.
func (s *PostgreSQLStoryStatusStore) FindStory(ctx context.Context, projectID, storyID int64) (StoryState, error) {
	var (
		status    string
		sprintID  sql.NullInt64
		active    sql.NullBool
		hasEffort bool
	)
	err := s.db.QueryRowContext(ctx, findStoryQuery, storyID, projectID).Scan(&status, &sprintID, &active, &hasEffort)
	if errors.Is(err, sql.ErrNoRows) {
		return StoryState{}, ErrStoryNotFound
	}
	if err != nil {
		return StoryState{}, err
	}
	return StoryState{
		Status:       StoryStatus(status),
		SprintID:     sprintID.Int64,
		SprintActive: active.Valid && active.Bool,
		HasEffort:    hasEffort,
	}, nil
}

// updateStatusQuery cambia el estado en una sola sentencia atómica: solo si sigue siendo el
// esperado y la historia pertenece a un Sprint activo.
const updateStatusQuery = `
	UPDATE user_stories us
	SET status = $3
	WHERE us.id = $1 AND us.status = $2
	  AND EXISTS (SELECT 1 FROM sprint_story_assignments a
			JOIN sprints s ON s.id = a.sprint_id
	    	WHERE a.story_id = us.id AND s.status = 'Activo')`

// UpdateStatus cambia el estado con comparación atómica. Devuelve si se cambió una fila; si dos
// cambios compiten por la misma historia, solo uno la cambia.
func (s *PostgreSQLStoryStatusStore) UpdateStatus(ctx context.Context, storyID int64, from, to StoryStatus) (bool, error) {
	result, err := s.db.ExecContext(ctx, updateStatusQuery, storyID, string(from), string(to))
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}
