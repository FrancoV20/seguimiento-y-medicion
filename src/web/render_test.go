package web

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
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
