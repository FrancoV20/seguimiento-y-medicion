package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

func TestCloseSprintServiceClosesActiveSprintAndReturnsCounts(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	sprintID, storyIDs := fixture.activeSprintWithStories(
		t,
		fixture.projectA,
		[]closeSprintStoryFixture{
			{status: Done, storyPoints: intPointer(3)},
			{status: InProgress, storyPoints: intPointer(5)},
			{status: EnSprint, storyPoints: intPointer(8)},
		},
	)
	service := fixture.service()

	result, err := service.CloseSprint(context.Background(), fixture.projectA, sprintID)

	if err != nil {
		t.Fatalf("CloseSprint() devolvió error: %v", err)
	}
	if result.SprintNumber != 1 || result.CompletedCount != 1 || result.ReturnedCount != 2 {
		t.Errorf("CloseSprint() = %+v; se esperaba Sprint 1, 1 completada y 2 devueltas", result)
	}
	fixture.assertSprintStatus(t, sprintID, SprintFinished)
	fixture.assertStoryState(t, storyIDs[0], Done, sprintID)
	fixture.assertStoryState(t, storyIDs[1], Pendiente, 0)
	fixture.assertStoryState(t, storyIDs[2], Pendiente, 0)
}

func TestCloseSprintServiceValidatesProjectBeforeStatusAndHU07(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	sprintID, _ := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
		{status: Done, storyPoints: nil},
	})
	validator := &closeSprintTestValidator{db: fixture.db}
	service := NewCloseSprintService(fixture.store(), validator)

	_, err := service.CloseSprint(context.Background(), fixture.projectB, sprintID)

	if !errors.Is(err, ErrSprintNotFound) {
		t.Fatalf("error = %v, se esperaba ErrSprintNotFound antes de validar HU-07", err)
	}
	if validator.calls != 0 {
		t.Errorf("la validación HU-07 se llamó %d veces antes de confirmar el proyecto; se esperaba 0", validator.calls)
	}
	fixture.assertSprintStatus(t, sprintID, SprintActive)
}

func TestCloseSprintServiceValidatesActiveStatusBeforeHU07(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	sprintID, _ := fixture.sprintWithStories(t, fixture.projectA, SprintFinished, []closeSprintStoryFixture{
		{status: Done, storyPoints: nil},
	})
	validator := &closeSprintTestValidator{db: fixture.db}
	service := NewCloseSprintService(fixture.store(), validator)

	_, err := service.CloseSprint(context.Background(), fixture.projectA, sprintID)

	if !errors.Is(err, ErrSprintAlreadyClosed) {
		t.Fatalf("error = %v, se esperaba ErrSprintAlreadyClosed antes de validar HU-07", err)
	}
	if validator.calls != 0 {
		t.Errorf("la validación HU-07 se llamó %d veces para un Sprint finalizado; se esperaba 0", validator.calls)
	}
	fixture.assertSprintStatus(t, sprintID, SprintFinished)
}

func TestCloseSprintServiceRejectsMissingSprint(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	validator := &closeSprintTestValidator{db: fixture.db}
	service := NewCloseSprintService(fixture.store(), validator)

	_, err := service.CloseSprint(context.Background(), fixture.projectA, int64(9_000_000_000))

	if !errors.Is(err, ErrSprintNotFound) {
		t.Fatalf("error = %v, se esperaba ErrSprintNotFound", err)
	}
	if validator.calls != 0 {
		t.Errorf("la validación HU-07 se llamó %d veces para un Sprint inexistente; se esperaba 0", validator.calls)
	}
}

func TestCloseSprintServiceBlocksInvalidCompletedStoryWithoutChangingData(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	sprintID, storyIDs := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
		{status: Done, storyPoints: nil},
		{status: InProgress, storyPoints: intPointer(5)},
	})
	validator := &closeSprintTestValidator{db: fixture.db}
	service := NewCloseSprintService(fixture.store(), validator)

	_, err := service.CloseSprint(context.Background(), fixture.projectA, sprintID)

	if !errors.Is(err, ErrCloseBlocked) {
		t.Fatalf("error = %v, se esperaba ErrCloseBlocked", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("Historia %d", storyIDs[0])) ||
		!strings.Contains(err.Error(), metrics.ErrInvalidStoryPoints.Error()) {
		t.Errorf("el error de cierre no incluye la historia inválida y el motivo HU-07: %v", err)
	}
	if validator.calls != 1 {
		t.Errorf("la validación HU-07 se llamó %d veces; se esperaba 1", validator.calls)
	}
	fixture.assertSprintStatus(t, sprintID, SprintActive)
	fixture.assertStoryState(t, storyIDs[0], Done, sprintID)
	fixture.assertStoryState(t, storyIDs[1], InProgress, sprintID)
}

func TestCloseSprintServiceAllowsEmptySprintAndHU07VelocityIsZero(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	emptySprintID, _ := fixture.sprintWithStories(t, fixture.projectA, SprintActive, nil)
	service := fixture.service()

	result, err := service.CloseSprint(context.Background(), fixture.projectA, emptySprintID)
	if err != nil {
		t.Fatalf("CloseSprint() para Sprint vacío devolvió error: %v", err)
	}
	if result.CompletedCount != 0 || result.ReturnedCount != 0 {
		t.Errorf("resultado para Sprint vacío = %+v; se esperaban ambos conteos en 0", result)
	}
	fixture.assertSprintStatus(t, emptySprintID, SprintFinished)

	metricsResult, err := metrics.NewService(metrics.NewPostgreSQLMetricsStore(fixture.db)).
		CalculateMetrics(emptySprintID)
	if err != nil {
		t.Fatalf("HU-07 CalculateMetrics() para Sprint vacío devolvió error: %v", err)
	}
	if metricsResult.Velocity != 0 {
		t.Errorf("velocidad HU-07 = %d para Sprint vacío; se esperaba 0", metricsResult.Velocity)
	}
}

func TestCloseSprintServiceAllowsSprintWithoutCompletedStoriesAndHU07VelocityIsZero(t *testing.T) {
	fixture := newCloseSprintServiceFixture(t)
	sprintID, _ := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
		{status: InProgress, storyPoints: intPointer(3)},
		{status: EnSprint, storyPoints: intPointer(5)},
	})
	service := fixture.service()

	result, err := service.CloseSprint(context.Background(), fixture.projectA, sprintID)
	if err != nil {
		t.Fatalf("CloseSprint() sin historias Hecho devolvió error: %v", err)
	}
	if result.CompletedCount != 0 || result.ReturnedCount != 2 {
		t.Errorf("resultado sin historias Hecho = %+v; se esperaban 0 completadas y 2 devueltas", result)
	}
	fixture.assertSprintStatus(t, sprintID, SprintFinished)

	metricsResult, err := metrics.NewService(metrics.NewPostgreSQLMetricsStore(fixture.db)).
		CalculateMetrics(sprintID)
	if err != nil {
		t.Fatalf("HU-07 CalculateMetrics() sin historias Hecho devolvió error: %v", err)
	}
	if metricsResult.Velocity != 0 {
		t.Errorf("velocidad HU-07 = %d sin historias Hecho; se esperaba 0", metricsResult.Velocity)
	}
}

type closeSprintServiceFixture struct {
	db       *sql.DB
	projectA int64
	projectB int64
}

type closeSprintStoryFixture struct {
	status      StoryStatus
	storyPoints *int
}

func newCloseSprintServiceFixture(t *testing.T) *closeSprintServiceFixture {
	t.Helper()
	db := openStatusTestDB(t)
	projectA := newStatusProject(t, db, "Proyecto A de integración HU-15")
	projectB := newStatusProject(t, db, "Proyecto B de integración HU-15")
	return &closeSprintServiceFixture{db: db, projectA: projectA, projectB: projectB}
}

func (f *closeSprintServiceFixture) store() *postgresCloseSprintServiceStore {
	return &postgresCloseSprintServiceStore{
		db:     f.db,
		closer: NewPostgreSQLSprintCloser(f.db),
	}
}

func (f *closeSprintServiceFixture) service() *CloseSprintService {
	return NewCloseSprintService(f.store(), &closeSprintTestValidator{db: f.db})
}

func (f *closeSprintServiceFixture) sprintWithStories(
	t *testing.T,
	projectID int64,
	status SprintStatus,
	storyFixtures []closeSprintStoryFixture,
) (int64, []int64) {
	t.Helper()
	sprintID := newStatusSprint(t, f.db, projectID, status)
	storyIDs := make([]int64, 0, len(storyFixtures))
	for _, story := range storyFixtures {
		storyID := newStatusStory(t, f.db, projectID, story.status, sprintID)
		if _, err := f.db.Exec(
			`UPDATE user_stories SET story_points = $1 WHERE id = $2`,
			story.storyPoints,
			storyID,
		); err != nil {
			t.Fatalf("no se pudieron establecer Story Points de historia %d: %v", storyID, err)
		}
		storyIDs = append(storyIDs, storyID)
	}
	return sprintID, storyIDs
}

func (f *closeSprintServiceFixture) activeSprintWithStories(
	t *testing.T,
	projectID int64,
	stories []closeSprintStoryFixture,
) (int64, []int64) {
	t.Helper()
	return f.sprintWithStories(t, projectID, SprintActive, stories)
}

func (f *closeSprintServiceFixture) assertSprintStatus(t *testing.T, sprintID int64, want SprintStatus) {
	t.Helper()
	var got string
	if err := f.db.QueryRow(
		`SELECT status FROM sprints WHERE id = $1`,
		sprintID,
	).Scan(&got); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint %d: %v", sprintID, err)
	}
	if got != string(want) {
		t.Errorf("estado del Sprint %d = %q; se esperaba %q", sprintID, got, want)
	}
}

func (f *closeSprintServiceFixture) assertStoryState(
	t *testing.T,
	storyID int64,
	wantStatus StoryStatus,
	wantSprintID int64,
) {
	t.Helper()
	var status string
	var activeSprintID sql.NullInt64
	if err := f.db.QueryRow(
		`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`,
		storyID,
	).Scan(&status, &activeSprintID); err != nil {
		t.Fatalf("no se pudo leer la historia %d: %v", storyID, err)
	}
	if status != string(wantStatus) {
		t.Errorf("estado de historia %d = %q; se esperaba %q", storyID, status, wantStatus)
	}
	if wantSprintID == 0 {
		if activeSprintID.Valid {
			t.Errorf("historia %d conserva active_sprint_id=%d; se esperaba NULL", storyID, activeSprintID.Int64)
		}
		return
	}
	if !activeSprintID.Valid || activeSprintID.Int64 != wantSprintID {
		t.Errorf("active_sprint_id de historia %d = %v; se esperaba %d", storyID, activeSprintID, wantSprintID)
	}
}

type postgresCloseSprintServiceStore struct {
	db     *sql.DB
	closer *PostgreSQLSprintCloser
}

func (s *postgresCloseSprintServiceStore) FindSprint(
	ctx context.Context,
	projectID int64,
	sprintID int64,
) (Sprint, error) {
	var sprint Sprint
	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, status
		FROM sprints
		WHERE project_id = $1 AND id = $2`,
		projectID,
		sprintID,
	).Scan(&sprint.ID, &sprint.ProjectID, &sprint.Number, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return Sprint{}, ErrSprintNotFound
	}
	if err != nil {
		return Sprint{}, err
	}
	sprint.Status = SprintStatus(status)
	return sprint, nil
}

func (s *postgresCloseSprintServiceStore) CloseSprint(
	ctx context.Context,
	sprintID int64,
) (CloseResult, error) {
	return s.closer.CloseSprint(ctx, sprintID)
}

type closeSprintTestValidator struct {
	db    *sql.DB
	calls int
}

func (v *closeSprintTestValidator) ValidateSprintClosure(_ context.Context, sprintID int64) error {
	v.calls++
	sprintData, err := metrics.NewPostgreSQLMetricsStore(v.db).LoadSprintData(sprintID)
	if err != nil {
		return err
	}
	return metrics.ValidateSprintClosure(sprintData)
}

func intPointer(value int) *int {
	return &value
}
