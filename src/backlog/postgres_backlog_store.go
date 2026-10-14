package backlog

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para database/sql
)

// PostgreSQLBacklogStore persiste el Product Backlog en PostgreSQL (T010, T020).
type PostgreSQLBacklogStore struct {
	db *sql.DB
}

// OpenPostgreSQL abre una conexión usando la URL de DATABASE_URL.
func OpenPostgreSQL(databaseURL string) (*sql.DB, error) {
	return sql.Open("pgx", databaseURL)
}

// NewPostgreSQLBacklogStore crea el store sobre una conexión abierta.
func NewPostgreSQLBacklogStore(db *sql.DB) *PostgreSQLBacklogStore {
	return &PostgreSQLBacklogStore{db: db}
}

// ProjectIsActive indica si el proyecto existe y está en estado Activo.
func (s *PostgreSQLBacklogStore) ProjectIsActive(projectID int64) (bool, error) {
	var active bool
	err := s.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1 AND status = 'Activo')`,
		projectID,
	).Scan(&active)
	return active, err
}

// CreateStory guarda la historia y sus criterios en una sola transacción:
// si algo falla, no queda nada persistido.
func (s *PostgreSQLBacklogStore) CreateStory(projectID int64, story UserStory) (UserStory, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return UserStory{}, err
	}
	defer func() { _ = tx.Rollback() }() // no tiene efecto después del Commit

	var storyPoints sql.NullInt64
	if story.StoryPoints != nil {
		storyPoints = sql.NullInt64{Int64: int64(*story.StoryPoints), Valid: true}
	}

	err = tx.QueryRowContext(ctx,
		`INSERT INTO user_stories (project_id, title, description, priority, story_points, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		projectID, story.Title, story.Description, string(story.Priority), storyPoints, string(story.Status),
	).Scan(&story.ID, &story.CreatedAt)
	if err != nil {
		return UserStory{}, err
	}

	for _, c := range story.AcceptanceCriteria {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, $2, $3)`,
			story.ID, c.Position, c.Content,
		)
		if err != nil {
			return UserStory{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return UserStory{}, err
	}
	story.ProjectID = projectID
	return story, nil
}
