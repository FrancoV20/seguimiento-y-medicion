package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

const projectsTestTemplate = `{{define "content"}}
<h1>{{.Title}}</h1>
{{if .Form}}{{range $field, $message := .Form.Errors}}<p data-field="{{$field}}">{{$message}}</p>{{end}}{{end}}
{{if .Data}}
  {{if .Data.Members}}{{range .Data.Members}}<span>{{.DisplayName}}</span>{{end}}{{else}}<p>{{.Data.NoMembers}}</p>{{end}}
  {{range .Data.Projects}}<p>{{.Name}}|{{.StartDate.Format "2006-01-02"}}|{{.EndDate.Format "2006-01-02"}}|{{.MemberCount}}</p>{{end}}
{{end}}
{{if .Form}}{{range $name, $value := .Form.Values}}<input name="{{$name}}" value="{{$value}}">{{end}}{{end}}
{{end}}`

type projectsApp struct {
	db *sql.DB
	ts *httptest.Server
	fx *projectCatalogFixture
}

type emptyProjectMembersCatalog struct{ ProjectCatalog }

type emptyProjectListCatalog struct{ ProjectCatalog }

type failingProjectCatalog struct {
	ProjectCatalog
	projectsErr error
	membersErr  error
}

type failingCreateProjectStore struct {
	memberID int
}

func (s failingCreateProjectStore) Create(*project.Project) error {
	return fmt.Errorf("simulated project store failure: INSERT INTO projects; src/project/service.go; C:\\private\\config")
}

func (s failingCreateProjectStore) MemberExists(id int) (bool, error) {
	return id == s.memberID, nil
}

func (emptyProjectMembersCatalog) ListMembers(context.Context) ([]MemberOption, error) {
	return []MemberOption{}, nil
}

func (emptyProjectListCatalog) ListProjects(context.Context) ([]ProjectRow, error) {
	return []ProjectRow{}, nil
}

func (c failingProjectCatalog) ListProjects(ctx context.Context) ([]ProjectRow, error) {
	if c.projectsErr != nil {
		return nil, c.projectsErr
	}
	return c.ProjectCatalog.ListProjects(ctx)
}

func (c failingProjectCatalog) ListMembers(ctx context.Context) ([]MemberOption, error) {
	if c.membersErr != nil {
		return nil, c.membersErr
	}
	return c.ProjectCatalog.ListMembers(ctx)
}

func newProjectsApp(t *testing.T, emptyMembers bool) *projectsApp {
	t.Helper()

	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)
	service := project.NewProjectService(project.NewPostgreSQLProjectStore(db))
	return newProjectsAppWithService(t, db, fx, emptyMembers, service)
}

func newProjectsAppWithService(
	t *testing.T,
	db *sql.DB,
	fx *projectCatalogFixture,
	emptyMembers bool,
	service *project.ProjectService,
	catalogOverride ...ProjectCatalog,
) *projectsApp {
	t.Helper()

	var catalog ProjectCatalog = NewPostgreSQLProjectCatalog(db)
	if emptyMembers {
		catalog = emptyProjectMembersCatalog{ProjectCatalog: catalog}
	}
	if len(catalogOverride) > 0 {
		catalog = catalogOverride[0]
	}
	return newProjectsAppWithTemplates(t, db, fx, service, catalog, projectsTestTemplates(t))
}

func newProjectsAppWithTemplates(
	t *testing.T,
	db *sql.DB,
	fx *projectCatalogFixture,
	service *project.ProjectService,
	catalog ProjectCatalog,
	templates fs.FS,
) *projectsApp {
	t.Helper()
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.renderer.files = templates
	srv.Use(CrossOriginGuard(srv.Renderer()))
	if err := RegisterProjects(srv, service, catalog); err != nil {
		t.Fatalf("no se pudo registrar la sección Proyectos: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return &projectsApp{db: db, ts: ts, fx: fx}
}

func TestProjects_ListShowsDataNewestFirstWithin500Milliseconds(t *testing.T) {
	app := newProjectsApp(t, false)
	client := projectsTestClient(t, app.ts)

	started := time.Now()
	resp, body := projectsGet(t, client, app, "/proyectos")
	elapsed := time.Since(started)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("la respuesta debía tardar menos de 500 ms; tardó %s", elapsed)
	}

	for _, want := range app.fx.projects {
		line := fmt.Sprintf("%s|%s|%s|%d",
			want.name,
			want.startDate.Format("2006-01-02"),
			want.endDate.Format("2006-01-02"),
			want.memberCount,
		)
		if !strings.Contains(body, line) {
			t.Errorf("la lista debía mostrar los datos %q; respuesta: %s", line, body)
		}
	}

	positionC := strings.Index(body, app.fx.projects[app.fx.projectC].name)
	positionB := strings.Index(body, app.fx.projects[app.fx.projectB].name)
	positionA := strings.Index(body, app.fx.projects[app.fx.projectA].name)
	if positionC < 0 || positionB < 0 || positionA < 0 ||
		!(positionC < positionB && positionB < positionA) {
		t.Errorf("los proyectos más recientes debían aparecer primero; posiciones C=%d, B=%d, A=%d",
			positionC, positionB, positionA)
	}
}

func TestProjects_EmptyListShowsMessageAndCreationForm(t *testing.T) {
	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)
	service := project.NewProjectService(project.NewPostgreSQLProjectStore(db))
	catalog := emptyProjectListCatalog{ProjectCatalog: NewPostgreSQLProjectCatalog(db)}
	app := newProjectsAppWithTemplates(t, db, fx, service, catalog, templateFiles)

	resp, body := projectsGet(t, projectsTestClient(t, app.ts), app, "/proyectos")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200 para la lista vacía, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Aún no hay proyectos") {
		t.Errorf("se esperaba el mensaje de lista vacía; respuesta: %s", body)
	}
	for _, field := range []string{ProjectNameField, ProjectStartDateField, ProjectEndDateField} {
		if !strings.Contains(body, `name="`+field+`"`) {
			t.Errorf("la lista vacía debía conservar el formulario; falta el campo %q", field)
		}
	}
}

func TestProjects_CatalogReadFailureShowsGenericErrorPage(t *testing.T) {
	tests := []struct {
		name        string
		projectsErr error
		membersErr  error
	}{
		{name: "lectura de proyectos", projectsErr: Unavailable(errors.New("SELECT secret_projects from src/web/private.go"))},
		{name: "lectura de integrantes", membersErr: Unavailable(errors.New("SELECT secret_members from src/web/private.go"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := openContextTestDB(t)
			fx := newProjectCatalogFixture(t, db)
			service := project.NewProjectService(project.NewPostgreSQLProjectStore(db))
			catalog := failingProjectCatalog{
				ProjectCatalog: NewPostgreSQLProjectCatalog(db),
				projectsErr:    tc.projectsErr,
				membersErr:     tc.membersErr,
			}
			app := newProjectsAppWithService(t, db, fx, false, service, catalog)

			resp, body := projectsGet(t, projectsTestClient(t, app.ts), app, "/proyectos")

			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Errorf("se esperaba la página de servicio no disponible, se obtuvo %d", resp.StatusCode)
			}
			if !strings.Contains(body, TitleUnavailable) || !strings.Contains(body, MsgUnavailable) {
				t.Errorf("se esperaba la página de error común; respuesta: %s", body)
			}
			for _, detail := range []string{"secret_projects", "secret_members", "src/web/private.go", "<form"} {
				if strings.Contains(body, detail) {
					t.Errorf("la respuesta no debía contener %q; respuesta: %s", detail, body)
				}
			}
		})
	}
}

func projectsTestTemplates(t *testing.T) fs.FS {
	t.Helper()
	files, ok := formsTemplates(t).(fstest.MapFS)
	if !ok {
		t.Fatal("se esperaba el sistema de plantillas de prueba")
	}
	files["templates/projects.html"] = &fstest.MapFile{Data: []byte(projectsTestTemplate)}
	return files
}

func projectsTestClient(t *testing.T, ts *httptest.Server) *http.Client {
	t.Helper()
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
	return client
}

func projectsTestValues(name string, memberID int64, start, end string) url.Values {
	return url.Values{
		ProjectNameField:      {name},
		ProjectStartDateField: {start},
		ProjectEndDateField:   {end},
		ProjectMembersField:   {fmt.Sprint(memberID)},
	}
}

func projectsPost(t *testing.T, client *http.Client, app *projectsApp, values url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := client.PostForm(app.ts.URL+"/proyectos", values)
	if err != nil {
		t.Fatalf("POST /proyectos: %v", err)
	}
	return resp, formsReadBody(t, resp)
}

func projectsGet(t *testing.T, client *http.Client, app *projectsApp, path string) (*http.Response, string) {
	t.Helper()
	resp, err := client.Get(app.ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func countProjectByName(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE name = $1`, name).Scan(&count); err != nil {
		t.Fatalf("no se pudo consultar el proyecto de prueba: %v", err)
	}
	return count
}

func uniqueProjectsTestName(t *testing.T, app *projectsApp, suffix string) string {
	t.Helper()
	name := fmt.Sprintf("HU21 handler %d %s", time.Now().UnixNano(), suffix)
	t.Cleanup(func() {
		if _, err := app.db.Exec(`DELETE FROM projects WHERE name = $1`, name); err != nil {
			t.Errorf("no se pudo limpiar el proyecto de prueba %q: %v", name, err)
		}
	})
	return name
}

func TestProjects_CreateSuccess_RedirectsShowsFlashAndProjectWithoutChangingActiveProject(t *testing.T) {
	app := newProjectsApp(t, false)
	client := projectsTestClient(t, app.ts)
	baseURL, err := url.Parse(app.ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	client.Jar.SetCookies(baseURL, []*http.Cookie{
		{Name: CookieProject, Value: fmt.Sprint(app.fx.projectA), Path: "/"},
		{Name: CookieMember, Value: fmt.Sprint(app.fx.alphaOne), Path: "/"},
	})
	name := uniqueProjectsTestName(t, app, "creación exitosa")
	values := projectsTestValues(name, app.fx.alphaOne, "2026-10-23", "2026-10-24")

	resp, _ := projectsPost(t, client, app, values)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/proyectos" {
		t.Fatalf("se esperaba 303 a /proyectos, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == CookieProject || cookie.Name == CookieMember {
			t.Errorf("el alta no debía modificar la cookie de contexto %q", cookie.Name)
		}
	}
	if count := countProjectByName(t, app.db, name); count != 1 {
		t.Fatalf("se esperaba un proyecto persistido, se encontraron %d", count)
	}

	page, body := projectsGet(t, client, app, resp.Header.Get("Location"))
	if page.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200 al seguir la redirección, se obtuvo %d", page.StatusCode)
	}
	if strings.Count(body, MsgProjectCreated) != 1 || !strings.Contains(body, name) {
		t.Errorf("la página debía mostrar una vez el éxito y el proyecto; respuesta: %s", body)
	}
	secondPage, secondBody := projectsGet(t, client, app, "/proyectos")
	if secondPage.StatusCode != http.StatusOK || strings.Contains(secondBody, MsgProjectCreated) {
		t.Errorf("el mensaje de éxito no debía repetirse en la siguiente carga; código %d", secondPage.StatusCode)
	}
}

func TestProjects_EqualDatesAreRejectedAndNextDayIsAccepted(t *testing.T) {
	app := newProjectsApp(t, false)
	client := projectsTestClient(t, app.ts)
	equalName := uniqueProjectsTestName(t, app, "fechas iguales")
	values := projectsTestValues(equalName, app.fx.alphaOne, "2026-10-23", "2026-10-23")

	resp, body := projectsPost(t, client, app, values)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("se esperaba 422 para fechas iguales, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, project.ErrInvalidPeriod.Error()) ||
		!strings.Contains(body, `data-field="`+ProjectEndDateField+`"`) {
		t.Errorf("se esperaba el error real de HU-01 junto a fecha_fin, respuesta: %s", body)
	}
	if !strings.Contains(body, equalName) || !strings.Contains(body, "2026-10-23") {
		t.Errorf("se debían conservar los valores enviados, respuesta: %s", body)
	}
	if count := countProjectByName(t, app.db, equalName); count != 0 {
		t.Errorf("no se debía crear el proyecto con fechas iguales, se encontraron %d", count)
	}

	nextDayName := uniqueProjectsTestName(t, app, "fecha posterior")
	nextDay := projectsTestValues(nextDayName, app.fx.alphaOne, "2026-10-23", "2026-10-24")
	accepted, _ := projectsPost(t, client, app, nextDay)
	if accepted.StatusCode != http.StatusSeeOther {
		t.Errorf("se esperaba aceptar fin un día posterior con 303, se obtuvo %d", accepted.StatusCode)
	}
	if count := countProjectByName(t, app.db, nextDayName); count != 1 {
		t.Errorf("se debía crear el proyecto con fin posterior, se encontraron %d", count)
	}
}

func TestProjects_RefreshAfterCreationDoesNotDuplicateOrRepeatFlash(t *testing.T) {
	app := newProjectsApp(t, false)
	client := projectsTestClient(t, app.ts)
	name := uniqueProjectsTestName(t, app, "recarga")
	resp, _ := projectsPost(t, client, app,
		projectsTestValues(name, app.fx.alphaOne, "2026-10-23", "2026-10-24"))
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/proyectos" {
		t.Fatalf("se esperaba 303 a /proyectos, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, body := projectsGet(t, client, app, "/proyectos"); !strings.Contains(body, MsgProjectCreated) {
		t.Errorf("la primera carga debía mostrar el mensaje de éxito: %s", body)
	}
	if _, body := projectsGet(t, client, app, "/proyectos"); strings.Contains(body, MsgProjectCreated) {
		t.Errorf("la recarga no debía repetir el mensaje de éxito: %s", body)
	}
	if count := countProjectByName(t, app.db, name); count != 1 {
		t.Errorf("la recarga no debía duplicar el proyecto, se encontraron %d", count)
	}
}

func TestProjects_NoAvailableMembersShowsMessageAndRejectsDirectSubmission(t *testing.T) {
	app := newProjectsApp(t, true)
	client := projectsTestClient(t, app.ts)

	page, body := projectsGet(t, client, app, "/proyectos")
	if page.StatusCode != http.StatusOK || !strings.Contains(body, MsgNoMembersAvailable) {
		t.Fatalf("se esperaba 200 con %q, se obtuvo %d: %s", MsgNoMembersAvailable, page.StatusCode, body)
	}

	const name = "HU21 handler sin integrantes"
	resp, responseBody := projectsPost(t, client, app,
		projectsTestValues(name, app.fx.alphaOne, "2026-10-23", "2026-10-24"))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("el envío directo sin opciones debía rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if count := countProjectByName(t, app.db, name); count != 0 {
		t.Errorf("no se debía crear un proyecto sin integrantes disponibles, se encontraron %d", count)
	}
	if strings.Contains(responseBody, MsgProjectCreated) {
		t.Errorf("no debía mostrarse éxito tras el envío rechazado")
	}
}

func TestProjects_ValidationErrorsAreShownByFieldAndPreserveValues(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		message     string
		makeForm    func(int64) url.Values
		projectName string
	}{
		{
			name:    "nombre vacío",
			field:   ProjectNameField,
			message: project.ErrEmptyName.Error(),
			makeForm: func(memberID int64) url.Values {
				return projectsTestValues("", memberID, "2026-10-23", "2026-10-24")
			},
			projectName: "",
		},
		{
			name:    "nombre solo con espacios",
			field:   ProjectNameField,
			message: project.ErrEmptyName.Error(),
			makeForm: func(memberID int64) url.Values {
				return projectsTestValues("   ", memberID, "2026-10-23", "2026-10-24")
			},
			projectName: "   ",
		},
		{
			name:    "fecha de fin anterior",
			field:   ProjectEndDateField,
			message: project.ErrInvalidPeriod.Error(),
			makeForm: func(memberID int64) url.Values {
				return projectsTestValues("HU21 fin anterior", memberID, "2026-10-24", "2026-10-23")
			},
			projectName: "HU21 fin anterior",
		},
		{
			name:    "sin integrantes marcados",
			field:   ProjectMembersField,
			message: project.ErrMissingMembers.Error(),
			makeForm: func(memberID int64) url.Values {
				values := projectsTestValues("HU21 sin integrantes marcados", memberID, "2026-10-23", "2026-10-24")
				values.Del(ProjectMembersField)
				return values
			},
			projectName: "HU21 sin integrantes marcados",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newProjectsApp(t, false)
			values := tc.makeForm(app.fx.alphaOne)

			resp, body := projectsPost(t, projectsTestClient(t, app.ts), app, values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422, se obtuvo %d; respuesta: %s", resp.StatusCode, body)
			}
			if !strings.Contains(body, `data-field="`+tc.field+`"`) ||
				!strings.Contains(body, tc.message) {
				t.Errorf("se esperaba %q junto a %q; respuesta: %s", tc.message, tc.field, body)
			}
			if !strings.Contains(body, values.Get(ProjectNameField)) ||
				!strings.Contains(body, values.Get(ProjectStartDateField)) ||
				!strings.Contains(body, values.Get(ProjectEndDateField)) {
				t.Errorf("se debían conservar los valores enviados; respuesta: %s", body)
			}
			if count := countProjectByName(t, app.db, tc.projectName); count != 0 {
				t.Errorf("no debía persistirse el proyecto, se encontraron %d", count)
			}
			if strings.Contains(body, MsgProjectCreated) {
				t.Errorf("no debía mostrarse éxito tras el rechazo")
			}
		})
	}
}

func TestProjects_MissingDatesAreShownByFieldAndPreserveValues(t *testing.T) {
	tests := []struct {
		name    string
		missing string
		field   string
		message string
	}{
		{name: "falta fecha de inicio", missing: ProjectStartDateField, field: ProjectStartDateField, message: MsgMissingStartDate},
		{name: "falta fecha de fin", missing: ProjectEndDateField, field: ProjectEndDateField, message: MsgMissingEndDate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newProjectsApp(t, false)
			values := projectsTestValues("HU21 fecha ausente", app.fx.alphaOne, "2026-10-23", "2026-10-24")
			values.Del(tc.missing)

			resp, body := projectsPost(t, projectsTestClient(t, app.ts), app, values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422, se obtuvo %d; respuesta: %s", resp.StatusCode, body)
			}
			if !strings.Contains(body, `data-field="`+tc.field+`"`) ||
				!strings.Contains(body, tc.message) {
				t.Errorf("se esperaba %q junto a %q; respuesta: %s", tc.message, tc.field, body)
			}
			if !strings.Contains(body, values.Get(ProjectNameField)) ||
				!strings.Contains(body, values.Get(ProjectStartDateField)) ||
				!strings.Contains(body, values.Get(ProjectEndDateField)) {
				t.Errorf("se debían conservar los valores enviados; respuesta: %s", body)
			}
			if count := countProjectByName(t, app.db, "HU21 fecha ausente"); count != 0 {
				t.Errorf("no debía persistirse el proyecto, se encontraron %d", count)
			}
		})
	}
}

func TestProjects_InvalidDateFormatIsShownByFieldAndPreservesValues(t *testing.T) {
	app := newProjectsApp(t, false)
	values := projectsTestValues("HU21 fecha con formato inválido", app.fx.alphaOne, "23/10/2026", "2026-10-24")

	resp, body := projectsPost(t, projectsTestClient(t, app.ts), app, values)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("se esperaba 422, se obtuvo %d; respuesta: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `data-field="`+ProjectStartDateField+`"`) ||
		!strings.Contains(body, MsgInvalidDateFormat) {
		t.Errorf("se esperaba el error de formato junto a fecha_inicio; respuesta: %s", body)
	}
	if !strings.Contains(body, values.Get(ProjectNameField)) ||
		!strings.Contains(body, values.Get(ProjectStartDateField)) {
		t.Errorf("se debían conservar los valores enviados; respuesta: %s", body)
	}
	if count := countProjectByName(t, app.db, values.Get(ProjectNameField)); count != 0 {
		t.Errorf("no debía persistirse el proyecto, se encontraron %d", count)
	}
}

func TestProjects_InvalidMemberIsShownByFieldAndPreservesValues(t *testing.T) {
	tests := []struct {
		name     string
		memberID string
	}{
		{name: "integrante no numérico", memberID: "no-numérico"},
		{name: "integrante inexistente", memberID: "999999999"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newProjectsApp(t, false)
			values := projectsTestValues("HU21 integrante inválido", app.fx.alphaOne, "2026-10-23", "2026-10-24")
			values.Set(ProjectMembersField, tc.memberID)

			resp, body := projectsPost(t, projectsTestClient(t, app.ts), app, values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("se esperaba 422, se obtuvo %d; respuesta: %s", resp.StatusCode, body)
			}
			if !strings.Contains(body, `data-field="`+ProjectMembersField+`"`) ||
				!strings.Contains(body, MsgInvalidMember) {
				t.Errorf("se esperaba %q junto a integrantes; respuesta: %s", MsgInvalidMember, body)
			}
			if !strings.Contains(body, tc.memberID) || !strings.Contains(body, values.Get(ProjectNameField)) {
				t.Errorf("se debían conservar los valores enviados; respuesta: %s", body)
			}
			if count := countProjectByName(t, app.db, values.Get(ProjectNameField)); count != 0 {
				t.Errorf("no debía persistirse el proyecto, se encontraron %d", count)
			}
		})
	}
}

func TestProjects_SaveFailureShowsGenericErrorAndPreservesValues(t *testing.T) {
	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)
	store := failingCreateProjectStore{memberID: int(fx.alphaOne)}
	service := project.NewProjectService(store)
	app := newProjectsAppWithService(t, db, fx, false, service)
	values := projectsTestValues("HU21 falla de guardado", fx.alphaOne, "2026-10-23", "2026-10-24")

	resp, body := projectsPost(t, projectsTestClient(t, app.ts), app, values)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("se esperaba 500, se obtuvo %d; respuesta: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, MsgOperationFailed) {
		t.Errorf("se esperaba %q; respuesta: %s", MsgOperationFailed, body)
	}
	for _, detail := range []string{"INSERT INTO projects", "src/project/service.go", `C:\private\config`} {
		if strings.Contains(body, detail) {
			t.Errorf("la respuesta no debía exponer detalles internos %q; respuesta: %s", detail, body)
		}
	}
	if !strings.Contains(body, values.Get(ProjectNameField)) ||
		!strings.Contains(body, values.Get(ProjectStartDateField)) ||
		!strings.Contains(body, values.Get(ProjectEndDateField)) {
		t.Errorf("se debían conservar los valores enviados; respuesta: %s", body)
	}
	if strings.Contains(body, MsgProjectCreated) {
		t.Errorf("no debía mostrarse éxito tras la falla")
	}
	if count := countProjectByName(t, app.db, values.Get(ProjectNameField)); count != 0 {
		t.Errorf("no debía persistirse el proyecto, se encontraron %d", count)
	}
}

var _ ProjectCatalog = emptyProjectMembersCatalog{}
