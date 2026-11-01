package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// HU-13, T031: servidor de prueba del Backlog con historias de distintas prioridades y estados.
//
//	Proyecto A: «Login» (Media, la primera, Pendiente), «Pagos» (Alta, Hecho),
//	            «Reportes» (Baja, En Sprint) y «Perfil» (Media, después de «Login», Pendiente).
//	Proyecto B: «Facturas» (Media, Pendiente).
type hu13BacklogApp struct {
	*hu22BacklogApp
	stories map[string]int64
}

func newHU13BacklogApp(t *testing.T) *hu13BacklogApp {
	return newHU13BacklogAppWithStore(t, func(app *ctxApp) backlog.BacklogStore {
		return backlog.NewPostgreSQLBacklogStore(app.db)
	})
}

func newHU13BacklogAppWithStore(t *testing.T, storeFactory func(*ctxApp) backlog.BacklogStore) *hu13BacklogApp {
	t.Helper()
	app := &hu13BacklogApp{
		hu22BacklogApp: newHU22BacklogAppWithDependencies(t, storeFactory, func(app *ctxApp) StoryCatalog {
			return NewPostgreSQLStoryCatalog(app.db)
		}),
		stories: map[string]int64{},
	}
	for _, s := range []struct {
		project           int64
		title, priority   string
		status, createdAt string
		points            any
		hours             any
		criteria          []string
	}{
		{app.fx.projectA, "Login", "Media", "Pendiente", "2026-09-01 10:00:00+00", 5, 4, []string{"Acepta email", "Rechaza clave vacía"}},
		{app.fx.projectA, "Pagos", "Alta", "Hecho", "2026-09-02 10:00:00+00", 3, nil, []string{"Cobra con tarjeta"}},
		{app.fx.projectA, "Reportes", "Baja", "En Sprint", "2026-09-03 10:00:00+00", nil, nil, []string{"Exporta a PDF"}},
		{app.fx.projectA, "Perfil", "Media", "Pendiente", "2026-09-04 10:00:00+00", 2, nil, []string{"Cambia la foto"}},
		{app.fx.projectB, "Facturas", "Media", "Pendiente", "2026-09-01 10:00:00+00", 8, nil, []string{"Emite la factura"}},
	} {
		var id int64
		if err := app.db.QueryRow(`INSERT INTO user_stories
			(project_id, title, description, priority, status, story_points, estimated_hours, created_at)
			VALUES ($1, $2, 'Descripción de '||$2, $3, $4, $5, $6, $7::timestamptz) RETURNING id`,
			s.project, s.title, s.priority, s.status, s.points, s.hours, s.createdAt).Scan(&id); err != nil {
			t.Fatalf("create HU-13 story %q: %v", s.title, err)
		}
		for i, c := range s.criteria {
			if _, err := app.db.Exec(`INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, $2, $3)`,
				id, i+1, c); err != nil {
				t.Fatalf("create HU-13 criterion: %v", err)
			}
		}
		app.stories[s.title] = id
	}
	app.chooseProject(t, app.fx.projectA)
	return app
}

// tableTitles devuelve los títulos de la tabla del Backlog en el orden en que aparecen.
func (a *hu13BacklogApp) tableTitles(t *testing.T, body string) []string {
	t.Helper()
	if !strings.Contains(body, "<tbody>") {
		return nil
	}
	rows := extractHU22TableRows(t, body)
	type found struct {
		title string
		at    int
	}
	var got []found
	for title := range a.stories {
		if i := strings.Index(rows, "<td>"+title+"</td>"); i >= 0 {
			got = append(got, found{title, i})
		}
	}
	for i := 1; i < len(got); i++ {
		for j := i; j > 0 && got[j].at < got[j-1].at; j-- {
			got[j], got[j-1] = got[j-1], got[j]
		}
	}
	titles := make([]string, len(got))
	for i, f := range got {
		titles[i] = f.title
	}
	return titles
}

func assertTitles(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ", ") != strings.Join(want, ", ") {
		t.Errorf("historias = [%s]; se esperaba [%s]", strings.Join(got, ", "), strings.Join(want, ", "))
	}
}

// T032
func TestHU13_ListadoOrdenado(t *testing.T) {
	app := newHU13BacklogApp(t)
	resp, body := app.get(t, app.client, "/backlog")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /backlog = %d; se esperaba 200", resp.StatusCode)
	}
	assertTitles(t, app.tableTitles(t, body), "Pagos", "Login", "Perfil", "Reportes")
}

// T033
func TestHU13_FiltroPorEstado(t *testing.T) {
	app := newHU13BacklogApp(t)

	resp, body := app.get(t, app.client, "/backlog?estado=Pendiente")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /backlog?estado=Pendiente = %d; se esperaba 200", resp.StatusCode)
	}
	assertTitles(t, app.tableTitles(t, body), "Login", "Perfil")
	if !strings.Contains(body, `<option value="Pendiente" selected>`) {
		t.Error("el filtro no muestra «Pendiente» seleccionado")
	}
	if !strings.Contains(body, `<form method="get" action="/backlog"`) || !strings.Contains(body, `name="estado"`) ||
		!strings.Contains(body, ">Filtrar</button>") {
		t.Error("falta el formulario GET del filtro con el selector «estado» y el botón «Filtrar»")
	}
	for _, option := range []string{"Todos", "Pendiente", "En Sprint", "En progreso", "Hecho"} {
		if !strings.Contains(body, `<option value="`+option+`"`) {
			t.Errorf("el filtro no ofrece la opción %q", option)
		}
	}

	for _, path := range []string{"/backlog?estado=Todos", "/backlog"} {
		_, body := app.get(t, app.client, path)
		assertTitles(t, app.tableTitles(t, body), "Pagos", "Login", "Perfil", "Reportes")
	}
}

// T034
func TestHU13_FiltroSinResultados(t *testing.T) {
	app := newHU13BacklogApp(t)
	if _, err := app.db.Exec(`UPDATE user_stories SET status = 'En progreso' WHERE id = $1`, app.stories["Pagos"]); err != nil {
		t.Fatal(err)
	}
	resp, body := app.get(t, app.client, "/backlog?estado=Hecho")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /backlog?estado=Hecho = %d; se esperaba 200", resp.StatusCode)
	}
	if !strings.Contains(body, MsgNoStoriesWithStatus) {
		t.Errorf("no aparece %q", MsgNoStoriesWithStatus)
	}
	if strings.Contains(body, TitleInternal) || strings.Contains(body, "<tbody>") {
		t.Error("el filtro sin resultados mostró un error o una tabla")
	}
}

// T035
func TestHU13_FiltroEstadoInvalido(t *testing.T) {
	app := newHU13BacklogApp(t)
	for _, path := range []string{"/backlog?estado=Archivada", "/backlog?estado=pendiente"} {
		resp, body := app.get(t, app.client, path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d; se esperaba 200", path, resp.StatusCode)
		}
		if !strings.Contains(body, `id="estado-error"`) || !strings.Contains(body, MsgInvalidStatusFilter) {
			t.Errorf("%s: no aparece %q junto al filtro", path, MsgInvalidStatusFilter)
		}
		assertTitles(t, app.tableTitles(t, body), "Pagos", "Login", "Perfil", "Reportes")
	}

	_, body := app.get(t, app.client, "/backlog?estado=Hecho&estado=Pendiente")
	assertTitles(t, app.tableTitles(t, body), "Pagos")
}

// T036
func TestHU13_BacklogVacioConFiltro(t *testing.T) {
	app := newHU13BacklogApp(t)
	if _, err := app.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, app.fx.projectA); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/backlog", "/backlog?estado=Hecho"} {
		resp, body := app.get(t, app.client, path)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d; se esperaba 200", path, resp.StatusCode)
		}
		if strings.Count(body, "Aún no hay historias") != 1 || !strings.Contains(body, `action="/backlog"`) {
			t.Errorf("%s: falta «Aún no hay historias» o el formulario de alta", path)
		}
		if strings.Contains(body, MsgNoStoriesWithStatus) {
			t.Errorf("%s: un Backlog vacío no debe mostrar %q", path, MsgNoStoriesWithStatus)
		}
	}
}
