package web

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// HU-09, T026: datos de prueba de "Reportes" (research.md §15), como los dejan HU-04, HU-06, HU-14,
// HU-15 y HU-17. Son los mismos «Alfa», «Beta», «Gamma» y «Delta» de src/report/fixtures_test.go,
// más «Largo» para el rendimiento y los saltos de página.
//
//	«Alfa» (18/08/2026 al 30/11/2026, Ana y Beto): Sprints 1 y 2 'Finalizado' y 3 'Activo', con las
//	  historias, el esfuerzo y los defectos de "Datos de ejemplo" de la spec.
//	«Beta» (01/10/2026 al 31/12/2026, Ana): sin historias ni Sprints.
//	«Gamma» (Ana): Sprints 1 y 2 'Finalizado' con una historia "Hecho" de 1 SP cada uno, sin horas; un
//	  defecto detectado en el 1 y resuelto en el 2; Sprint 3 'Finalizado' sin registro de cierre.
//	«Delta» (Ana): Sprint 1 'Finalizado' con «D1» Hecho 2 SP, 1 h est., 1 h de Caro y un defecto abierto.
//	«Largo» (Ana): Sprint 1 'Finalizado' con 120 historias (90 "Hecho" de 2 SP, 4 h est. y 4 h de Ana;
//	  30 que volvieron al Backlog, de 3 SP y 4 h est., sin horas; una con un título de 300 caracteres) y
//	  50 defectos detectados en él, 20 resueltos en él.
type reportWebFixture struct {
	db       *sql.DB
	projects map[string]int64
	sprints  map[string]int64
	stories  map[string]int64
	members  map[string]int64
}

// largoLongTitle es el título de 300 caracteres de «Largo».
var largoLongTitle = strings.Repeat("Historia con un titulo muy largo ", 10)[:299] + "x"

func newReportWebFixture(t *testing.T, db *sql.DB) *reportWebFixture {
	t.Helper()
	f := &reportWebFixture{db: db, projects: map[string]int64{}, sprints: map[string]int64{},
		stories: map[string]int64{}, members: map[string]int64{}}
	t.Cleanup(f.cleanup)

	for _, name := range []string{"Ana", "Beto", "Caro"} {
		f.members[name] = f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	}
	alfa := f.project(t, "Alfa", "2026-08-18", "2026-11-30", "Ana", "Beto")
	f.project(t, "Beta", "2026-10-01", "2026-12-31", "Ana")
	gamma := f.project(t, "Gamma", "2026-09-01", "2026-12-31", "Ana")
	delta := f.project(t, "Delta", "2026-09-01", "2026-12-31", "Ana")
	largo := f.project(t, "Largo", "2026-09-01", "2026-12-31", "Ana")

	// «Alfa».
	f.story(t, alfa, "Alta", 3, "4")
	f.story(t, alfa, "Login", 5, "10")
	f.story(t, alfa, "Tablero", 3, "6")
	f.story(t, alfa, "Perfil", 2, "")
	f.story(t, alfa, "Reportes", 8, "12")
	f.story(t, alfa, "Exportar", 5, "6")
	f.story(t, alfa, "Ayuda", 0, "")

	a1 := f.sprint(t, alfa, "Alfa 1", 1, "Dar de alta usuarios", "2026-08-18", "2026-08-31", "Finalizado")
	f.closed(t, a1, "Alta", "Hecho", 3)
	f.effort(t, "Alta", "Ana", "4")

	a2 := f.sprint(t, alfa, "Alfa 2", 2, "Gestionar perfiles y reportes", "2026-09-01", "2026-09-14", "Finalizado")
	f.closed(t, a2, "Login", "Hecho", 5)
	f.closed(t, a2, "Tablero", "Hecho", 3)
	f.closed(t, a2, "Perfil", "Hecho", 2)
	f.closed(t, a2, "Reportes", "En progreso", 8)
	f.effort(t, "Login", "Ana", "8")
	f.effort(t, "Login", "Beto", "3")
	f.effort(t, "Tablero", "Ana", "6")
	f.effort(t, "Perfil", "Ana", "3")
	f.effort(t, "Perfil", "Beto", "1")
	f.effort(t, "Reportes", "Beto", "3")

	a3 := f.sprint(t, alfa, "Alfa 3", 3, "Exportar e informar", "2026-09-15", "2026-09-28", "Activo")
	f.assign(t, a3, "Exportar", "En progreso")
	f.assign(t, a3, "Ayuda", "En Sprint")
	f.effort(t, "Exportar", "Ana", "2")

	f.defect(t, alfa, "Login", "El login no valida la contraseña", "Alta", a2, a2)
	f.defect(t, alfa, "Perfil", "La foto de perfil no se guarda", "Media", a2, 0)
	f.defect(t, alfa, "Exportar", "El PDF sale en blanco", "Crítica", a3, 0)

	// «Gamma».
	f.story(t, gamma, "G1", 1, "")
	f.story(t, gamma, "G2", 1, "")
	g1 := f.sprint(t, gamma, "Gamma 1", 1, "Migrar datos", "2026-09-01", "2026-09-14", "Finalizado")
	f.closed(t, g1, "G1", "Hecho", 1)
	g2 := f.sprint(t, gamma, "Gamma 2", 2, "Auditar accesos", "2026-09-15", "2026-09-28", "Finalizado")
	f.closed(t, g2, "G2", "Hecho", 1)
	f.sprint(t, gamma, "Gamma 3", 3, "Cerrar el proyecto", "2026-09-29", "2026-10-12", "Finalizado")
	f.defect(t, gamma, "G1", "La migración pierde acentos", "Baja", g1, g2)

	// «Delta».
	f.story(t, delta, "D1", 2, "1")
	d1 := f.sprint(t, delta, "Delta 1", 1, "Probar otro proyecto", "2026-09-01", "2026-09-14", "Finalizado")
	f.closed(t, d1, "D1", "Hecho", 2)
	f.effort(t, "D1", "Caro", "1")
	f.defect(t, delta, "D1", "Defecto de Delta", "Media", d1, 0)

	// «Largo»: los datos se cargan en bloque para que la preparación sea rápida.
	l1 := f.sprint(t, largo, "Largo 1", 1, "Probar un reporte largo", "2026-09-01", "2026-09-14", "Finalizado")
	f.exec(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours, status, active_sprint_id)
		SELECT $1::bigint, CASE WHEN n = 1 THEN $3::text ELSE 'L' || lpad(n::text, 3, '0') END, 'd', 'Media',
		       CASE WHEN n <= 90 THEN 2 ELSE 3 END, 4,
		       CASE WHEN n <= 90 THEN 'Hecho' ELSE 'Pendiente' END,
		       CASE WHEN n <= 90 THEN $2::bigint END
		FROM generate_series(1, 120) AS n
		ORDER BY n`, largo, l1, largoLongTitle)
	f.exec(t, `INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		SELECT $1::bigint, id, CASE WHEN status = 'Hecho' THEN 'Hecho' ELSE 'En progreso' END, story_points
		FROM user_stories WHERE project_id = $2::bigint`, l1, largo)
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id)
		SELECT $1::bigint, id FROM user_stories WHERE project_id = $2::bigint AND status = 'Hecho'`, l1, largo)
	f.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		SELECT id, $2::int, '2026-09-10', 'Desarrollo', 4 FROM user_stories WHERE project_id = $1::bigint AND status = 'Hecho'`,
		largo, f.members["Ana"])
	f.exec(t, `INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		SELECT $1::int, (SELECT MIN(id) FROM user_stories WHERE project_id = $1::int), 'Defecto ' || n, 'Media',
		       CASE WHEN n <= 20 THEN 'Resuelto' ELSE 'Abierto' END, $2::bigint, CASE WHEN n <= 20 THEN $2::bigint END
		FROM generate_series(1, 50) AS n`, largo, l1)
	return f
}

func (f *reportWebFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *reportWebFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *reportWebFixture) project(t *testing.T, name, start, end string, members ...string) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, $2, $3) RETURNING id`, name, start, end)
	f.projects[name] = id
	for _, m := range members {
		f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, id, f.members[m])
	}
	return id
}

// story crea una historia "Pendiente"; points 0 = sin Story Points y estimated "" = sin horas.
func (f *reportWebFixture) story(t *testing.T, project int64, title string, points int, estimated string) {
	t.Helper()
	var sp, est any
	if points != 0 {
		sp = points
	}
	if estimated != "" {
		est = estimated
	}
	f.stories[title] = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours)
		VALUES ($1, $2, 'd', 'Media', $3, $4) RETURNING id`, project, title, sp, est)
}

func (f *reportWebFixture) sprint(t *testing.T, project int64, key string, number int, goal, start, end, status string) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, project, number, goal, start, end, status)
	f.sprints[key] = id
	return id
}

func (f *reportWebFixture) assign(t *testing.T, sprintID int64, title, status string) {
	t.Helper()
	f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, f.stories[title])
	f.exec(t, `UPDATE user_stories SET status = $1, active_sprint_id = $2 WHERE id = $3`, status, sprintID, f.stories[title])
}

// closed deja la historia como la deja el cierre de HU-15.
func (f *reportWebFixture) closed(t *testing.T, sprintID int64, title, finalStatus string, points int) {
	t.Helper()
	f.exec(t, `INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points) VALUES ($1, $2, $3, $4)`,
		sprintID, f.stories[title], finalStatus, points)
	if finalStatus == "Hecho" {
		f.assign(t, sprintID, title, finalStatus)
	}
}

func (f *reportWebFixture) effort(t *testing.T, title, member, hours string) {
	t.Helper()
	f.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, '2026-09-10', 'Desarrollo', $3)`, f.stories[title], f.members[member], hours)
}

// defect registra un defecto; resolution 0 = "Abierto".
func (f *reportWebFixture) defect(t *testing.T, project int64, title, description, severity string, detection, resolution int64) {
	t.Helper()
	status, res := "Abierto", any(nil)
	if resolution != 0 {
		status, res = "Resuelto", resolution
	}
	f.exec(t, `INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, project, f.stories[title], description, severity, status, detection, res)
}

// metricsRows cuenta las filas de sprint_metrics de los Sprints del proyecto.
func (f *reportWebFixture) metricsRows(t *testing.T, project string) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM sprint_metrics WHERE sprint_id IN (SELECT id FROM sprints WHERE project_id = $1)`,
		f.projects[project]).Scan(&n); err != nil {
		t.Fatalf("no se pudo contar sprint_metrics: %v", err)
	}
	return n
}

func (f *reportWebFixture) sprintPath(key string) string {
	return fmt.Sprintf("/reportes/sprints/%d/pdf", f.sprints[key])
}

// cleanup: defectos, registro de cierre, asignaciones, sprint_metrics, effort_logs, historias,
// Sprints, proyectos (con project_members en cascada) e integrantes.
func (f *reportWebFixture) cleanup() {
	const sprints = `SELECT id FROM sprints WHERE project_id = $1`
	for _, p := range f.projects {
		_, _ = f.db.Exec(`DELETE FROM defects WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM sprint_story_results WHERE sprint_id IN (`+sprints+`)`, p)
		_, _ = f.db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id IN (`+sprints+`)`, p)
		_, _ = f.db.Exec(`DELETE FROM sprint_metrics WHERE sprint_id IN (`+sprints+`)`, p)
		_, _ = f.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, p)
		_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM projects WHERE id = $1`, p)
	}
	for _, m := range f.members {
		_, _ = f.db.Exec(`DELETE FROM members WHERE id = $1`, m)
	}
}
