package web

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// HU-08 (T009): datos de prueba compartidos del Dashboard (los usan el catálogo y la pantalla).
// Cada proyecto tiene sus propios Sprints. Los Sprints finalizados quedan como los deja el cierre de
// HU-15: las historias "Hecho" conservan su asignación y las demás vuelven al Backlog como
// "Pendiente", con su estado final en sprint_story_results.

// dashboardFixture agrupa los proyectos de los escenarios de la spec.
type dashboardFixture struct {
	db     *sql.DB
	stamp  int64
	member int64

	alfa, beta, empty, noSprint, onlyActive, noHours, partial, twelve, noStories int64

	// Sprints de «Alfa»: 1 y 2 finalizados, 3 activo.
	alfaSprint1, alfaSprint2, alfaSprint3 int64
	// Historias de «Alfa» (tabla "Datos de ejemplo" de la spec).
	alfaA, alfaB int64

	projects []int64
}

// newDashboardFixture arma todos los proyectos y registra su limpieza, en este orden: defectos,
// registro de cierre, asignaciones, sprint_metrics, esfuerzo, historias, Sprints, proyectos e
// integrante.
func newDashboardFixture(t *testing.T, db *sql.DB) *dashboardFixture {
	t.Helper()
	f := &dashboardFixture{db: db, stamp: time.Now().UnixNano()}
	t.Cleanup(f.cleanup(t))

	f.member = f.insertID(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("Ana HU-08 %d", f.stamp))

	// «Alfa»: escenarios 2, 3, 5, 7 y 9.
	f.alfa = f.project(t, "Alfa")
	f.alfaSprint1 = f.sprint(t, f.alfa, 1, "Finalizado")
	f.alfaSprint2 = f.sprint(t, f.alfa, 2, "Finalizado")
	f.alfaSprint3 = f.sprint(t, f.alfa, 3, "Activo")
	f.alfaA = f.closedStory(t, f.alfa, f.alfaSprint1, "A", 8, "Hecho", "")
	f.closedStory(t, f.alfa, f.alfaSprint1, "P1", 3, "En progreso", "")
	f.alfaB = f.closedStory(t, f.alfa, f.alfaSprint2, "B", 13, "Hecho", "10")
	f.effort(t, f.alfaB, "8")
	f.effort(t, f.alfaB, "4")
	f.closedStory(t, f.alfa, f.alfaSprint2, "P2", 2, "En Sprint", "")
	c := f.activeStory(t, f.alfa, f.alfaSprint3, "C", 1, "En Sprint")
	d := f.activeStory(t, f.alfa, f.alfaSprint3, "D", 3, "En progreso")
	f.activeStory(t, f.alfa, f.alfaSprint3, "E", 5, "En progreso")
	f.activeStory(t, f.alfa, f.alfaSprint3, "F", 2, "Hecho")
	f.defect(t, f.alfa, c, f.alfaSprint3, 0)
	f.defect(t, f.alfa, d, f.alfaSprint3, 0)
	f.defect(t, f.alfa, f.alfaB, f.alfaSprint2, 0)
	f.defect(t, f.alfa, f.alfaA, f.alfaSprint1, f.alfaSprint2)
	f.defect(t, f.alfa, f.alfaB, f.alfaSprint2, f.alfaSprint2)

	// «Beta»: escenario 10. Si se mezclara con «Alfa», cambiarían todas las cuentas.
	f.beta = f.project(t, "Beta")
	betaSprint1 := f.sprint(t, f.beta, 1, "Finalizado")
	f.pendingStory(t, f.beta, "R", 3)
	q := f.closedStory(t, f.beta, betaSprint1, "Q", 5, "Hecho", "")
	f.defect(t, f.beta, q, betaSprint1, 0)

	// «Vacío»: escenario 1, sin historias, Sprints ni defectos.
	f.empty = f.project(t, "Vacío")

	// «Sin Sprint»: escenario 4, dos historias "Pendiente" y ningún Sprint.
	f.noSprint = f.project(t, "Sin Sprint")
	f.pendingStory(t, f.noSprint, "S1", 2)
	f.pendingStory(t, f.noSprint, "S2", 3)

	// «Solo activo»: escenario 6, una "Pendiente" y el Sprint 1 activo con una "En Sprint" de 3.
	f.onlyActive = f.project(t, "Solo activo")
	onlyActiveSprint := f.sprint(t, f.onlyActive, 1, "Activo")
	f.pendingStory(t, f.onlyActive, "O1", 2)
	f.activeStory(t, f.onlyActive, onlyActiveSprint, "O2", 3, "En Sprint")

	// «Sin horas»: escenario 8, Sprint 1 finalizado con una "Hecho" de 5 sin horas estimadas.
	f.noHours = f.project(t, "Sin horas")
	noHoursSprint := f.sprint(t, f.noHours, 1, "Finalizado")
	f.closedStory(t, f.noHours, noHoursSprint, "N1", 5, "Hecho", "")

	// «Parcial»: G (3, 10 horas estimadas, 12 reales) y H (2, sin horas estimadas).
	f.partial = f.project(t, "Parcial")
	partialSprint := f.sprint(t, f.partial, 1, "Finalizado")
	g := f.closedStory(t, f.partial, partialSprint, "G", 3, "Hecho", "10")
	f.effort(t, g, "8")
	f.effort(t, g, "4")
	f.closedStory(t, f.partial, partialSprint, "H", 2, "Hecho", "")

	// «Doce»: Sprints 1 a 12 finalizados, 21 en los dos primeros y 5 en los demás (acumulada 92).
	f.twelve = f.project(t, "Doce")
	for number := 1; number <= 12; number++ {
		sprintID := f.sprint(t, f.twelve, number, "Finalizado")
		points := 5
		if number <= 2 {
			points = 21
		}
		f.closedStory(t, f.twelve, sprintID, fmt.Sprintf("D%d", number), points, "Hecho", "")
	}

	// «Sin historias»: Sprint 1 finalizado sin historias.
	f.noStories = f.project(t, "Sin historias")
	f.sprint(t, f.noStories, 1, "Finalizado")

	return f
}

func (f *dashboardFixture) insertID(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar los datos del Dashboard: %v", err)
	}
	return id
}

func (f *dashboardFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := f.db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("no se pudo preparar los datos del Dashboard: %v", err)
	}
}

func (f *dashboardFixture) project(t *testing.T, name string) int64 {
	t.Helper()
	id := f.insertID(t,
		`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU08 %s %d", name, f.stamp))
	f.projects = append(f.projects, id)
	return id
}

func (f *dashboardFixture) sprint(t *testing.T, projectID int64, number int, status string) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, $2, $3, '2026-10-01', '2026-10-15', $4) RETURNING id`,
		projectID, number, fmt.Sprintf("Objetivo HU-08 %d", number), status)
}

func (f *dashboardFixture) story(t *testing.T, projectID int64, title string, points int, status string, activeSprint int64, estimatedHours string) int64 {
	t.Helper()
	active := sql.NullInt64{Int64: activeSprint, Valid: activeSprint > 0}
	estimated := sql.NullString{String: estimatedHours, Valid: estimatedHours != ""}
	return f.insertID(t,
		`INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id, estimated_hours)
		 VALUES ($1, $2, 'Historia de prueba HU-08', 'Media', $3, $4, $5, $6::numeric) RETURNING id`,
		projectID, fmt.Sprintf("HU08 %s %d", title, f.stamp), points, status, active, estimated)
}

// pendingStory crea una historia "Pendiente" en el Backlog, sin Sprint.
func (f *dashboardFixture) pendingStory(t *testing.T, projectID int64, title string, points int) int64 {
	t.Helper()
	return f.story(t, projectID, title, points, "Pendiente", 0, "")
}

// activeStory crea una historia asignada al Sprint activo, con el estado indicado.
func (f *dashboardFixture) activeStory(t *testing.T, projectID, sprintID int64, title string, points int, status string) int64 {
	t.Helper()
	id := f.story(t, projectID, title, points, status, sprintID, "")
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id)
	return id
}

// closedStory crea una historia de un Sprint finalizado como la deja el cierre de HU-15: si su estado
// final es "Hecho" conserva la asignación; si no, vuelve al Backlog como "Pendiente". En los dos
// casos queda su fila en sprint_story_results.
func (f *dashboardFixture) closedStory(t *testing.T, projectID, sprintID int64, title string, points int, finalStatus, estimatedHours string) int64 {
	t.Helper()
	var id int64
	if finalStatus == "Hecho" {
		id = f.story(t, projectID, title, points, "Hecho", sprintID, estimatedHours)
		f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id)
	} else {
		id = f.story(t, projectID, title, points, "Pendiente", 0, estimatedHours)
	}
	f.exec(t, `INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points) VALUES ($1, $2, $3, $4)`,
		sprintID, id, finalStatus, points)
	return id
}

// effort registra horas reales de la integrante del fixture en la historia.
func (f *dashboardFixture) effort(t *testing.T, storyID int64, hours string) {
	t.Helper()
	f.exec(t,
		`INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		 VALUES ($1, $2, '2026-10-05', 'Trabajo de HU-08', $3::numeric)`,
		storyID, f.member, hours)
}

// defect crea un defecto detectado en detection; con resolution distinto de 0 queda "Resuelto" en él.
func (f *dashboardFixture) defect(t *testing.T, projectID, storyID, detection, resolution int64) {
	t.Helper()
	status, resolved := "Abierto", sql.NullInt64{}
	if resolution != 0 {
		status, resolved = "Resuelto", sql.NullInt64{Int64: resolution, Valid: true}
	}
	f.exec(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		 VALUES ($1, $2, 'Defecto de prueba HU-08', 'Media', $3, $4, $5)`,
		projectID, storyID, status, detection, resolved)
}

// cleanup borra todo lo que creó el fixture, en el orden del encabezado de tasks.md.
func (f *dashboardFixture) cleanup(t *testing.T) func() {
	return func() {
		ctx := context.Background()
		const sprints = `SELECT id FROM sprints WHERE project_id = ANY($1)`
		const stories = `SELECT id FROM user_stories WHERE project_id = ANY($1)`
		ids := make([]int64, len(f.projects))
		copy(ids, f.projects)
		for _, query := range []string{
			`DELETE FROM defects WHERE project_id = ANY($1)`,
			`DELETE FROM sprint_story_results WHERE sprint_id IN (` + sprints + `)`,
			`DELETE FROM sprint_story_assignments WHERE sprint_id IN (` + sprints + `)`,
			`DELETE FROM sprint_metrics WHERE sprint_id IN (` + sprints + `)`,
			`DELETE FROM effort_logs WHERE story_id IN (` + stories + `)`,
			`DELETE FROM user_stories WHERE project_id = ANY($1)`,
			`DELETE FROM sprints WHERE project_id = ANY($1)`,
			`DELETE FROM projects WHERE id = ANY($1)`,
		} {
			if _, err := f.db.ExecContext(ctx, query, ids); err != nil {
				t.Errorf("no se pudieron borrar los datos del Dashboard: %v", err)
			}
		}
		if f.member != 0 {
			if _, err := f.db.ExecContext(ctx, `DELETE FROM members WHERE id = $1`, f.member); err != nil {
				t.Errorf("no se pudo borrar la integrante del Dashboard: %v", err)
			}
		}
	}
}
