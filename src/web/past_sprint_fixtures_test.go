package web

import (
	"database/sql"
	"testing"
)

// HU-16, T005: datos de prueba de "Sprints anteriores", cargados directamente como los deja el
// cierre de HU-15 (research.md §10): las historias "Hecho" conservan su asignación y su
// active_sprint_id; las demás vuelven al Backlog en "Pendiente", sin asignación.
//
//	«Alfa»:  Sprint 1 'Finalizado' («Alta» Hecho 3);
//	         Sprint 2 'Finalizado' («Login» Hecho 5, «Tablero» Hecho 3, «Perfil» Hecho 2, «Reportes» En progreso 8);
//	         Sprint 3 'Activo' («Exportar» En progreso 5, «Ayuda» En Sprint sin estimar).
//	«Beta»:  sin Sprints ni historias.
//	«Gamma»: Sprint 1 'Finalizado' («Migración» En progreso 5, «Respaldo» En Sprint sin estimar);
//	         Sprint 2 'Finalizado' sin registro de cierre, con «Auditoría» Hecho 2 asignada.
type pastSprintFixture struct {
	db                *sql.DB
	alfa, beta, gamma int64
	sprints           map[string]int64 // "Alfa 1", "Alfa 2", "Alfa 3", "Gamma 1", "Gamma 2"
	stories           map[string]int64 // por título
	extraProjects     []int64
}

func newPastSprintFixture(t *testing.T, db *sql.DB) *pastSprintFixture {
	t.Helper()
	f := &pastSprintFixture{db: db, sprints: map[string]int64{}, stories: map[string]int64{}}
	t.Cleanup(f.cleanup)

	f.alfa = f.project(t, "Alfa", "2026-08-18", "2026-11-30")
	f.beta = f.project(t, "Beta", "2026-10-01", "2026-12-31")
	f.gamma = f.project(t, "Gamma", "2026-09-01", "2026-12-31")

	// Historias de «Alfa» en el orden de la spec (el orden de creación es el de las tablas).
	for _, s := range []struct {
		title  string
		points int
	}{{"Alta", 3}, {"Login", 5}, {"Tablero", 3}, {"Perfil", 2}, {"Reportes", 8}, {"Exportar", 5}, {"Ayuda", 0}} {
		f.story(t, f.alfa, s.title, "Pendiente", s.points)
	}
	for _, s := range []struct {
		title  string
		points int
	}{{"Migración", 5}, {"Respaldo", 0}, {"Auditoría", 2}} {
		f.story(t, f.gamma, s.title, "Pendiente", s.points)
	}

	a1 := f.sprint(t, f.alfa, "Alfa 1", 1, "Dar de alta usuarios", "2026-08-18", "2026-08-31", "Finalizado")
	f.closed(t, a1, "Alta", "Hecho", 3, true)

	a2 := f.sprint(t, f.alfa, "Alfa 2", 2, "Gestionar perfiles y reportes", "2026-09-01", "2026-09-14", "Finalizado")
	f.closed(t, a2, "Login", "Hecho", 5, true)
	f.closed(t, a2, "Tablero", "Hecho", 3, true)
	f.closed(t, a2, "Perfil", "Hecho", 2, true)
	f.closed(t, a2, "Reportes", "En progreso", 8, false)

	a3 := f.sprint(t, f.alfa, "Alfa 3", 3, "Exportar e informar", "2026-09-15", "2026-09-28", "Activo")
	f.assign(t, a3, "Exportar", "En progreso")
	f.assign(t, a3, "Ayuda", "En Sprint")

	g1 := f.sprint(t, f.gamma, "Gamma 1", 1, "Migrar datos", "2026-09-01", "2026-09-14", "Finalizado")
	f.closed(t, g1, "Migración", "En progreso", 5, false)
	f.closed(t, g1, "Respaldo", "En Sprint", 0, false)

	g2 := f.sprint(t, f.gamma, "Gamma 2", 2, "Auditar accesos", "2026-09-15", "2026-09-28", "Finalizado")
	f.assign(t, g2, "Auditoría", "Hecho")
	return f
}

func (f *pastSprintFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *pastSprintFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *pastSprintFixture) project(t *testing.T, name, start, end string) int64 {
	t.Helper()
	return f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, $2, $3) RETURNING id`, name, start, end)
}

// story crea una historia; points 0 = sin Story Points.
func (f *pastSprintFixture) story(t *testing.T, project int64, title, status string, points int) int64 {
	t.Helper()
	var sp any
	if points != 0 {
		sp = points
	}
	id := f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, story_points)
		VALUES ($1, $2, 'd', 'Media', $3, $4) RETURNING id`, project, title, status, sp)
	f.stories[title] = id
	return id
}

func (f *pastSprintFixture) sprint(t *testing.T, project int64, key string, number int, goal, start, end, status string) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, project, number, goal, start, end, status)
	f.sprints[key] = id
	return id
}

// assign asigna la historia al Sprint con ese estado (como HU-04 y HU-14).
func (f *pastSprintFixture) assign(t *testing.T, sprintID int64, title, status string) {
	t.Helper()
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, f.stories[title])
	f.exec(t, `UPDATE user_stories SET status = $1, active_sprint_id = $2 WHERE id = $3`, status, sprintID, f.stories[title])
}

// closed deja la historia como la deja el cierre de HU-15: su fila en sprint_story_results y, si
// quedó "Hecho", asignada al Sprint; si no, "Pendiente" en el Backlog. points 0 = sin estimar.
func (f *pastSprintFixture) closed(t *testing.T, sprintID int64, title, finalStatus string, points int, done bool) {
	t.Helper()
	var sp any
	if points != 0 {
		sp = points
	}
	f.exec(t, `INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points) VALUES ($1, $2, $3, $4)`,
		sprintID, f.stories[title], finalStatus, sp)
	if done {
		f.assign(t, sprintID, title, finalStatus)
	}
}

// addProject crea un proyecto adicional para una prueba y lo borra al final.
func (f *pastSprintFixture) addProject(t *testing.T, name string) int64 {
	t.Helper()
	id := f.project(t, name, "2026-01-01", "2026-12-31")
	f.extraProjects = append(f.extraProjects, id)
	return id
}

// storyPointsOfDone suma, con SQL, los Story Points de las filas 'Hecho' del registro de cierre.
func (f *pastSprintFixture) storyPointsOfDone(t *testing.T, sprintID int64) int {
	t.Helper()
	var total int
	if err := f.db.QueryRow(`SELECT COALESCE(SUM(story_points), 0) FROM sprint_story_results
		WHERE sprint_id = $1 AND final_status = 'Hecho'`, sprintID).Scan(&total); err != nil {
		t.Fatalf("no se pudo sumar el registro de cierre: %v", err)
	}
	return total
}

// cleanup: registro de cierre → asignaciones → sprint_metrics → historias → Sprints → proyectos.
func (f *pastSprintFixture) cleanup() {
	projects := append([]int64{f.alfa, f.beta, f.gamma}, f.extraProjects...)
	for _, p := range projects {
		_, _ = f.db.Exec(`DELETE FROM sprint_story_results WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`, p)
		_, _ = f.db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`, p)
		_, _ = f.db.Exec(`DELETE FROM sprint_metrics WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`, p)
		_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM projects WHERE id = $1`, p)
	}
}
