package defect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func openDefectTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no está definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}
	db.SetMaxOpenConns(10)
	return db
}

type defectStoreFixture struct {
	projectA, projectB                              int64
	activeSprintA, finishedSprintA, finishedSprintB int64
	storyPending, storyInProgress, storyDone        int64
	storyWithoutSprint, storyProjectB               int64
}

func newDefectStoreFixture(t *testing.T, db *sql.DB) defectStoreFixture {
	t.Helper()
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	fixture := defectStoreFixture{}
	insertID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear un dato de prueba: %v", err)
		}
		return id
	}
	insertProject := func(name string) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO projects (name, start_date, end_date) VALUES ($1, $2, $3) RETURNING id`,
			fmt.Sprintf("%s %d", name, stamp), "2026-01-01", "2026-12-31",
		)
	}
	insertSprint := func(projectID int64, number int, status string) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			 VALUES ($1, $2, $3, '2026-01-01', '2026-01-14', $4) RETURNING id`,
			projectID, number, fmt.Sprintf("goal %d %d", projectID, number), status,
		)
	}
	insertStory := func(projectID int64, title, status string, sprintID *int64) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
			 VALUES ($1, $2, 'Descripción de prueba', 'Media', $3, $4) RETURNING id`,
			projectID, fmt.Sprintf("%s %d", title, stamp), status, sprintID,
		)
	}

	fixture.projectA = insertProject("HU06 Store A")
	fixture.projectB = insertProject("HU06 Store B")
	fixture.activeSprintA = insertSprint(fixture.projectA, 1, "Activo")
	fixture.finishedSprintA = insertSprint(fixture.projectA, 2, "Finalizado")
	fixture.finishedSprintB = insertSprint(fixture.projectB, 1, "Finalizado")
	fixture.storyPending = insertStory(fixture.projectA, "Pendiente", "Pendiente", nil)
	fixture.storyInProgress = insertStory(fixture.projectA, "En progreso", "En progreso", &fixture.activeSprintA)
	fixture.storyDone = insertStory(fixture.projectA, "Hecho anterior", "Hecho", &fixture.finishedSprintA)
	fixture.storyWithoutSprint = insertStory(fixture.projectA, "Sin Sprint", "Pendiente", nil)
	fixture.storyProjectB = insertStory(fixture.projectB, "Proyecto B", "Pendiente", nil)

	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `DELETE FROM user_stories WHERE project_id IN ($1, $2)`, fixture.projectA, fixture.projectB); err != nil {
			t.Errorf("no se pudieron borrar las historias de prueba: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM sprints WHERE project_id IN ($1, $2)`, fixture.projectA, fixture.projectB); err != nil {
			t.Errorf("no se pudieron borrar los Sprints de prueba: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM projects WHERE id IN ($1, $2)`, fixture.projectA, fixture.projectB); err != nil {
			t.Errorf("no se pudieron borrar los proyectos de prueba: %v", err)
		}
	})
	return fixture
}

func TestPostgreSQLStore_CreateTracksActiveSprintAndPreservesStory(t *testing.T) {
	db := openDefectTestDB(t)
	fixture := newDefectStoreFixture(t, db)
	store := NewPostgreSQLStore(db)
	ctx := context.Background()

	for _, storyID := range []int64{fixture.storyPending, fixture.storyInProgress, fixture.storyDone, fixture.storyWithoutSprint} {
		var beforeStatus string
		var beforeSprintID sql.NullInt64
		if err := db.QueryRowContext(ctx,
			`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`, storyID,
		).Scan(&beforeStatus, &beforeSprintID); err != nil {
			t.Fatalf("no se pudo leer la historia antes del registro: %v", err)
		}

		input := validRegisterInput()
		input.ProjectID = fixture.projectA
		input.StoryID = storyID
		registered, err := store.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create para historia %d devolvió error: %v", storyID, err)
		}
		var status string
		var detectionSprintID int64
		var resolutionSprintID sql.NullInt64
		if err := db.QueryRowContext(ctx,
			`SELECT status, detection_sprint_id, resolution_sprint_id FROM defects WHERE id = $1`,
			registered.ID,
		).Scan(&status, &detectionSprintID, &resolutionSprintID); err != nil {
			t.Fatalf("no se pudo leer el defecto creado: %v", err)
		}
		if status != string(StatusOpen) || detectionSprintID != fixture.activeSprintA || resolutionSprintID.Valid {
			t.Errorf("datos del defecto: estado=%q sprint detección=%d sprint resolución=%+v",
				status, detectionSprintID, resolutionSprintID)
		}
		if registered.SprintNumber != 1 {
			t.Errorf("número de Sprint registrado = %d; se esperaba 1", registered.SprintNumber)
		}

		var unchangedStatus string
		var unchangedSprintID sql.NullInt64
		if err := db.QueryRowContext(ctx,
			`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`, storyID,
		).Scan(&unchangedStatus, &unchangedSprintID); err != nil {
			t.Fatalf("no se pudo volver a leer la historia: %v", err)
		}
		if unchangedStatus != beforeStatus || unchangedSprintID != beforeSprintID {
			t.Errorf("la historia %d cambió: antes=(%q, %+v), después=(%q, %+v)",
				storyID, beforeStatus, beforeSprintID, unchangedStatus, unchangedSprintID)
		}
	}

	if _, err := store.Create(ctx, RegisterInput{ProjectID: fixture.projectB, StoryID: fixture.storyProjectB, Description: "sin Sprint", Severity: SeverityLow}); !errors.Is(err, ErrNoActiveSprint) {
		t.Errorf("proyecto B sin Sprint activo: se esperaba ErrNoActiveSprint, se obtuvo %v", err)
	}
}

func TestPostgreSQLStore_CreateRejectsForeignStoryAndRollsBack(t *testing.T) {
	db := openDefectTestDB(t)
	fixture := newDefectStoreFixture(t, db)
	store := NewPostgreSQLStore(db)
	ctx := context.Background()
	var before int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM defects WHERE project_id = $1`, fixture.projectA).Scan(&before); err != nil {
		t.Fatal(err)
	}

	input := validRegisterInput()
	input.ProjectID = fixture.projectA
	input.StoryID = fixture.storyProjectB
	if _, err := store.Create(ctx, input); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("se esperaba ErrStoryNotFound, se obtuvo %v", err)
	}

	var after int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM defects WHERE project_id = $1`, fixture.projectA).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("la transacción fallida dejó defectos: antes=%d después=%d", before, after)
	}
}

func TestPostgreSQLStore_CreateSerializesWithSprintClosure(t *testing.T) {
	db := openDefectTestDB(t)
	fixture := newDefectStoreFixture(t, db)
	store := NewPostgreSQLStore(db)
	ctx := context.Background()

	closure, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var lockedSprintID int64
	if err := closure.QueryRowContext(ctx,
		`SELECT id FROM sprints WHERE id = $1 FOR UPDATE`, fixture.activeSprintA,
	).Scan(&lockedSprintID); err != nil {
		_ = closure.Rollback()
		t.Fatalf("no se pudo bloquear el Sprint para simular el cierre: %v", err)
	}

	result := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		input := validRegisterInput()
		input.ProjectID = fixture.projectA
		input.StoryID = fixture.storyPending
		_, createErr := store.Create(ctx, input)
		result <- createErr
	}()
	<-started
	select {
	case createErr := <-result:
		_ = closure.Rollback()
		t.Fatalf("Create no esperó el bloqueo FOR UPDATE del cierre: %v", createErr)
	case <-time.After(100 * time.Millisecond):
	}

	if _, err := closure.ExecContext(ctx, `UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, lockedSprintID); err != nil {
		_ = closure.Rollback()
		t.Fatalf("no se pudo finalizar el Sprint: %v", err)
	}
	if err := closure.Commit(); err != nil {
		t.Fatalf("no se pudo confirmar el cierre: %v", err)
	}

	select {
	case createErr := <-result:
		if !errors.Is(createErr, ErrNoActiveSprint) {
			t.Errorf("tras el cierre se esperaba ErrNoActiveSprint, se obtuvo %v", createErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Create no terminó después de confirmar el cierre")
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM defects WHERE project_id = $1`, fixture.projectA).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("no debe quedar un defecto asociado al Sprint cerrado, hay %d", count)
	}
}
