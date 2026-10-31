package backlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"
)

// Pruebas del servicio UpdateEstimatedHours (HU-17, T005) contra PostgreSQL real.
// Usan openTestDB y crearProyectoActivo de postgres_backlog_store_integration_test.go.

// estimateFixture tiene dos proyectos con historias en cada situación que importa a HU-17.
type estimateFixture struct {
	db             *sql.DB
	service        *EstimateService
	store          *PostgreSQLEstimateStore
	projectID      int64 // proyecto activo de las pruebas
	otherID        int64 // otro proyecto
	sinEstimar     int64 // Pendiente, sin horas estimadas
	estimada       int64 // Pendiente, 8 horas estimadas y 5 Story Points
	conHoras       int64 // 3 horas estimadas y horas reales registradas
	enActivo       int64 // asignada al Sprint activo, 4 horas estimadas
	finalizada     int64 // asignada a un Sprint finalizado, 5 horas estimadas
	ajena          int64 // del otro proyecto, 6 horas estimadas
	activeSprintID int64
}

func newEstimateFixture(t *testing.T) *estimateFixture {
	t.Helper()
	db := openTestDB(t)
	store := NewPostgreSQLEstimateStore(db)
	f := &estimateFixture{db: db, store: store, service: NewEstimateService(store)}
	f.projectID = crearProyectoActivo(t, db)
	f.otherID = crearProyectoActivo(t, db)

	f.sinEstimar = f.insertStory(t, f.projectID, "HU17 sin estimar", nil, nil)
	f.estimada = f.insertStory(t, f.projectID, "HU17 estimada", floatPtr(8), intPtr(5))
	f.conHoras = f.insertStory(t, f.projectID, "HU17 con horas reales", floatPtr(3), nil)
	f.enActivo = f.insertStory(t, f.projectID, "HU17 en Sprint activo", floatPtr(4), nil)
	f.finalizada = f.insertStory(t, f.projectID, "HU17 en Sprint finalizado", floatPtr(5), nil)
	f.ajena = f.insertStory(t, f.otherID, "HU17 de otro proyecto", floatPtr(6), nil)

	finishedSprintID := f.insertSprint(t, 1, "Finalizado")
	f.activeSprintID = f.insertSprint(t, 2, "Activo")
	f.assign(t, finishedSprintID, f.finalizada, "Hecho")
	f.assign(t, f.activeSprintID, f.enActivo, "En Sprint")

	var memberID int64
	if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ('HU17 integrante') RETURNING id`).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
	}
	// Se registra después de crearProyectoActivo: corre antes y libera las historias.
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, memberID)
		_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, memberID)
	})
	if _, err := db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, CURRENT_DATE, 'Desarrollo', 1.5), ($1, $2, CURRENT_DATE, 'Pruebas', 2.25)`,
		f.conHoras, memberID); err != nil {
		t.Fatalf("no se pudieron registrar horas reales: %v", err)
	}
	return f
}

func (f *estimateFixture) insertStory(t *testing.T, projectID int64, title string, hours *float64, points *int) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours)
		VALUES ($1, $2, 'Historia de prueba HU-17', 'Media', $3, $4)
		RETURNING id`, projectID, title, points, hours).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear la historia %q: %v", title, err)
	}
	return id
}

func (f *estimateFixture) insertSprint(t *testing.T, number int, status string) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, 'Sprint de prueba HU-17', DATE '2026-10-01', DATE '2026-10-15', $3)
		RETURNING id`, f.projectID, number, status).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear el Sprint %d: %v", number, err)
	}
	return id
}

func (f *estimateFixture) assign(t *testing.T, sprintID, storyID int64, status string) {
	t.Helper()
	if _, err := f.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID); err != nil {
		t.Fatalf("no se pudo asignar la historia %d: %v", storyID, err)
	}
	if _, err := f.db.Exec(`UPDATE user_stories SET status = $1, active_sprint_id = $2 WHERE id = $3`, status, sprintID, storyID); err != nil {
		t.Fatalf("no se pudo actualizar la historia %d: %v", storyID, err)
	}
}

// hoursText devuelve estimated_hours como texto exacto de NUMERIC ("" si es NULL).
func (f *estimateFixture) hoursText(t *testing.T, storyID int64) string {
	t.Helper()
	var h sql.NullString
	if err := f.db.QueryRow(`SELECT estimated_hours::text FROM user_stories WHERE id = $1`, storyID).Scan(&h); err != nil {
		t.Fatalf("no se pudieron leer las horas de %d: %v", storyID, err)
	}
	return h.String
}

// storySnapshot devuelve todas las columnas de la historia salvo estimated_hours.
func (f *estimateFixture) storySnapshot(t *testing.T, storyID int64) string {
	t.Helper()
	var projectID int64
	var title, description, priority, status string
	var points, activeSprint sql.NullInt64
	var createdAt time.Time
	if err := f.db.QueryRow(`
		SELECT project_id, title, description, priority, story_points, status, active_sprint_id, created_at
		FROM user_stories WHERE id = $1`, storyID).Scan(
		&projectID, &title, &description, &priority, &points, &status, &activeSprint, &createdAt); err != nil {
		t.Fatalf("no se pudo leer la historia %d: %v", storyID, err)
	}
	return fmt.Sprintf("%d|%s|%s|%s|%v|%s|%v|%s", projectID, title, description, priority, points, status, activeSprint, createdAt.UTC().Format(time.RFC3339Nano))
}

func TestUpdateEstimatedHours_RegistraEnHistoriaSinEstimacion(t *testing.T) {
	f := newEstimateFixture(t)
	result, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.sinEstimar, "8")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if result.Previous != nil || result.Current != 8 {
		t.Errorf("resultado = %+v; se esperaba Previous nil (registradas) y Current 8", result)
	}
	if got := f.hoursText(t, f.sinEstimar); got != "8.00" {
		t.Errorf("estimated_hours = %q; se esperaba 8.00", got)
	}
}

func TestUpdateEstimatedHours_ModificaUnaEstimacionExistente(t *testing.T) {
	f := newEstimateFixture(t)
	result, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.estimada, "12,5")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if result.Previous == nil || *result.Previous != 8 || result.Current != 12.5 {
		t.Errorf("resultado = %+v; se esperaba Previous 8 (actualizadas) y Current 12.5", result)
	}
	if got := f.hoursText(t, f.estimada); got != "12.50" {
		t.Errorf("estimated_hours = %q; se esperaba 12.50", got)
	}
}

func TestUpdateEstimatedHours_GuardaValoresValidosExactos(t *testing.T) {
	cases := []struct {
		input   string
		current float64
		stored  string
	}{
		{"8", 8, "8.00"},
		{"1,5", 1.5, "1.50"},
		{"1.5", 1.5, "1.50"},
		{"0,01", 0.01, "0.01"},
		{"9999,99", 9999.99, "9999.99"},
		{" 2,75 ", 2.75, "2.75"},
	}
	f := newEstimateFixture(t)
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			result, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.estimada, c.input)
			if err != nil {
				t.Fatalf("%q: no se esperaba error: %v", c.input, err)
			}
			if result.Current != c.current {
				t.Errorf("%q: Current = %v; se esperaba %v", c.input, result.Current, c.current)
			}
			if got := f.hoursText(t, f.estimada); got != c.stored {
				t.Errorf("%q: estimated_hours = %q; se esperaba %q (sin redondeo)", c.input, got, c.stored)
			}
		})
	}
}

func TestUpdateEstimatedHours_RechazaValoresInvalidosSinCambiarElAnterior(t *testing.T) {
	invalid := []string{"0", "0,00", "-1", "abc", "", "   ", "1,555", "9999,995", "1.000", "1.000,5", "1e3", "+5", "8 h", "0x10"}
	f := newEstimateFixture(t)
	for _, input := range invalid {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			_, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.estimada, input)
			if !errors.Is(err, ErrInvalidEstimatedHours) || errors.Is(err, ErrEstimatedHoursTooLarge) {
				t.Errorf("%q: error = %v; se esperaba ErrInvalidEstimatedHours (no por máximo)", input, err)
			}
			if got := f.hoursText(t, f.estimada); got != "8.00" {
				t.Errorf("%q: el valor anterior cambió a %q", input, got)
			}
		})
	}
}

func TestUpdateEstimatedHours_RechazaValoresPorEncimaDelMaximo(t *testing.T) {
	f := newEstimateFixture(t)
	for _, input := range []string{"10000", "10000,00", "12345,5"} {
		t.Run(input, func(t *testing.T) {
			_, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.estimada, input)
			if !errors.Is(err, ErrEstimatedHoursTooLarge) || !errors.Is(err, ErrInvalidEstimatedHours) {
				t.Errorf("%q: error = %v; se esperaba ErrEstimatedHoursTooLarge (reconocido como ErrInvalidEstimatedHours)", input, err)
			}
			if got := f.hoursText(t, f.estimada); got != "8.00" {
				t.Errorf("%q: el valor anterior cambió a %q", input, got)
			}
		})
	}
}

func TestUpdateEstimatedHours_RechazaHistoriaDeSprintFinalizado(t *testing.T) {
	f := newEstimateFixture(t)
	before := f.storySnapshot(t, f.finalizada)
	_, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.finalizada, "9")
	if !errors.Is(err, ErrStoryInFinishedSprint) {
		t.Fatalf("error = %v; se esperaba ErrStoryInFinishedSprint", err)
	}
	if got := f.hoursText(t, f.finalizada); got != "5.00" {
		t.Errorf("la historia bloqueada cambió a %q", got)
	}
	if after := f.storySnapshot(t, f.finalizada); after != before {
		t.Errorf("la historia bloqueada cambió: antes=%s después=%s", before, after)
	}
}

func TestUpdateEstimatedHours_PermiteHistoriaDelSprintActivoYConHorasReales(t *testing.T) {
	f := newEstimateFixture(t)
	for _, storyID := range []int64{f.enActivo, f.conHoras} {
		if _, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, storyID, "7,25"); err != nil {
			t.Errorf("historia %d: no se esperaba error: %v", storyID, err)
		}
		if got := f.hoursText(t, storyID); got != "7.25" {
			t.Errorf("historia %d: estimated_hours = %q; se esperaba 7.25", storyID, got)
		}
	}
}

func TestUpdateEstimatedHours_RechazaHistoriaInexistenteODeOtroProyecto(t *testing.T) {
	f := newEstimateFixture(t)
	for name, storyID := range map[string]int64{"inexistente": 9223372036854775807, "de otro proyecto": f.ajena} {
		t.Run(name, func(t *testing.T) {
			_, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, storyID, "3")
			if !errors.Is(err, ErrStoryNotFound) {
				t.Errorf("error = %v; se esperaba ErrStoryNotFound", err)
			}
		})
	}
	if got := f.hoursText(t, f.ajena); got != "6.00" {
		t.Errorf("la historia de otro proyecto cambió a %q", got)
	}
}

func TestUpdateEstimatedHours_MismoValorSeInformaComoActualizado(t *testing.T) {
	f := newEstimateFixture(t)
	result, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.estimada, "8,00")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if result.Previous == nil || *result.Previous != 8 || result.Current != 8 {
		t.Errorf("resultado = %+v; se esperaba Previous 8 y Current 8", result)
	}
	if got := f.hoursText(t, f.estimada); got != "8.00" {
		t.Errorf("estimated_hours = %q; se esperaba 8.00", got)
	}
}

func TestUpdateEstimatedHours_SoloCambiaEstimatedHours(t *testing.T) {
	f := newEstimateFixture(t)
	others := []int64{f.sinEstimar, f.conHoras, f.enActivo, f.finalizada, f.ajena}
	before := f.storySnapshot(t, f.estimada)
	othersBefore := make(map[int64]string, len(others))
	for _, id := range others {
		othersBefore[id] = f.storySnapshot(t, id) + "|" + f.hoursText(t, id)
	}

	if _, err := f.service.UpdateEstimatedHours(context.Background(), f.projectID, f.estimada, "21"); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	if after := f.storySnapshot(t, f.estimada); after != before {
		t.Errorf("cambiaron otras columnas de la historia: antes=%s después=%s", before, after)
	}
	if got := f.hoursText(t, f.estimada); got != "21.00" {
		t.Errorf("estimated_hours = %q; se esperaba 21.00", got)
	}
	for _, id := range others {
		if now := f.storySnapshot(t, id) + "|" + f.hoursText(t, id); now != othersBefore[id] {
			t.Errorf("cambió otra historia (%d): antes=%s después=%s", id, othersBefore[id], now)
		}
	}
}
