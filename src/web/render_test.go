package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

// T025: estructura común, contexto vacío, UTF-8 y caracteres especiales (FR-008, FR-011).

func TestPage_InicioMuestraSinProyectoSeleccionadoYUTF8(t *testing.T) {
	rec := httptest.NewRecorder()

	NewServer(quietLogger()).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type inesperado: %q", ct)
	}
	body := rec.Body.String()
	for _, esperado := range []string{`<meta charset="utf-8">`, MsgNoProjectSelected, `<nav`, `aria-current="page"`, `<main`} {
		if !strings.Contains(body, esperado) {
			t.Errorf("la página debe contener %q", esperado)
		}
	}
}

func TestPage_MuestraElContextoActivo(t *testing.T) {
	s := NewServer(quietLogger())
	ctx := WithActiveContext(t.Context(), &ActiveContext{ProjectID: 3, ProjectName: "Medición Ñandú", MemberID: 2, MemberName: "Giuliana"})
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	s.Renderer().Page(rec, req, "home.html", PageData{Title: "Inicio"})

	body := rec.Body.String()
	if !strings.Contains(body, "Medición Ñandú") || !strings.Contains(body, "Giuliana") {
		t.Error("el espacio de contexto debe mostrar el proyecto y el integrante activos")
	}
	if strings.Contains(body, MsgNoProjectSelected) {
		t.Error("con contexto activo no debe decir 'Sin proyecto seleccionado'")
	}
}

func TestPage_TildesEnieYSimbolosSeVenCorrectamente(t *testing.T) {
	s := NewServer(quietLogger())
	rec := httptest.NewRecorder()
	texto := "Añadir canción — ¿Qué tal? ¡Sí! € 50 % «ok» ✓"

	s.Renderer().Page(rec, httptest.NewRequest(http.MethodGet, "/", nil), "home.html", PageData{Title: texto})

	if !strings.Contains(rec.Body.String(), "<title>"+texto) {
		t.Errorf("el texto con caracteres especiales debe verse tal cual; cuerpo:\n%s", rec.Body.String())
	}
}

// T026: sin recursos de otros orígenes y estilos servidos localmente (FR-010, SC-006).

var recursoExterno = regexp.MustCompile(`(?i)(src|href|action)\s*=\s*["']?\s*(https?:)?//`)

func TestPage_NoReferenciaOtrosOrigenes(t *testing.T) {
	rec := httptest.NewRecorder()
	NewServer(quietLogger()).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	if m := recursoExterno.FindString(body); m != "" {
		t.Errorf("la página referencia un recurso externo: %q", m)
	}
	if !strings.Contains(body, `href="/static/app.css"`) {
		t.Error("la página debe usar la hoja de estilos local /static/app.css")
	}
}

func TestStatic_SirveLosEstilosLocales(t *testing.T) {
	h := NewServer(quietLogger()).Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.css", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("se esperaba el CSS con 200 y text/css, se obtuvo %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), ":root") || recursoCSSExterno.MatchString(rec.Body.String()) {
		t.Error("el CSS debe ser el local y no importar recursos externos")
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("no debe listarse el contenido de /static/, se obtuvo %d", rec.Code)
	}
}

var recursoCSSExterno = regexp.MustCompile(`(?i)(@import|url\()\s*["']?\s*(https?:)?//`)

// T032: una plantilla que falla a mitad del render da la página de error completa (FR-017).

func TestPage_FalloAMitadDelRenderMuestraPaginaDeErrorCompleta(t *testing.T) {
	archivo := func(name string) *fstest.MapFile {
		b, err := fs.ReadFile(templateFiles, "templates/"+name)
		if err != nil {
			t.Fatalf("no se pudo leer la plantilla %s: %v", name, err)
		}
		return &fstest.MapFile{Data: b}
	}
	rd := NewRenderer(NewRegistry(), quietLogger())
	rd.files = fstest.MapFS{
		"templates/layout.html": archivo("layout.html"),
		"templates/error.html":  archivo("error.html"),
		"templates/rota.html":   {Data: []byte(`{{define "content"}}<p>contenido parcial</p>{{.Data.NoExiste}}{{end}}`)},
	}
	rec := httptest.NewRecorder()

	rd.Page(rec, httptest.NewRequest(http.MethodGet, "/", nil), "rota.html", PageData{Title: "Rota", Data: struct{}{}})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("se esperaba 500, se obtuvo %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "contenido parcial") {
		t.Error("no debe enviarse el HTML que se alcanzó a generar antes del fallo")
	}
	if !strings.Contains(body, TitleInternal) || !strings.Contains(body, "</html>") {
		t.Error("debe enviarse la página de error completa con la estructura común")
	}
}
