package web

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// T006: PostgreSQLEffortCatalog contra PostgreSQL real, con fixtures propias que se borran al
// terminar (primero los registros y las historias, después proyectos e integrantes).

// effortFixture:
//   - Proyecto A (Ana y Beto): Sprint 1 finalizado y Sprint 2 activo con tres historias
//     ("En Sprint" sin horas, "En progreso" con tres cargas de Ana y Beto, "Hecho" con una
//     carga de Beto), una historia "Pendiente" sin Sprint y una del Sprint finalizado.
//   - Proyecto B (Carla): Sprint 1 activo con una historia con horas.
//   - Proyecto C (Ana): sin Sprint activo.
type effortFixture struct {
	db                                *sql.DB
	stamp                             string
	projectA, projectB, projectC      int64
	ana, beto, carla                  int64
	anaName, betoName, carlaName      string
	sprintA, finishedSprintA, sprintB int64
	enSprint, enProgreso, hecho       int64
	pendiente, finalizada, ajena      int64
	enSprintTitle, enProgresoTitle    string
	hechoTitle, pendienteTitle        string
	finalizadaTitle, ajenaTitle       string
}

func newEffortFixture(t *testing.T, db *sql.DB) *effortFixture {
	t.Helper()
	f := &effortFixture{db: db, stamp: fmt.Sprint(time.Now().UnixNano())}
	f.anaName, f.betoName, f.carlaName = "Ana HU24 "+f.stamp, "Beto HU24 "+f.stamp, "Carla HU24 "+f.stamp
	t.Cleanup(f.cleanup)

	newProject := func(name string) int64 {
		return f.id(t, `INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, name+" "+f.stamp)
	}
	newMember := func(name string) int64 {
		return f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	}
	link := func(project, member int64) {
		f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, project, member)
	}
	newSprint := func(project int64, number int, status string) int64 {
		return f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, 'Objetivo HU24', DATE '2026-10-01', DATE '2026-10-31', $3) RETURNING id`,
			project, number, status)
	}
	newStory := func(project int64, title, status string, sprint int64) int64 {
		var active any
		if sprint != 0 {
			active = sprint
		}
		id := f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
			VALUES ($1, $2, 'Descripción HU24', 'Media', $3, $4) RETURNING id`, project, title, status, active)
		if sprint != 0 {
			f.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprint, id)
		}
		return id
	}

	f.projectA, f.projectB, f.projectC = newProject("HU24 Proyecto A"), newProject("HU24 Proyecto B"), newProject("HU24 Proyecto C")
	f.ana, f.beto, f.carla = newMember(f.anaName), newMember(f.betoName), newMember(f.carlaName)
	link(f.projectA, f.ana)
	link(f.projectA, f.beto)
	link(f.projectB, f.carla)
	link(f.projectC, f.ana)

	f.finishedSprintA = newSprint(f.projectA, 1, "Finalizado")
	f.sprintA = newSprint(f.projectA, 2, "Activo")
	f.sprintB = newSprint(f.projectB, 1, "Activo")

	f.enSprintTitle, f.enProgresoTitle, f.hechoTitle = "HU24 en Sprint", "HU24 en progreso", "HU24 hecha"
	f.pendienteTitle, f.finalizadaTitle, f.ajenaTitle = "HU24 pendiente", "HU24 de Sprint finalizado", "HU24 de otro proyecto"
	f.enSprint = newStory(f.projectA, f.enSprintTitle, "En Sprint", f.sprintA)
	f.enProgreso = newStory(f.projectA, f.enProgresoTitle, "En progreso", f.sprintA)
	f.hecho = newStory(f.projectA, f.hechoTitle, "Hecho", f.sprintA)
	f.pendiente = newStory(f.projectA, f.pendienteTitle, "Pendiente", 0)
	f.finalizada = newStory(f.projectA, f.finalizadaTitle, "En Sprint", f.finishedSprintA)
	f.ajena = newStory(f.projectB, f.ajenaTitle, "En progreso", f.sprintB)

	// Mismo día para Beto y Ana el 12/10: Ana carga después, así que aparece primero.
	f.logEffort(t, f.enProgreso, f.ana, "2026-10-10", "Diseño", "2.5")
	f.logEffort(t, f.enProgreso, f.beto, "2026-10-12", "Pruebas", "1.25")
	f.logEffort(t, f.enProgreso, f.ana, "2026-10-12", "Código", "3")
	f.logEffort(t, f.hecho, f.beto, "2026-10-11", "Revisión", "1")
	f.logEffort(t, f.finalizada, f.ana, "2026-10-02", "Cierre", "4")
	f.logEffort(t, f.ajena, f.carla, "2026-10-05", "Ajena", "2")
	return f
}

func (f *effortFixture) id(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(query, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *effortFixture) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *effortFixture) logEffort(t *testing.T, story, member int64, date, activity, hours string) {
	t.Helper()
	f.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, $3::date, $4, $5::numeric)`, story, member, date, activity, hours)
}

// cleanup borra todo lo creado, incluso las cargas hechas por las pruebas a través de la web.
func (f *effortFixture) cleanup() {
	projects := []any{f.projectA, f.projectB, f.projectC}
	_, _ = f.db.Exec(`DELETE FROM effort_logs WHERE story_id IN
		(SELECT id FROM user_stories WHERE project_id IN ($1, $2, $3))`, projects...)
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2, $3)`, projects...)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2, $3)`, projects...)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3)`, f.ana, f.beto, f.carla)
}

// countEffort cuenta las cargas de una historia.
func (f *effortFixture) countEffort(t *testing.T, story int64) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM effort_logs WHERE story_id = $1`, story).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar las cargas: %v", err)
	}
	return n
}

// countAllEffort cuenta las cargas de todas las historias de la fixture.
func (f *effortFixture) countAllEffort(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM effort_logs WHERE story_id IN
		(SELECT id FROM user_stories WHERE project_id IN ($1, $2, $3))`, f.projectA, f.projectB, f.projectC).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar las cargas: %v", err)
	}
	return n
}

func TestPostgreSQLEffortCatalog_ActiveSprintStories_SoloDelSprintActivoConHoras(t *testing.T) {
	db := openContextTestDB(t)
	fx := newEffortFixture(t, db)

	sprint, err := NewPostgreSQLEffortCatalog(db).ActiveSprintStories(context.Background(), fx.projectA)
	if err != nil {
		t.Fatalf("ActiveSprintStories devolvió error: %v", err)
	}
	if sprint == nil {
		t.Fatal("se esperaba el Sprint activo del proyecto A")
	}
	if sprint.Number != 2 {
		t.Errorf("número del Sprint activo: se esperaba 2, se obtuvo %d", sprint.Number)
	}
	want := []EffortStoryRow{
		{ID: fx.enSprint, Title: fx.enSprintTitle, Status: "En Sprint", ActualHours: 0},
		{ID: fx.enProgreso, Title: fx.enProgresoTitle, Status: "En progreso", ActualHours: 675},
		{ID: fx.hecho, Title: fx.hechoTitle, Status: "Hecho", ActualHours: 100},
	}
	if len(sprint.Stories) != len(want) {
		t.Fatalf("se esperaban %d historias (sin pendiente, sin Sprint finalizado ni de otro proyecto), se obtuvieron %+v", len(want), sprint.Stories)
	}
	for i, w := range want {
		if sprint.Stories[i] != w {
			t.Errorf("historia %d: se esperaba %+v, se obtuvo %+v", i, w, sprint.Stories[i])
		}
	}
	if got := sprint.Stories[1].HoursText(); got != "6,75" {
		t.Errorf("horas con coma decimal: se esperaba 6,75, se obtuvo %q", got)
	}
}

func TestPostgreSQLEffortCatalog_ActiveSprintStories_OtroProyectoYSinSprint(t *testing.T) {
	db := openContextTestDB(t)
	fx := newEffortFixture(t, db)
	catalog := NewPostgreSQLEffortCatalog(db)

	sprintB, err := catalog.ActiveSprintStories(context.Background(), fx.projectB)
	if err != nil || sprintB == nil {
		t.Fatalf("proyecto B: se esperaba su Sprint activo, se obtuvo %+v, %v", sprintB, err)
	}
	if len(sprintB.Stories) != 1 || sprintB.Stories[0].ID != fx.ajena || sprintB.Stories[0].ActualHours != 200 {
		t.Errorf("proyecto B: se esperaba solo su historia con 2 h, se obtuvo %+v", sprintB.Stories)
	}

	none, err := catalog.ActiveSprintStories(context.Background(), fx.projectC)
	if err != nil {
		t.Fatalf("proyecto sin Sprint activo devolvió error: %v", err)
	}
	if none != nil {
		t.Errorf("proyecto sin Sprint activo: se esperaba nil, se obtuvo %+v", none)
	}
}

func TestPostgreSQLEffortCatalog_ConDosSprintsActivosTomaElDeMayorNumero(t *testing.T) {
	db := openContextTestDB(t)
	fx := newEffortFixture(t, db)
	// FIX-HU-04 lo impide desde el servicio; acá se fuerza por SQL para probar la consulta.
	sprint3 := fx.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 3, 'Objetivo HU24', DATE '2026-11-01', DATE '2026-11-30', 'Activo') RETURNING id`, fx.projectA)
	nueva := fx.id(t, `INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
		VALUES ($1, 'HU24 del Sprint 3', 'Descripción HU24', 'Media', 'En Sprint', $2) RETURNING id`, fx.projectA, sprint3)

	catalog := NewPostgreSQLEffortCatalog(db)
	sprint, err := catalog.ActiveSprintStories(context.Background(), fx.projectA)
	if err != nil || sprint == nil {
		t.Fatalf("se esperaba un Sprint activo, se obtuvo %+v, %v", sprint, err)
	}
	if sprint.Number != 3 || len(sprint.Stories) != 1 || sprint.Stories[0].ID != nueva {
		t.Errorf("se esperaba el Sprint 3 con su única historia, se obtuvo %+v", sprint)
	}
	if h, err := catalog.StoryHistory(context.Background(), fx.projectA, fx.enProgreso); err != nil || h != nil {
		t.Errorf("una historia del Sprint activo de menor número no debe tener historial, se obtuvo %+v, %v", h, err)
	}
}

func TestPostgreSQLEffortCatalog_StoryHistory_MasRecientesPrimeroYTotal(t *testing.T) {
	db := openContextTestDB(t)
	fx := newEffortFixture(t, db)

	h, err := NewPostgreSQLEffortCatalog(db).StoryHistory(context.Background(), fx.projectA, fx.enProgreso)
	if err != nil || h == nil {
		t.Fatalf("se esperaba el historial, se obtuvo %+v, %v", h, err)
	}
	if h.StoryID != fx.enProgreso || h.Title != fx.enProgresoTitle {
		t.Errorf("historia: se obtuvo %d %q", h.StoryID, h.Title)
	}
	day := func(s string) time.Time {
		d, _ := time.Parse("2006-01-02", s)
		return d
	}
	want := []EffortEntryRow{
		{MemberName: fx.anaName, Date: day("2026-10-12"), Activity: "Código", Hours: 300},
		{MemberName: fx.betoName, Date: day("2026-10-12"), Activity: "Pruebas", Hours: 125},
		{MemberName: fx.anaName, Date: day("2026-10-10"), Activity: "Diseño", Hours: 250},
	}
	if len(h.Entries) != len(want) {
		t.Fatalf("se esperaban %d cargas, se obtuvieron %+v", len(want), h.Entries)
	}
	var sum effort.Hours
	for i, w := range want {
		got := h.Entries[i]
		if got.MemberName != w.MemberName || !sameDay(got.Date, w.Date) || got.Activity != w.Activity || got.Hours != w.Hours {
			t.Errorf("carga %d: se esperaba %+v, se obtuvo %+v", i, w, got)
		}
		sum += got.Hours
	}
	if h.Total != 675 || h.Total != sum {
		t.Errorf("total: se esperaba 6,75 = suma de las cargas, se obtuvo %s", h.TotalText())
	}
}

func TestPostgreSQLEffortCatalog_StoryHistory_SinCargasYNoEncontradas(t *testing.T) {
	db := openContextTestDB(t)
	fx := newEffortFixture(t, db)
	catalog := NewPostgreSQLEffortCatalog(db)

	h, err := catalog.StoryHistory(context.Background(), fx.projectA, fx.enSprint)
	if err != nil || h == nil {
		t.Fatalf("historia sin cargas: se esperaba historial vacío, se obtuvo %+v, %v", h, err)
	}
	if len(h.Entries) != 0 || h.Total != 0 || h.TotalText() != "0" {
		t.Errorf("historia sin cargas: se esperaba lista vacía y total 0, se obtuvo %+v", h)
	}

	cases := map[string][2]int64{
		"pendiente sin Sprint":       {fx.projectA, fx.pendiente},
		"de un Sprint finalizado":    {fx.projectA, fx.finalizada},
		"de otro proyecto":           {fx.projectA, fx.ajena},
		"propia pedida desde otro":   {fx.projectB, fx.enProgreso},
		"inexistente":                {fx.projectA, 2147483000},
		"proyecto sin Sprint activo": {fx.projectC, fx.enSprint},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			h, err := catalog.StoryHistory(context.Background(), c[0], c[1])
			if err != nil {
				t.Fatalf("no se esperaba error: %v", err)
			}
			if h != nil {
				t.Errorf("se esperaba \"no encontrada\" (nil), se obtuvo %+v", h)
			}
		})
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
