package metrics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Integración de PostgreSQLQualityStore (HU-10, T011–T016) contra PostgreSQL real, con datos propios
// que se borran al terminar.

// qualityFixture son los datos de T011:
//   - proyectos «Alfa» y «Beta»; en «Alfa», Sprints 1 y 2 "Finalizado" y Sprint 3 "Activo";
//   - Sprint 1: historias de 3, 5, 8 y 2 Story Points, "Hecho" las de 3 y 5;
//   - Sprint 2: tres "Hecho" de 1 Story Point y una sin estimar que volvió al Backlog;
//   - defectos de «Alfa»: Sprint 2 → 3 detectados y 2 resueltos; Sprint 1 → 2 detectados (uno de
//     ellos reabierto) y 0 resueltos;
//   - un defecto de «Beta» que apunta al Sprint 2 de «Alfa» y no debe contarse.
type qualityFixture struct {
	db    *sql.DB
	store *PostgreSQLQualityStore

	alfaID, betaID             int64
	sprint1, sprint2, sprint3  int64
	sprint1Stories             []int64 // en orden de id: 3, 5, 8 y 2 Story Points
	sprint2Stories             []int64 // en orden de id: tres de 1 Story Point y una sin estimar
	sprint3Story, betaStory    int64
	reopenedDefect, betaDefect int64
}

func openQualityTestDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL no está definida; se omiten las pruebas de integración de HU-10")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL no está disponible: %v", err)
	}
	var missing string
	if err := db.QueryRow(`
		SELECT COALESCE(string_agg(required.name, ', '), '')
		FROM unnest(ARRAY['sprints', 'sprint_story_assignments', 'sprint_story_results', 'sprint_metrics', 'defects']::text[]) AS required(name)
		WHERE to_regclass('public.' || required.name) IS NULL`).Scan(&missing); err != nil {
		t.Fatalf("no se pudo verificar el esquema: %v", err)
	}
	if missing != "" {
		t.Skipf("faltan tablas (¿se aplicaron las migraciones?): %s", missing)
	}
	return db
}

func newQualityFixture(t *testing.T) *qualityFixture {
	t.Helper()
	db := openQualityTestDB(t)
	f := &qualityFixture{db: db, store: NewPostgreSQLQualityStore(db)}
	stamp := time.Now().UnixNano()

	var projectIDs []int64
	t.Cleanup(func() { cleanupQualityFixture(t, db, projectIDs) })

	insertProject := func(name string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO projects (name, start_date, end_date, status)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31', 'Activo') RETURNING id`,
			fmt.Sprintf("HU10 %s %d", name, stamp)).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear el proyecto %q: %v", name, err)
		}
		projectIDs = append(projectIDs, id)
		return id
	}

	f.alfaID = insertProject("Alfa")
	f.betaID = insertProject("Beta")
	f.sprint1 = f.finishedSprint(t, 1)
	f.sprint2 = f.finishedSprint(t, 2)
	f.sprint3 = insertQualitySprint(t, db, f.alfaID, 3, "Activo")

	f.sprint1Stories = []int64{
		f.closedStory(t, f.sprint1, "S1 3 SP", intPointer(3), "Hecho"),
		f.closedStory(t, f.sprint1, "S1 5 SP", intPointer(5), "Hecho"),
		f.closedStory(t, f.sprint1, "S1 8 SP", intPointer(8), "En progreso"),
		f.closedStory(t, f.sprint1, "S1 2 SP", intPointer(2), "En Sprint"),
	}
	f.sprint2Stories = []int64{
		f.closedStory(t, f.sprint2, "S2 1 SP a", intPointer(1), "Hecho"),
		f.closedStory(t, f.sprint2, "S2 1 SP b", intPointer(1), "Hecho"),
		f.closedStory(t, f.sprint2, "S2 1 SP c", intPointer(1), "Hecho"),
		f.closedStory(t, f.sprint2, "S2 sin estimar", nil, "En Sprint"),
	}

	// El Sprint 3 sigue activo: su historia está asignada y no hay registro de cierre.
	f.sprint3Story = insertQualityStory(t, db, f.alfaID, "S3 en curso", intPointer(5), "En Sprint", f.sprint3)
	assignQualityStory(t, db, f.sprint3, f.sprint3Story)

	f.betaStory = insertQualityStory(t, db, f.betaID, "Beta historia", intPointer(3), "Pendiente", 0)

	defectStory := f.sprint2Stories[0]
	// Tres detectados en el Sprint 2; uno de ellos resuelto en el mismo Sprint.
	insertQualityDefect(t, db, f.alfaID, defectStory, f.sprint2, f.sprint2)
	insertQualityDefect(t, db, f.alfaID, defectStory, f.sprint2, 0)
	insertQualityDefect(t, db, f.alfaID, defectStory, f.sprint2, 0)
	// Detectado en el Sprint 1 y resuelto en el 2.
	insertQualityDefect(t, db, f.alfaID, f.sprint1Stories[0], f.sprint1, f.sprint2)
	// Detectado en el Sprint 1, resuelto y reabierto: vuelve a 'Abierto' sin Sprint de resolución.
	f.reopenedDefect = insertQualityDefect(t, db, f.alfaID, f.sprint1Stories[1], f.sprint1, f.sprint2)
	if _, err := db.Exec(`UPDATE defects SET status = 'Abierto', resolution_sprint_id = NULL WHERE id = $1`, f.reopenedDefect); err != nil {
		t.Fatalf("no se pudo reabrir el defecto: %v", err)
	}
	// Defecto de «Beta» cuyos Sprints de detección y resolución son el Sprint 2 de «Alfa».
	f.betaDefect = insertQualityDefect(t, db, f.betaID, f.betaStory, f.sprint2, f.sprint2)
	return f
}

// cleanupQualityFixture borra, en este orden: defectos, registro de cierre, asignaciones,
// sprint_metrics, historias, Sprints y proyectos.
func cleanupQualityFixture(t *testing.T, db *sql.DB, projectIDs []int64) {
	t.Helper()
	if len(projectIDs) == 0 {
		return
	}
	const sprintsOfProjects = `SELECT id FROM sprints WHERE project_id = ANY($1)`
	steps := []struct{ name, query string }{
		{"defectos", `DELETE FROM defects WHERE project_id = ANY($1)
			OR detection_sprint_id IN (` + sprintsOfProjects + `)
			OR resolution_sprint_id IN (` + sprintsOfProjects + `)`},
		{"registro de cierre", `DELETE FROM sprint_story_results WHERE sprint_id IN (` + sprintsOfProjects + `)`},
		{"asignaciones", `DELETE FROM sprint_story_assignments WHERE sprint_id IN (` + sprintsOfProjects + `)`},
		{"sprint_metrics", `DELETE FROM sprint_metrics WHERE sprint_id IN (` + sprintsOfProjects + `)`},
		{"historias", `DELETE FROM user_stories WHERE project_id = ANY($1)`},
		{"Sprints", `DELETE FROM sprints WHERE project_id = ANY($1)`},
		{"proyectos", `DELETE FROM projects WHERE id = ANY($1)`},
	}
	for _, step := range steps {
		if _, err := db.Exec(step.query, projectIDs); err != nil {
			t.Errorf("no se pudieron borrar los %s de la fixture: %v", step.name, err)
		}
	}
}

func insertQualitySprint(t *testing.T, db *sql.DB, projectID int64, number int, status string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, DATE '2026-10-01', DATE '2026-10-15', $4) RETURNING id`,
		projectID, number, fmt.Sprintf("Sprint %d HU-10", number), status).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear el Sprint %d: %v", number, err)
	}
	return id
}

// insertQualityStory crea una historia; activeSprintID 0 la deja sin Sprint.
func insertQualityStory(t *testing.T, db *sql.DB, projectID int64, title string, points *int, status string, activeSprintID int64) int64 {
	t.Helper()
	var active sql.NullInt64
	if activeSprintID != 0 {
		active = sql.NullInt64{Int64: activeSprintID, Valid: true}
	}
	var id int64
	if err := db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status, estimated_hours, active_sprint_id)
		VALUES ($1, $2, 'Historia de prueba HU-10', 'Media', $3, $4, 4, $5) RETURNING id`,
		projectID, "HU10 "+title, points, status, active).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear la historia %q: %v", title, err)
	}
	return id
}

func assignQualityStory(t *testing.T, db *sql.DB, sprintID, storyID int64) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID); err != nil {
		t.Fatalf("no se pudo asignar la historia %d al Sprint %d: %v", storyID, sprintID, err)
	}
}

// insertQualityDefect crea un defecto detectado en detectionSprintID; con resolutionSprintID distinto
// de 0 queda 'Resuelto' en ese Sprint.
func insertQualityDefect(t *testing.T, db *sql.DB, projectID, storyID, detectionSprintID, resolutionSprintID int64) int64 {
	t.Helper()
	status, resolution := "Abierto", sql.NullInt64{}
	if resolutionSprintID != 0 {
		status, resolution = "Resuelto", sql.NullInt64{Int64: resolutionSprintID, Valid: true}
	}
	var id int64
	if err := db.QueryRow(`
		INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		VALUES ($1, $2, 'Defecto de prueba HU-10', 'Media', $3, $4, $5) RETURNING id`,
		projectID, storyID, status, detectionSprintID, resolution).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear el defecto: %v", err)
	}
	return id
}

// finishedSprint crea un Sprint "Finalizado" de «Alfa» con el número indicado.
func (f *qualityFixture) finishedSprint(t *testing.T, number int) int64 {
	t.Helper()
	return insertQualitySprint(t, f.db, f.alfaID, number, "Finalizado")
}

// closedStory crea una historia de «Alfa» como la deja el cierre de HU-15: una fila en
// sprint_story_results con su estado final; si terminó "Hecho" conserva la asignación y el Sprint,
// y si no, vuelve al Backlog como "Pendiente" sin Sprint.
func (f *qualityFixture) closedStory(t *testing.T, sprintID int64, title string, points *int, finalStatus string) int64 {
	t.Helper()
	var id int64
	if finalStatus == "Hecho" {
		id = insertQualityStory(t, f.db, f.alfaID, title, points, "Hecho", sprintID)
		assignQualityStory(t, f.db, sprintID, id)
	} else {
		id = insertQualityStory(t, f.db, f.alfaID, title, points, "Pendiente", 0)
	}
	if _, err := f.db.Exec(`
		INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		VALUES ($1, $2, $3, $4)`, sprintID, id, finalStatus, points); err != nil {
		t.Fatalf("no se pudo registrar el cierre de la historia %q: %v", title, err)
	}
	return id
}

func (f *qualityFixture) load(t *testing.T, projectID, sprintID int64) QualityData {
	t.Helper()
	data, err := f.store.LoadSprintQuality(context.Background(), projectID, sprintID)
	if err != nil {
		t.Fatalf("LoadSprintQuality(%d, %d) devolvió error: %v", projectID, sprintID, err)
	}
	return data
}

func (f *qualityFixture) quality(t *testing.T, sprintID int64) SprintQuality {
	t.Helper()
	result, err := NewQualityService(f.store).SprintQuality(context.Background(), f.alfaID, sprintID)
	if err != nil {
		t.Fatalf("SprintQuality(%d, %d) devolvió error: %v", f.alfaID, sprintID, err)
	}
	return result
}

// T012: métricas de historias, porcentajes y advertencias.
func TestPostgresQuality_Historias(t *testing.T) {
	f := newQualityFixture(t)

	t.Run("Sprint 2: 3 de 4 historias completadas da 75.00", func(t *testing.T) {
		got := f.quality(t, f.sprint2)
		if got.StoriesPlanned != 4 || got.StoriesCompleted != 3 || got.CompletionPercentage != "75.00" {
			t.Errorf("Sprint 2 = %d de %d, %q; se esperaba 3 de 4, \"75.00\"", got.StoriesCompleted, got.StoriesPlanned, got.CompletionPercentage)
		}
	})

	t.Run("Sprint sin historias Hecho da 0.00", func(t *testing.T) {
		sprintID := f.finishedSprint(t, 4)
		f.closedStory(t, sprintID, "S4 en progreso", intPointer(5), "En progreso")
		f.closedStory(t, sprintID, "S4 en Sprint", intPointer(3), "En Sprint")
		got := f.quality(t, sprintID)
		if got.StoriesPlanned != 2 || got.StoriesCompleted != 0 || got.CompletionPercentage != "0.00" {
			t.Errorf("Sprint sin Hecho = %d de %d, %q; se esperaba 0 de 2, \"0.00\"", got.StoriesCompleted, got.StoriesPlanned, got.CompletionPercentage)
		}
		if got.PointsPlanned != 8 || got.PointsCompleted != 0 {
			t.Errorf("Story Points = %d planificados y %d completados; se esperaban 8 y 0", got.PointsPlanned, got.PointsCompleted)
		}
	})

	t.Run("Sprint finalizado sin registro de cierre da No disponible", func(t *testing.T) {
		sprintID := f.finishedSprint(t, 5)
		got := f.quality(t, sprintID)
		want := SprintQuality{SprintID: sprintID, CompletionPercentage: "No disponible"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Sprint sin registro\n got  %+v\n want %+v", got, want)
		}
	})

	t.Run("Sprint 1: 3, 5, 8 y 2 Story Points dan 18 planificados y 8 completados", func(t *testing.T) {
		got := f.quality(t, f.sprint1)
		if got.PointsPlanned != 18 || got.PointsCompleted != 8 {
			t.Errorf("Story Points = %d planificados y %d completados; se esperaban 18 y 8", got.PointsPlanned, got.PointsCompleted)
		}
		if got.StoriesPlanned != 4 || got.StoriesCompleted != 2 || got.CompletionPercentage != "50.00" || got.StoriesWithoutPoints != 0 {
			t.Errorf("Sprint 1 = %+v; se esperaban 2 de 4, \"50.00\" y ninguna sin Story Points", got)
		}
	})

	t.Run("Sprint 1: el registro de cierre se lee completo y ordenado por historia", func(t *testing.T) {
		got := f.load(t, f.alfaID, f.sprint1)
		want := []QualityStoryResult{
			{FinalStatus: "Hecho", StoryPoints: intPointer(3)},
			{FinalStatus: "Hecho", StoryPoints: intPointer(5)},
			{FinalStatus: "En progreso", StoryPoints: intPointer(8)},
			{FinalStatus: "En Sprint", StoryPoints: intPointer(2)},
		}
		if got.SprintID != f.sprint1 || got.Status != "Finalizado" || !reflect.DeepEqual(got.Stories, want) {
			t.Errorf("LoadSprintQuality(Sprint 1)\n got  %+v\n want SprintID %d, Finalizado, %+v", got, f.sprint1, want)
		}
	})

	t.Run("Sprint 2: una historia sin Story Points da advertencia de 1 y 3 planificados", func(t *testing.T) {
		got := f.quality(t, f.sprint2)
		if got.StoriesWithoutPoints != 1 || got.PointsPlanned != 3 || got.PointsCompleted != 3 {
			t.Errorf("Sprint 2 = %d sin Story Points, %d planificados y %d completados; se esperaban 1, 3 y 3",
				got.StoriesWithoutPoints, got.PointsPlanned, got.PointsCompleted)
		}
	})
}

// T013: conteo de defectos detectados y resueltos, incluido el reabierto.
func TestPostgresQuality_Defectos(t *testing.T) {
	f := newQualityFixture(t)

	t.Run("Sprint 2: 3 detectados y 2 resueltos", func(t *testing.T) {
		got := f.load(t, f.alfaID, f.sprint2)
		if got.DefectsDetected != 3 || got.DefectsResolved != 2 {
			t.Errorf("Sprint 2 = %d detectados y %d resueltos; se esperaban 3 y 2", got.DefectsDetected, got.DefectsResolved)
		}
		if quality := f.quality(t, f.sprint2); quality.DefectsDetected != 3 || quality.DefectsResolved != 2 {
			t.Errorf("SprintQuality(Sprint 2) = %d detectados y %d resueltos; se esperaban 3 y 2", quality.DefectsDetected, quality.DefectsResolved)
		}
	})

	t.Run("Sprint sin defectos da 0 y 0", func(t *testing.T) {
		sprintID := f.finishedSprint(t, 6)
		f.closedStory(t, sprintID, "S6 Hecho", intPointer(2), "Hecho")
		got := f.load(t, f.alfaID, sprintID)
		if got.DefectsDetected != 0 || got.DefectsResolved != 0 {
			t.Errorf("Sprint sin defectos = %d detectados y %d resueltos; se esperaban 0 y 0", got.DefectsDetected, got.DefectsResolved)
		}
	})

	t.Run("el reabierto cuenta como detectado en su Sprint y no como resuelto", func(t *testing.T) {
		got := f.load(t, f.alfaID, f.sprint1)
		if got.DefectsDetected != 2 || got.DefectsResolved != 0 {
			t.Errorf("Sprint 1 = %d detectados y %d resueltos; se esperaban 2 (incluido el reabierto) y 0",
				got.DefectsDetected, got.DefectsResolved)
		}
		// Estaba resuelto en el Sprint 2; reabierto, ya no suma a los 2 resueltos de ese Sprint.
		if sprint2 := f.load(t, f.alfaID, f.sprint2); sprint2.DefectsResolved != 2 {
			t.Errorf("Sprint 2 = %d resueltos; el defecto reabierto no debía contarse (se esperaban 2)", sprint2.DefectsResolved)
		}
	})
}

// T014: aislamiento por proyecto.
func TestPostgresQuality_OtroProyecto(t *testing.T) {
	f := newQualityFixture(t)

	t.Run("un Sprint de Alfa pedido con el proyecto Beta da ErrSprintNotFound", func(t *testing.T) {
		for _, sprintID := range []int64{f.sprint1, f.sprint2, f.sprint3} {
			_, err := f.store.LoadSprintQuality(context.Background(), f.betaID, sprintID)
			if !errors.Is(err, ErrSprintNotFound) {
				t.Errorf("LoadSprintQuality(Beta, %d): error = %v; se esperaba ErrSprintNotFound", sprintID, err)
			}
			if _, err := NewQualityService(f.store).SprintQuality(context.Background(), f.betaID, sprintID); !errors.Is(err, ErrSprintNotFound) {
				t.Errorf("SprintQuality(Beta, %d): error = %v; se esperaba ErrSprintNotFound", sprintID, err)
			}
		}
	})

	t.Run("un defecto de Beta que apunta a un Sprint de Alfa no cuenta", func(t *testing.T) {
		var detection, resolution int64
		if err := f.db.QueryRow(`SELECT detection_sprint_id, resolution_sprint_id FROM defects WHERE id = $1 AND project_id = $2`,
			f.betaDefect, f.betaID).Scan(&detection, &resolution); err != nil || detection != f.sprint2 || resolution != f.sprint2 {
			t.Fatalf("la fixture debía tener un defecto de Beta en el Sprint 2 de Alfa: detección %d, resolución %d, error %v", detection, resolution, err)
		}
		got := f.load(t, f.alfaID, f.sprint2)
		if got.DefectsDetected != 3 || got.DefectsResolved != 2 {
			t.Errorf("Sprint 2 de Alfa = %d detectados y %d resueltos; se esperaban 3 y 2 sin el defecto de Beta",
				got.DefectsDetected, got.DefectsResolved)
		}
	})
}

// T015: Sprint activo e identificador inexistente.
func TestPostgresQuality_SprintActivoEInexistente(t *testing.T) {
	f := newQualityFixture(t)

	t.Run("el Sprint 3 activo devuelve solo su estado y el servicio responde ErrSprintNotFinished", func(t *testing.T) {
		got := f.load(t, f.alfaID, f.sprint3)
		want := QualityData{SprintID: f.sprint3, Status: "Activo"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("LoadSprintQuality(Sprint 3)\n got  %+v\n want %+v", got, want)
		}
		result, err := NewQualityService(f.store).SprintQuality(context.Background(), f.alfaID, f.sprint3)
		if !errors.Is(err, ErrSprintNotFinished) {
			t.Errorf("SprintQuality(Sprint 3): error = %v; se esperaba ErrSprintNotFinished", err)
		}
		if !reflect.DeepEqual(result, SprintQuality{}) {
			t.Errorf("no debía devolver métricas de un Sprint activo: %+v", result)
		}
	})

	t.Run("el identificador 2147483647 da ErrSprintNotFound", func(t *testing.T) {
		_, err := f.store.LoadSprintQuality(context.Background(), f.alfaID, 2147483647)
		if !errors.Is(err, ErrSprintNotFound) {
			t.Errorf("LoadSprintQuality(2147483647): error = %v; se esperaba ErrSprintNotFound", err)
		}
		if _, err := NewQualityService(f.store).SprintQuality(context.Background(), f.alfaID, 2147483647); !errors.Is(err, ErrSprintNotFound) {
			t.Errorf("SprintQuality(2147483647): error = %v; se esperaba ErrSprintNotFound", err)
		}
	})
}

// T016: PointsCompleted coincide con la velocidad que calcula HU-07 para el mismo Sprint.
func TestQuality_CoherenciaConVelocidadHU07(t *testing.T) {
	f := newQualityFixture(t)
	t.Cleanup(func() {
		if _, err := f.db.Exec(`DELETE FROM sprint_metrics WHERE sprint_id = $1`, f.sprint1); err != nil {
			t.Errorf("no se pudo borrar la fila de sprint_metrics que guardó HU-07: %v", err)
		}
	})

	metrics, err := NewService(NewPostgreSQLMetricsStore(f.db)).CalculateMetrics(f.sprint1)
	if err != nil {
		t.Fatalf("HU-07 CalculateMetrics(Sprint 1) devolvió error: %v", err)
	}
	quality := f.quality(t, f.sprint1)
	if quality.PointsCompleted != metrics.Velocity {
		t.Errorf("PointsCompleted = %d; la velocidad de HU-07 es %d y debían coincidir", quality.PointsCompleted, metrics.Velocity)
	}
	if metrics.Velocity != 8 {
		t.Errorf("velocidad de HU-07 = %d; se esperaba 8 (historias Hecho de 3 y 5 Story Points)", metrics.Velocity)
	}
}
