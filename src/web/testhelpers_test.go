package web

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

const formsSuccessText = "Guardado correctamente"

const (
	formsPageTemplate   = `{{define "content"}}<h1>{{.Title}}</h1>{{end}}`
	formsResultTemplate = formsPageTemplate
	formsFormTemplate   = `{{define "content"}}<h1>{{.Title}}</h1>
<form method="post" action="/prueba/form">
{{template "field" (.Form.Field "nombre" "Nombre" "text")}}
{{template "field" (.Form.Field "detalle" "Detalle" "text")}}
<label><input type="checkbox" name="falla" value="1"> Simular falla</label>
<button type="submit">Guardar</button>
</form>{{end}}`
)

// formsApp es el servidor real con tres secciones de prueba (sin requisitos, con
// RequiresProject y con RequiresMember) y un formulario de prueba (HU-20b, T005).
type formsApp struct {
	srv   *Server
	ts    *httptest.Server
	saved atomic.Int32
}

// newFormsApp arma y arranca el servidor; setup permite agregar middleware antes de arrancar.
func newFormsApp(t *testing.T, setup ...func(*Server)) *formsApp {
	t.Helper()
	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.renderer.files = formsTemplates(t)
	a := &formsApp{srv: srv}
	a.registerSections(t)
	for _, f := range setup {
		f(srv)
	}
	a.ts = httptest.NewServer(srv.Handler())
	t.Cleanup(a.ts.Close)
	return a
}

// formsTemplates copia las plantillas reales y suma las de prueba.
func formsTemplates(t *testing.T) fs.FS {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(templateFiles, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(templateFiles, p)
		if err != nil {
			return err
		}
		m[p] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatalf("no se pudieron leer las plantillas: %v", err)
	}
	m["templates/test_page.html"] = &fstest.MapFile{Data: []byte(formsPageTemplate)}
	m["templates/test_form.html"] = &fstest.MapFile{Data: []byte(formsFormTemplate)}
	return m
}

func (a *formsApp) registerSections(t *testing.T) {
	t.Helper()
	page := func(title string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			a.srv.renderer.Page(w, r, "test_page.html", PageData{Title: title, Form: NewFormView()})
		}
	}
	register := func(s Section, routes Routes) {
		t.Helper()
		if err := a.srv.Register(s, routes); err != nil {
			t.Fatalf("no se pudo registrar la sección %q: %v", s.Name, err)
		}
	}
	register(Section{Name: "Libre", Path: "/libre"}, Routes{"GET /libre": page("Libre")})
	register(Section{Name: "Con proyecto", Path: "/con-proyecto", RequiresProject: true},
		Routes{"GET /con-proyecto": page("Con proyecto")})
	register(Section{Name: "Con integrante", Path: "/con-integrante", RequiresMember: true},
		Routes{"GET /con-integrante": page("Con integrante")})
	register(Section{Name: "Prueba", Path: "/prueba"}, Routes{
		"GET /prueba/form":      http.HandlerFunc(a.formGet),
		"POST /prueba/form":     http.HandlerFunc(a.formPost),
		"GET /prueba/resultado": page("Resultado"),
		"GET /prueba/alerta":    http.HandlerFunc(a.alertGet),
	})
}

func (a *formsApp) formGet(w http.ResponseWriter, r *http.Request) {
	a.srv.renderer.Page(w, r, "test_form.html", PageData{Title: "Formulario de prueba", Form: NewFormView()})
}

func (a *formsApp) formPost(w http.ResponseWriter, r *http.Request) {
	form := FormFromRequest(r, "nombre", "detalle")
	if strings.TrimSpace(form.Values["nombre"]) == "" {
		form.AddError("nombre", "El nombre es obligatorio")
	}
	if form.HasErrors() {
		a.srv.renderer.PageStatus(w, r, http.StatusUnprocessableEntity, "test_form.html",
			PageData{Title: "Formulario de prueba", Form: form})
		return
	}
	if r.PostFormValue("falla") == "1" {
		a.srv.renderer.PageStatus(w, r, http.StatusInternalServerError, "test_form.html",
			PageData{Title: "Formulario de prueba", Form: form, Flash: &Flash{Kind: FlashError, Text: MsgOperationFailed}})
		return
	}
	a.saved.Add(1)
	SetFlash(w, r, FlashSuccess, formsSuccessText)
	RedirectSeeOther(w, r, "/prueba/resultado")
}

// alertGet muestra el mensaje pedido por la consulta, en la página común o en la del formulario.
func (a *formsApp) alertGet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tmpl := "test_page.html"
	if q.Get("pagina") == "form" {
		tmpl = "test_form.html"
	}
	a.srv.renderer.Page(w, r, tmpl, PageData{
		Title: "Alerta",
		Form:  NewFormView(),
		Flash: &Flash{Kind: FlashKind(q.Get("tipo")), Text: q.Get("texto")},
	})
}

// client devuelve un navegador simulado con cookies; si follow es false, no sigue redirecciones.
func (a *formsApp) client(follow bool) *http.Client {
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	if !follow {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return c
}

func formsReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return string(b)
}

func (a *formsApp) get(t *testing.T, c *http.Client, path string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *formsApp) post(t *testing.T, c *http.Client, path string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.ts.URL+path, v)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}
