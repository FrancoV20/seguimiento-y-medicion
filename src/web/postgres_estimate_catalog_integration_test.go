package web

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// Integración de PostgreSQLEstimateCatalog (HU-17, T008) contra PostgreSQL real, con fixtures
// propias que se borran al terminar (primero las horas, después las historias y los proyectos).

type estimateCatalogFixture struct {
	db                                       *sql.DB
	projectID, otherID, emptyID              int64
	sinEstimar, conHoras, finalizada, activo int64
	ajena                                    int64
}

func newEstimateCatalogFixture(t *testing.T) *estimateCatalogFixture {
	t.Helper()
	db := openContextTestDB(t)
	f := &estimateCatalogFixture{db: db}
	stamp := time.Now().UnixNano()
	var projectIDs []int64
	var memberID int64
	t.Cleanup(func() {
		if memberID != 0 {
			_, _ = db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, memberID)
		}
		for _, id := range projectIDs {
			_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, id)
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, id)
		}
		if memberID != 0 {
			_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, memberID)
		}
	})
	insertProject := func(name string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO projects (name, start_date, end_date, status)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31', 'Activo') RETURNING id`,
			fmt.Sprintf("%s %d", name, stamp)).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear el proyecto %q: %v", name, err)
		}
		projectIDs = append(projectIDs, id)
		return id
	}
	insertStory := func(projectID int64, title string, points *int, hours *float64, status string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours, status)
			VALUES ($1, $2, 'Historia de prueba HU-17', 'Media', $3, $4, $5) RETURNING id`,
			projectID, title, points, hours, status).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear la historia %q: %v", title, err)
		}
		return id
	}
	insertSprint := func(number int, status string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, 'Sprint HU-17', DATE '2026-10-01', DATE '2026-10-15', $3) RETURNING id`,
			f.projectID, number, status).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear el Sprint %d: %v", number, err)
		}
		return id
	}
	assign := func(sprintID, storyID int64) {
		if _, err := db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID); err != nil {
			t.Fatalf("no se pudo asignar la historia %d: %v", storyID, err)
		}
		if _, err := db.Exec(`UPDATE user_stories SET active_sprint_id = $1 WHERE id = $2`, sprintID, storyID); err != nil {
			t.Fatalf("no se pudo actualizar la historia %d: %v", storyID, err)
		}
	}
	points, hours := 5, 8.5
	finishedHours, activeHours, foreignHours := 13.0, 2.0, 6.0

	f.projectID = insertProject("HU17 Estimaciones A")
	f.otherID = insertProject("HU17 Estimaciones B")
	f.emptyID = insertProject("HU17 Estimaciones vacío")
	f.sinEstimar = insertStory(f.projectID, "HU17 sin estimar", nil, nil, "Pendiente")
	f.conHoras = insertStory(f.projectID, "HU17 con horas reales", &points, &hours, "En progreso")
	f.finalizada = insertStory(f.projectID, "HU17 Sprint finalizado", nil, &finishedHours, "Hecho")
	f.activo = insertStory(f.projectID, "HU17 Sprint activo", nil, &activeHours, "En Sprint")
	f.ajena = insertStory(f.otherID, "HU17 de otro proyecto", nil, &foreignHours, "Pendiente")
	assign(insertSprint(1, "Finalizado"), f.finalizada)
	assign(insertSprint(2, "Activo"), f.activo)

	if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ('HU17 catálogo') RETURNING id`).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $3, CURRENT_DATE, 'Desarrollo', 1.5), ($1, $3, CURRENT_DATE, 'Pruebas', 2.25),
		       ($2, $3, CURRENT_DATE, 'Ajena', 4)`,
		f.conHoras, f.ajena, memberID); err != nil {
		t.Fatalf("no se pudieron registrar horas reales: %v", err)
	}
	return f
}

func TestPostgreSQLEstimateCatalog_ListaSoloElProyectoPedidoEnOrdenDescendente(t *testing.T) {
	f := newEstimateCatalogFixture(t)
	rows, err := NewPostgreSQLEstimateCatalog(f.db).ListEstimates(context.Background(), f.projectID)
	if err != nil {
		t.Fatalf("ListEstimates devolvió error: %v", err)
	}
	want := []int64{f.activo, f.finalizada, f.conHoras, f.sinEstimar}
	if len(rows) != len(want) {
		t.Fatalf("se esperaban %d historias del proyecto, se obtuvieron %d: %+v", len(want), len(rows), rows)
	}
	for i, row := range rows {
		if row.ID != want[i] {
			t.Errorf("posición %d: id %d; se esperaba %d (orden descendente por id)", i, row.ID, want[i])
		}
		if row.ID == f.ajena {
			t.Errorf("apareció la historia de otro proyecto: %+v", row)
		}
	}
}

func TestPostgreSQLEstimateCatalog_InformaHorasRealesYBloqueo(t *testing.T) {
	f := newEstimateCatalogFixture(t)
	rows, err := NewPostgreSQLEstimateCatalog(f.db).ListEstimates(context.Background(), f.projectID)
	if err != nil {
		t.Fatalf("ListEstimates devolvió error: %v", err)
	}
	byID := make(map[int64]EstimateRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	sin := byID[f.sinEstimar]
	if sin.Title != "HU17 sin estimar" || sin.Status != backlog.StatusPending || sin.StoryPoints != nil ||
		sin.EstimatedHours != nil || sin.ActualHours != 0 || sin.Locked || sin.LockedReason != "" {
		t.Errorf("historia sin estimar: %+v", sin)
	}

	con := byID[f.conHoras]
	if con.StoryPoints == nil || *con.StoryPoints != 5 || con.EstimatedHours == nil || *con.EstimatedHours != 8.5 ||
		con.ActualHours != 3.75 || con.Status != "En progreso" || con.Locked {
		t.Errorf("historia con horas reales (8,5 estimadas y 1,5 + 2,25 reales): %+v", con)
	}

	fin := byID[f.finalizada]
	if !fin.Locked || fin.LockedReason != EstimateLockedReason || fin.EstimatedHours == nil || *fin.EstimatedHours != 13 {
		t.Errorf("historia de un Sprint finalizado: %+v; se esperaba bloqueada con %q", fin, EstimateLockedReason)
	}

	act := byID[f.activo]
	if act.Locked || act.LockedReason != "" || act.ActualHours != 0 {
		t.Errorf("historia del Sprint activo no debe estar bloqueada: %+v", act)
	}
}

func TestPostgreSQLEstimateCatalog_ProyectoSinHistoriasNoEsError(t *testing.T) {
	f := newEstimateCatalogFixture(t)
	rows, err := NewPostgreSQLEstimateCatalog(f.db).ListEstimates(context.Background(), f.emptyID)
	if err != nil {
		t.Fatalf("un proyecto sin historias no debe ser un error: %v", err)
	}
	if rows == nil || len(rows) != 0 {
		t.Errorf("se esperaba una lista vacía (no nil), se obtuvo %#v", rows)
	}
}
