package metrics

import (
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgreSQLMetricsStoreLoadsAndSavesMetrics(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:44984954@127.0.0.1:5433/seguimiento_y_medicion?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL integration database is unavailable: %v", err)
	}

	fixtureID := fmt.Sprintf("metrics_it_%d", time.Now().UnixNano())
	projectID := fixtureID + "_project"
	sprintID := fixtureID + "_sprint"
	storyID := fixtureID + "_story"

	_, err = db.Exec(`
		INSERT INTO projects (id, name, start_date, end_date)
		VALUES ($1, $2, CURRENT_DATE, CURRENT_DATE)`,
		projectID, "Metrics integration test")
	if err != nil {
		t.Fatalf("insert project fixture: %v", err)
	}
	defer func() {
		if _, err := db.Exec(`DELETE FROM projects WHERE id = $1`, projectID); err != nil {
			t.Errorf("delete project fixture: %v", err)
		}
	}()

	_, err = db.Exec(`
		INSERT INTO sprints (id, project_id, number, display_id, status)
		VALUES ($1, $2, 1, $3, 'Finalizado')`,
		sprintID, projectID, "Integration "+fixtureID)
	if err != nil {
		t.Fatalf("insert sprint fixture: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO backlog_stories (
			id, project_id, title, story_points, estimated_hours, actual_hours, status
		)
		VALUES ($1, $2, $3, 5, 8.00, 10.00, 'Terminada')`,
		storyID, projectID, "Metrics integration story")
	if err != nil {
		t.Fatalf("insert story fixture: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sprint_stories (sprint_id, story_id)
		VALUES ($1, $2)`,
		sprintID, storyID)
	if err != nil {
		t.Fatalf("associate story with sprint fixture: %v", err)
	}

	store := NewPostgreSQLMetricsStore(db)
	loaded, err := store.LoadSprintData(sprintID)
	if err != nil {
		t.Fatalf("LoadSprintData() error = %v", err)
	}
	if loaded.ID != sprintID {
		t.Errorf("loaded sprint ID = %q, want %q", loaded.ID, sprintID)
	}
	if loaded.Status != "Finalizado" {
		t.Errorf("loaded sprint status = %q, want %q", loaded.Status, "Finalizado")
	}
	if len(loaded.Stories) != 1 {
		t.Fatalf("loaded stories count = %d, want 1", len(loaded.Stories))
	}
	if loaded.Stories[0].ID != storyID {
		t.Errorf("loaded story ID = %q, want %q", loaded.Stories[0].ID, storyID)
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
