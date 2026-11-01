package web

import (
	"database/sql"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// HU-13, T046: servidor de prueba de la edición, con el servicio y el catálogo reales y las
// historias de T031, más el Sprint 1 'Finalizado' con «Pagos» (Hecho) y el Sprint 2 'Activo' con
// «Reportes» (En Sprint).
func newHU13EditApp(t *testing.T) *hu13BacklogApp {
	return newHU13EditAppWithStore(t, func(app *ctxApp) backlog.BacklogStore {
		return backlog.NewPostgreSQLBacklogStore(app.db)
	})
}

func newHU13EditAppWithStore(t *testing.T, storeFactory func(*ctxApp) backlog.BacklogStore) *hu13BacklogApp {
	t.Helper()
	app := newHU13BacklogAppWithStore(t, storeFactory)
	for _, s := range []struct {
		number int
		status string
		story  string
	}{{1, "Finalizado", "Pagos"}, {2, "Activo", "Reportes"}} {
		var sprintID int64
		if err := app.db.QueryRow(`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, 'Objetivo HU-13', CURRENT_DATE, CURRENT_DATE + 14, $3) RETURNING id`,
			app.fx.projectA, s.number, s.status).Scan(&sprintID); err != nil {
			t.Fatalf("create HU-13 sprint: %v", err)
		}
		if _, err := app.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`,
			sprintID, app.stories[s.story]); err != nil {
			t.Fatalf("assign HU-13 story: %v", err)
		}
		if _, err := app.db.Exec(`UPDATE user_stories SET active_sprint_id = $1 WHERE id = $2`,
			sprintID, app.stories[s.story]); err != nil {
			t.Fatalf("assign HU-13 story: %v", err)
		}
	}
	return app
}

func (a *hu13BacklogApp) editPath(title string) string {
	return "/backlog/" + id64(a.stories[title]) + "/editar"
}

// editValues es un envío válido del formulario de edición de «Login» con el título indicado.
func editValues(title string) url.Values {
	return url.Values{
		BacklogFieldTitle:       {title},
		BacklogFieldDescription: {"Descripción de Login"},
		BacklogFieldPriority:    {"Media"},
		BacklogFieldStoryPoints: {"5"},
		BacklogFieldCriterion:   {"Acepta email", "Rechaza clave vacía"},
		BacklogFieldStatus:      {"Pendiente"},
	}
}

// hu13Story es lo que hay en la base de una historia.
type hu13Story struct {
	Title, Description, Priority, Status string
	StoryPoints                          sql.NullInt64
	Hours                                sql.NullFloat64
	CreatedAt                            time.Time
	Criteria                             []string
}

func (a *hu13BacklogApp) stored(t *testing.T, title string) hu13Story {
	t.Helper()
	var s hu13Story
	if err := a.db.QueryRow(`SELECT title, description, priority, status, story_points, estimated_hours::float8, created_at
		FROM user_stories WHERE id = $1`, a.stories[title]).Scan(&s.Title, &s.Description, &s.Priority, &s.Status,
		&s.StoryPoints, &s.Hours, &s.CreatedAt); err != nil {
		t.Fatalf("read HU-13 story %q: %v", title, err)
	}
	rows, err := a.db.Query(`SELECT content FROM acceptance_criteria WHERE story_id = $1 ORDER BY position`, a.stories[title])
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		s.Criteria = append(s.Criteria, c)
	}
	return s
}

func (a *hu13BacklogApp) assertUnchanged(t *testing.T, title string, before hu13Story) {
	t.Helper()
	if after := a.stored(t, title); !reflect.DeepEqual(after, before) {
		t.Errorf("«%s» cambió:\nantes   %+v\ndespués %+v", title, before, after)
	}
}

// rowOf devuelve la fila <tr> de la tabla del Backlog que tiene el título indicado.
func rowOf(t *testing.T, body, title string) string {
	t.Helper()
	rows := extractHU22TableRows(t, body)
	i := strings.Index(rows, "<td>"+title+"</td>")
	if i < 0 {
		t.Fatalf("la lista no tiene la historia %q", title)
	}
	start := strings.LastIndex(rows[:i], "<tr>")
	end := strings.Index(rows[i:], "</tr>")
	return rows[start : i+end]
}

// follow sigue una redirección 303 y devuelve la página de destino.
func (a *hu13BacklogApp) follow(t *testing.T, resp *http.Response) string {
	t.Helper()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d; se esperaba 303", resp.StatusCode)
	}
	_, body := a.get(t, a.client, resp.Header.Get("Location"))
	return body
}

// ---- Bloque B: edición (T047–T053)

// T047
func TestHU13_FormularioConDatosActuales(t *testing.T) {
	app := newHU13EditApp(t)
	resp, body := app.get(t, app.client, app.editPath("Login")+"?estado=Pendiente")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET edición = %d; se esperaba 200", resp.StatusCode)
	}
	for _, want := range []string{
		"<h1>" + TitleStoryEdit + "</h1>",
		`action="` + app.editPath("Login") + `"`,
		`name="titulo" value="Login"`,
		`>Descripción de Login</textarea>`,
		`<option value="Media" selected>`,
		`<option value="5" selected>`,
		`name="estado" value="Pendiente"`,
		`href="/backlog?estado=Pendiente">Cancelar</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("el formulario no contiene %q", want)
		}
	}
	first, second := strings.Index(body, `value="Acepta email"`), strings.Index(body, `value="Rechaza clave vacía"`)
	if first < 0 || second < 0 || first > second {
		t.Error("los criterios no aparecen en su orden")
	}
	if strings.Contains(body, BacklogFieldEstimatedHours) {
		t.Error("el formulario de edición no debe tener el campo de horas estimadas")
	}
}

// T048
func TestHU13_EdicionExitosa(t *testing.T) {
	assertSaved := func(t *testing.T, app *hu13BacklogApp, resp *http.Response) string {
		t.Helper()
		if loc := resp.Header.Get("Location"); resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/backlog") {
			t.Fatalf("POST = %d a %q; se esperaba 303 a /backlog", resp.StatusCode, loc)
		}
		body := app.follow(t, resp)
		if strings.Count(body, MsgStoryUpdated) != 1 || strings.Contains(body, MsgStoryUpdatedMissingPoints) {
			t.Errorf("se esperaba %q una sola vez", MsgStoryUpdated)
		}
		return body
	}
	assertKept := func(t *testing.T, app *hu13BacklogApp, title string, before hu13Story) {
		t.Helper()
		after := app.stored(t, title)
		if after.Status != before.Status || after.Hours != before.Hours || !after.CreatedAt.Equal(before.CreatedAt) {
			t.Errorf("cambiaron el estado, las horas o la fecha de creación:\nantes   %+v\ndespués %+v", before, after)
		}
	}

	t.Run("7a título", func(t *testing.T) {
		app := newHU13EditApp(t)
		before := app.stored(t, "Login")
		resp, _ := app.post(t, app.client, app.editPath("Login"), editValues("Inicio de sesión"))
		body := assertSaved(t, app, resp)
		if !strings.Contains(body, "<td>Inicio de sesión</td>") {
			t.Error("la lista no muestra «Inicio de sesión»")
		}
		assertKept(t, app, "Login", before)
	})
	t.Run("7b descripción", func(t *testing.T) {
		app := newHU13EditApp(t)
		before := app.stored(t, "Login")
		values := editValues("Login")
		values.Set(BacklogFieldDescription, "Como usuario quiero ingresar con mi email")
		resp, _ := app.post(t, app.client, app.editPath("Login"), values)
		assertSaved(t, app, resp)
		_, form := app.get(t, app.client, app.editPath("Login"))
		if !strings.Contains(form, ">Como usuario quiero ingresar con mi email</textarea>") {
			t.Error("la edición no muestra la descripción nueva")
		}
		assertKept(t, app, "Login", before)
	})
	t.Run("7c prioridad", func(t *testing.T) {
		app := newHU13EditApp(t)
		before := app.stored(t, "Reportes")
		values := url.Values{
			BacklogFieldTitle:       {"Reportes"},
			BacklogFieldDescription: {"Descripción de Reportes"},
			BacklogFieldPriority:    {"Alta"},
			BacklogFieldStoryPoints: {"3"},
			BacklogFieldCriterion:   {"Exporta a PDF"},
		}
		resp, _ := app.post(t, app.client, app.editPath("Reportes"), values)
		body := assertSaved(t, app, resp)
		assertTitles(t, app.tableTitles(t, body), "Pagos", "Reportes", "Login", "Perfil")
		if !strings.Contains(rowOf(t, body, "Reportes"), "<td>Alta</td>") {
			t.Error("la lista no muestra «Reportes» con prioridad Alta")
		}
		assertKept(t, app, "Reportes", before)
	})
	t.Run("7d Story Points", func(t *testing.T) {
		app := newHU13EditApp(t)
		before := app.stored(t, "Login")
		values := editValues("Login")
		values.Set(BacklogFieldStoryPoints, "8")
		resp, _ := app.post(t, app.client, app.editPath("Login"), values)
		body := assertSaved(t, app, resp)
		if !strings.Contains(rowOf(t, body, "Login"), "<td>8</td>") {
			t.Error("la lista no muestra 8 Story Points en «Login»")
		}
		assertKept(t, app, "Login", before)
	})
}

// T049
func TestHU13_VuelveALaListaFiltrada(t *testing.T) {
	app := newHU13EditApp(t)
	_, list := app.get(t, app.client, "/backlog?estado=Pendiente")
	if !strings.Contains(list, `href="`+app.editPath("Login")+`?estado=Pendiente"`) {
		t.Error("el enlace «Editar» de la lista filtrada no lleva el filtro")
	}

	resp, _ := app.post(t, app.client, app.editPath("Login"), editValues("Inicio de sesión"))
	if loc := resp.Header.Get("Location"); loc != "/backlog?estado=Pendiente" {
		t.Fatalf("Location = %q; se esperaba /backlog?estado=Pendiente", loc)
	}
	body := app.follow(t, resp)
	if !strings.Contains(body, MsgStoryUpdated) || !strings.Contains(body, "<td>Inicio de sesión</td>") {
		t.Error("la lista filtrada no muestra el mensaje o la historia editada")
	}

	for _, estado := range []string{"Archivada", "", "https://otro.sitio"} {
		values := editValues("Login")
		values.Set(BacklogFieldStatus, estado)
		resp, _ := app.post(t, app.client, app.editPath("Login"), values)
		if loc := resp.Header.Get("Location"); resp.StatusCode != http.StatusSeeOther || loc != "/backlog" {
			t.Errorf("estado=%q: %d a %q; se esperaba 303 a /backlog", estado, resp.StatusCode, loc)
		}
	}
}

// T050
func TestHU13_AgregarYQuitarCriterio(t *testing.T) {
	app := newHU13EditApp(t)
	before := app.stored(t, "Login")

	values := editValues("Título sin guardar")
	values.Set(BacklogFieldAction, "agregar_criterio")
	resp, body := app.post(t, app.client, app.editPath("Login"), values)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("agregar criterio = %d; se esperaba 200", resp.StatusCode)
	}
	if got := strings.Count(body, `name="criterio"`); got != 3 {
		t.Errorf("campos de criterio = %d; se esperaban 3", got)
	}
	if !strings.Contains(body, `value="Título sin guardar"`) {
		t.Error("no se conservó el título escrito")
	}
	app.assertUnchanged(t, "Login", before)

	values = editValues("Login")
	values.Set(BacklogFieldAction, "quitar_criterio:0")
	resp, body = app.post(t, app.client, app.editPath("Login"), values)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("quitar criterio = %d; se esperaba 200", resp.StatusCode)
	}
	if strings.Contains(body, `value="Acepta email"`) || !strings.Contains(body, `value="Rechaza clave vacía"`) {
		t.Error("no se quitó «Acepta email» o se perdió «Rechaza clave vacía»")
	}
	app.assertUnchanged(t, "Login", before)
}

// T051
func TestHU13_QuitarStoryPoints(t *testing.T) {
	app := newHU13EditApp(t)
	values := editValues("Login")
	values.Set(BacklogFieldStoryPoints, "")
	resp, _ := app.post(t, app.client, app.editPath("Login"), values)
	body := app.follow(t, resp)
	if !strings.Contains(body, MsgStoryUpdatedMissingPoints) || !strings.Contains(body, "alert-warning") {
		t.Errorf("se esperaba la advertencia %q", MsgStoryUpdatedMissingPoints)
	}
	if !strings.Contains(rowOf(t, body, "Login"), "<td>Sin estimar</td>") {
		t.Error("la lista no muestra «Login» como «Sin estimar»")
	}
	if sp := app.stored(t, "Login").StoryPoints; sp.Valid {
		t.Errorf("story_points = %d; se esperaba NULL", sp.Int64)
	}
}

// T052
func TestHU13_EdicionSinProyecto(t *testing.T) {
	app := newHU13EditApp(t)
	client := app.ctxApp.client(false)
	path := app.editPath("Login")

	resp, _ := app.get(t, client, path)
	want := "/contexto/proyecto?volver_a=" + url.QueryEscape(path)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
		t.Fatalf("sin proyecto: %d a %q; se esperaba 303 a %q", resp.StatusCode, resp.Header.Get("Location"), want)
	}
	resp, _ = app.selectProject(t, client, id64(app.fx.projectA), path)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != path {
		t.Fatalf("al elegir el proyecto: %d a %q; se esperaba volver a %q", resp.StatusCode, resp.Header.Get("Location"), path)
	}
	resp, body := app.get(t, client, path)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, TitleStoryEdit) {
		t.Errorf("la edición no se muestra después de elegir el proyecto: %d", resp.StatusCode)
	}
}

// T053
func TestHU13_IgnoraHorasEstimadas(t *testing.T) {
	app := newHU13EditApp(t)
	for _, hours := range []string{"abc", "7"} {
		values := editValues("Login")
		values.Set(BacklogFieldEstimatedHours, hours)
		resp, _ := app.post(t, app.client, app.editPath("Login"), values)
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("horas_estimadas=%s: %d; se esperaba 303", hours, resp.StatusCode)
		}
		if h := app.stored(t, "Login").Hours; !h.Valid || h.Float64 != 4 {
			t.Errorf("horas_estimadas=%s: las horas quedaron en %v; se esperaba 4", hours, h)
		}
	}
}

func TestHU13_TextoEscapado(t *testing.T) {
	app := newHU13EditApp(t)
	values := editValues("<script>alert(1)</script>")
	values[BacklogFieldCriterion] = []string{"<b>negrita</b>"}
	resp, _ := app.post(t, app.client, app.editPath("Login"), values)
	list := app.follow(t, resp)
	_, form := app.get(t, app.client, app.editPath("Login"))
	for name, body := range map[string]string{"lista": list, "formulario": form} {
		if strings.Contains(body, "<script>alert(1)</script>") || strings.Contains(body, "<b>negrita</b>") {
			t.Errorf("%s: el texto no está escapado", name)
		}
		if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
			t.Errorf("%s: no muestra el título como texto", name)
		}
	}
	if !strings.Contains(form, "&lt;b&gt;negrita&lt;/b&gt;") {
		t.Error("formulario: no muestra el criterio como texto")
	}
}

func TestHU13_RecargaSinReenviar(t *testing.T) {
	app := newHU13EditApp(t)
	resp, _ := app.post(t, app.client, app.editPath("Login"), editValues("Inicio de sesión"))
	first := app.follow(t, resp)
	_, second := app.get(t, app.client, "/backlog?estado=Pendiente")
	if !strings.Contains(first, MsgStoryUpdated) || strings.Contains(second, MsgStoryUpdated) {
		t.Error("el mensaje no apareció una sola vez")
	}
	if n := hu22CountStories(t, app.hu22BacklogApp, app.fx.projectA); n != 4 {
		t.Errorf("historias del proyecto = %d; se esperaban 4", n)
	}
}

// ---- Bloque C: validaciones (T062–T065)

// T062
func TestHU13_TituloODescripcionVacios(t *testing.T) {
	cases := []struct {
		field, value, message string
	}{
		{BacklogFieldTitle, "", backlog.ErrTitleRequired.Error()},
		{BacklogFieldDescription, "   ", backlog.ErrDescriptionRequired.Error()},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			app := newHU13EditApp(t)
			before := app.stored(t, "Login")
			values := editValues("Título escrito")
			values.Set(tc.field, tc.value)
			resp, body := app.post(t, app.client, app.editPath("Login"), values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST = %d; se esperaba 422", resp.StatusCode)
			}
			if !strings.Contains(body, `id="`+tc.field+`-error"`) || !strings.Contains(body, tc.message) {
				t.Errorf("no aparece %q junto a %s", tc.message, tc.field)
			}
			if !strings.Contains(body, `value="Acepta email"`) || !strings.Contains(body, `name="estado" value="Pendiente"`) {
				t.Error("no se conservaron los datos enviados o el filtro oculto")
			}
			if tc.field == BacklogFieldDescription && !strings.Contains(body, `value="Título escrito"`) {
				t.Error("no se conservó el título escrito")
			}
			app.assertUnchanged(t, "Login", before)
		})
	}
}

// T063
func TestHU13_SinCriterios(t *testing.T) {
	app := newHU13EditApp(t)
	before := app.stored(t, "Login")
	values := editValues("Login")
	values[BacklogFieldCriterion] = []string{"", "   "}
	resp, body := app.post(t, app.client, app.editPath("Login"), values)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("POST = %d; se esperaba 422", resp.StatusCode)
	}
	if !strings.Contains(body, `id="criterio-error"`) || !strings.Contains(body, backlog.ErrAcceptanceCriteriaRequired.Error()) {
		t.Error("no aparece el mensaje de criterios junto a los criterios")
	}
	app.assertUnchanged(t, "Login", before)
}

// T064
func TestHU13_PrioridadInvalida(t *testing.T) {
	app := newHU13EditApp(t)
	before := app.stored(t, "Login")
	values := editValues("Login")
	values.Set(BacklogFieldPriority, "Urgente")
	resp, body := app.post(t, app.client, app.editPath("Login"), values)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("POST = %d; se esperaba 422", resp.StatusCode)
	}
	if !strings.Contains(body, `id="prioridad-error"`) || !strings.Contains(body, backlog.ErrInvalidPriority.Error()) {
		t.Error("no aparece el mensaje de prioridad junto a la prioridad")
	}
	if !strings.Contains(body, `<option value="Urgente" selected>`) {
		t.Error("la prioridad enviada no quedó seleccionada")
	}
	app.assertUnchanged(t, "Login", before)
}

// T065
func TestHU13_StoryPointsInvalidos(t *testing.T) {
	for _, points := range []string{"4", "abc"} {
		t.Run(points, func(t *testing.T) {
			app := newHU13EditApp(t)
			before := app.stored(t, "Login")
			values := editValues("Login")
			values.Set(BacklogFieldStoryPoints, points)
			resp, body := app.post(t, app.client, app.editPath("Login"), values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST = %d; se esperaba 422", resp.StatusCode)
			}
			if !strings.Contains(body, `id="story_points-error"`) || !strings.Contains(body, MsgInvalidStoryPoints) {
				t.Errorf("no aparece %q junto a los Story Points", MsgInvalidStoryPoints)
			}
			if !strings.Contains(body, `<option value="`+points+`" selected>`) {
				t.Error("el valor enviado no quedó seleccionado")
			}
			app.assertUnchanged(t, "Login", before)
		})
	}
}
