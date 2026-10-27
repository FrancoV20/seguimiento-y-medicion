package test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

type webProjectsBDDApp struct {
	db            *sql.DB
	server        *httptest.Server
	client        *http.Client
	memberID      int64
	projectPrefix string
}

type noMembersBDDProjectCatalog struct{ web.ProjectCatalog }

type noProjectsBDDProjectCatalog struct{ web.ProjectCatalog }

type failingBDDProjectStore struct {
	memberID int
}

func (failingBDDProjectStore) Create(*project.Project) error {
	return fmt.Errorf("simulated project store failure")
}

func (s failingBDDProjectStore) MemberExists(id int) (bool, error) {
	return id == s.memberID, nil
}

func (noMembersBDDProjectCatalog) ListMembers(_ context.Context) ([]web.MemberOption, error) {
	return []web.MemberOption{}, nil
}

func (noProjectsBDDProjectCatalog) ListProjects(context.Context) ([]web.ProjectRow, error) {
	return []web.ProjectRow{}, nil
}

func newWebProjectsBDDApp(t *testing.T, noMembers, failSave bool, noProjects ...bool) *webProjectsBDDApp {
	t.Helper()

	databaseURL := os.Getenv(web.EnvDatabaseURL)
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite el escenario con PostgreSQL")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("no se pudo abrir PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}

	stamp := time.Now().UnixNano()
	memberName := fmt.Sprintf("HU21 BDD integrante %d", stamp)
	projectPrefix := fmt.Sprintf("HU21 BDD %d", stamp)
	var memberID int64
	if err := db.QueryRow(
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, memberName,
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante BDD: %v", err)
	}

	catalog := web.ProjectCatalog(web.NewPostgreSQLProjectCatalog(db))
	if noMembers {
		catalog = noMembersBDDProjectCatalog{ProjectCatalog: catalog}
	}
	if len(noProjects) > 0 && noProjects[0] {
		catalog = noProjectsBDDProjectCatalog{ProjectCatalog: catalog}
	}
	srv := web.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	var store project.ProjectStore = project.NewPostgreSQLProjectStore(db)
	if failSave {
		store = failingBDDProjectStore{memberID: int(memberID)}
	}
	if err := web.RegisterProjects(srv, project.NewProjectService(store), catalog); err != nil {
		t.Fatalf("no se pudo registrar la sección Proyectos: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("no se pudo crear el almacén de cookies: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	app := &webProjectsBDDApp{db: db, server: ts, client: client, memberID: memberID}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM projects WHERE name LIKE $1`, projectPrefix+" %")
		_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, memberID)
	})
	app.projectPrefix = projectPrefix
	return app
}

func webProjectsBDDGet(t *testing.T, app *webProjectsBDDApp, path string) (*http.Response, string) {
	t.Helper()
	resp, err := app.client.Get(app.server.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return resp, string(body)
}

func webProjectsBDDPost(t *testing.T, app *webProjectsBDDApp, name, start, end string) (*http.Response, string) {
	t.Helper()
	form := url.Values{
		web.ProjectNameField:      {name},
		web.ProjectStartDateField: {start},
		web.ProjectEndDateField:   {end},
		web.ProjectMembersField:   {fmt.Sprint(app.memberID)},
	}
	return webProjectsBDDPostValues(t, app, form)
}

func webProjectsBDDPostValues(t *testing.T, app *webProjectsBDDApp, form url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := app.client.PostForm(app.server.URL+"/proyectos", form)
	if err != nil {
		t.Fatalf("POST /proyectos: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return resp, string(body)
}

func bddHasFieldError(body, field, message string) bool {
	if field == web.ProjectMembersField {
		return strings.Contains(body, `class="picker-list field-invalid"`) && strings.Contains(body, message)
	}
	association := `aria-describedby="` + field + `-error"`
	errorID := `id="` + field + `-error"`
	return strings.Contains(body, association) && strings.Contains(body, errorID) && strings.Contains(body, message)
}

func countBDDProject(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE name = $1`, name).Scan(&count); err != nil {
		t.Fatalf("no se pudo consultar el proyecto BDD: %v", err)
	}
	return count
}

func projectTableRow(body, projectName string) string {
	namePosition := strings.Index(body, projectName)
	if namePosition < 0 {
		return ""
	}
	rowStart := strings.LastIndex(body[:namePosition], "<tr")
	rowEnd := strings.Index(body[namePosition:], "</tr>")
	if rowStart < 0 || rowEnd < 0 {
		return ""
	}
	return body[rowStart : namePosition+rowEnd+len("</tr>")]
}

func TestWebProjects_CreateValidProject_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " alta"

	resp, _ := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-24")

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/proyectos" {
		t.Fatalf("al crear se esperaba 303 a /proyectos, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	page, err := app.client.Get(app.server.URL + resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("GET /proyectos: %v", err)
	}
	defer page.Body.Close()
	body, err := io.ReadAll(page.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la página de proyectos: %v", err)
	}
	if !strings.Contains(string(body), web.MsgProjectCreated) || !strings.Contains(string(body), name) {
		t.Errorf("la página debía mostrar el éxito y el proyecto; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 1 {
		t.Errorf("se esperaba un proyecto creado, se encontraron %d", count)
	}
}

func TestWebProjects_EqualDatesAreRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " fechas iguales"

	resp, body := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-23")

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("las fechas iguales debían rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, project.ErrInvalidPeriod.Error()) || !strings.Contains(body, web.ProjectEndDateField) {
		t.Errorf("se esperaba el error de fechas de HU-01 junto a fecha_fin; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con fechas iguales, se encontraron %d", count)
	}
}

func TestWebProjects_NoAvailableMembersPreventsCreation_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, true, false)
	name := app.projectPrefix + " sin integrantes"

	resp, body := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-24")

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("sin integrantes disponibles se esperaba 422, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, web.MsgNoMembersAvailable) {
		t.Errorf("se esperaba %q; respuesta: %s", web.MsgNoMembersAvailable, body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse un proyecto sin integrantes disponibles, se encontraron %d", count)
	}
}

func TestWebProjects_EmptyNameIsRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	form := url.Values{
		web.ProjectNameField:      {""},
		web.ProjectStartDateField: {"2026-10-23"},
		web.ProjectEndDateField:   {"2026-10-24"},
		web.ProjectMembersField:   {fmt.Sprint(app.memberID)},
	}

	resp, body := webProjectsBDDPostValues(t, app, form)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("el nombre vacío debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !bddHasFieldError(body, web.ProjectNameField, project.ErrEmptyName.Error()) {
		t.Errorf("se esperaba el error de nombre junto a nombre; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, ""); count != 0 {
		t.Errorf("no debía persistirse un proyecto sin nombre, se encontraron %d", count)
	}
}

func TestWebProjects_EarlierEndDateIsRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " fin anterior"

	resp, body := webProjectsBDDPost(t, app, name, "2026-10-24", "2026-10-23")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("la fecha de fin anterior debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !bddHasFieldError(body, web.ProjectEndDateField, project.ErrInvalidPeriod.Error()) {
		t.Errorf("se esperaba el error de período junto a fecha_fin; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con fechas inválidas, se encontraron %d", count)
	}
}

func TestWebProjects_MissingDateIsRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " fecha faltante"
	form := url.Values{
		web.ProjectNameField:    {name},
		web.ProjectEndDateField: {"2026-10-24"},
		web.ProjectMembersField: {fmt.Sprint(app.memberID)},
	}

	resp, body := webProjectsBDDPostValues(t, app, form)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("la fecha de inicio faltante debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !bddHasFieldError(body, web.ProjectStartDateField, web.MsgMissingStartDate) {
		t.Errorf("se esperaba el mensaje de fecha faltante junto a fecha_inicio; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con fecha faltante, se encontraron %d", count)
	}
}

func TestWebProjects_NoSelectedMembersIsRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " sin integrantes marcados"
	form := url.Values{
		web.ProjectNameField:      {name},
		web.ProjectStartDateField: {"2026-10-23"},
		web.ProjectEndDateField:   {"2026-10-24"},
	}

	resp, body := webProjectsBDDPostValues(t, app, form)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("el envío sin integrantes debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !bddHasFieldError(body, web.ProjectMembersField, project.ErrMissingMembers.Error()) {
		t.Errorf("se esperaba el error de integrantes junto al campo; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto sin integrantes, se encontraron %d", count)
	}
}

func TestWebProjects_InvalidDateFormatIsRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " formato inválido"
	form := url.Values{
		web.ProjectNameField:      {name},
		web.ProjectStartDateField: {"23/10/2026"},
		web.ProjectEndDateField:   {"2026-10-24"},
		web.ProjectMembersField:   {fmt.Sprint(app.memberID)},
	}

	resp, body := webProjectsBDDPostValues(t, app, form)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("la fecha mal formada debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !bddHasFieldError(body, web.ProjectStartDateField, web.MsgInvalidDateFormat) {
		t.Errorf("se esperaba el error de formato junto a fecha_inicio; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con fecha mal formada, se encontraron %d", count)
	}
}

func TestWebProjects_InvalidMemberIsRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	name := app.projectPrefix + " integrante inválido"
	form := url.Values{
		web.ProjectNameField:      {name},
		web.ProjectStartDateField: {"2026-10-23"},
		web.ProjectEndDateField:   {"2026-10-24"},
		web.ProjectMembersField:   {"no-numérico"},
	}

	resp, body := webProjectsBDDPostValues(t, app, form)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("el integrante inválido debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !bddHasFieldError(body, web.ProjectMembersField, web.MsgInvalidMember) {
		t.Errorf("se esperaba el error del integrante junto al campo; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con integrante inválido, se encontraron %d", count)
	}
}

func TestWebProjects_SaveFailureShowsGenericError_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, true)
	name := app.projectPrefix + " falla al guardar"

	resp, body := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-24")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("la falla de guardado debía responder 500, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, web.MsgOperationFailed) || strings.Contains(body, web.MsgProjectCreated) {
		t.Errorf("se esperaba únicamente el error genérico, respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con falla de guardado, se encontraron %d", count)
	}
}

func TestWebProjects_ListShowsProjectDetailsNewestFirst_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false)
	anotherMemberName := app.projectPrefix + " segundo integrante"
	var anotherMemberID int64
	if err := app.db.QueryRow(
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, anotherMemberName,
	).Scan(&anotherMemberID); err != nil {
		t.Fatalf("no se pudo crear el segundo integrante BDD: %v", err)
	}
	t.Cleanup(func() {
		if _, err := app.db.Exec(`DELETE FROM projects WHERE name LIKE $1`, app.projectPrefix+" %"); err != nil {
			t.Errorf("no se pudieron limpiar los proyectos BDD: %v", err)
		}
		if _, err := app.db.Exec(`DELETE FROM members WHERE id = $1`, anotherMemberID); err != nil {
			t.Errorf("no se pudo limpiar el segundo integrante BDD: %v", err)
		}
	})

	olderName := app.projectPrefix + " proyecto anterior"
	newerName := app.projectPrefix + " proyecto reciente"
	var olderID, newerID int64
	for _, project := range []struct {
		name          string
		start, end    string
		idDestination *int64
	}{
		{name: olderName, start: "2026-10-20", end: "2026-10-22", idDestination: &olderID},
		{name: newerName, start: "2026-10-23", end: "2026-10-24", idDestination: &newerID},
	} {
		if err := app.db.QueryRow(
			`INSERT INTO projects (name, start_date, end_date) VALUES ($1, $2, $3) RETURNING id`,
			project.name, project.start, project.end,
		).Scan(project.idDestination); err != nil {
			t.Fatalf("no se pudo crear el proyecto BDD: %v", err)
		}
	}
	for _, link := range []struct {
		projectID, memberID int64
	}{
		{olderID, app.memberID},
		{newerID, app.memberID},
		{newerID, anotherMemberID},
	} {
		if _, err := app.db.Exec(
			`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`,
			link.projectID, link.memberID,
		); err != nil {
			t.Fatalf("no se pudo asociar el integrante BDD: %v", err)
		}
	}

	resp, body := webProjectsBDDGet(t, app, "/proyectos")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200 para la lista, se obtuvo %d", resp.StatusCode)
	}
	for _, tc := range []struct {
		name   string
		values []string
	}{
		{name: olderName, values: []string{olderName, "20/10/2026", "22/10/2026", "1"}},
		{name: newerName, values: []string{newerName, "23/10/2026", "24/10/2026", "2"}},
	} {
		row := projectTableRow(body, tc.name)
		if row == "" {
			t.Errorf("se esperaba una fila de tabla para %q; respuesta: %s", tc.name, body)
			continue
		}
		for _, expected := range tc.values {
			if !strings.Contains(row, expected) {
				t.Errorf("la fila de %q debía incluir %q; fila: %s", tc.name, expected, row)
			}
		}
	}
	if strings.Index(body, newerName) < 0 || strings.Index(body, olderName) < 0 ||
		strings.Index(body, newerName) >= strings.Index(body, olderName) {
		t.Errorf("el proyecto más reciente debía aparecer primero; respuesta: %s", body)
	}
}

func TestWebProjects_EmptyListShowsMessageAndCreationForm_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false, false, true)

	resp, body := webProjectsBDDGet(t, app, "/proyectos")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200 para la lista vacía, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Aún no hay proyectos") {
		t.Errorf("se esperaba el mensaje de lista vacía; respuesta: %s", body)
	}
	for _, field := range []string{web.ProjectNameField, web.ProjectStartDateField, web.ProjectEndDateField} {
		if !strings.Contains(body, `name="`+field+`"`) {
			t.Errorf("la lista vacía debía conservar el formulario; falta %q", field)
		}
	}
}
