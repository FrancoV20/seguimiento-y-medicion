package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// HU-11, T010-T021: PostgreSQLProjectStore.Update contra PostgreSQL real.

// updateFixture (T010): «Alfa» (2026-03-01 a 2026-06-30) con Ana y Beto y el Sprint 2
// (2026-06-01 a 2026-06-14); «Beta» con Beto y horas de Beto; Carla sin proyecto.
type updateFixture struct {
	db                           *sql.DB
	alfa, beta                   int
	ana, beto, carla             int
	anaName, betoName, carlaName string
	alfaName                     string
	storyAlfa, storyBeta         int64
}

func openUpdateTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newUpdateFixture(t *testing.T) *updateFixture {
	t.Helper()
	db := openUpdateTestDB(t)
	stamp := fmt.Sprint(time.Now().UnixNano())
	f := &updateFixture{db: db, anaName: "Ana HU11 " + stamp, betoName: "Beto HU11 " + stamp,
		carlaName: "Carla HU11 " + stamp, alfaName: "Alfa HU11 " + stamp}
	t.Cleanup(f.cleanup)

	f.alfa = int(f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-03-01', '2026-06-30') RETURNING id`, f.alfaName))
	f.beta = int(f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, "Beta HU11 "+stamp))
	f.ana = int(f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, f.anaName))
	f.beto = int(f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, f.betoName))
	f.carla = int(f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, f.carlaName))
	for _, pm := range [][2]int{{f.alfa, f.ana}, {f.alfa, f.beto}, {f.beta, f.beto}} {
		f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, pm[0], pm[1])
	}
	f.storyAlfa = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, 'HU11 historia Alfa', 'd', 'Alta') RETURNING id`, f.alfa)
	f.storyBeta = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, 'HU11 historia Beta', 'd', 'Alta') RETURNING id`, f.beta)
	f.sprint(t, 2, "2026-06-01", "2026-06-14")
	f.hours(t, f.storyBeta, f.beto)
	return f
}

func (f *updateFixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *updateFixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *updateFixture) sprint(t *testing.T, number int, start, end string) {
	t.Helper()
	f.exec(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date) VALUES ($1, $2, 'Objetivo HU11', $3, $4)`, f.alfa, number, start, end)
}

func (f *updateFixture) hours(t *testing.T, story int64, member int) {
	t.Helper()
	f.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours) VALUES ($1, $2, '2026-03-10', 'HU11', 2)`, story, member)
}

// cleanup borra en orden: horas, historias (las asignaciones caen en cascada), proyectos (con
// sus Sprints e integrantes) e integrantes propios.
func (f *updateFixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2))`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3)`, f.ana, f.beto, f.carla)
}

// projectState es todo lo que una modificación puede cambiar, más lo que no debe cambiar.
type projectState struct {
	Name, Start, End, Status string
	NextSprint               int
	Members                  []int
	Sprints                  string
	Effort                   int
}

func (f *updateFixture) state(t *testing.T, project int) projectState {
	t.Helper()
	var s projectState
	if err := f.db.QueryRow(`SELECT name, to_char(start_date, 'YYYY-MM-DD'), to_char(end_date, 'YYYY-MM-DD'), status, next_sprint_number FROM projects WHERE id = $1`, project).
		Scan(&s.Name, &s.Start, &s.End, &s.Status, &s.NextSprint); err != nil {
		t.Fatalf("no se pudo leer el proyecto: %v", err)
	}
	rows, err := f.db.Query(`SELECT member_id FROM project_members WHERE project_id = $1 ORDER BY member_id`, project)
	if err != nil {
		t.Fatalf("no se pudieron leer los integrantes: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		s.Members = append(s.Members, id)
	}
	_ = f.db.QueryRow(`SELECT COALESCE(string_agg(number || ':' || start_date || '/' || end_date, ',' ORDER BY number), '') FROM sprints WHERE project_id = $1`, project).Scan(&s.Sprints)
	_ = f.db.QueryRow(`SELECT COUNT(*) FROM effort_logs el JOIN user_stories us ON us.id = el.story_id WHERE us.project_id IN ($1, $2)`, f.alfa, f.beta).Scan(&s.Effort)
	return s
}

func (f *updateFixture) update(name string, start, end string, members ...int) ProjectUpdate {
	return ProjectUpdate{ID: f.alfa, Name: name, MemberIDs: members, StartDate: day(start), EndDate: day(end)}
}

func (f *updateFixture) assertUnchanged(t *testing.T, before projectState) {
	t.Helper()
	if after := f.state(t, f.alfa); !reflect.DeepEqual(after, before) {
		t.Errorf("el proyecto no debía cambiar:\nantes   %+v\ndespués %+v", before, after)
	}
}

func sortedInts(ids ...int) []int {
	out := append([]int(nil), ids...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// T011: cambia nombre y fechas; lo demás no cambia. Es la primera prueba con arreglos int[].
func TestPostgresUpdate_NombreYFechas(t *testing.T) {
	f := newUpdateFixture(t)
	before := f.state(t, f.alfa)
	store := NewPostgreSQLProjectStore(f.db)

	if err := store.Update(context.Background(), f.update("Alfa 2", "2026-02-01", "2026-07-31", f.ana, f.beto)); err != nil {
		t.Fatalf("Update devolvió error: %v", err)
	}
	after := f.state(t, f.alfa)
	if after.Name != "Alfa 2" || after.Start != "2026-02-01" || after.End != "2026-07-31" {
		t.Errorf("no se guardaron el nombre y las fechas: %+v", after)
	}
	if after.Status != before.Status || after.NextSprint != before.NextSprint || after.Sprints != before.Sprints ||
		after.Effort != before.Effort || !reflect.DeepEqual(after.Members, before.Members) {
		t.Errorf("cambió algo que no debía:\nantes   %+v\ndespués %+v", before, after)
	}
}

// T012
func TestPostgresUpdate_Alta(t *testing.T) {
	f := newUpdateFixture(t)
	if err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update(f.alfaName, "2026-03-01", "2026-06-30", f.ana, f.beto, f.carla)); err != nil {
		t.Fatalf("Update devolvió error: %v", err)
	}
	if got := f.state(t, f.alfa).Members; !reflect.DeepEqual(got, sortedInts(f.ana, f.beto, f.carla)) {
		t.Errorf("se esperaban Ana, Beto y Carla, se obtuvo %v", got)
	}
}

// T013: Beto tiene horas solo en «Beta»: se puede quitar de «Alfa».
func TestPostgresUpdate_BajaSinHoras(t *testing.T) {
	f := newUpdateFixture(t)
	before := f.state(t, f.alfa)
	if err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update(f.alfaName, "2026-03-01", "2026-06-30", f.ana)); err != nil {
		t.Fatalf("Update devolvió error: %v", err)
	}
	after := f.state(t, f.alfa)
	if !reflect.DeepEqual(after.Members, []int{f.ana}) || after.Effort != before.Effort {
		t.Errorf("Beto debía salir de «Alfa» y sus horas en «Beta» seguir: %+v", after)
	}
}

// T014: Beto con horas en «Alfa»: se bloquea todo el cambio.
func TestPostgresUpdate_BajaConHoras(t *testing.T) {
	f := newUpdateFixture(t)
	f.hours(t, f.storyAlfa, f.beto)
	before := f.state(t, f.alfa)

	err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update("Otro nombre", "2026-03-01", "2026-06-30", f.ana, f.carla))
	var hasEffort *MemberHasEffortError
	if !errors.As(err, &hasEffort) || hasEffort.MemberName != f.betoName {
		t.Fatalf("se esperaba MemberHasEffortError con Beto, se obtuvo %v", err)
	}
	f.assertUnchanged(t, before)
}

// T015: con varios integrantes con horas se nombra al primero por nombre.
func TestPostgresUpdate_VariosConHoras_NombraAlPrimero(t *testing.T) {
	f := newUpdateFixture(t)
	f.hours(t, f.storyAlfa, f.beto)
	f.hours(t, f.storyAlfa, f.ana)
	err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update(f.alfaName, "2026-03-01", "2026-06-30", f.carla))
	var hasEffort *MemberHasEffortError
	if !errors.As(err, &hasEffort) || hasEffort.MemberName != f.anaName {
		t.Errorf("se esperaba nombrar a Ana, se obtuvo %v", err)
	}
}

// T016
func TestPostgresUpdate_IntegranteInexistente(t *testing.T) {
	f := newUpdateFixture(t)
	before := f.state(t, f.alfa)
	err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update("Otro", "2026-03-01", "2026-06-30", f.ana, 2147483647))
	if !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("se esperaba ErrMemberNotFound, se obtuvo %v", err)
	}
	f.assertUnchanged(t, before)
}

// T017
func TestPostgresUpdate_ProyectoInexistente(t *testing.T) {
	f := newUpdateFixture(t)
	u := f.update("Fantasma", "2026-03-01", "2026-06-30", f.ana)
	u.ID = 2147483000
	if err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), u); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("se esperaba ErrProjectNotFound, se obtuvo %v", err)
	}
	var n int
	_ = f.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE name = 'Fantasma'`).Scan(&n)
	if n != 0 {
		t.Error("no debía crearse ningún proyecto")
	}
}

// T018
func TestPostgresUpdate_SprintFuera(t *testing.T) {
	f := newUpdateFixture(t)
	before := f.state(t, f.alfa)
	err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update("Otro", "2026-03-01", "2026-06-10", f.ana, f.beto))
	var out *SprintOutOfRangeError
	if !errors.As(err, &out) || *out != (SprintOutOfRangeError{Number: 2, Field: DateFieldEnd}) {
		t.Fatalf("se esperaba el Sprint 2 fuera por el fin, se obtuvo %v", err)
	}
	f.assertUnchanged(t, before)
}

func TestPostgresUpdate_SprintYaFuera(t *testing.T) {
	f := newUpdateFixture(t)
	f.sprint(t, 1, "2026-02-20", "2026-03-05")
	if err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update(f.alfaName, "2026-03-01", "2026-07-31", f.ana, f.beto)); err != nil {
		t.Fatalf("un Sprint que ya estaba fuera no debía bloquear: %v", err)
	}
	if got := f.state(t, f.alfa); got.End != "2026-07-31" || !strings.HasPrefix(got.Sprints, "1:2026-02-20/2026-03-05") {
		t.Errorf("se esperaba el fin nuevo y el Sprint 1 sin cambios: %+v", got)
	}
}

func TestPostgresUpdate_SoloNombreConSprints(t *testing.T) {
	f := newUpdateFixture(t)
	f.sprint(t, 1, "2026-02-20", "2026-03-05") // dato viejo fuera de rango: no se revisa
	if err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update("Solo nombre", "2026-03-01", "2026-06-30", f.ana, f.beto)); err != nil {
		t.Fatalf("cambiar solo el nombre debía guardarse: %v", err)
	}
	if got := f.state(t, f.alfa).Name; got != "Solo nombre" {
		t.Errorf("se esperaba el nombre nuevo, se obtuvo %q", got)
	}
}

// T019
func TestPostgresUpdate_SinCambios(t *testing.T) {
	f := newUpdateFixture(t)
	before := f.state(t, f.alfa)
	if err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update(f.alfaName, "2026-03-01", "2026-06-30", f.beto, f.ana)); err != nil {
		t.Fatalf("guardar sin cambios debía ser un éxito: %v", err)
	}
	f.assertUnchanged(t, before)
}

func TestPostgresUpdate_FalloNoDejaCambios(t *testing.T) {
	f := newUpdateFixture(t)
	before := f.state(t, f.alfa)
	// 256 caracteres: el UPDATE (paso 7) falla después de quitar a Beto (paso 4).
	err := NewPostgreSQLProjectStore(f.db).Update(context.Background(), f.update(strings.Repeat("x", 256), "2026-03-01", "2026-06-30", f.ana))
	if err == nil {
		t.Fatal("se esperaba una falla de almacenamiento")
	}
	f.assertUnchanged(t, before)
}

// T020: dos guardados simultáneos: gana uno completo, nunca una mezcla.
func TestPostgresUpdate_ConcurrenteUltimaGana(t *testing.T) {
	f := newUpdateFixture(t)
	store := NewPostgreSQLProjectStore(f.db)
	a := f.update("Versión A", "2026-03-01", "2026-06-30", f.ana, f.carla)
	b := f.update("Versión B", "2026-02-01", "2026-07-31", f.ana, f.beto, f.carla)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, u := range []ProjectUpdate{a, b} {
		wg.Add(1)
		go func(i int, u ProjectUpdate) {
			defer wg.Done()
			errs[i] = store.Update(context.Background(), u)
		}(i, u)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("los dos guardados debían terminar sin error: %v", errs)
	}
	got := f.state(t, f.alfa)
	isA := got.Name == "Versión A" && got.Start == "2026-03-01" && got.End == "2026-06-30" && reflect.DeepEqual(got.Members, sortedInts(f.ana, f.carla))
	isB := got.Name == "Versión B" && got.Start == "2026-02-01" && got.End == "2026-07-31" && reflect.DeepEqual(got.Members, sortedInts(f.ana, f.beto, f.carla))
	if !isA && !isB {
		t.Errorf("el resultado mezcla los dos guardados: %+v", got)
	}
}

// waitForLock espera hasta que alguna sesión de la base quede esperando un bloqueo.
func waitForLock(t *testing.T, db *sql.DB) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&n)
		if n > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("la operación no quedó esperando el bloqueo")
}

// T021: una carga de horas en curso (con la pertenencia bloqueada) hace esperar a la baja, que
// después ve las horas y se rechaza.
func TestPostgresUpdate_EsperaCargaDeHoras(t *testing.T) {
	f := newUpdateFixture(t)
	ctx := context.Background()
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT 1 FROM project_members WHERE project_id = $1 AND member_id = $2 FOR SHARE`, f.alfa, f.beto); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours) VALUES ($1, $2, '2026-03-11', 'HU11 en curso', 1)`, f.storyAlfa, f.beto); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- NewPostgreSQLProjectStore(f.db).Update(ctx, f.update(f.alfaName, "2026-03-01", "2026-06-30", f.ana))
	}()
	waitForLock(t, f.db)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var hasEffort *MemberHasEffortError
	if err := <-done; !errors.As(err, &hasEffort) {
		t.Fatalf("se esperaba MemberHasEffortError, se obtuvo %v", err)
	}
	if got := f.state(t, f.alfa).Members; !reflect.DeepEqual(got, sortedInts(f.ana, f.beto)) {
		t.Errorf("Beto debía seguir en «Alfa», se obtuvo %v", got)
	}
}
