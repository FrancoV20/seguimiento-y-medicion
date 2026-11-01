package metrics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLQualityStore implementa QualityStore con PostgreSQL (HU-10, T018). Solo lee: el Sprint,
// el registro de cierre de HU-15 (sprint_story_results) y los defectos de HU-06.
type PostgreSQLQualityStore struct {
	db *sql.DB
}

var _ QualityStore = (*PostgreSQLQualityStore)(nil)

// NewPostgreSQLQualityStore crea el adaptador sobre una conexión abierta.
func NewPostgreSQLQualityStore(db *sql.DB) *PostgreSQLQualityStore {
	return &PostgreSQLQualityStore{db: db}
}

// LoadSprintQuality lee los datos de calidad del Sprint del proyecto en una única transacción de solo
// lectura con REPEATABLE READ, para que el Sprint, el registro de cierre y los defectos correspondan
// al mismo instante. Devuelve ErrSprintNotFound si el Sprint no existe en el proyecto; si no está
// "Finalizado", devuelve solo SprintID y Status, sin leer el registro ni los defectos.
func (s *PostgreSQLQualityStore) LoadSprintQuality(ctx context.Context, projectID, sprintID int64) (QualityData, error) {
	if s == nil || s.db == nil {
		return QualityData{}, errors.New("PostgreSQL database connection is required")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return QualityData{}, fmt.Errorf("load sprint quality %d: begin transaction: %w", sprintID, err)
	}
	defer func() { _ = tx.Rollback() }() // sin efecto después del Commit

	data := QualityData{SprintID: sprintID}
	err = tx.QueryRowContext(ctx, `
		SELECT status
		FROM sprints
		WHERE id = $1 AND project_id = $2`,
		sprintID, projectID,
	).Scan(&data.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return QualityData{}, ErrSprintNotFound
	}
	if err != nil {
		return QualityData{}, fmt.Errorf("load sprint quality %d: read sprint: %w", sprintID, err)
	}
	if data.Status != qualityFinishedStatus {
		return data, nil
	}

	if data.Stories, err = loadQualityStories(ctx, tx, sprintID); err != nil {
		return QualityData{}, err
	}

	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE detection_sprint_id = $1),
		       COUNT(*) FILTER (WHERE status = 'Resuelto' AND resolution_sprint_id = $1)
		FROM defects
		WHERE project_id = $2
		  AND (detection_sprint_id = $1 OR resolution_sprint_id = $1)`,
		sprintID, projectID,
	).Scan(&data.DefectsDetected, &data.DefectsResolved); err != nil {
		return QualityData{}, fmt.Errorf("load sprint quality %d: count defects: %w", sprintID, err)
	}

	if err := tx.Commit(); err != nil {
		return QualityData{}, fmt.Errorf("load sprint quality %d: commit: %w", sprintID, err)
	}
	return data, nil
}

// loadQualityStories lee el registro de cierre del Sprint ordenado por historia.
func loadQualityStories(ctx context.Context, tx *sql.Tx, sprintID int64) ([]QualityStoryResult, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT final_status, story_points
		FROM sprint_story_results
		WHERE sprint_id = $1
		ORDER BY story_id`,
		sprintID,
	)
	if err != nil {
		return nil, fmt.Errorf("load sprint quality %d: read closing record: %w", sprintID, err)
	}
	defer rows.Close()

	var stories []QualityStoryResult
	for rows.Next() {
		var (
			story  QualityStoryResult
			points sql.NullInt64
		)
		if err := rows.Scan(&story.FinalStatus, &points); err != nil {
			return nil, fmt.Errorf("load sprint quality %d: scan closing record: %w", sprintID, err)
		}
		if points.Valid {
			value := int(points.Int64)
			story.StoryPoints = &value
		}
		stories = append(stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load sprint quality %d: iterate closing record: %w", sprintID, err)
	}
	return stories, nil
}
