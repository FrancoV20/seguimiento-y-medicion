package defect

import (
	"context"
	"database/sql"
)

// PostgreSQLStore persiste defectos en PostgreSQL.
type PostgreSQLStore struct {
	db *sql.DB
}

// NewPostgreSQLStore crea un store sobre una conexión abierta.
func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// Create registra un defecto en PostgreSQL.
func (s *PostgreSQLStore) Create(ctx context.Context, input RegisterInput) (Registered, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Registered{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var sprintID, sprintNumber int64
	err = tx.QueryRowContext(ctx,
		`SELECT id, number
		 FROM sprints
		 WHERE project_id = $1 AND status = 'Activo'
		 ORDER BY number DESC
		 LIMIT 1
		 FOR SHARE`,
		input.ProjectID,
	).Scan(&sprintID, &sprintNumber)
	if err == sql.ErrNoRows {
		return Registered{}, ErrNoActiveSprint
	}
	if err != nil {
		return Registered{}, err
	}

	var storyID int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM user_stories WHERE id = $1 AND project_id = $2`,
		input.StoryID, input.ProjectID,
	).Scan(&storyID)
	if err == sql.ErrNoRows {
		return Registered{}, ErrStoryNotFound
	}
	if err != nil {
		return Registered{}, err
	}

	var registered Registered
	registered.SprintNumber = sprintNumber
	err = tx.QueryRowContext(ctx,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		input.ProjectID, input.StoryID, input.Description, string(input.Severity), string(StatusOpen), sprintID,
	).Scan(&registered.ID)
	if err != nil {
		return Registered{}, err
	}
	if err := tx.Commit(); err != nil {
		return Registered{}, err
	}
	return registered, nil
}
