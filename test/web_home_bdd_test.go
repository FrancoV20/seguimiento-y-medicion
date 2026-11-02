package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// MEJORA-01, Escenario 7 (specs/mejora-01-diseno/inicio.md), con el servidor real y sus secciones.

// Escenario 7: Inicio con secciones bien diferenciadas.
func TestWebHomeBDD_InicioConSeccionesDiferenciadas(t *testing.T) {
	// Dado la aplicación con sus secciones registradas
	s := web.NewServer(silencioso())
	secciones := []web.Section{
		{Name: "Proyectos", Path: "/proyectos"},
		{Name: "Tablero", Path: "/tablero"},
		{Name: "Métricas del Sprint", Path: "/metricas"},
	}
	for _, sec := range secciones {
		page := func(w http.ResponseWriter, r *http.Request) {
			s.Renderer().Page(w, r, "home.html", web.PageData{Title: "Sección"})
		}
		if err := s.Register(sec, web.Routes{"GET " + sec.Path: http.HandlerFunc(page)}); err != nil {
			t.Fatalf("no se pudo registrar %s: %v", sec.Path, err)
		}
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// Cuando abro el Inicio
	body := leer(t, srv.URL+"/")

	// Entonces veo la bienvenida, los accesos, "Cómo se trabaja" y las secciones agrupadas
	for _, esperado := range []string{
		"Bienvenido a Software Metrics &amp; Estimation",
		"Scrum · SDD · BDD · TDD",
		">Elegir proyecto</a>",
		">Crear un proyecto</a>",
		"Cómo se trabaja",
		"Trabajo diario",
		"Cierre y métricas",
		"<h3>Planificar</h3>", "<h3>Ejecutar</h3>", "<h3>Medir</h3>",
		"UTN FRSR · Ingeniería y Calidad de Software · 2026",
	} {
		if !strings.Contains(body, esperado) {
			t.Errorf("el Inicio debía mostrar %q", esperado)
		}
	}
	planificar := strings.Index(body, "<h3>Planificar</h3>")
	ejecutar := strings.Index(body, "<h3>Ejecutar</h3>")
	medir := strings.Index(body, "<h3>Medir</h3>")
	for _, c := range []struct {
		path     string
		from, to int
	}{
		{`href="/proyectos"`, planificar, ejecutar},
		{`href="/tablero"`, ejecutar, medir},
		{`href="/metricas"`, medir, len(body)},
	} {
		if c.from < 0 || c.to < 0 {
			t.Errorf("la sección %s debía estar en su grupo, pero el grupo no se ve", c.path)
			continue
		}
		if i := strings.Index(body[c.from:], c.path); i < 0 || c.from+i > c.to {
			t.Errorf("la sección %s debía estar en su grupo", c.path)
		}
	}
}
