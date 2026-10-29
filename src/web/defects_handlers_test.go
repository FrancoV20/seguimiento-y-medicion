package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

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

func TestDefectsHandler_RequiredFieldsReturnAllErrorsAndPreserveValues(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)
	values := url.Values{
		DefectDescriptionField: {"   "},
		DefectSeverityField:    {""},
		DefectStoryField:       {""},
	}

	resp, body := fixture.app.post(t, client, "/defectos", values)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("el formulario inválido debía responder 422; obtuvo %d", resp.StatusCode)
	}
	for _, field := range []struct{ name, message string }{
		{DefectDescriptionField, MsgDefectDescriptionRequired},
		{DefectSeverityField, MsgDefectSeverityRequired},
		{DefectStoryField, MsgDefectStoryRequired},
	} {
		if !strings.Contains(body, field.message) || !strings.Contains(body, `id="`+field.name+`-error"`) {
			t.Errorf("el error %q debía aparecer junto al campo %q", field.message, field.name)
		}
	}
	if !strings.Contains(body, `name="descripcion" value="   "`) {
		t.Errorf("el valor de descripción debía conservarse")
	}
	assertNoDefects(t, fixture.app.db, fixture.app.fx.projectA)
}

func TestDefectsHandler_DescriptionLengthAndWhitespace(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)

	for _, tc := range []struct {
		name, description, message string
		status                     int
	}{
		{"solo espacios", " \t ", MsgDefectDescriptionRequired, http.StatusUnprocessableEntity},
		{"1001 caracteres", strings.Repeat("ñ", 1001), MsgDefectDescriptionTooLong, http.StatusUnprocessableEntity},
		{"carácter NUL (%00)", "falla\x00al guardar", MsgDefectDescriptionInvalid, http.StatusUnprocessableEntity},
		{"UTF-8 inválido (%FF)", "falla \xff al guardar", MsgDefectDescriptionInvalid, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{
				DefectDescriptionField: {tc.description},
				DefectSeverityField:    {"Media"},
				DefectStoryField:       {id64(fixture.storyActiveA)},
			}
			resp, body := fixture.app.post(t, client, "/defectos", values)
			if resp.StatusCode != tc.status || !strings.Contains(body, tc.message) ||
				!strings.Contains(body, `id="descripcion-error"`) {
				t.Errorf("status=%d debía ser %d y mostrar el error %q junto a descripción",
					resp.StatusCode, tc.status, tc.message)
			}
			assertNoDefects(t, fixture.app.db, fixture.app.fx.projectA)
		})
	}

	description := strings.Repeat("ñ", 1000)
	values := url.Values{
		DefectDescriptionField: {description},
		DefectSeverityField:    {"Media"},
		DefectStoryField:       {id64(fixture.storyActiveA)},
	}
	resp, _ := fixture.app.post(t, client, "/defectos", values)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/defectos" {
		t.Fatalf("1000 caracteres debían aceptarse con redirección 303; obtuvo %d", resp.StatusCode)
	}
	var storedLength int
	if err := fixture.app.db.QueryRow(
		`SELECT char_length(description) FROM defects WHERE story_id = $1`, fixture.storyActiveA,
	).Scan(&storedLength); err != nil {
		t.Fatal(err)
	}
	if storedLength != 1000 {
		t.Errorf("se guardaron %d caracteres; se esperaban 1000", storedLength)
	}
}

func TestDefectsHandler_InvalidStoryAndSeverityReturnFieldErrors(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)

	cases := []struct {
		name, story, severity, field, message string
	}{
		{"historia de otro proyecto", id64(fixture.storyB), "Alta", DefectStoryField, MsgDefectStoryNotInProject},
		{"historia inexistente", "999999999", "Alta", DefectStoryField, MsgDefectStoryNotInProject},
		{"historia no numérica", "abc", "Alta", DefectStoryField, MsgDefectStoryNotInProject},
		{"severidad inválida", id64(fixture.storyActiveA), "Urgente", DefectSeverityField, MsgDefectSeverityInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{
				DefectDescriptionField: {"Descripción conservada"},
				DefectSeverityField:    {tc.severity},
				DefectStoryField:       {tc.story},
			}
			resp, body := fixture.app.post(t, client, "/defectos", values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("el envío inválido debía responder 422; obtuvo %d", resp.StatusCode)
			}
			if !strings.Contains(body, tc.message) || !strings.Contains(body, `id="`+tc.field+`-error"`) {
				t.Errorf("faltó el error %q junto al campo %q", tc.message, tc.field)
			}
			for _, value := range []string{"Descripción conservada", tc.severity, tc.story} {
				if value != "" && !strings.Contains(body, `value="`+value+`"`) {
					t.Errorf("no se conservó el valor enviado %q", value)
				}
			}
			assertNoDefects(t, fixture.app.db, fixture.app.fx.projectA)
		})
	}
}

func TestDefectsHandler_NoActiveSprintRejectsGetAndPost(t *testing.T) {
	for _, sprintStatus := range []string{"sin Sprints", "todos finalizados"} {
		t.Run(sprintStatus, func(t *testing.T) {
			fixture := newDefectsHandlerFixture(t)
			client := fixture.app.client(false)
			fixture.selectProjectA(t, client)
			if sprintStatus == "todos finalizados" {
				if _, err := fixture.app.db.Exec(
					`UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, fixture.activeSprintA,
				); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := fixture.app.db.Exec(
					`DELETE FROM user_stories WHERE id = $1`, fixture.storyActiveA,
				); err != nil {
					t.Fatal(err)
				}
				fixture.storyActiveA = 0
				if _, err := fixture.app.db.Exec(
					`DELETE FROM sprints WHERE id = $1`, fixture.activeSprintA,
				); err != nil {
					t.Fatal(err)
				}
				fixture.activeSprintA = 0
			}

			getResp, getBody := fixture.app.get(t, client, "/defectos")
			if getResp.StatusCode != http.StatusConflict || !strings.Contains(getBody, MsgDefectNoSprint) {
				t.Errorf("GET sin Sprint activo debía responder 409 con %q; obtuvo %d", MsgDefectNoSprint, getResp.StatusCode)
			}
			if strings.Contains(getBody, `<form`) || strings.Contains(getBody, `name="descripcion"`) {
				t.Errorf("GET sin Sprint activo no debía mostrar el formulario")
			}

			values := url.Values{
				DefectDescriptionField: {"Descripción"},
				DefectSeverityField:    {"Alta"},
				DefectStoryField:       {id64(fixture.storyDone)},
			}
			postResp, postBody := fixture.app.post(t, client, "/defectos", values)
			if postResp.StatusCode != http.StatusConflict || !strings.Contains(postBody, MsgDefectNoSprint) {
				t.Errorf("POST sin Sprint activo debía responder 409 con %q; obtuvo %d", MsgDefectNoSprint, postResp.StatusCode)
			}
			assertNoDefects(t, fixture.app.db, fixture.app.fx.projectA)
		})
	}
}

type failingDefectStore struct{ err error }

func (s failingDefectStore) Create(context.Context, defect.RegisterInput) (defect.Registered, error) {
	return defect.Registered{}, s.err
}

func TestDefectsHandler_StorageFailurePreservesFormAndHidesDetails(t *testing.T) {
	var testDB *sql.DB
	app := newCtxApp(t, nil, func(server *Server) {
		testDB = openContextTestDB(t)
		if err := RegisterDefects(
			server,
			defect.NewService(failingDefectStore{err: errors.New("SELECT * FROM defects at /internal/config/database")}),
			NewPostgreSQLDefectCatalog(testDB),
		); err != nil {
			t.Fatalf("no se pudo registrar la sección Defectos: %v", err)
		}
	})
	var sprintID, storyID int64
	if err := app.db.QueryRow(
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, 80, 'HU06 falla de guardado', '2026-01-01', '2026-01-14', 'Activo') RETURNING id`,
		app.fx.projectA,
	).Scan(&sprintID); err != nil {
		t.Fatal(err)
	}
	if err := app.db.QueryRow(
		`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
		 VALUES ($1, 'HU06 historia falla de guardado', 'Descripción', 'Media', 'En progreso', $2) RETURNING id`,
		app.fx.projectA, sprintID,
	).Scan(&storyID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := app.db.Exec(`DELETE FROM user_stories WHERE id = $1`, storyID); err != nil {
			t.Errorf("no se pudo borrar la historia de prueba: %v", err)
		}
		if _, err := app.db.Exec(`DELETE FROM sprints WHERE id = $1`, sprintID); err != nil {
			t.Errorf("no se pudo borrar el Sprint de prueba: %v", err)
		}
	})
	client := app.client(false)
	resp, _ := app.selectProject(t, client, id64(app.fx.projectA), "/defectos")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("la selección del proyecto debía redirigir; obtuvo %d", resp.StatusCode)
	}

	values := url.Values{
		DefectDescriptionField: {"Descripción conservada ante falla"},
		DefectSeverityField:    {"Baja"},
		DefectStoryField:       {id64(storyID)},
	}
	resp, body := app.post(t, client, "/defectos", values)
	if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(body, MsgOperationFailed) {
		t.Errorf("la falla de guardado debía responder 500 con %q; obtuvo %d", MsgOperationFailed, resp.StatusCode)
	}
	if !strings.Contains(body, `value="Descripción conservada ante falla"`) ||
		!strings.Contains(body, `value="Baja"`) ||
		!strings.Contains(body, `value="`+id64(storyID)+`"`) {
		t.Errorf("la respuesta debía conservar los valores enviados")
	}
	if strings.Contains(body, "SELECT * FROM defects") ||
		strings.Contains(body, "/internal/config/database") ||
		strings.Contains(body, "Defecto registrado correctamente.") {
		t.Errorf("la respuesta no debe exponer detalles internos ni mostrar éxito")
	}
	assertNoDefects(t, testDB, app.fx.projectA)
}

func TestDefectsHandler_ErrorResponseDoesNotExposeSQLOrInternalPaths(t *testing.T) {
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		if err := RegisterDefects(
			server,
			defect.NewService(failingDefectStore{err: errors.New("SELECT * FROM defects at /internal/config/database")}),
			NewPostgreSQLDefectCatalog(db),
		); err != nil {
			t.Fatalf("no se pudo registrar la sección Defectos: %v", err)
		}
	})
	client := app.client(false)
	resp, _ := app.selectProject(t, client, id64(app.fx.projectA), "/defectos")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("la selección del proyecto debía redirigir; obtuvo %d", resp.StatusCode)
	}

	resp, body := app.post(t, client, "/defectos", url.Values{
		DefectDescriptionField: {"Descripción"},
		DefectSeverityField:    {"Alta"},
		DefectStoryField:       {"1"},
	})
	if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(body, MsgOperationFailed) {
		t.Fatalf("se esperaba el error genérico de operación; status=%d body=%s", resp.StatusCode, body)
	}
	for _, privateText := range []string{"SELECT * FROM defects", "/internal/config/database"} {
		if strings.Contains(body, privateText) {
			t.Errorf("la respuesta de error expuso detalles internos %q", privateText)
		}
	}
}

func TestDefectsHandler_NoProjectContextRedirectsWithoutSaving(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	postValues := url.Values{
		DefectDescriptionField: {"No debe guardarse"},
		DefectSeverityField:    {"Alta"},
		DefectStoryField:       {id64(fixture.storyActiveA)},
	}
	cases := []struct {
		name  string
		setup func(*http.Client)
	}{
		{"sin selección", func(*http.Client) {}},
		{"cookie inválida", func(client *http.Client) {
			fixture.app.setCookies(t, client, &http.Cookie{Name: CookieProject, Value: "invalid", Path: "/"})
		}},
		{"proyecto eliminado", func(client *http.Client) {
			var deletedProjectID int64
			if err := fixture.app.db.QueryRow(
				`INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`,
				fmt.Sprintf("HU06 proyecto eliminado %d", fixture.storyActiveA),
			).Scan(&deletedProjectID); err != nil {
				t.Fatalf("no se pudo crear el proyecto temporal: %v", err)
			}
			fixture.app.setCookies(t, client, idCookie(CookieProject, deletedProjectID))
			if _, err := fixture.app.db.Exec(`DELETE FROM projects WHERE id = $1`, deletedProjectID); err != nil {
				t.Fatalf("no se pudo eliminar el proyecto temporal: %v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := fixture.app.client(false)
			tc.setup(client)
			resp, body := fixture.app.post(t, client, "/defectos", postValues)
			if resp.StatusCode != http.StatusSeeOther ||
				!strings.HasPrefix(resp.Header.Get("Location"), pathSelectProject+"?volver_a=") {
				t.Errorf("sin proyecto activo debía redirigir a selección; status=%d location=%q",
					resp.StatusCode, resp.Header.Get("Location"))
			}
			if strings.Contains(body, "No debe guardarse") {
				t.Errorf("el formulario no debía procesarse sin proyecto activo")
			}
			assertNoDefects(t, fixture.app.db, fixture.app.fx.projectA)
		})
	}
}

type failingDefectCatalog struct {
	DefectCatalog
	err error
}

func (c failingDefectCatalog) Defects(context.Context, int64) ([]DefectRow, error) {
	return nil, c.err
}

func TestDefectsHandler_CatalogFailureUsesApplicationErrorPage(t *testing.T) {
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		if err := RegisterDefects(
			server,
			defect.NewService(defect.NewPostgreSQLStore(db)),
			failingDefectCatalog{DefectCatalog: NewPostgreSQLDefectCatalog(db), err: errors.New("private query detail")},
		); err != nil {
			t.Fatalf("no se pudo registrar la sección Defectos: %v", err)
		}
	})
	client := app.client(false)
	resp, _ := app.selectProject(t, client, id64(app.fx.projectA), "/defectos")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("la selección del proyecto debía redirigir; obtuvo %d", resp.StatusCode)
	}

	resp, body := app.get(t, client, "/defectos")
	if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(body, MsgInternal) {
		t.Errorf("la falla de lectura debía mostrar la página de error común; obtuvo %d", resp.StatusCode)
	}
	if strings.Contains(body, "private query detail") || strings.Contains(body, `<form`) || strings.Contains(body, `<table`) {
		t.Errorf("la respuesta no debe exponer el error interno ni renderizar una pantalla incompleta")
	}
}

func TestDefectsHandler_ListShowsTraceabilityForActiveProjectWithin500ms(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)
	insert := func(description, severity string, storyID, sprintID int64) int64 {
		t.Helper()
		var id int64
		if err := fixture.app.db.QueryRow(
			`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
			 VALUES ($1, $2, $3, $4, 'Abierto', $5) RETURNING id`,
			fixture.app.fx.projectA, storyID, description, severity, sprintID,
		).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear el defecto de prueba: %v", err)
		}
		return id
	}
	olderID := insert("Descripción anterior", "Baja", fixture.storyDone, fixture.finishedSprintA)
	newerID := insert("Descripción reciente", "Crítica", fixture.storyActiveA, fixture.activeSprintA)
	if _, err := fixture.app.db.Exec(
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		 VALUES ($1, $2, 'No debe aparecer en el proyecto activo', 'Media', 'Abierto', $3)`,
		fixture.app.fx.projectB, fixture.storyB, fixture.activeSprintB,
	); err != nil {
		t.Fatalf("no se pudo crear el defecto del proyecto B: %v", err)
	}

	start := time.Now()
	resp, body := fixture.app.get(t, client, "/defectos")
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /defectos debía responder 200; obtuvo %d", resp.StatusCode)
	}
	if duration >= 500*time.Millisecond {
		t.Errorf("la lista tardó %s; debía responder en menos de 500 ms", duration)
	}
	for _, want := range []string{
		"Identificador", "Descripción", "Severidad", "Estado", "Historia relacionada",
		"Sprint de detección", "Sprint de resolución",
		id64(newerID), "Descripción reciente", "Crítica", "Abierto",
		"HU06 handler historia activa", "Sprint 1", "—",
		id64(olderID), "Descripción anterior", "Baja", "HU06 handler historia Hecho",
		"Sprint 2",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("la lista debía incluir %q; body=%s", want, body)
		}
	}
	if strings.Contains(body, "No debe aparecer en el proyecto activo") {
		t.Errorf("la lista filtró un defecto de otro proyecto")
	}
	newerPos, olderPos := strings.Index(body, "Descripción reciente"), strings.Index(body, "Descripción anterior")
	if newerPos < 0 || olderPos < 0 || newerPos >= olderPos {
		t.Errorf("los defectos no se mostraron de más reciente a más antiguo")
	}
}

func TestDefectsHandler_EmptyListShowsMessageAndForm(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)

	resp, body := fixture.app.get(t, client, "/defectos")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /defectos sin defectos debía responder 200; obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, MsgDefectNone) {
		t.Errorf("la lista vacía debía mostrar %q", MsgDefectNone)
	}
	for _, field := range []string{`name="descripcion"`, `name="severidad"`, `name="historia"`} {
		if !strings.Contains(body, field) {
			t.Errorf("la lista vacía debía mantener el formulario; falta %q", field)
		}
	}
}

func TestDefectsHandler_ListEscapesDescriptionAndStoryTitle(t *testing.T) {
	fixture := newDefectsHandlerFixture(t)
	client := fixture.app.client(false)
	fixture.selectProjectA(t, client)
	const descriptionScript = "<script>alert('description')</script>"
	const titleScript = "<script>alert('title')</script>"
	var storyID int64
	if err := fixture.app.db.QueryRow(
		`INSERT INTO user_stories (project_id, title, description, priority, status, active_sprint_id)
		 VALUES ($1, $2, 'Descripción', 'Media', 'Pendiente', $3) RETURNING id`,
		fixture.app.fx.projectA, titleScript, fixture.activeSprintA,
	).Scan(&storyID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := fixture.app.db.Exec(`DELETE FROM user_stories WHERE id = $1`, storyID); err != nil {
			t.Errorf("no se pudo borrar la historia de prueba: %v", err)
		}
	})
	if _, err := fixture.app.db.Exec(
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		 VALUES ($1, $2, $3, 'Alta', 'Abierto', $4)`,
		fixture.app.fx.projectA, storyID, descriptionScript, fixture.activeSprintA,
	); err != nil {
		t.Fatal(err)
	}

	resp, body := fixture.app.get(t, client, "/defectos")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /defectos debía responder 200; obtuvo %d", resp.StatusCode)
	}
	if strings.Contains(body, descriptionScript) || strings.Contains(body, titleScript) ||
		!strings.Contains(body, "&lt;script&gt;alert(&#39;description&#39;)&lt;/script&gt;") ||
		!strings.Contains(body, "&lt;script&gt;alert(&#39;title&#39;)&lt;/script&gt;") {
		t.Errorf("la descripción y el título deben mostrarse escapados como texto")
	}
}

type failingStoryOptionsCatalog struct {
	DefectCatalog
	err error
}

func (c failingStoryOptionsCatalog) StoryOptions(context.Context, int64) ([]DefectStoryOption, error) {
	return nil, c.err
}

func TestDefectsHandler_StoryCatalogFailureUsesApplicationErrorPage(t *testing.T) {
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		if err := RegisterDefects(
			server,
			defect.NewService(defect.NewPostgreSQLStore(db)),
			failingStoryOptionsCatalog{DefectCatalog: NewPostgreSQLDefectCatalog(db), err: errors.New("private story query detail")},
		); err != nil {
			t.Fatalf("no se pudo registrar la sección Defectos: %v", err)
		}
	})
	client := app.client(false)
	resp, _ := app.selectProject(t, client, id64(app.fx.projectA), "/defectos")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("la selección del proyecto debía redirigir; obtuvo %d", resp.StatusCode)
	}

	resp, body := app.get(t, client, "/defectos")
	if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(body, MsgInternal) {
		t.Errorf("la falla al leer historias debía mostrar la página de error común; obtuvo %d", resp.StatusCode)
	}
	if strings.Contains(body, "private story query detail") || strings.Contains(body, `<form`) || strings.Contains(body, `<table`) {
		t.Errorf("la respuesta no debe exponer detalles ni mostrar una pantalla incompleta")
	}
}

func assertNoDefects(t *testing.T, db *sql.DB, projectID int64) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM defects WHERE project_id = $1`, projectID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("no debía guardarse ningún defecto; se encontraron %d", count)
	}
}
