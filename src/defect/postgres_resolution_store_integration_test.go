package defect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// HU-19 (T011): fixtures de integración para resolver y reabrir defectos. Cada prueba crea dos
// proyectos con una historia cada uno; los Sprints y defectos se agregan según el escenario. Al
// terminar se borran las historias (los defectos caen en cascada), los Sprints y los proyectos.

type resolutionFixture struct {
	db                 *sql.DB
	projectA, projectB int64
	storyA, storyB     int64
	stamp              int64
}

// defectSnapshot es el estado completo de un defecto, para comprobar que no cambió.
type defectSnapshot struct {
	Status           string
	ResolutionSprint sql.NullInt64
	DetectionSprint  int64
	Description      string
	Severity         string
	StoryID          int64
	ProjectID        int64
}

func newResolutionFixture(t *testing.T) *resolutionFixture {
	t.Helper()
	db := openDefectTestDB(t)
	f := &resolutionFixture{db: db, stamp: time.Now().UnixNano()}
	f.projectA = f.insertID(t,
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU19 Resolución A %d", f.stamp),
	)
	f.projectB = f.insertID(t,
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU19 Resolución B %d", f.stamp),
	)
	t.Cleanup(func() {
		ctx := context.Background()
		for _, query := range []string{
			`DELETE FROM user_stories WHERE project_id IN ($1, $2)`,
			`DELETE FROM sprints WHERE project_id IN ($1, $2)`,
			`DELETE FROM projects WHERE id IN ($1, $2)`,
		} {
			if _, err := db.ExecContext(ctx, query, f.projectA, f.projectB); err != nil {
				t.Errorf("no se pudieron borrar los datos de prueba de HU-19: %v", err)
			}
		}
	})
	f.storyA = f.story(t, f.projectA)
	f.storyB = f.story(t, f.projectB)
	return f
}

func (f *resolutionFixture) insertID(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear un dato de prueba de HU-19: %v", err)
	}
	return id
}

// story crea una historia "Pendiente" en el proyecto.
func (f *resolutionFixture) story(t *testing.T, projectID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO user_stories (project_id, title, description, priority, status)
		 VALUES ($1, $2, 'Historia con defectos', 'Media', 'Pendiente') RETURNING id`,
		projectID, fmt.Sprintf("HU19 historia %d %d", projectID, f.stamp),
	)
}

// sprint crea un Sprint con el número y el estado ("Activo" o "Finalizado") indicados.
func (f *resolutionFixture) sprint(t *testing.T, projectID int64, number int, status string) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, $2, $3, '2026-01-01', '2026-01-14', $4) RETURNING id`,
		projectID, number, fmt.Sprintf("Objetivo HU19 %d", number), status,
	)
}

// openDefect crea un defecto "Abierto" detectado en el Sprint indicado.
func (f *resolutionFixture) openDefect(t *testing.T, projectID, storyID, detectionSprintID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		 VALUES ($1, $2, 'Defecto de HU-19', 'Alta', 'Abierto', $3) RETURNING id`,
		projectID, storyID, detectionSprintID,
	)
}

// resolvedDefect crea un defecto "Resuelto" con su Sprint de detección y de resolución.
func (f *resolutionFixture) resolvedDefect(t *testing.T, projectID, storyID, detectionSprintID, resolutionSprintID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		 VALUES ($1, $2, 'Defecto resuelto de HU-19', 'Media', 'Resuelto', $3, $4) RETURNING id`,
		projectID, storyID, detectionSprintID, resolutionSprintID,
	)
}

// snapshot lee el estado completo del defecto.
func (f *resolutionFixture) snapshot(t *testing.T, defectID int64) defectSnapshot {
	t.Helper()
	var s defectSnapshot
	if err := f.db.QueryRowContext(context.Background(),
		`SELECT status, resolution_sprint_id, detection_sprint_id, description, severity, story_id, project_id
		 FROM defects WHERE id = $1`,
		defectID,
	).Scan(&s.Status, &s.ResolutionSprint, &s.DetectionSprint, &s.Description, &s.Severity, &s.StoryID, &s.ProjectID); err != nil {
		t.Fatalf("no se pudo leer el defecto %d: %v", defectID, err)
	}
	return s
}

// assertUnchanged comprueba que el defecto conserva exactamente el estado anterior.
func (f *resolutionFixture) assertUnchanged(t *testing.T, defectID int64, before defectSnapshot) {
	t.Helper()
	if after := f.snapshot(t, defectID); after != before {
		t.Errorf("el defecto %d cambió: antes %+v, después %+v", defectID, before, after)
	}
}

// T012 [E1]: resolver registra el Sprint activo y no toca el resto del defecto.
func TestPostgresResolve_OK(t *testing.T) {
	f := newResolutionFixture(t)
	f.sprint(t, f.projectA, 2, "Finalizado")
	active := f.sprint(t, f.projectA, 3, "Activo")
	defectID := f.openDefect(t, f.projectA, f.storyA, active)
	before := f.snapshot(t, defectID)

	got, err := NewPostgreSQLResolutionStore(f.db).Resolve(context.Background(), f.projectA, defectID)

	if err != nil {
		t.Fatalf("Resolve() devolvió error: %v", err)
	}
	if got != (Resolved{DefectID: defectID, SprintNumber: 3}) {
		t.Errorf("Resolve() = %+v; se esperaba el defecto %d resuelto en el Sprint 3", got, defectID)
	}
	after := f.snapshot(t, defectID)
	if after.Status != string(StatusResolved) || !after.ResolutionSprint.Valid || after.ResolutionSprint.Int64 != active {
		t.Errorf("después de resolver: estado=%q Sprint de resolución=%v; se esperaba %q y %d",
			after.Status, after.ResolutionSprint, StatusResolved, active)
	}
	before.Status, before.ResolutionSprint = after.Status, after.ResolutionSprint
	if after != before {
		t.Errorf("resolver cambió otros datos del defecto: antes %+v, después %+v", before, after)
	}
}

// T013 [E2]: sin Sprint activo no se resuelve.
func TestPostgresResolve_SinSprintActivo(t *testing.T) {
	f := newResolutionFixture(t)
	finished := f.sprint(t, f.projectA, 1, "Finalizado")
	defectID := f.openDefect(t, f.projectA, f.storyA, finished)
	before := f.snapshot(t, defectID)

	_, err := NewPostgreSQLResolutionStore(f.db).Resolve(context.Background(), f.projectA, defectID)

	if !errors.Is(err, ErrNoActiveSprint) {
		t.Fatalf("Resolve() = %v; se esperaba ErrNoActiveSprint", err)
	}
	f.assertUnchanged(t, defectID, before)
}

// T013 [E2]: el Sprint activo de otro proyecto no sirve para resolver.
func TestPostgresResolve_SprintActivoDeOtroProyecto(t *testing.T) {
	f := newResolutionFixture(t)
	finished := f.sprint(t, f.projectA, 1, "Finalizado")
	f.sprint(t, f.projectB, 5, "Activo")
	defectID := f.openDefect(t, f.projectA, f.storyA, finished)
	before := f.snapshot(t, defectID)

	_, err := NewPostgreSQLResolutionStore(f.db).Resolve(context.Background(), f.projectA, defectID)

	if !errors.Is(err, ErrNoActiveSprint) {
		t.Fatalf("Resolve() = %v; se esperaba ErrNoActiveSprint", err)
	}
	f.assertUnchanged(t, defectID, before)
}

// T014 [E4]: detectado en el Sprint 3 (finalizado) con el Sprint 2 activo: se rechaza.
func TestPostgresResolve_Anterior(t *testing.T) {
	f := newResolutionFixture(t)
	detection := f.sprint(t, f.projectA, 3, "Finalizado")
	f.sprint(t, f.projectA, 2, "Activo")
	defectID := f.openDefect(t, f.projectA, f.storyA, detection)
	before := f.snapshot(t, defectID)

	_, err := NewPostgreSQLResolutionStore(f.db).Resolve(context.Background(), f.projectA, defectID)

	if !errors.Is(err, ErrResolutionBeforeDetection) {
		t.Fatalf("Resolve() = %v; se esperaba ErrResolutionBeforeDetection", err)
	}
	f.assertUnchanged(t, defectID, before)
}

// [E7]: resolver un defecto ya resuelto no cambia su Sprint de resolución.
func TestPostgresResolve_YaResuelto(t *testing.T) {
	f := newResolutionFixture(t)
	sprint2 := f.sprint(t, f.projectA, 2, "Finalizado")
	f.sprint(t, f.projectA, 3, "Activo")
	defectID := f.resolvedDefect(t, f.projectA, f.storyA, sprint2, sprint2)
	before := f.snapshot(t, defectID)

	_, err := NewPostgreSQLResolutionStore(f.db).Resolve(context.Background(), f.projectA, defectID)

	if !errors.Is(err, ErrAlreadyResolved) {
		t.Fatalf("Resolve() = %v; se esperaba ErrAlreadyResolved", err)
	}
	f.assertUnchanged(t, defectID, before)
}

// T015 [E5]: un identificador que no existe da ErrDefectNotFound al resolver y al reabrir.
func TestPostgresResolve_Inexistente(t *testing.T) {
	f := newResolutionFixture(t)
	f.sprint(t, f.projectA, 1, "Activo")
	store := NewPostgreSQLResolutionStore(f.db)
	const missing = int64(9_000_000_000)

	if _, err := store.Resolve(context.Background(), f.projectA, missing); !errors.Is(err, ErrDefectNotFound) {
		t.Errorf("Resolve() = %v; se esperaba ErrDefectNotFound", err)
	}
	if err := store.Reopen(context.Background(), f.projectA, missing); !errors.Is(err, ErrDefectNotFound) {
		t.Errorf("Reopen() = %v; se esperaba ErrDefectNotFound", err)
	}
}

// T015 [E5]: un defecto de otro proyecto se trata como inexistente y no se modifica.
func TestPostgresResolve_OtroProyecto(t *testing.T) {
	f := newResolutionFixture(t)
	f.sprint(t, f.projectA, 1, "Activo")
	sprintB := f.sprint(t, f.projectB, 1, "Activo")
	openB := f.openDefect(t, f.projectB, f.storyB, sprintB)
	resolvedB := f.resolvedDefect(t, f.projectB, f.storyB, sprintB, sprintB)
	beforeOpen, beforeResolved := f.snapshot(t, openB), f.snapshot(t, resolvedB)
	store := NewPostgreSQLResolutionStore(f.db)

	if _, err := store.Resolve(context.Background(), f.projectA, openB); !errors.Is(err, ErrDefectNotFound) {
		t.Errorf("Resolve() de un defecto de otro proyecto = %v; se esperaba ErrDefectNotFound", err)
	}
	if err := store.Reopen(context.Background(), f.projectA, resolvedB); !errors.Is(err, ErrDefectNotFound) {
		t.Errorf("Reopen() de un defecto de otro proyecto = %v; se esperaba ErrDefectNotFound", err)
	}
	f.assertUnchanged(t, openB, beforeOpen)
	f.assertUnchanged(t, resolvedB, beforeResolved)
}

// T016 [E6]: si falla el UPDATE, el defecto conserva su estado y su Sprint de resolución.
func TestPostgresResolve_FalloNoDejaCambios(t *testing.T) {
	f := newResolutionFixture(t)
	active := f.sprint(t, f.projectA, 1, "Activo")
	openID := f.openDefect(t, f.projectA, f.storyA, active)
	resolvedID := f.resolvedDefect(t, f.projectA, f.storyA, active, active)
	beforeOpen, beforeResolved := f.snapshot(t, openID), f.snapshot(t, resolvedID)
	injectDefectUpdateFailure(t, f, openID, resolvedID)
	store := NewPostgreSQLResolutionStore(f.db)

	_, err := store.Resolve(context.Background(), f.projectA, openID)
	if err == nil || !strings.Contains(err.Error(), "injected HU-19 failure") {
		t.Errorf("Resolve() = %v; se esperaba la falla inyectada en el UPDATE", err)
	}
	err = store.Reopen(context.Background(), f.projectA, resolvedID)
	if err == nil || !strings.Contains(err.Error(), "injected HU-19 failure") {
		t.Errorf("Reopen() = %v; se esperaba la falla inyectada en el UPDATE", err)
	}
	f.assertUnchanged(t, openID, beforeOpen)
	f.assertUnchanged(t, resolvedID, beforeResolved)
}

// T017 [E3]: reabrir vuelve a "Abierto" y borra el Sprint de resolución, también sin Sprint activo.
func TestPostgresReopen_OK(t *testing.T) {
	f := newResolutionFixture(t)
	sprint2 := f.sprint(t, f.projectA, 2, "Finalizado")
	defectID := f.resolvedDefect(t, f.projectA, f.storyA, sprint2, sprint2)
	before := f.snapshot(t, defectID)

	if err := NewPostgreSQLResolutionStore(f.db).Reopen(context.Background(), f.projectA, defectID); err != nil {
		t.Fatalf("Reopen() devolvió error: %v", err)
	}

	after := f.snapshot(t, defectID)
	if after.Status != string(StatusOpen) || after.ResolutionSprint.Valid {
		t.Errorf("después de reabrir: estado=%q Sprint de resolución=%v; se esperaba %q y NULL",
			after.Status, after.ResolutionSprint, StatusOpen)
	}
	before.Status, before.ResolutionSprint = after.Status, after.ResolutionSprint
	if after != before {
		t.Errorf("reabrir cambió otros datos del defecto: antes %+v, después %+v", before, after)
	}
}

// [E8]: reabrir un defecto abierto no lo modifica.
func TestPostgresReopen_YaAbierto(t *testing.T) {
	f := newResolutionFixture(t)
	active := f.sprint(t, f.projectA, 1, "Activo")
	defectID := f.openDefect(t, f.projectA, f.storyA, active)
	before := f.snapshot(t, defectID)

	err := NewPostgreSQLResolutionStore(f.db).Reopen(context.Background(), f.projectA, defectID)

	if !errors.Is(err, ErrAlreadyOpen) {
		t.Fatalf("Reopen() = %v; se esperaba ErrAlreadyOpen", err)
	}
	f.assertUnchanged(t, defectID, before)
}

// T018: dos resoluciones simultáneas del mismo defecto: una termina bien y la otra informa que ya
// está resuelto.
func TestPostgresResolve_Concurrente(t *testing.T) {
	f := newResolutionFixture(t)
	active := f.sprint(t, f.projectA, 1, "Activo")
	defectID := f.openDefect(t, f.projectA, f.storyA, active)
	store := NewPostgreSQLResolutionStore(f.db)

	const attempts = 2
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.Resolve(context.Background(), f.projectA, defectID)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	ok, already := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrAlreadyResolved):
			already++
		default:
			t.Errorf("resolución concurrente con error inesperado: %v", err)
		}
	}
	if ok != 1 || already != 1 {
		t.Errorf("resoluciones correctas=%d ya resueltas=%d; se esperaba 1 y 1", ok, already)
	}
	after := f.snapshot(t, defectID)
	if after.Status != string(StatusResolved) || !after.ResolutionSprint.Valid || after.ResolutionSprint.Int64 != active {
		t.Errorf("estado final = %q con Sprint %v; se esperaba %q en el Sprint %d",
			after.Status, after.ResolutionSprint, StatusResolved, active)
	}
}

// injectDefectUpdateFailure hace fallar cualquier UPDATE de los defectos indicados y lo deshace al
// terminar la prueba (antes de la limpieza de los fixtures).
func injectDefectUpdateFailure(t *testing.T, f *resolutionFixture, defectIDs ...int64) {
	t.Helper()
	ids := make([]string, 0, len(defectIDs))
	for _, id := range defectIDs {
		ids = append(ids, fmt.Sprint(id))
	}
	suffix := fmt.Sprint(f.stamp)
	functionName := "hu19_fail_fn_" + suffix
	triggerName := "hu19_fail_" + suffix
	if _, err := f.db.Exec(fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger
		LANGUAGE plpgsql
		AS $body$
		BEGIN
			RAISE EXCEPTION 'injected HU-19 failure';
		END;
		$body$`, functionName)); err != nil {
		t.Fatalf("no se pudo crear la función de falla: %v", err)
	}
	if _, err := f.db.Exec(fmt.Sprintf(`
		CREATE TRIGGER %s BEFORE UPDATE ON defects
		FOR EACH ROW WHEN (OLD.id IN (%s))
		EXECUTE FUNCTION %s()`, triggerName, strings.Join(ids, ", "), functionName)); err != nil {
		_, _ = f.db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName))
		t.Fatalf("no se pudo crear el trigger de falla: %v", err)
	}
	t.Cleanup(func() {
		if _, err := f.db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON defects`, triggerName)); err != nil {
			t.Errorf("no se pudo quitar el trigger de falla: %v", err)
		}
		if _, err := f.db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName)); err != nil {
			t.Errorf("no se pudo quitar la función de falla: %v", err)
		}
	})
}
