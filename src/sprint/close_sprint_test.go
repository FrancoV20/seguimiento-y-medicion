package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
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

func TestCloseSprint_ReasignaHistoriasDevueltasYConservaElRegistroCongelado(t *testing.T) {
	// Dado un Sprint cerrado que devolvió historias al Backlog y cuyo registro fue guardado.
	fixture := newCloseSprintServiceFixture(t)
	sprintID, storyIDs := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
		{status: Done, storyPoints: intPointer(3)},
		{status: InProgress, storyPoints: intPointer(5)},
		{status: EnSprint, storyPoints: intPointer(8)},
	})
	result, err := fixture.service().CloseSprint(context.Background(), fixture.projectA, sprintID)
	if err != nil {
		t.Fatalf("CloseSprint() devolvió error: %v", err)
	}
	if result.ReturnedCount != 2 {
		t.Fatalf("historias devueltas = %d; se esperaban 2", result.ReturnedCount)
	}
	snapshotBefore := readCloseSprintResults(t, fixture.db, sprintID)

	// Cuando el servicio real de HU-04 crea un Sprint nuevo asignando las devueltas y
	// luego se editan los Story Points de una de ellas.
	storiesToAssign := make([]UserStory, 0, 2)
	for _, storyID := range storyIDs[1:] {
		storiesToAssign = append(storiesToAssign, UserStory{
			ID:        storyID,
			ProjectID: fixture.projectA,
			Status:    Pendiente,
		})
	}
	var nextNumber int64
	if err := fixture.db.QueryRow(
		`SELECT COALESCE(MAX(number), 0) + 1 FROM sprints WHERE project_id = $1`,
		fixture.projectA,
	).Scan(&nextNumber); err != nil {
		t.Fatalf("no se pudo leer el siguiente número de Sprint: %v", err)
	}
	sprintStore, err := NewPostgreSQLSprintStore(testDatabaseURL(t))
	if err != nil {
		t.Fatalf("NewPostgreSQLSprintStore() devolvió error: %v", err)
	}
	t.Cleanup(func() {
		if err := sprintStore.Close(); err != nil {
			t.Errorf("no se pudo cerrar el store PostgreSQL: %v", err)
		}
	})
	newSprint, err := NewService(sprintStore).CreateSprint(
		context.Background(),
		SprintCreationRequest{
			ProjectID: fixture.projectA,
			Goal:      "Replanificar historias devueltas",
			StartDate: time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, time.November, 14, 0, 0, 0, 0, time.UTC),
			StoryIDs:  []int64{storyIDs[1], storyIDs[2]},
		},
		Project{
			ID:               fixture.projectA,
			Status:           "Activo",
			NextSprintNumber: nextNumber,
		},
		storiesToAssign,
	)
	if err != nil {
		t.Fatalf("HU-04 rechazó las historias devueltas al crear el Sprint nuevo: %v", err)
	}
	if len(newSprint.Sprint.StoryIDs) != 2 {
		t.Errorf("historias asignadas al Sprint nuevo = %v; se esperaban %v",
			newSprint.Sprint.StoryIDs, []int64{storyIDs[1], storyIDs[2]})
	}
	for _, storyID := range storyIDs[1:] {
		fixture.assertStoryState(t, storyID, EnSprint, newSprint.Sprint.ID)
	}
	const editedPoints = 13
	if _, err := fixture.db.Exec(
		`UPDATE user_stories SET story_points = $1 WHERE id = $2`,
		editedPoints,
		storyIDs[1],
	); err != nil {
		t.Fatalf("no se pudieron editar los Story Points de la historia devuelta: %v", err)
	}

	// Entonces el Sprint nuevo contiene las historias y la foto del Sprint anterior
	// conserva exactamente sus historias, estados finales y puntos originales.
	snapshotAfter := readCloseSprintResults(t, fixture.db, sprintID)
	if !reflect.DeepEqual(snapshotAfter, snapshotBefore) {
		t.Errorf("el registro del Sprint cerrado cambió después de reasignar o editar puntos:\nantes=%+v\ndespués=%+v",
			snapshotBefore, snapshotAfter)
	}
}

func TestCloseSprint_EsfuerzoPosteriorAlCierreSeRechazaPorHU03(t *testing.T) {
	// Dado un Sprint finalizado con una historia Hecho y otra devuelta al Backlog.
	fixture := newCloseSprintServiceFixture(t)
	sprintID, storyIDs := fixture.activeSprintWithStories(t, fixture.projectA, []closeSprintStoryFixture{
		{status: Done, storyPoints: intPointer(3)},
		{status: InProgress, storyPoints: intPointer(5)},
	})
	if _, err := fixture.service().CloseSprint(context.Background(), fixture.projectA, sprintID); err != nil {
		t.Fatalf("no se pudo finalizar el Sprint de prueba: %v", err)
	}
	var memberID int64
	if err := fixture.db.QueryRow(
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("HU15 esfuerzo posterior %d", time.Now().UnixNano()),
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
	}
	t.Cleanup(func() {
		_, _ = fixture.db.Exec(`DELETE FROM members WHERE id = $1`, memberID)
	})
	if _, err := fixture.db.Exec(
		`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`,
		fixture.projectA, memberID,
	); err != nil {
		t.Fatalf("no se pudo vincular el integrante al proyecto: %v", err)
	}

	// Cuando el servicio real de HU-03 intenta registrar horas en la historia Hecho
	// que sigue asignada al Sprint finalizado.
	result := effort.NewService(
		effort.NewPostgreSQLEffortStore(fixture.db),
		func() time.Time { return time.Date(2026, time.October, 29, 12, 0, 0, 0, time.UTC) },
	).LogEffort(context.Background(), effort.LogEffortRequest{
		StoryID:  fmt.Sprintf("%d", storyIDs[0]),
		MemberID: fmt.Sprintf("%d", memberID),
		WorkDate: time.Date(2026, time.October, 28, 0, 0, 0, 0, time.UTC),
		Activity: "Intento posterior al cierre",
		Hours:    "2",
	})

	// Entonces HU-03 rechaza la carga con su error de dominio y no guarda horas.
	// Prueba de caracterización: la regla ya existe en HU-03 (T001); coordinar con su responsable.
	foundDomainError := false
	for _, validationError := range result.ValidationErrors {
		if validationError.Field == effort.FieldStory && errors.Is(validationError.Err, effort.ErrStoryNotInSprint) {
			foundDomainError = true
			break
		}
	}
	if !foundDomainError {
		t.Errorf("HU-03 debía rechazar la historia del Sprint finalizado con %q; resultado=%+v",
			effort.ErrStoryNotInSprint, result)
	}
	if result.Succeeded() {
		t.Errorf("HU-03 informó éxito al intentar cargar esfuerzo después del cierre")
	}
	var storedEffort int
	if err := fixture.db.QueryRow(
		`SELECT count(*) FROM effort_logs WHERE story_id = $1`,
		storyIDs[0],
	).Scan(&storedEffort); err != nil {
		t.Fatalf("no se pudo comprobar el esfuerzo persistido: %v", err)
	}
	if storedEffort != 0 {
		t.Errorf("se guardaron %d registros de esfuerzo después del cierre; se esperaba 0", storedEffort)
	}
	fixture.assertStoryState(t, storyIDs[1], Pendiente, 0)
}

type closeSprintSnapshotRow struct {
	StoryID     int64
	FinalStatus string
	StoryPoints sql.NullInt64
}

func readCloseSprintResults(t *testing.T, db *sql.DB, sprintID int64) []closeSprintSnapshotRow {
	t.Helper()
	rows, err := db.Query(`
		SELECT story_id, final_status, story_points
		FROM sprint_story_results
		WHERE sprint_id = $1
		ORDER BY story_id`, sprintID)
	if err != nil {
		t.Fatalf("no se pudo leer la foto del Sprint %d: %v", sprintID, err)
	}
	defer rows.Close()

	var results []closeSprintSnapshotRow
	for rows.Next() {
		var row closeSprintSnapshotRow
		if err := rows.Scan(&row.StoryID, &row.FinalStatus, &row.StoryPoints); err != nil {
			t.Fatalf("no se pudo leer una fila de la foto del Sprint %d: %v", sprintID, err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("no se pudo recorrer la foto del Sprint %d: %v", sprintID, err)
	}
	return results
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
	check ClosureCheck,
) (CloseResult, error) {
	return s.closer.CloseSprint(ctx, sprintID, check)
}

type closeSprintTestValidator struct {
	db    *sql.DB
	calls int
}

// ValidateSprintClosure aplica HU-07 a las historias que el cierre bloqueó, como el adaptador de main.
func (v *closeSprintTestValidator) ValidateSprintClosure(_ context.Context, stories []ClosingStory) error {
	v.calls++
	data := metrics.SprintData{Stories: make([]metrics.CompletedStory, 0, len(stories))}
	for _, story := range stories {
		data.Stories = append(data.Stories, metrics.CompletedStory{
			ID:          story.ID,
			Status:      backlog.StoryStatus(story.Status),
			StoryPoints: story.StoryPoints,
			ActualHours: story.ActualHours,
		})
	}
	return metrics.ValidateSprintClosure(data)
}

func intPointer(value int) *int {
	return &value
}
