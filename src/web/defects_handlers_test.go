package web

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

type defectsHandlerFixture struct {
	app                                            *ctxApp
	activeSprintA, activeSprintB                   int64
	finishedSprintA                                int64
	storyActiveA, storyDone, storyNoSprint, storyB int64
}

func newDefectsHandlerFixture(t *testing.T) *defectsHandlerFixture {
	t.Helper()
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		if err := RegisterDefects(
			server,
			defect.NewService(defect.NewPostgreSQLStore(db)),
			NewPostgreSQLDefectCatalog(db),
		); err != nil {
			t.Fatalf("no se pudo registrar la sección Defectos: %v", err)
		}
	})
	fixture := &defectsHandlerFixture{app: app}
	insertID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := app.db.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
			t.Fatalf("no se pudo preparar el escenario de defectos: %v", err)
		}
		return id
	}
	insertSprint := func(projectID int64, number int, status string) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			 VALUES ($1, $2, $3, '2026-01-01', '2026-01-14', $4) RETURNING id`,
			projectID, number, fmt.Sprintf("HU06 handler %d %d", projectID, number), status,
		)
	}
	insertStory := func(projectID int64, title, status string, sprintID *int64) int64 {
		t.Helper()
		return insertID(
			`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
			 VALUES ($1, $2, 'Descripción de prueba', 'Media', $3, $4) RETURNING id`,
			projectID, title, status, sprintID,
		)
	}

	fixture.activeSprintA = insertSprint(app.fx.projectA, 1, "Activo")
	fixture.activeSprintB = insertSprint(app.fx.projectB, 1, "Activo")
	fixture.finishedSprintA = insertSprint(app.fx.projectA, 2, "Finalizado")
	fixture.storyActiveA = insertStory(app.fx.projectA, "HU06 handler historia activa", "En progreso", &fixture.activeSprintA)
	fixture.storyDone = insertStory(app.fx.projectA, "HU06 handler historia Hecho", "Hecho", &fixture.finishedSprintA)
	fixture.storyNoSprint = insertStory(app.fx.projectA, "HU06 handler historia sin Sprint", "Pendiente", nil)
	fixture.storyB = insertStory(app.fx.projectB, "HU06 handler historia B", "Pendiente", nil)
	t.Cleanup(func() {
		if _, err := app.db.Exec(
			`DELETE FROM user_stories WHERE id IN ($1, $2, $3, $4)`,
			fixture.storyActiveA, fixture.storyDone, fixture.storyNoSprint, fixture.storyB,
		); err != nil {
			t.Errorf("no se pudieron borrar las historias del handler: %v", err)
		}
		if _, err := app.db.Exec(
			`DELETE FROM sprints WHERE id IN ($1, $2, $3)`,
			fixture.activeSprintA, fixture.activeSprintB, fixture.finishedSprintA,
		); err != nil {
			t.Errorf("no se pudieron borrar los Sprints del handler: %v", err)
		}
	})
	return fixture
}

func (f *defectsHandlerFixture) selectProjectA(t *testing.T, client *http.Client) {
	t.Helper()
	resp, _ := f.app.selectProject(t, client, id64(f.app.fx.projectA), "/defectos")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/defectos" {
		t.Fatalf("el flujo de HU-20b debía seleccionar el proyecto y volver a Defectos; obtuvo %d a %q",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

func postDefect(t *testing.T, app *ctxApp, client *http.Client, storyID int64, severity string, extra url.Values) (*http.Response, string) {
	t.Helper()
	values := url.Values{
		DefectDescriptionField: {"Botón de guardar bloqueado"},
		DefectSeverityField:    {severity},
		DefectStoryField:       {id64(storyID)},
	}
	for key, entries := range extra {
		for _, entry := range entries {
			values.Add(key, entry)
		}
	}
	return app.post(t, client, "/defectos", values)
}

func TestDefectsHandler_RegisterSuccessUsesActiveContextAndOneTimeFlash(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)
	resp, _ := postDefect(t, fixture.app, client, fixture.storyActiveA, "Alta", url.Values{
		"sprint_id":  {id64(fixture.finishedSprintA)},
		"project_id": {id64(fixture.app.fx.projectB)},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/defectos" {
		t.Fatalf("POST válido debía redirigir con 303 a /defectos; obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, body := fixture.app.get(t, client, "/defectos")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Defecto registrado correctamente. Sprint de detección: Sprint 1") {
		t.Errorf("la página debía mostrar una vez el éxito con Sprint 1; status=%d body=%s", resp.StatusCode, body)
	}
	if _, body = fixture.app.get(t, client, "/defectos"); strings.Contains(body, "Defecto registrado correctamente.") {
		t.Errorf("el mensaje flash no debe repetirse al recargar")
	}

	var countA, countB int
	if err := fixture.app.db.QueryRow(`SELECT count(*) FROM defects WHERE project_id = $1`, fixture.app.fx.projectA).Scan(&countA); err != nil {
		t.Fatal(err)
	}
	if err := fixture.app.db.QueryRow(`SELECT count(*) FROM defects WHERE project_id = $1`, fixture.app.fx.projectB).Scan(&countB); err != nil {
		t.Fatal(err)
	}
	if countA != 1 || countB != 0 {
		t.Errorf("el envío debe registrar solo en el proyecto activo: A=%d B=%d", countA, countB)
	}
	var projectID, sprintID int64
	if err := fixture.app.db.QueryRow(`SELECT project_id, detection_sprint_id FROM defects WHERE story_id = $1`, fixture.storyActiveA).Scan(&projectID, &sprintID); err != nil {
		t.Fatal(err)
	}
	if projectID != fixture.app.fx.projectA || sprintID != fixture.activeSprintA {
		t.Errorf("el proyecto o Sprint enviado por formulario alteró el contexto: project=%d sprint=%d", projectID, sprintID)
	}
}

func TestDefectsHandler_RegisterDoneAndUnassignedStories(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)

	for _, storyID := range []int64{fixture.storyDone, fixture.storyNoSprint} {
		resp, _ := postDefect(t, fixture.app, client, storyID, "Crítica", nil)
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/defectos" {
			t.Fatalf("la historia %d debía aceptar el registro; obtuvo %d", storyID, resp.StatusCode)
		}
	}
	for _, storyID := range []int64{fixture.storyDone, fixture.storyNoSprint} {
		var status string
		var activeSprint sql.NullInt64
		if err := fixture.app.db.QueryRow(
			`SELECT us.status, us.active_sprint_id FROM user_stories us WHERE us.id = $1`, storyID,
		).Scan(&status, &activeSprint); err != nil {
			t.Fatal(err)
		}
		if storyID == fixture.storyDone && (status != "Hecho" || !activeSprint.Valid || activeSprint.Int64 != fixture.finishedSprintA) {
			t.Errorf("el registro cambió estado/asignación de historia Hecho: status=%q sprint=%+v", status, activeSprint)
		}
		if storyID == fixture.storyNoSprint && (status != "Pendiente" || activeSprint.Valid) {
			t.Errorf("el registro cambió historia sin Sprint: status=%q sprint=%+v", status, activeSprint)
		}
		var defectStatus string
		var detectionSprintID int64
		if err := fixture.app.db.QueryRow(
			`SELECT status, detection_sprint_id FROM defects WHERE story_id = $1`, storyID,
		).Scan(&defectStatus, &detectionSprintID); err != nil {
			t.Fatal(err)
		}
		if defectStatus != "Abierto" || detectionSprintID != fixture.activeSprintA {
			t.Errorf("defecto de historia %d: estado=%q Sprint detección=%d", storyID, defectStatus, detectionSprintID)
		}
	}
}

func TestDefectsHandler_RegisterEverySeverity(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)
	for _, severity := range []string{"Crítica", "Alta", "Media", "Baja"} {
		resp, _ := postDefect(t, fixture.app, client, fixture.storyActiveA, severity, nil)
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/defectos" {
			t.Fatalf("la severidad %q debía registrarse; obtuvo %d", severity, resp.StatusCode)
		}
	}
	rows, err := fixture.app.db.Query(`SELECT severity FROM defects WHERE project_id = $1 ORDER BY id`, fixture.app.fx.projectA)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var severity string
		if err := rows.Scan(&severity); err != nil {
			t.Fatal(err)
		}
		got = append(got, severity)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{"Crítica", "Alta", "Media", "Baja"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("severidades persistidas = %v; se esperaban %v", got, want)
	}
}
