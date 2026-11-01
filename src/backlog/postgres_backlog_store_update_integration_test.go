package backlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

// HU-13, T009: datos de prueba de la edición en PostgreSQL. Los Sprints 'Finalizado' se preparan
// directamente en la base, como los deja el cierre de HU-15.
//
//	«Alfa»: Sprint 1 'Finalizado' («Pagos», Hecho); Sprint 2 'Activo' («Login», En Sprint, Media,
//	        5 SP, 4 horas, criterios «Acepta email» y «Rechaza clave vacía»; «Cierre», Hecho);
//	        Sprint 3 'Activo' sin historias (destino de T020 y T021).
//	«Beta»: «Facturas», sin Sprint.
type editFixture struct {
	db                          *sql.DB
	store                       *PostgreSQLBacklogStore
	alfa, beta                  int64
	sprint1, sprint2, sprint3   int64
	login, pagos, cierre, factu int64
}

func newEditFixture(t *testing.T) *editFixture {
	t.Helper()
	db := openTestDB(t)
	f := &editFixture{db: db, store: NewPostgreSQLBacklogStore(db)}
	t.Cleanup(f.cleanup)

	stamp := time.Now().UnixNano()
	f.alfa = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, CURRENT_DATE, CURRENT_DATE + 60) RETURNING id`,
		fmt.Sprintf("HU13 Alfa %d", stamp))
	f.beta = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, CURRENT_DATE, CURRENT_DATE + 60) RETURNING id`,
		fmt.Sprintf("HU13 Beta %d", stamp))
	f.sprint1 = f.sprint(t, 1, "Finalizado")
	f.sprint2 = f.sprint(t, 2, "Activo")
	f.sprint3 = f.sprint(t, 3, "Activo")

	f.login = f.story(t, f.alfa, "Login", "Media", 5, 4, "En Sprint", f.sprint2, "Acepta email", "Rechaza clave vacía")
	f.pagos = f.story(t, f.alfa, "Pagos", "Alta", 3, 0, "Hecho", f.sprint1, "Cobra con tarjeta")
	f.cierre = f.story(t, f.alfa, "Cierre", "Baja", 2, 0, "Hecho", f.sprint2, "Cierra la caja")
	f.factu = f.story(t, f.beta, "Facturas", "Media", 8, 0, "Pendiente", 0, "Emite la factura")
	return f
}

func (f *editFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *editFixture) sprint(t *testing.T, number int, status string) int64 {
	t.Helper()
	return f.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, 'Objetivo', CURRENT_DATE, CURRENT_DATE + 14, $3) RETURNING id`, f.alfa, number, status)
}

// story crea una historia con sus criterios; points u hours en 0 = sin valor; sprintID 0 = sin Sprint.
func (f *editFixture) story(t *testing.T, project int64, title, priority string, points int, hours float64,
	status string, sprintID int64, criteria ...string) int64 {
	t.Helper()
	var sp, h, active any
	if points != 0 {
		sp = points
	}
	if hours != 0 {
		h = hours
	}
	if sprintID != 0 {
		active = sprintID
	}
	id := f.id(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours, status, active_sprint_id)
		VALUES ($1, $2, 'Descripción de '||$2, $3, $4, $5, $6, $7) RETURNING id`, project, title, priority, sp, h, status, active)
	for i, c := range criteria {
		if _, err := f.db.Exec(`INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, $2, $3)`, id, i+1, c); err != nil {
			t.Fatalf("no se pudo preparar el criterio: %v", err)
		}
	}
	if sprintID != 0 {
		if _, err := f.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id); err != nil {
			t.Fatalf("no se pudo asignar la historia: %v", err)
		}
	}
	return id
}

// cleanup: asignaciones, criterios, historias, Sprints y proyectos.
func (f *editFixture) cleanup() {
	for _, p := range []int64{f.alfa, f.beta} {
		_, _ = f.db.Exec(`DELETE FROM sprint_story_assignments WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, p)
		_, _ = f.db.Exec(`DELETE FROM acceptance_criteria WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, p)
		_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, p)
		_, _ = f.db.Exec(`DELETE FROM projects WHERE id = $1`, p)
	}
}

// storedStory es lo que hay en la base de una historia.
type storedStory struct {
	Title, Description, Priority, Status string
	StoryPoints                          sql.NullInt64
	Hours                                sql.NullFloat64
	ActiveSprint                         sql.NullInt64
	CreatedAt                            time.Time
	Sprint                               sql.NullInt64
	Criteria                             []string
}

func (f *editFixture) read(t *testing.T, storyID int64) storedStory {
	t.Helper()
	var s storedStory
	if err := f.db.QueryRow(`
		SELECT us.title, us.description, us.priority, us.status, us.story_points, us.estimated_hours::float8,
		       us.active_sprint_id, us.created_at, a.sprint_id
		FROM user_stories us LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
		WHERE us.id = $1`, storyID).Scan(&s.Title, &s.Description, &s.Priority, &s.Status, &s.StoryPoints,
		&s.Hours, &s.ActiveSprint, &s.CreatedAt, &s.Sprint); err != nil {
		t.Fatalf("no se pudo leer la historia %d: %v", storyID, err)
	}
	rows, err := f.db.Query(`SELECT position, content FROM acceptance_criteria WHERE story_id = $1 ORDER BY position`, storyID)
	if err != nil {
		t.Fatalf("no se pudieron leer los criterios: %v", err)
	}
	defer rows.Close()
	for want := 1; rows.Next(); want++ {
		var pos int
		var content string
		if err := rows.Scan(&pos, &content); err != nil {
			t.Fatal(err)
		}
		if pos != want {
			t.Errorf("criterio en la posición %d; se esperaba %d", pos, want)
		}
		s.Criteria = append(s.Criteria, content)
	}
	return s
}

// edit arma la historia que recibe el almacén (los criterios ya numerados, como los deja el servicio).
func edit(id int64, title, priority string, points *int, criteria ...string) UserStory {
	story := UserStory{ID: id, Title: title, Description: "Descripción nueva de " + title, Priority: Priority(priority), StoryPoints: points}
	for i, c := range criteria {
		story.AcceptanceCriteria = append(story.AcceptanceCriteria, AcceptanceCriterion{Position: i + 1, Content: c})
	}
	return story
}

// T010
func TestPostgresUpdateStory_DatosYSinEfectos(t *testing.T) {
	f := newEditFixture(t)
	before := f.read(t, f.login)

	if err := f.store.UpdateStory(context.Background(), f.alfa,
		edit(f.login, "Inicio de sesión", "Alta", intPtr(8), "Acepta email", "Rechaza clave vacía")); err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}
	after := f.read(t, f.login)
	if after.Title != "Inicio de sesión" || after.Description != "Descripción nueva de Inicio de sesión" ||
		after.Priority != "Alta" || !after.StoryPoints.Valid || after.StoryPoints.Int64 != 8 {
		t.Errorf("datos guardados = %+v", after)
	}
	if after.Status != before.Status || after.Hours != before.Hours || after.ActiveSprint != before.ActiveSprint ||
		!after.CreatedAt.Equal(before.CreatedAt) || after.Sprint != before.Sprint {
		t.Errorf("cambiaron datos fuera de alcance:\nantes   %+v\ndespués %+v", before, after)
	}
}

// T011
func TestPostgresUpdateStory_ReemplazaCriterios(t *testing.T) {
	f := newEditFixture(t)
	ctx := context.Background()

	three := []string{"Rechaza clave vacía", "Bloquea tras tres intentos", "Acepta email"}
	if err := f.store.UpdateStory(ctx, f.alfa, edit(f.login, "Login", "Media", intPtr(5), three...)); err != nil {
		t.Fatalf("UpdateStory con tres criterios: %v", err)
	}
	if got := f.read(t, f.login).Criteria; !reflect.DeepEqual(got, three) {
		t.Errorf("criterios = %v; se esperaba %v", got, three)
	}

	if err := f.store.UpdateStory(ctx, f.alfa, edit(f.login, "Login", "Media", intPtr(5), "Solo uno")); err != nil {
		t.Fatalf("UpdateStory con un criterio: %v", err)
	}
	if got := f.read(t, f.login).Criteria; !reflect.DeepEqual(got, []string{"Solo uno"}) {
		t.Errorf("criterios = %v; se esperaba [Solo uno]", got)
	}
}

// T012
func TestPostgresUpdateStory_SinStoryPoints(t *testing.T) {
	f := newEditFixture(t)
	if err := f.store.UpdateStory(context.Background(), f.alfa, edit(f.login, "Login", "Media", nil, "Acepta email")); err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}
	if sp := f.read(t, f.login).StoryPoints; sp.Valid {
		t.Errorf("story_points = %d; se esperaba NULL", sp.Int64)
	}
}

// T013
func TestPostgresUpdateStory_InexistenteYOtroProyecto(t *testing.T) {
	f := newEditFixture(t)
	ctx := context.Background()
	if err := f.store.UpdateStory(ctx, f.alfa, edit(2147483647, "X", "Alta", nil, "C")); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("historia 2147483647: error = %v; se esperaba ErrStoryNotFound", err)
	}
	before := f.read(t, f.factu)
	if err := f.store.UpdateStory(ctx, f.alfa, edit(f.factu, "Cambiada", "Alta", nil, "C")); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("«Facturas» con «Alfa»: error = %v; se esperaba ErrStoryNotFound", err)
	}
	if after := f.read(t, f.factu); !reflect.DeepEqual(after, before) {
		t.Errorf("«Facturas» cambió:\nantes   %+v\ndespués %+v", before, after)
	}
}

// T014
func TestPostgresUpdateStory_SprintFinalizado(t *testing.T) {
	f := newEditFixture(t)
	before := f.read(t, f.pagos)
	if err := f.store.UpdateStory(context.Background(), f.alfa, edit(f.pagos, "Pagos nuevo", "Baja", intPtr(13), "Otro")); !errors.Is(err, ErrStoryInFinishedSprint) {
		t.Fatalf("error = %v; se esperaba ErrStoryInFinishedSprint", err)
	}
	if after := f.read(t, f.pagos); !reflect.DeepEqual(after, before) {
		t.Errorf("«Pagos» cambió:\nantes   %+v\ndespués %+v", before, after)
	}
}

// T015
func TestPostgresUpdateStory_HechoEnSprintActivo(t *testing.T) {
	f := newEditFixture(t)
	if err := f.store.UpdateStory(context.Background(), f.alfa, edit(f.cierre, "Cierre de caja", "Baja", intPtr(3), "Cierra la caja")); err != nil {
		t.Fatalf("UpdateStory de una historia Hecho de un Sprint activo: %v", err)
	}
	if after := f.read(t, f.cierre); after.Title != "Cierre de caja" || after.Status != "Hecho" {
		t.Errorf("«Cierre» = %+v; se esperaba el título nuevo y el estado Hecho", after)
	}
}

// T016
func TestPostgresUpdateStory_FalloNoDejaCambios(t *testing.T) {
	f := newEditFixture(t)
	before := f.read(t, f.login)
	// El criterio "   " llega directo al almacén y lo rechaza el CHECK en el INSERT (paso 5),
	// después del UPDATE y del DELETE.
	err := f.store.UpdateStory(context.Background(), f.alfa, edit(f.login, "Título que no queda", "Alta", intPtr(13), "Nuevo", "   "))
	if err == nil || errors.Is(err, ErrStoryNotFound) || errors.Is(err, ErrStoryInFinishedSprint) {
		t.Fatalf("error = %v; se esperaba una falla de almacenamiento", err)
	}
	if after := f.read(t, f.login); !reflect.DeepEqual(after, before) {
		t.Errorf("quedaron cambios parciales:\nantes   %+v\ndespués %+v", before, after)
	}
}

// T017
func TestPostgresUpdateStory_SinCambios(t *testing.T) {
	f := newEditFixture(t)
	before := f.read(t, f.login)
	same := UserStory{ID: f.login, Title: before.Title, Description: before.Description, Priority: Priority(before.Priority), StoryPoints: intPtr(5)}
	for i, c := range before.Criteria {
		same.AcceptanceCriteria = append(same.AcceptanceCriteria, AcceptanceCriterion{Position: i + 1, Content: c})
	}
	if err := f.store.UpdateStory(context.Background(), f.alfa, same); err != nil {
		t.Fatalf("UpdateStory sin cambios: %v", err)
	}
	if after := f.read(t, f.login); !reflect.DeepEqual(after, before) {
		t.Errorf("la historia cambió:\nantes   %+v\ndespués %+v", before, after)
	}
}

// T018
func TestPostgresUpdateStory_ConcurrenteUltimaGana(t *testing.T) {
	f := newEditFixture(t)
	a := edit(f.login, "Versión A", "Alta", intPtr(3), "A1", "A2")
	b := edit(f.login, "Versión B", "Baja", intPtr(13), "B1", "B2", "B3")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, story := range []UserStory{a, b} {
		wg.Add(1)
		go func(i int, story UserStory) {
			defer wg.Done()
			errs[i] = f.store.UpdateStory(context.Background(), f.alfa, story)
		}(i, story)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("guardado %d: %v", i, err)
		}
	}
	got := f.read(t, f.login)
	switch got.Title {
	case "Versión A":
		if !reflect.DeepEqual(got.Criteria, []string{"A1", "A2"}) {
			t.Errorf("quedó la versión A con los criterios %v", got.Criteria)
		}
	case "Versión B":
		if !reflect.DeepEqual(got.Criteria, []string{"B1", "B2", "B3"}) {
			t.Errorf("quedó la versión B con los criterios %v", got.Criteria)
		}
	default:
		t.Errorf("título = %q; se esperaba una de las dos versiones", got.Title)
	}
}

// T019
func TestPostgresUpdateStory_EsperaAlCierre(t *testing.T) {
	f := newEditFixture(t)
	ctx := context.Background()
	before := f.read(t, f.login)

	closer, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closer.Rollback() }()
	if _, err := closer.ExecContext(ctx, `SELECT 1 FROM sprints WHERE id = $1 FOR UPDATE`, f.sprint2); err != nil {
		t.Fatal(err)
	}
	if _, err := closer.ExecContext(ctx, `UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, f.sprint2); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- f.store.UpdateStory(ctx, f.alfa, edit(f.login, "Después del cierre", "Alta", intPtr(8), "C"))
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int
		if err := f.db.QueryRow(`SELECT COUNT(*) FROM pg_stat_activity
			WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%FOR SHARE%'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("UpdateStory terminó sin esperar al cierre: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("UpdateStory no quedó esperando el bloqueo del Sprint")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := closer.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := <-done; !errors.Is(err, ErrStoryInFinishedSprint) {
		t.Fatalf("error = %v; se esperaba ErrStoryInFinishedSprint", err)
	}
	after := f.read(t, f.login)
	if after.Title != before.Title || !reflect.DeepEqual(after.Criteria, before.Criteria) || after.StoryPoints != before.StoryPoints {
		t.Errorf("«Login» cambió:\nantes   %+v\ndespués %+v", before, after)
	}
}

// moveLogin cambia el Sprint de «Login» desde otra conexión, como haría HU-23.
func (f *editFixture) moveLogin(t *testing.T, to int64) {
	t.Helper()
	if _, err := f.db.Exec(`UPDATE sprint_story_assignments SET sprint_id = $1 WHERE story_id = $2`, to, f.login); err != nil {
		t.Errorf("no se pudo mover «Login»: %v", err)
	}
}

// T020
func TestPostgresUpdateStory_AsignacionCambiaReintenta(t *testing.T) {
	f := newEditFixture(t)
	calls := 0
	f.store.beforeStoryLock = func() {
		calls++
		if calls == 1 {
			f.moveLogin(t, f.sprint3)
		}
	}
	if err := f.store.UpdateStory(context.Background(), f.alfa, edit(f.login, "Con reintento", "Alta", intPtr(8), "C")); err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}
	if calls != 2 {
		t.Errorf("el gancho corrió %d veces; se esperaban 2 (un reintento)", calls)
	}
	if after := f.read(t, f.login); after.Title != "Con reintento" || !after.Sprint.Valid || after.Sprint.Int64 != f.sprint3 {
		t.Errorf("«Login» = %+v; se esperaba guardada en el Sprint 3", after)
	}
}

// T021
func TestPostgresUpdateStory_AsignacionCambiaDosVeces(t *testing.T) {
	f := newEditFixture(t)
	before := f.read(t, f.login)
	targets := []int64{f.sprint3, f.sprint2}
	calls := 0
	f.store.beforeStoryLock = func() {
		if calls < len(targets) {
			f.moveLogin(t, targets[calls])
		}
		calls++
	}
	err := f.store.UpdateStory(context.Background(), f.alfa, edit(f.login, "No queda", "Alta", intPtr(8), "C"))
	if err == nil || errors.Is(err, ErrStoryNotFound) || errors.Is(err, ErrStoryInFinishedSprint) {
		t.Fatalf("error = %v; se esperaba un error que no es de dominio", err)
	}
	after := f.read(t, f.login)
	if after.Title != before.Title || !reflect.DeepEqual(after.Criteria, before.Criteria) {
		t.Errorf("«Login» cambió:\nantes   %+v\ndespués %+v", before, after)
	}
}
