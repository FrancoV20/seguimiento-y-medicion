package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

// Pruebas de handler de HU-11 (T035-T043, T053-T058, T064-T066, T071-T074). Usan el servidor
// real con las plantillas reales, el servicio y el almacenamiento de HU-01 sobre PostgreSQL y el
// contexto de HU-20b. Solo se reemplazan el almacenamiento o el catálogo para simular fallas.

// hu11Fixture: «Alfa» (2026-03-01 a 2026-06-30) con Ana y Beto y el Sprint 2 (2026-06-01 a
// 2026-06-14); «Beta» con Beto y horas de Beto; Carla sin proyecto.
type hu11Fixture struct {
	db                           *sql.DB
	stamp                        string
	alfa, beta                   int64
	ana, beto, carla             int64
	alfaName                     string
	anaName, betoName, carlaName string
	storyAlfa, storyBeta         int64
}

func newHU11Fixture(t *testing.T, db *sql.DB) *hu11Fixture {
	t.Helper()
	f := &hu11Fixture{db: db, stamp: fmt.Sprint(time.Now().UnixNano())}
	f.alfaName = "Alfa HU11 " + f.stamp
	f.anaName, f.betoName, f.carlaName = "Ana HU11 "+f.stamp, "Beto HU11 "+f.stamp, "Carla HU11 "+f.stamp
	t.Cleanup(f.cleanup)

	f.alfa = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-03-01', '2026-06-30') RETURNING id`, f.alfaName)
	f.beta = f.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-01-01', '2026-12-31') RETURNING id`, "Beta HU11 "+f.stamp)
	f.ana = f.member(t, f.anaName, f.alfa)
	f.beto = f.member(t, f.betoName, f.alfa, f.beta)
	f.carla = f.member(t, f.carlaName)
	f.storyAlfa = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, 'HU11 Alfa', 'd', 'Alta') RETURNING id`, f.alfa)
	f.storyBeta = f.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, 'HU11 Beta', 'd', 'Alta') RETURNING id`, f.beta)
	f.sprint(t, 2, "2026-06-01", "2026-06-14")
	f.hours(t, f.storyBeta, f.beto)
	return f
}

func (f *hu11Fixture) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
	return id
}

func (f *hu11Fixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
	}
}

func (f *hu11Fixture) member(t *testing.T, name string, projects ...int64) int64 {
	t.Helper()
	id := f.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	for _, p := range projects {
		f.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, p, id)
	}
	return id
}

func (f *hu11Fixture) sprint(t *testing.T, number int, start, end string) {
	t.Helper()
	f.exec(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date) VALUES ($1, $2, 'Objetivo HU11', $3, $4)`, f.alfa, number, start, end)
}

func (f *hu11Fixture) hours(t *testing.T, story, member int64) {
	t.Helper()
	f.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours) VALUES ($1, $2, '2026-03-10', 'HU11', 2)`, story, member)
}

func (f *hu11Fixture) cleanup() {
	_, _ = f.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id IN ($1, $2))`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM projects WHERE id IN ($1, $2)`, f.alfa, f.beta)
	_, _ = f.db.Exec(`DELETE FROM members WHERE id IN ($1, $2, $3)`, f.ana, f.beto, f.carla)
	_, _ = f.db.Exec(`DELETE FROM members WHERE display_name LIKE $1`, "HU11 extra "+f.stamp+"%")
}

// snapshot resume lo que una modificación puede cambiar en «Alfa».
func (f *hu11Fixture) snapshot(t *testing.T) string {
	t.Helper()
	var s string
	err := f.db.QueryRow(`
		SELECT p.name || '|' || p.start_date || '|' || p.end_date || '|' ||
		       COALESCE((SELECT string_agg(member_id::text, ',' ORDER BY member_id) FROM project_members WHERE project_id = p.id), '')
		FROM projects p WHERE p.id = $1`, f.alfa).Scan(&s)
	if err != nil {
		t.Fatalf("no se pudo leer «Alfa»: %v", err)
	}
	return s
}

func (f *hu11Fixture) assertUnchanged(t *testing.T, before string) {
	t.Helper()
	if after := f.snapshot(t); after != before {
		t.Errorf("«Alfa» no debía cambiar:\nantes   %s\ndespués %s", before, after)
	}
}

type hu11App struct {
	ts *httptest.Server
	fx *hu11Fixture
}

type hu11Option struct {
	store   func(project.ProjectStore) project.ProjectStore
	catalog func(ProjectCatalog) ProjectCatalog
}

// newHU11Server (T035) arma el servidor real con Proyectos y el contexto de HU-20b.
func newHU11Server(t *testing.T, opts ...hu11Option) *hu11App {
	t.Helper()
	db := openContextTestDB(t)
	fx := newHU11Fixture(t, db)
	var store project.ProjectStore = project.NewPostgreSQLProjectStore(db)
	var catalog ProjectCatalog = NewPostgreSQLProjectCatalog(db)
	for _, o := range opts {
		if o.store != nil {
			store = o.store(store)
		}
		if o.catalog != nil {
			catalog = o.catalog(catalog)
		}
	}
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.Use(CrossOriginGuard(srv.Renderer()))
	RegisterContext(srv, NewPostgreSQLContextStore(db))
	if err := RegisterProjects(srv, project.NewProjectService(store), catalog); err != nil {
		t.Fatalf("no se pudo registrar Proyectos: %v", err)
	}
	app := &hu11App{ts: httptest.NewServer(srv.Handler()), fx: fx}
	t.Cleanup(app.ts.Close)
	return app
}

func (a *hu11App) client() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func (a *hu11App) get(t *testing.T, c *http.Client, path string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *hu11App) post(t *testing.T, c *http.Client, path string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.ts.URL+path, v)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *hu11App) editPath() string { return fmt.Sprintf("/proyectos/%d/modificar", a.fx.alfa) }

// form arma el envío de la modificación de «Alfa».
func (a *hu11App) form(name, start, end string, members ...int64) url.Values {
	v := url.Values{ProjectNameField: {name}, ProjectStartDateField: {start}, ProjectEndDateField: {end}}
	for _, m := range members {
		v.Add(ProjectMembersField, strconv.FormatInt(m, 10))
	}
	return v
}

func checkedHTML(id int64) string   { return fmt.Sprintf(`value="%d" checked>`, id) }
func uncheckedHTML(id int64) string { return fmt.Sprintf(`value="%d">`, id) }

// ---------------------------------------------------------------- Bloque A

// T036
func TestHU11_EnlaceModificarEnLista(t *testing.T) {
	a := newHU11Server(t)
	resp, body := a.get(t, a.client(), "/proyectos")
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, `<th scope="col">Acciones</th>`,
		fmt.Sprintf(`<a href="/proyectos/%d/modificar">%s</a>`, a.fx.alfa, LinkEditProject),
		fmt.Sprintf(`<a href="/proyectos/%d/modificar">%s</a>`, a.fx.beta, LinkEditProject))
}

// T037
func TestHU11_FormularioConDatosActuales(t *testing.T) {
	a := newHU11Server(t)
	resp, body := a.get(t, a.client(), a.editPath())
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "<h1>"+TitleProjectEdit+"</h1>", `action="`+a.editPath()+`"`,
		inputValueHTML(ProjectNameField, a.fx.alfaName),
		inputValueHTML(ProjectStartDateField, "2026-03-01"), inputValueHTML(ProjectEndDateField, "2026-06-30"),
		checkedHTML(a.fx.ana), checkedHTML(a.fx.beto), uncheckedHTML(a.fx.carla))
}

// T038
func TestHU11_ModificacionExitosa(t *testing.T) {
	a := newHU11Server(t)
	c := a.client()
	resp, _ := a.post(t, c, a.editPath(), a.form("Alfa nuevo "+a.fx.stamp, "2026-02-01", "2026-07-31", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/proyectos" {
		t.Fatalf("se esperaba volver a /proyectos, se obtuvo %q", loc)
	}
	_, body := a.get(t, c, "/proyectos")
	if n := strings.Count(body, MsgProjectUpdated); n != 1 {
		t.Errorf("el mensaje de éxito debía verse una vez, se vio %d", n)
	}
	assertContains(t, body, "Alfa nuevo "+a.fx.stamp, "<td>01/02/2026</td>", "<td>31/07/2026</td>")
}

// T043: después del 303, recargar la lista no repite el mensaje ni vuelve a guardar.
func TestHU11_RecargaSinReenviar(t *testing.T) {
	a := newHU11Server(t)
	c := a.client()
	resp, _ := a.post(t, c, a.editPath(), a.form("Recarga "+a.fx.stamp, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	_, body := a.get(t, c, "/proyectos")
	assertContains(t, body, MsgProjectUpdated)
	after := a.fx.snapshot(t)
	_, body = a.get(t, c, "/proyectos")
	assertNotContains(t, body, MsgProjectUpdated)
	a.fx.assertUnchanged(t, after)
}

func TestHU11_SinCambiosEsExito(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	resp, _ := a.post(t, a.client(), a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	a.fx.assertUnchanged(t, before)
}

// T039: el nombre nuevo se ve en la estructura común sin volver a elegir el proyecto.
func TestHU11_NombreEnEstructuraComun(t *testing.T) {
	a := newHU11Server(t)
	c := a.client()
	resp, _ := a.post(t, c, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(a.fx.alfa, 10)}})
	assertStatus(t, resp, http.StatusSeeOther)

	nuevo := "Alfa 2 " + a.fx.stamp
	resp, _ = a.post(t, c, a.editPath(), a.form(nuevo, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	_, body := a.get(t, c, "/")
	assertContains(t, body, "Proyecto: <strong>"+nuevo+"</strong>")
}

// T040
func TestHU11_NombreVacio(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	for _, name := range []string{"", "   "} {
		resp, body := a.post(t, a.client(), a.editPath(), a.form(name, "2026-03-02", "2026-06-29", a.fx.ana, a.fx.carla))
		assertStatus(t, resp, http.StatusUnprocessableEntity)
		assertContains(t, body, fieldErrorHTML(ProjectNameField, project.ErrEmptyName.Error()),
			inputValueHTML(ProjectStartDateField, "2026-03-02"), inputValueHTML(ProjectEndDateField, "2026-06-29"),
			checkedHTML(a.fx.ana), checkedHTML(a.fx.carla), uncheckedHTML(a.fx.beto))
	}
	a.fx.assertUnchanged(t, before)
}

// T041
func TestHU11_FechasInvalidas(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	for _, end := range []string{"2026-03-01", "2026-02-15"} {
		resp, body := a.post(t, a.client(), a.editPath(), a.form("Otro", "2026-03-01", end, a.fx.ana))
		assertStatus(t, resp, http.StatusUnprocessableEntity)
		assertContains(t, body, fieldErrorHTML(ProjectEndDateField, project.ErrInvalidPeriod.Error()),
			inputValueHTML(ProjectNameField, "Otro"), inputValueHTML(ProjectEndDateField, end))
	}
	a.fx.assertUnchanged(t, before)
}

// T042
func TestHU11_FechaFaltanteOFormato(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	cases := []struct{ start, end, field, msg string }{
		{"", "2026-06-30", ProjectStartDateField, MsgMissingStartDate},
		{"2026-03-01", "", ProjectEndDateField, MsgMissingEndDate},
		{"01/03/2026", "2026-06-30", ProjectStartDateField, MsgInvalidDateFormat},
	}
	for _, c := range cases {
		resp, body := a.post(t, a.client(), a.editPath(), a.form("Otro", c.start, c.end, a.fx.ana))
		assertStatus(t, resp, http.StatusUnprocessableEntity)
		assertContains(t, body, fieldErrorHTML(c.field, c.msg), inputValueHTML(ProjectNameField, "Otro"))
	}
	a.fx.assertUnchanged(t, before)
}

// T043
func TestHU11_NombreEscapado(t *testing.T) {
	a := newHU11Server(t)
	c := a.client()
	name := `<script>alert("x")</script> ` + a.fx.stamp
	resp, _ := a.post(t, c, a.editPath(), a.form(name, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	for _, path := range []string{"/proyectos", a.editPath()} {
		_, body := a.get(t, c, path)
		assertContains(t, body, "&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;")
		assertNotContains(t, body, `<script>alert("x")</script>`)
	}
}

// failingEditCatalog falla al leer el proyecto a modificar.
type failingEditCatalog struct{ ProjectCatalog }

func (failingEditCatalog) ProjectForEdit(context.Context, int64) (ProjectEdit, error) {
	return ProjectEdit{}, errors.New("SELECT p.name FROM projects: conexión rechazada en postgres_project_catalog_edit.go")
}

func TestHU11_ErrorAlLeer(t *testing.T) {
	a := newHU11Server(t, hu11Option{catalog: func(c ProjectCatalog) ProjectCatalog { return failingEditCatalog{c} }})
	resp, body := a.get(t, a.client(), a.editPath())
	assertStatus(t, resp, http.StatusInternalServerError)
	assertContains(t, body, TitleInternal)
	assertNoInternals(t, body)
}

// ---------------------------------------------------------------- Bloque B

// T053
func TestHU11_AgregarIntegrante(t *testing.T) {
	a := newHU11Server(t)
	c := a.client()
	resp, _ := a.post(t, c, a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto, a.fx.carla))
	assertStatus(t, resp, http.StatusSeeOther)

	a.post(t, c, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(a.fx.alfa, 10)}})
	_, body := a.get(t, c, pathSelectMember)
	assertContains(t, body, a.fx.carlaName)

	ok, err := effort.NewPostgreSQLEffortStore(a.fx.db).MemberBelongsToStoryProject(context.Background(),
		strconv.FormatInt(a.fx.carla, 10), strconv.FormatInt(a.fx.storyAlfa, 10))
	if err != nil || !ok {
		t.Errorf("Carla debía poder cargar esfuerzo en «Alfa»: %v, %v", ok, err)
	}
}

// T054
func TestHU11_QuitarSinHoras(t *testing.T) {
	a := newHU11Server(t)
	c := a.client()
	resp, _ := a.post(t, c, a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-06-30", a.fx.ana))
	assertStatus(t, resp, http.StatusSeeOther)

	a.post(t, c, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(a.fx.alfa, 10)}})
	_, body := a.get(t, c, pathSelectMember)
	assertContains(t, body, a.fx.anaName)
	assertNotContains(t, body, a.fx.betoName)
	var n int
	_ = a.fx.db.QueryRow(`SELECT COUNT(*) FROM effort_logs WHERE story_id = $1 AND member_id = $2`, a.fx.storyBeta, a.fx.beto).Scan(&n)
	if n != 1 {
		t.Errorf("las horas de Beto en «Beta» debían seguir, hay %d", n)
	}
}

// T055
func TestHU11_QuitarUltimo(t *testing.T) {
	a := newHU11Server(t)
	a.fx.exec(t, `DELETE FROM project_members WHERE project_id = $1 AND member_id = $2`, a.fx.alfa, a.fx.beto)
	before := a.fx.snapshot(t)
	resp, body := a.post(t, a.client(), a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-06-30"))
	assertStatus(t, resp, http.StatusUnprocessableEntity)
	assertContains(t, body, fieldErrorHTML(ProjectMembersField, project.ErrMissingMembers.Error()))
	a.fx.assertUnchanged(t, before)
}

// T056
func TestHU11_QuitarConHoras(t *testing.T) {
	a := newHU11Server(t)
	a.fx.hours(t, a.fx.storyAlfa, a.fx.beto)
	before := a.fx.snapshot(t)
	resp, body := a.post(t, a.client(), a.editPath(), a.form("Nombre nuevo", "2026-03-01", "2026-06-30", a.fx.ana, a.fx.carla))
	assertStatus(t, resp, http.StatusConflict)
	assertContains(t, body, fieldErrorHTML(ProjectMembersField, fmt.Sprintf(MsgMemberHasEffort, a.fx.betoName)),
		inputValueHTML(ProjectNameField, "Nombre nuevo"), checkedHTML(a.fx.carla), uncheckedHTML(a.fx.beto))
	a.fx.assertUnchanged(t, before)
}

// T057
func TestHU11_IntegranteInvalido(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	for _, raw := range []string{"2147483647", "abc"} {
		v := a.form("Otro", "2026-03-01", "2026-06-30", a.fx.ana)
		v.Add(ProjectMembersField, raw)
		resp, body := a.post(t, a.client(), a.editPath(), v)
		assertStatus(t, resp, http.StatusUnprocessableEntity)
		assertContains(t, body, fieldErrorHTML(ProjectMembersField, MsgInvalidMember), inputValueHTML(ProjectNameField, "Otro"))
	}
	a.fx.assertUnchanged(t, before)
}

// T058: el integrante identificado que otra persona quitó se limpia con una advertencia.
func TestHU11_IdentificadoQuitado(t *testing.T) {
	a := newHU11Server(t)
	beto := a.client()
	a.post(t, beto, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(a.fx.alfa, 10)}})
	resp, _ := a.post(t, beto, pathSelectMember, loginAs(t, a.fx.db, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)

	resp, _ = a.post(t, a.client(), a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-06-30", a.fx.ana))
	assertStatus(t, resp, http.StatusSeeOther)

	resp, body := a.get(t, beto, "/")
	assertContains(t, body, MsgMemberGone)
	cleared := false
	for _, ck := range resp.Cookies() {
		if ck.Name == CookieMember && ck.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("se esperaba borrar la cookie del integrante")
	}
}

// ---------------------------------------------------------------- Bloque C

// T064
func TestHU11_SprintQuedaFuera(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	cases := []struct{ start, end, field string }{
		{"2026-03-01", "2026-06-10", ProjectEndDateField},
		{"2026-06-05", "2026-06-30", ProjectStartDateField},
	}
	for _, c := range cases {
		resp, body := a.post(t, a.client(), a.editPath(), a.form("Otro", c.start, c.end, a.fx.ana, a.fx.beto))
		assertStatus(t, resp, http.StatusConflict)
		assertContains(t, body, fieldErrorHTML(c.field, fmt.Sprintf(MsgSprintOutOfRange, 2)),
			inputValueHTML(ProjectStartDateField, c.start), inputValueHTML(ProjectEndDateField, c.end))
	}
	a.fx.assertUnchanged(t, before)
}

// T065
func TestHU11_SprintYaFueraNoBloquea(t *testing.T) {
	a := newHU11Server(t)
	a.fx.sprint(t, 1, "2026-02-20", "2026-03-05")
	resp, _ := a.post(t, a.client(), a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-07-31", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	var start, end string
	_ = a.fx.db.QueryRow(`SELECT start_date::text, end_date::text FROM sprints WHERE project_id = $1 AND number = 1`, a.fx.alfa).Scan(&start, &end)
	if start != "2026-02-20" || end != "2026-03-05" {
		t.Errorf("el Sprint 1 no debía cambiar: %s a %s", start, end)
	}
}

// T066
func TestHU11_SinCambioDeFechasConSprints(t *testing.T) {
	a := newHU11Server(t)
	a.fx.sprint(t, 1, "2026-02-20", "2026-03-05") // dato viejo fuera de rango
	resp, _ := a.post(t, a.client(), a.editPath(), a.form("Solo nombre "+a.fx.stamp, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto))
	assertStatus(t, resp, http.StatusSeeOther)
	resp, _ = a.post(t, a.client(), a.editPath(), a.form("Solo nombre "+a.fx.stamp, "2026-03-01", "2026-06-30", a.fx.ana, a.fx.beto, a.fx.carla))
	assertStatus(t, resp, http.StatusSeeOther)
}

// ---------------------------------------------------------------- Bloque D

// T071
func TestHU11_ProyectoInexistente(t *testing.T) {
	a := newHU11Server(t)
	before := a.fx.snapshot(t)
	for _, id := range []string{"999999", "abc", "0", "-1", "007", "2147483648"} {
		path := "/proyectos/" + id + "/modificar"
		resp, body := a.get(t, a.client(), path)
		assertStatus(t, resp, http.StatusNotFound)
		assertContains(t, body, TitleNotFound)
		resp, _ = a.post(t, a.client(), path, a.form("Otro", "2026-03-01", "2026-06-30", a.fx.ana))
		assertStatus(t, resp, http.StatusNotFound)
	}
	a.fx.assertUnchanged(t, before)
}

func TestHU11_404AntesQue422(t *testing.T) {
	a := newHU11Server(t)
	resp, _ := a.post(t, a.client(), "/proyectos/999999/modificar", a.form("", "", "", 0))
	assertStatus(t, resp, http.StatusNotFound)
}

// T072
func TestHU11_ProyectoBorradoAlGuardar(t *testing.T) {
	a := newHU11Server(t, hu11Option{store: func(project.ProjectStore) project.ProjectStore {
		return &project.TestProjectStore{UpdateErr: project.ErrProjectNotFound}
	}})
	resp, _ := a.post(t, a.client(), a.editPath(), a.form("Otro", "2026-03-01", "2026-06-30", a.fx.ana))
	assertStatus(t, resp, http.StatusNotFound)
}

// failingUpdateStore es el almacenamiento real salvo que la modificación falla con detalles internos.
type failingUpdateStore struct{ project.ProjectStore }

func (failingUpdateStore) Update(context.Context, project.ProjectUpdate) error {
	return errors.New("UPDATE projects SET name: SQLSTATE 08006 en postgres_store_update.go; DATABASE_URL=postgres://x")
}

// T073
func TestHU11_ErrorAlGuardar(t *testing.T) {
	a := newHU11Server(t, hu11Option{store: func(s project.ProjectStore) project.ProjectStore { return failingUpdateStore{s} }})
	before := a.fx.snapshot(t)
	resp, body := a.post(t, a.client(), a.editPath(), a.form("Nombre nuevo", "2026-03-02", "2026-06-29", a.fx.ana, a.fx.carla))
	assertStatus(t, resp, http.StatusInternalServerError)
	assertContains(t, body, MsgOperationFailed, inputValueHTML(ProjectNameField, "Nombre nuevo"),
		inputValueHTML(ProjectStartDateField, "2026-03-02"), checkedHTML(a.fx.carla))
	assertNotContains(t, body, MsgProjectUpdated)
	assertNoInternals(t, body)
	a.fx.assertUnchanged(t, before)
}

// T074 (SC-006): 50 integrantes en el sistema y 20 Sprints en el proyecto.
func TestHU11_Rendimiento(t *testing.T) {
	a := newHU11Server(t)
	for i := 0; i < 50; i++ {
		a.fx.member(t, fmt.Sprintf("HU11 extra %s %02d", a.fx.stamp, i))
	}
	for n := 3; n < 23; n++ {
		a.fx.sprint(t, n, "2026-03-10", "2026-03-20")
	}
	c := a.client()
	start := time.Now()
	resp, _ := a.get(t, c, a.editPath())
	if elapsed := time.Since(start); resp.StatusCode != http.StatusOK || elapsed > 500*time.Millisecond {
		t.Errorf("GET: %d en %v (máximo 500 ms)", resp.StatusCode, elapsed)
	}
	start = time.Now()
	resp, _ = a.post(t, c, a.editPath(), a.form(a.fx.alfaName, "2026-03-01", "2026-07-15", a.fx.ana, a.fx.beto))
	if elapsed := time.Since(start); resp.StatusCode != http.StatusSeeOther || elapsed > 500*time.Millisecond {
		t.Errorf("POST: %d en %v (máximo 500 ms)", resp.StatusCode, elapsed)
	}
}
