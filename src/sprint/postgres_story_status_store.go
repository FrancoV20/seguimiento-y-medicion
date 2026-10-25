package sprint

import (
	"context"
	"database/sql"
)

// PostgreSQLStoryStatusStore implementa StoryStatusStore mediante PostgreSQL (HU-14).
type PostgreSQLStoryStatusStore struct {
	db *sql.DB
}

var _ StoryStatusStore = (*PostgreSQLStoryStatusStore)(nil)

// NewPostgreSQLStoryStatusStore crea el almacenamiento sobre una conexión ya abierta.
func NewPostgreSQLStoryStatusStore(db *sql.DB) *PostgreSQLStoryStatusStore {
	return &PostgreSQLStoryStatusStore{db: db}
}

// FindStory lee la historia dentro del proyecto. RED: esqueleto sin implementar.
func (s *PostgreSQLStoryStatusStore) FindStory(ctx context.Context, projectID, storyID int64) (StoryState, error) {
	return StoryState{}, errNotImplemented
}

// UpdateStatus cambia el estado con comparación atómica. RED: esqueleto sin implementar.
func (s *PostgreSQLStoryStatusStore) UpdateStatus(ctx context.Context, storyID int64, from, to StoryStatus) (bool, error) {
	return false, errNotImplemented
}
