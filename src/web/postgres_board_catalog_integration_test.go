package web

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T009): PostgreSQLBoardCatalog contra PostgreSQL real. Cada prueba crea sus propios
// proyectos, Sprints, historias y horas, y los elimina al terminar.

const (
	catPending    = sprint.StoryStatus("Pendiente")
	catEnSprint   = sprint.StoryStatus("En Sprint")
	catInProgress = sprint.StoryStatus("En progreso")
	catDone       = sprint.StoryStatus("Hecho")
)

// newCatProject crea un proyecto y, al terminar, borra sus horas, historias y el proyecto
// (sus Sprints y asignaciones caen en cascada).
func newCatProject(t *testing.T, db *sql.DB, label string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, DATE '2026-10-01', DATE '2026-12-31', 'Activo')
		RETURNING id`,
		fmt.Sprintf("HU14 catalogo %s %d", label, time.Now().UnixNano()),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba: %v", err)
	}
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM projects WHERE id = $1`,
		} {
			if _, err := db.Exec(query, id); err != nil {
				t.Errorf("no se pudo limpiar el proyecto de prueba %d: %v", id, err)
			}
		}
	})
	return id
}

// newCatSprint crea un Sprint del proyecto con el estado indicado ("Activo" o "Finalizado").
func newCatSprint(t *testing.T, db *sql.DB, projectID int64, status string) (id int64, number int) {
	t.Helper()
	if err := db.QueryRow(
		`SELECT COALESCE(MAX(number), 0) + 1 FROM sprints WHERE project_id = $1`, projectID,
	).Scan(&number); err != nil {
		t.Fatalf("no se pudo calcular el número de Sprint: %v", err)
	}
	err := db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, 'Objetivo del Tablero', DATE '2026-10-01', DATE '2026-10-15', $3)
		RETURNING id`,
		projectID, number, status,
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el Sprint de prueba: %v", err)
	}
	return id, number
}

// newCatStory crea una historia; si sprintID es mayor que cero, la asigna a ese Sprint.
func newCatStory(t *testing.T, db *sql.DB, projectID, sprintID int64, title string, points *int, status sprint.StoryStatus) int64 {
	t.Helper()
	var storyPoints sql.NullInt64
	if points != nil {
		storyPoints = sql.NullInt64{Int64: int64(*points), Valid: true}
	}
	var id int64
	err := db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
		VALUES ($1, $2, 'Historia para probar el Tablero', 'Media', $3, $4, $5)
		RETURNING id`,
		projectID, title, storyPoints, string(status), sql.NullInt64{Int64: sprintID, Valid: sprintID > 0},
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear la historia de prueba %q: %v", title, err)
	}
	if sprintID > 0 {
		if _, err := db.Exec(
			`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id,
		); err != nil {
			t.Fatalf("no se pudo asignar la historia al Sprint: %v", err)
		}
	}
	return id
}

// addCatEffort registra horas de un integrante en la historia (HU-03).
func addCatEffort(t *testing.T, db *sql.DB, storyID int64) {
	t.Helper()
	var memberID int64
	if err := db.QueryRow(
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("Integrante catalogo %d", time.Now().UnixNano()),
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, memberID); err != nil {
			t.Errorf("no se pudieron limpiar las horas de prueba: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM members WHERE id = $1`, memberID); err != nil {
			t.Errorf("no se pudo limpiar el integrante de prueba: %v", err)
		}
	})
	if _, err := db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, DATE '2026-10-05', 'Trabajo de HU-14', 2)`,
		storyID, memberID,
	); err != nil {
		t.Fatalf("no se pudieron registrar las horas de prueba: %v", err)
	}
}

func catStoryIDs(g BoardGroup) []int64 {
	ids := make([]int64, 0, len(g.Stories))
	for _, s := range g.Stories {
		ids = append(ids, s.ID)
	}
	return ids
}

func catSamePoints(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestPostgreSQLBoardCatalog_SprintActivoAgrupadoPorEstado(t *testing.T) {
	db := openContextTestDB(t)
	catalog := NewPostgreSQLBoardCatalog(db)
	ctx := context.Background()
	projectA := newCatProject(t, db, "A")
	projectB := newCatProject(t, db, "B")

	// Proyecto A: un Sprint finalizado (no debe aparecer) y el Sprint activo (el número 2).
	oldSprint, _ := newCatSprint(t, db, projectA, "Finalizado")
	newCatStory(t, db, projectA, oldSprint, "Historia del Sprint finalizado", boardPuntos(21), catDone)
	activeSprint, number := newCatSprint(t, db, projectA, "Activo")
	s1 := newCatStory(t, db, projectA, activeSprint, "Historia En Sprint", boardPuntos(3), catEnSprint)
	s2 := newCatStory(t, db, projectA, activeSprint, "Historia En progreso", boardPuntos(5), catInProgress)
	addCatEffort(t, db, s2)
	s3 := newCatStory(t, db, projectA, activeSprint, "Historia Hecho con puntos", boardPuntos(8), catDone)
	s4 := newCatStory(t, db, projectA, activeSprint, "Historia Hecho sin puntos", nil, catDone)
	newCatStory(t, db, projectA, 0, "Historia sin Sprint", boardPuntos(2), catPending)

	// Proyecto B: otro Sprint activo con una historia propia.
	sprintB, numberB := newCatSprint(t, db, projectB, "Activo")
	storyB := newCatStory(t, db, projectB, sprintB, "Historia de otro proyecto", boardPuntos(13), catDone)

	t.Run("proyecto A", func(t *testing.T) {
		view, found, err := catalog.ActiveBoard(ctx, projectA)
		if err != nil || !found {
			t.Fatalf("ActiveBoard() = (found %v, error %v), se esperaba el Sprint activo", found, err)
		}
		if view.ID != activeSprint || view.Number != int64(number) || view.Label != fmt.Sprintf("Sprint %d", number) {
			t.Errorf("Sprint = (%d, %d, %q), se esperaba (%d, %d, %q)",
				view.ID, view.Number, view.Label, activeSprint, number, fmt.Sprintf("Sprint %d", number))
		}
		if view.Goal != "Objetivo del Tablero" {
			t.Errorf("Goal = %q", view.Goal)
		}
		if got := view.StartDate.Format("2006-01-02"); got != "2026-10-01" {
			t.Errorf("StartDate = %s, se esperaba 2026-10-01", got)
		}
		if got := view.EndDate.Format("2006-01-02"); got != "2026-10-15" {
			t.Errorf("EndDate = %s, se esperaba 2026-10-15", got)
		}

		// Tres grupos, en este orden, con las historias ordenadas por id.
		wantStatuses := []sprint.StoryStatus{catEnSprint, catInProgress, catDone}
		wantIDs := [][]int64{{s1}, {s2}, {s3, s4}}
		if len(view.Groups) != 3 {
			t.Fatalf("grupos = %d, se esperaban 3", len(view.Groups))
		}
		for i, g := range view.Groups {
			if g.Status != wantStatuses[i] {
				t.Errorf("grupo %d: estado %q, se esperaba %q", i, g.Status, wantStatuses[i])
			}
			if got := catStoryIDs(g); !slices.Equal(got, wantIDs[i]) {
				t.Errorf("grupo %q: historias %v, se esperaban %v", g.Status, got, wantIDs[i])
			}
		}

		// Datos de cada historia: puntos, horas y la única acción válida de su estado.
		byID := map[int64]BoardStory{}
		for _, g := range view.Groups {
			for _, st := range g.Stories {
				byID[st.ID] = st
			}
		}
		if len(byID) != 4 {
			t.Errorf("el Tablero tiene %d historias, se esperaban 4 (sin las de otro Sprint, otro proyecto ni sin Sprint)", len(byID))
		}
		checks := []struct {
			id     int64
			title  string
			points *int
			status sprint.StoryStatus
			effort bool
			action sprint.Action
			label  string
		}{
			{s1, "Historia En Sprint", boardPuntos(3), catEnSprint, false, sprint.ActionStart, "Comenzar"},
			{s2, "Historia En progreso", boardPuntos(5), catInProgress, true, sprint.ActionComplete, "Completar"},
			{s3, "Historia Hecho con puntos", boardPuntos(8), catDone, false, sprint.ActionReopen, "Reabrir"},
			{s4, "Historia Hecho sin puntos", nil, catDone, false, sprint.ActionReopen, "Reabrir"},
		}
		for _, c := range checks {
			st, ok := byID[c.id]
			if !ok {
				t.Errorf("falta la historia %q", c.title)
				continue
			}
			if st.Title != c.title || st.Status != c.status || st.HasEffort != c.effort {
				t.Errorf("%q: (%q, %q, horas %v), se esperaba (%q, %q, horas %v)",
					c.title, st.Title, st.Status, st.HasEffort, c.title, c.status, c.effort)
			}
			if !catSamePoints(st.StoryPoints, c.points) {
				t.Errorf("%q: Story Points distintos de los esperados", c.title)
			}
			if st.Action != c.action || st.ActionLabel != c.label {
				t.Errorf("%q: acción (%q, %q), se esperaba (%q, %q)", c.title, st.Action, st.ActionLabel, c.action, c.label)
			}
		}

		// Planificados: 3 + 5 + 8 + 0 (sin estimar cuenta 0); completados: solo las Hecho (8).
		if view.PlannedPoints != 16 || view.CompletedPoints != 8 {
			t.Errorf("puntos (planificados %d, completados %d), se esperaba (16, 8)", view.PlannedPoints, view.CompletedPoints)
		}
	})

	t.Run("proyecto B no ve datos del A y viceversa", func(t *testing.T) {
		view, found, err := catalog.ActiveBoard(ctx, projectB)
		if err != nil || !found {
			t.Fatalf("ActiveBoard() = (found %v, error %v), se esperaba el Sprint activo", found, err)
		}
		if view.ID != sprintB || view.Number != int64(numberB) {
			t.Errorf("Sprint = (%d, %d), se esperaba (%d, %d)", view.ID, view.Number, sprintB, numberB)
		}
		if got := catStoryIDs(view.Groups[2]); !slices.Equal(got, []int64{storyB}) {
			t.Errorf("grupo Hecho = %v, se esperaba solo la historia %d", got, storyB)
		}
		if len(view.Groups[0].Stories) != 0 || len(view.Groups[1].Stories) != 0 {
			t.Errorf("los grupos En Sprint y En progreso debían estar vacíos")
		}
		if view.PlannedPoints != 13 || view.CompletedPoints != 13 {
			t.Errorf("puntos (%d, %d), se esperaba (13, 13)", view.PlannedPoints, view.CompletedPoints)
		}
	})
}

func TestPostgreSQLBoardCatalog_SinSprintActivoNoEsUnError(t *testing.T) {
	db := openContextTestDB(t)
	catalog := NewPostgreSQLBoardCatalog(db)
	sinSprints := newCatProject(t, db, "sin Sprints")
	soloFinalizado := newCatProject(t, db, "solo finalizado")
	newCatSprint(t, db, soloFinalizado, "Finalizado")

	casos := []struct {
		nombre    string
		proyectoID int64
	}{
		{"proyecto sin Sprints", sinSprints},
		{"proyecto con solo un Sprint finalizado", soloFinalizado},
		{"proyecto inexistente", MaxID},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			view, found, err := catalog.ActiveBoard(context.Background(), tc.proyectoID)

			if err != nil {
				t.Fatalf("no tener Sprint activo no es un error, se obtuvo: %v", err)
			}
			if found || view.ID != 0 {
				t.Errorf("se esperaba found=false y una vista vacía, se obtuvo (%v, %+v)", found, view)
			}
		})
	}
}

func TestPostgreSQLBoardCatalog_SprintActivoSinHistorias(t *testing.T) {
	db := openContextTestDB(t)
	projectID := newCatProject(t, db, "sin historias")
	sprintID, number := newCatSprint(t, db, projectID, "Activo")

	view, found, err := NewPostgreSQLBoardCatalog(db).ActiveBoard(context.Background(), projectID)

	if err != nil || !found {
		t.Fatalf("ActiveBoard() = (found %v, error %v), se esperaba el Sprint activo", found, err)
	}
	if view.ID != sprintID || view.Number != int64(number) {
		t.Errorf("Sprint = (%d, %d), se esperaba (%d, %d)", view.ID, view.Number, sprintID, number)
	}
	if len(view.Groups) != 3 {
		t.Fatalf("grupos = %d, se esperaban 3 aunque estén vacíos", len(view.Groups))
	}
	for _, g := range view.Groups {
		if len(g.Stories) != 0 {
			t.Errorf("el grupo %q debía estar vacío", g.Status)
		}
	}
	if view.PlannedPoints != 0 || view.CompletedPoints != 0 {
		t.Errorf("puntos (%d, %d), se esperaba (0, 0)", view.PlannedPoints, view.CompletedPoints)
	}
}

func TestPostgreSQLBoardCatalog_FallaDeLecturaEsUnError(t *testing.T) {
	db := openContextTestDB(t)
	_ = db.Close() // toda consulta falla

	_, found, err := NewPostgreSQLBoardCatalog(db).ActiveBoard(context.Background(), 1)

	if err == nil {
		t.Errorf("una falla de lectura debe devolver un error, no confundirse con \"sin Sprint activo\"")
	}
	if found {
		t.Errorf("ante una falla de lectura found debe ser falso")
	}
}