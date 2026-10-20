package web

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// T023: el menú se arma solo con las secciones registradas (FR-009).

func registroDePrueba(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	for _, s := range []Section{{Name: "Proyectos", Path: "/proyectos"}, {Name: "Backlog", Path: "/backlog"}} {
		if err := r.Add(s); err != nil {
			t.Fatalf("no se pudo registrar %q: %v", s.Name, err)
		}
	}
	return r
}

func nombres(items []MenuItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func activos(items []MenuItem) []string {
	var out []string
	for _, it := range items {
		if it.Active {
			out = append(out, it.Name)
		}
	}
	return out
}

func TestMenu_SoloSeccionesRegistradasConInicioPrimero(t *testing.T) {
	menu := registroDePrueba(t).Menu("/")

	got := strings.Join(nombres(menu), ",")
	if got != "Inicio,Proyectos,Backlog" {
		t.Errorf("menú inesperado: %s", got)
	}
}

func TestMenu_ResaltaLaSeccionActual(t *testing.T) {
	reg := registroDePrueba(t)
	casos := map[string]string{
		"/":              "Inicio",
		"/proyectos":     "Proyectos",
		"/proyectos/5":   "Proyectos",
		"/backlog/nuevo": "Backlog",
		"/proyectosx":    "",
	}
	for path, esperado := range casos {
		got := strings.Join(activos(reg.Menu(path)), ",")
		if got != esperado {
			t.Errorf("%s: se esperaba resaltada %q, se obtuvo %q", path, esperado, got)
		}
	}
}

func TestServer_NuevaSeccionApareceEnElMenuSinTocarElLayout(t *testing.T) {
	s := NewServer(quietLogger())
	err := s.Register(Section{Name: "Sprints", Path: "/sprints"}, Routes{
		"GET /sprints": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Renderer().Page(w, r, "home.html", PageData{Title: "Sprints"})
		}),
	})
	if err != nil {
		t.Fatalf("no se pudo registrar la sección: %v", err)
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	inicio := leerPagina(t, srv.URL+"/")
	sprints := leerPagina(t, srv.URL+"/sprints")

	if !strings.Contains(inicio, `href="/sprints"`) || !strings.Contains(inicio, ">Sprints<") {
		t.Error("la sección registrada debe aparecer en el menú de la página de inicio")
	}
	if !strings.Contains(sprints, `href="/sprints" class="active" aria-current="page"`) {
		t.Error("en su propia página, la sección debe aparecer resaltada")
	}
}

// T024: dos secciones con la misma dirección hacen fallar el arranque.

func TestRegistry_DireccionDuplicadaFalla(t *testing.T) {
	reg := registroDePrueba(t)

	err := reg.Add(Section{Name: "Otra", Path: "/proyectos"})

	var dup *DuplicateSectionError
	if !errors.As(err, &dup) || !strings.Contains(err.Error(), "/proyectos") {
		t.Errorf("se esperaba DuplicateSectionError que nombre /proyectos, se obtuvo: %v", err)
	}
	if len(reg.Sections()) != 2 {
		t.Error("una sección rechazada no debe agregarse al menú")
	}
}

func TestServer_RegisterConDireccionDuplicadaDevuelveError(t *testing.T) {
	s := NewServer(quietLogger())
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if err := s.Register(Section{Name: "Proyectos", Path: "/proyectos"}, Routes{"GET /proyectos": ok}); err != nil {
		t.Fatalf("el primer registro debía funcionar: %v", err)
	}

	err := s.Register(Section{Name: "Proyectos 2", Path: "/proyectos"}, Routes{"GET /proyectos/otra": ok})

	if err == nil || !strings.Contains(err.Error(), "/proyectos") {
		t.Errorf("se esperaba un error que nombre la dirección duplicada, se obtuvo: %v", err)
	}
}

func TestServer_RegisterConRutaEnConflictoDevuelveErrorSinPanico(t *testing.T) {
	s := NewServer(quietLogger())
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if err := s.Register(Section{Name: "A", Path: "/a"}, Routes{"GET /a": ok}); err != nil {
		t.Fatalf("el primer registro debía funcionar: %v", err)
	}

	err := s.Register(Section{Name: "B", Path: "/b"}, Routes{"GET /a": ok})

	if err == nil || !strings.Contains(err.Error(), "GET /a") {
		t.Errorf("se esperaba un error que nombre la ruta en conflicto, se obtuvo: %v", err)
	}
	if strings.Contains(strings.Join(nombres(s.registry.Menu("/")), ","), ",B") {
		t.Error("una sección cuyo registro falló no debe aparecer en el menú")
	}
}

func leerPagina(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}
