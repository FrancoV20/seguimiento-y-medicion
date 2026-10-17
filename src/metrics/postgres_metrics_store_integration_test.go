package metrics

import (
	"database/sql"
	"math/big"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgreSQLMetricsStoreLoadsAndSavesMetrics(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL =  "postgres://USUARIO:CLAVE@localhost:PUERTO/seguimiento_y_medicion?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL connection: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL integration database is unavailable: %v", err)
	}

	var missingTables string
	if err := db.QueryRow(`
		SELECT COALESCE(string_agg(required.name, ', '), '')
		FROM unnest(ARRAY['sprints', 'sprint_story_assignments', 'sprint_metrics']::text[]) AS required(name)
		WHERE to_regclass('public.' || required.name) IS NULL`).Scan(&missingTables); err != nil {
		t.Fatalf("check Sprint schema prerequisites: %v", err)
	}
	if missingTables != "" {
		t.Skipf("Sprint integration schema is unavailable; missing tables: %s", missingTables)
	}

	var hasEstimatedHours bool
	if err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'user_stories'
			  AND column_name = 'estimated_hours'
		)`).Scan(&hasEstimatedHours); err != nil {
		t.Fatalf("check estimated-hours column: %v", err)
	}

	var projectID, memberID, sprintID, storyID int64
	if err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date)
		VALUES ('Metrics integration test', CURRENT_DATE, CURRENT_DATE)
		RETURNING id`).Scan(&projectID); err != nil {
		t.Fatalf("insert project fixture: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO members (display_name)
		VALUES ('Metrics integration member')
		RETURNING id`).Scan(&memberID); err != nil {
		t.Fatalf("insert member fixture: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Sprint de prueba', CURRENT_DATE, CURRENT_DATE + INTERVAL '1 day', 'Finalizado')
		RETURNING id`, projectID).Scan(&sprintID); err != nil {
		t.Fatalf("insert sprint fixture: %v", err)
	}

	if err := db.QueryRow(`
		INSERT INTO user_stories (
			project_id, active_sprint_id, title, description, priority, story_points, status
		)
		VALUES ($1, $2, 'Metrics integration story', 'Test story', 'Media', 5, 'Hecho')
		RETURNING id`,
		projectID, sprintID).Scan(&storyID); err != nil {
		t.Fatalf("insert story fixture: %v", err)
	}

	if hasEstimatedHours {
		if _, err := db.Exec(`UPDATE user_stories SET estimated_hours = 8.00 WHERE id = $1`, storyID); err != nil {
			t.Fatalf("set estimated-hours fixture: %v", err)
		}
	}

	if _, err := db.Exec(`
		INSERT INTO sprint_story_assignments (sprint_id, story_id)
		VALUES ($1, $2)`,
		sprintID, storyID); err != nil {
		t.Fatalf("associate story with sprint fixture: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, CURRENT_DATE, 'Metrics integration work', 10.00)`,
		storyID, memberID); err != nil {
		t.Fatalf("insert effort-log fixture: %v", err)
	}

	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM sprint_metrics WHERE sprint_id = $1`, sprintID); err != nil {
			t.Errorf("delete sprint-metrics fixture: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id = $1`, sprintID); err != nil {
			t.Errorf("delete sprint-story fixture: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM sprints WHERE id = $1`, sprintID); err != nil {
			t.Errorf("delete sprint fixture: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM effort_logs WHERE story_id = $1`, storyID); err != nil {
			t.Errorf("delete effort-log fixture: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM user_stories WHERE id = $1`, storyID); err != nil {
			t.Errorf("delete story fixture: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM projects WHERE id = $1`, projectID); err != nil {
			t.Errorf("delete project fixture: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM members WHERE id = $1`, memberID); err != nil {
			t.Errorf("delete member fixture: %v", err)
		}
	})

	store := NewPostgreSQLMetricsStore(db)
	loaded, err := store.LoadSprintData(sprintID)
	if err != nil {
		t.Fatalf("LoadSprintData() error = %v", err)
	}
	if loaded.ID != sprintID {
		t.Errorf("loaded sprint ID = %d, want %d", loaded.ID, sprintID)
	}
	if loaded.Status != "Finalizado" {
		t.Errorf("loaded sprint status = %q, want %q", loaded.Status, "Finalizado")
	}
	if len(loaded.Stories) != 1 {
		t.Fatalf("loaded stories count = %d, want 1", len(loaded.Stories))
	}
	if loaded.Stories[0].ID != storyID {
		t.Errorf("loaded story ID = %d, want %d", loaded.Stories[0].ID, storyID)
	}
	if loaded.Stories[0].Status != "Hecho" {
		t.Errorf("loaded story status = %q, want %q", loaded.Stories[0].Status, "Hecho")
	}
	if loaded.Stories[0].ActualHours == nil || loaded.Stories[0].ActualHours.Cmp(big.NewRat(10, 1)) != 0 {
		t.Errorf("loaded accumulated actual hours = %v, want 10", loaded.Stories[0].ActualHours)
	}
	if hasEstimatedHours {
		if loaded.Stories[0].EstimatedHours == nil || loaded.Stories[0].EstimatedHours.Cmp(big.NewRat(8, 1)) != 0 {
			t.Errorf("loaded estimated hours = %v, want 8", loaded.Stories[0].EstimatedHours)
		}
	} else if loaded.Stories[0].EstimatedHours != nil {
		t.Errorf("loaded estimated hours = %v, want nil when schema has no estimated_hours column", loaded.Stories[0].EstimatedHours)
	}

	metrics := SprintMetrics{
		SprintID:              sprintID,
		Velocity:              5,
		EstimatedHoursTotal:   integrationTestRat(t, "8"),
		ActualHoursTotal:      integrationTestRat(t, "10"),
		DeviationPercentage:   "25.00",
		CalculationUsedCount:  1,
		CalculationTotalCount: 1,
	}
	if err := store.SaveMetrics(metrics); err != nil {
		t.Fatalf("SaveMetrics() error = %v", err)
	}

	var (
		velocity            int
		estimatedTotal      string
		actualTotal         string
		deviationPercentage string
	)
	err = db.QueryRow(`
		SELECT velocity, estimated_hours_total, actual_hours_total, deviation_percentage
		FROM sprint_metrics
		WHERE sprint_id = $1`,
		sprintID).Scan(&velocity, &estimatedTotal, &actualTotal, &deviationPercentage)
	if err != nil {
		t.Fatalf("read persisted metrics: %v", err)
	}
	if velocity != metrics.Velocity {
		t.Errorf("persisted velocity = %d, want %d", velocity, metrics.Velocity)
	}
	if estimatedTotal != "8.00" {
		t.Errorf("persisted estimated hours = %q, want %q", estimatedTotal, "8.00")
	}
	if actualTotal != "10.00" {
		t.Errorf("persisted actual hours = %q, want %q", actualTotal, "10.00")
	}
	if deviationPercentage != metrics.DeviationPercentage {
		t.Errorf("persisted deviation = %q, want %q", deviationPercentage, metrics.DeviationPercentage)
	}
}

func integrationTestRat(t *testing.T, value string) *big.Rat {
	t.Helper()

	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		t.Fatalf("invalid rational number %q", value)
	}
	return rat
}