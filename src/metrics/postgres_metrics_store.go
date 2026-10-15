package metrics

import (
	"database/sql"
	"fmt"
	"math/big"
)

type PostgreSQLMetricsStore struct {
	db *sql.DB
}

func NewPostgreSQLMetricsStore(db *sql.DB) *PostgreSQLMetricsStore {
	return &PostgreSQLMetricsStore{db: db}
}

func (s *PostgreSQLMetricsStore) LoadSprintData(sprintID int64) (SprintData, error) {
	if s == nil || s.db == nil {
		return SprintData{}, fmt.Errorf("PostgreSQL database connection is required")
	}

	sprint := SprintData{}
	err := s.db.QueryRow(`
		SELECT id, status
		FROM sprints
		WHERE id = $1`,
		sprintID).Scan(&sprint.ID, &sprint.Status)
	if err != nil {
		return SprintData{}, fmt.Errorf("load sprint %d: %w", sprintID, err)
	}

	var hasEstimatedHours bool
	if err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'user_stories'
			  AND column_name = 'estimated_hours'
		)`).Scan(&hasEstimatedHours); err != nil {
		return SprintData{}, fmt.Errorf("check estimated-hours column: %w", err)
	}

	estimatedHoursColumn := "NULL::text"
	if hasEstimatedHours {
		estimatedHoursColumn = "us.estimated_hours::text"
	}

	query := fmt.Sprintf(`
		SELECT us.id, us.status, us.story_points,
		       %s AS estimated_hours, SUM(el.hours)::text AS actual_hours
		FROM sprint_story_assignments AS ss
		JOIN user_stories AS us ON us.id = ss.story_id
		LEFT JOIN effort_logs AS el ON el.story_id = us.id
		WHERE ss.sprint_id = $1
		GROUP BY us.id, us.status, us.story_points, %s
		ORDER BY us.id`,
		estimatedHoursColumn,
		estimatedHoursColumn,
	)
	rows, err := s.db.Query(query, sprintID)
	if err != nil {
		return SprintData{}, fmt.Errorf("load stories for sprint %d: %w", sprintID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			story         CompletedStory
			storyPoints   sql.NullInt64
			estimatedText sql.NullString
			actualText    sql.NullString
		)
		if err := rows.Scan(
			&story.ID,
			&story.Status,
			&storyPoints,
			&estimatedText,
			&actualText,
		); err != nil {
			return SprintData{}, fmt.Errorf("scan story for sprint %d: %w", sprintID, err)
		}

		if storyPoints.Valid {
			value := int(storyPoints.Int64)
			story.StoryPoints = &value
		}
		if estimatedText.Valid {
			value, ok := new(big.Rat).SetString(estimatedText.String)
			if !ok {
				return SprintData{}, fmt.Errorf("parse estimated hours for story %d: invalid decimal %q", story.ID, estimatedText.String)
			}
			story.EstimatedHours = value
		}
		if actualText.Valid {
			value, ok := new(big.Rat).SetString(actualText.String)
			if !ok {
				return SprintData{}, fmt.Errorf("parse actual hours for story %d: invalid decimal %q", story.ID, actualText.String)
			}
			story.ActualHours = value
		}

		sprint.Stories = append(sprint.Stories, story)
	}
	if err := rows.Err(); err != nil {
		return SprintData{}, fmt.Errorf("iterate stories for sprint %d: %w", sprintID, err)
	}

	return sprint, nil
}

func (s *PostgreSQLMetricsStore) SaveMetrics(metrics SprintMetrics) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("%w: PostgreSQL database connection is required", ErrMetricsPersistence)
	}
	if metrics.EstimatedHoursTotal == nil || metrics.ActualHoursTotal == nil {
		return fmt.Errorf("%w: effort totals must not be nil", ErrMetricsPersistence)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%w: begin transaction: %w", ErrMetricsPersistence, err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO sprint_metrics (
			sprint_id, velocity, estimated_hours_total, actual_hours_total,
			deviation_percentage, calculation_used_count, calculation_total_count,
			is_partial, warning
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (sprint_id) DO UPDATE SET
			velocity = EXCLUDED.velocity,
			estimated_hours_total = EXCLUDED.estimated_hours_total,
			actual_hours_total = EXCLUDED.actual_hours_total,
			deviation_percentage = EXCLUDED.deviation_percentage,
			calculation_used_count = EXCLUDED.calculation_used_count,
			calculation_total_count = EXCLUDED.calculation_total_count,
			is_partial = EXCLUDED.is_partial,
			warning = EXCLUDED.warning,
			calculated_at = NOW()`,
		metrics.SprintID,
		metrics.Velocity,
		metrics.EstimatedHoursTotal.FloatString(2),
		metrics.ActualHoursTotal.FloatString(2),
		metrics.DeviationPercentage,
		metrics.CalculationUsedCount,
		metrics.CalculationTotalCount,
		metrics.IsPartial,
		metrics.Warning)
	if err != nil {
		return fmt.Errorf("%w: save snapshot: %w", ErrMetricsPersistence, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit snapshot transaction: %w", ErrMetricsPersistence, err)
	}

	return nil
}
