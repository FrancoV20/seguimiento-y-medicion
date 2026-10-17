package effort

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Estas pruebas corren contra PostgreSQL real. Necesitan la variable DATABASE_URL y las
// migraciones 001 a 004 aplicadas. Si DATABASE_URL no está definida, se omiten.

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no está definida: se omite la prueba de integración con PostgreSQL")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL (¿Docker levantado y DATABASE_URL correcta?): %v", err)
	}
	return db
}

// fixture son los datos de prueba creados en la base; se borran al terminar cada prueba.
type fixture struct {
	project, otherProject                int64
	member, foreignMember                int64
	activeStory, noSprintStory, finished int64
}

func mustID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(query, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v\nconsulta: %s", err, query)
	}
	return id
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("no se pudo preparar el escenario: %v\nconsulta: %s", err, query)
	}
}

func newFixture(t *testing.T, db *sql.DB) fixture {
	t.Helper()
	var f fixture

	const insProject = `INSERT INTO projects (name, start_date, end_date) VALUES ('test-effort', '2026-01-01', '2026-12-31') RETURNING id`
	f.project = mustID(t, db, insProject)
	f.otherProject = mustID(t, db, insProject)

	const insMember = `INSERT INTO members (display_name) VALUES ('Integrante de prueba') RETURNING id`
	f.member = mustID(t, db, insMember)
	f.foreignMember = mustID(t, db, insMember)
	mustExec(t, db, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, f.project, f.member)
	mustExec(t, db, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, f.otherProject, f.foreignMember)

	const insSprint = `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, 'meta de prueba', '2026-10-01', '2026-10-14', $3) RETURNING id`
	activeSprint := mustID(t, db, insSprint, f.project, 1, "Activo")
	finishedSprint := mustID(t, db, insSprint, f.project, 2, "Finalizado")

	const insStory = `INSERT INTO user_stories (project_id, title, description, priority, active_sprint_id)
		VALUES ($1, 'historia de prueba', 'descripcion', 'Alta', $2) RETURNING id`
	f.activeStory = mustID(t, db, insStory, f.project, activeSprint)
	f.noSprintStory = mustID(t, db, insStory, f.project, nil)
	f.finished = mustID(t, db, insStory, f.project, finishedSprint)

	t.Cleanup(func() {
		db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2))`, f.project, f.otherProject)
		db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.project, f.otherProject)
		db.Exec(`DELETE FROM sprints WHERE project_id IN ($1, $2)`, f.project, f.otherProject)
		db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.project, f.otherProject)
		db.Exec(`DELETE FROM members WHERE id IN ($1, $2)`, f.member, f.foreignMember)
	})
	return f
}

func idStr(n int64) string { return strconv.FormatInt(n, 10) }

func TestPostgresStoryAcceptsEffort(t *testing.T) {
	db := openTestDB(t)
	f := newFixture(t, db)
	store := NewPostgreSQLEffortStore(db)

	cases := []struct {
		name    string
		storyID string
		want    bool
	}{
		{"historia en Sprint activo", idStr(f.activeStory), true},
		{"historia sin Sprint", idStr(f.noSprintStory), false},
		{"historia en Sprint finalizado", idStr(f.finished), false},
		{"historia inexistente", "999999999", false},
		{"identificador no numérico", "abc", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := store.StoryAcceptsEffort(context.Background(), c.storyID)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != c.want {
				t.Errorf("StoryAcceptsEffort = %v, se esperaba %v", got, c.want)
			}
		})
	}
}

func TestPostgresMemberBelongsToStoryProject(t *testing.T) {
	db := openTestDB(t)
	f := newFixture(t, db)
	store := NewPostgreSQLEffortStore(db)

	cases := []struct {
		name            string
		memberID, story string
		want            bool
	}{
		{"integrante del proyecto", idStr(f.member), idStr(f.activeStory), true},
		{"integrante de otro proyecto", idStr(f.foreignMember), idStr(f.activeStory), false},
		{"integrante inexistente", "999999999", idStr(f.activeStory), false},
		{"integrante no numérico", "abc", idStr(f.activeStory), false},
		{"historia inexistente", idStr(f.member), "999999999", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := store.MemberBelongsToStoryProject(context.Background(), c.memberID, c.story)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != c.want {
				t.Errorf("MemberBelongsToStoryProject = %v, se esperaba %v", got, c.want)
			}
		})
	}
}

func TestPostgresSaveEntryAccumulates(t *testing.T) {
	db := openTestDB(t)
	f := newFixture(t, db)
	store := NewPostgreSQLEffortStore(db)
	ctx := context.Background()

	save := func(activity string, hours Hours) (EffortEntry, Hours) {
		t.Helper()
		start := time.Now()
		entry, total, err := store.SaveEntry(ctx, EffortEntry{
			StoryID:  idStr(f.activeStory),
			MemberID: idStr(f.member),
			WorkDate: time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC),
			Activity: activity,
			Hours:    hours,
		})
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("SC-005: guardar y devolver el acumulado tardó %v, debía tardar menos de 2 s", elapsed)
		}
		if err != nil {
			t.Fatalf("SaveEntry falló: %v", err)
		}
		return entry, total
	}

	first, total := save("Desarrollo de endpoint en Go", 400)
	if total != 400 {
		t.Errorf("acumulado tras 4 h = %d, se esperaba 400", total)
	}
	if first.ID == "" || first.CreatedAt.IsZero() {
		t.Errorf("el registro guardado debe tener ID y CreatedAt: %+v", first)
	}
	if first.StoryID != idStr(f.activeStory) || first.MemberID != idStr(f.member) ||
		first.Activity != "Desarrollo de endpoint en Go" || first.Hours != 400 {
		t.Errorf("el registro devuelto no coincide con lo guardado: %+v", first)
	}

	var storedDate string
	err := db.QueryRow(`SELECT to_char(work_date, 'YYYY-MM-DD') FROM effort_logs WHERE id = $1`, first.ID).Scan(&storedDate)
	if err != nil {
		t.Fatalf("no se pudo leer el registro: %v", err)
	}
	if storedDate != "2026-10-10" {
		t.Errorf("work_date = %s, se esperaba 2026-10-10", storedDate)
	}

	if _, total = save("Revisión", 250); total != 650 {
		t.Errorf("acumulado tras 2,5 h más = %d, se esperaba 650", total)
	}
	if _, total = save("Ajuste menor", 5); total != 655 {
		t.Errorf("acumulado tras 0,05 h más = %d, se esperaba 655", total)
	}

	got, err := store.AccumulatedHours(ctx, idStr(f.activeStory))
	if err != nil {
		t.Fatalf("AccumulatedHours falló: %v", err)
	}
	if got != 655 {
		t.Errorf("AccumulatedHours = %d, se esperaba 655", got)
	}

	other, err := store.AccumulatedHours(ctx, idStr(f.noSprintStory))
	if err != nil {
		t.Fatalf("AccumulatedHours falló: %v", err)
	}
	if other != 0 {
		t.Errorf("una historia sin registros debe acumular 0, se obtuvo %d", other)
	}
}

func TestPostgresSaveEntryFailureLeavesNoPartialData(t *testing.T) {
	db := openTestDB(t)
	f := newFixture(t, db)
	store := NewPostgreSQLEffortStore(db)
	ctx := context.Background()
	date := time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC)

	if _, _, err := store.SaveEntry(ctx, EffortEntry{
		StoryID: idStr(f.activeStory), MemberID: idStr(f.member), WorkDate: date, Activity: "ok", Hours: 400,
	}); err != nil {
		t.Fatalf("el primer registro debía guardarse: %v", err)
	}

	// Integrante inexistente: viola la clave foránea y el guardado tiene que fallar.
	_, _, err := store.SaveEntry(ctx, EffortEntry{
		StoryID: idStr(f.activeStory), MemberID: "999999999", WorkDate: date, Activity: "falla", Hours: 200,
	})
	if err == nil {
		t.Fatal("se esperaba un error al guardar con un integrante inexistente")
	}

	got, err := store.AccumulatedHours(ctx, idStr(f.activeStory))
	if err != nil {
		t.Fatalf("AccumulatedHours falló: %v", err)
	}
	if got != 400 {
		t.Errorf("el acumulado debe seguir en 400 tras el fallo, se obtuvo %d", got)
	}
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM effort_logs WHERE story_id = $1`, f.activeStory).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Errorf("debe haber 1 registro, hay %d", rows)
	}
}

func TestPostgresCheckConstraintRejectsOutOfRangeHours(t *testing.T) {
	db := openTestDB(t)
	f := newFixture(t, db)

	insert := func(hours string) error {
		q := fmt.Sprintf(`INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
			VALUES (%d, %d, '2026-10-10', 'directo', %s)`, f.activeStory, f.member, hours)
		_, err := db.Exec(q)
		return err
	}

	for _, h := range []string{"0", "-1", "24.01"} {
		if err := insert(h); err == nil {
			t.Errorf("la base debía rechazar %s horas escritas directamente", h)
		}
	}
	for _, h := range []string{"24", "0.5"} {
		if err := insert(h); err != nil {
			t.Errorf("la base debía aceptar %s horas: %v", h, err)
		}
	}
}
