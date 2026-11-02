package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// MEJORA-01, Escenario 7 (specs/mejora-01-diseno/inicio.md): pantalla de Inicio con bienvenida,
// "Cómo se trabaja" y las secciones agrupadas en Planificar, Ejecutar y Medir.

func homeMenu(paths ...string) []MenuItem {
	names := map[string]string{
		"/": "Inicio", "/proyectos": "Proyectos", "/backlog": "Backlog", "/estimaciones": "Estimaciones",
		"/planning-poker": "Planning Poker", "/sprints": "Sprints", "/tablero": "Tablero",
		"/esfuerzo": "Registro de esfuerzo", "/defectos": "Defectos", "/cierre": "Cierre de Sprint",
		"/dashboard": "Dashboard", "/estado": "Estado del proyecto", "/metricas": "Métricas del Sprint",
		"/calidad": "Calidad y avance", "/sprints-anteriores": "Sprints anteriores", "/reportes": "Reportes",
		"/libre": "Libre",
	}
	menu := make([]MenuItem, 0, len(paths))
	for _, p := range paths {
		menu = append(menu, MenuItem{Name: names[p], Path: p})
	}
	return menu
}

func homeGroupPaths(groups []homeGroup) map[string][]string {
	got := map[string][]string{}
	for _, g := range groups {
		for _, it := range g.Items {
			got[g.Title] = append(got[g.Title], it.Path)
		}
	}
	return got
}

// R1 y R3: solo las secciones registradas, en el orden de cada grupo, sin Inicio y sin grupos vacíos.
func TestHomeGroups_AgrupaSoloLasSeccionesRegistradas(t *testing.T) {
	all := homeMenu("/", "/backlog", "/estimaciones", "/proyectos", "/tablero", "/planning-poker", "/esfuerzo",
		"/calidad", "/defectos", "/sprints", "/estado", "/cierre", "/metricas", "/sprints-anteriores",
		"/dashboard", "/reportes")
	groups := homeGroups(all)

	var titles []string
	for _, g := range groups {
		titles = append(titles, g.Title)
		if g.Subtitle == "" {
			t.Errorf("el grupo %q debe tener subtítulo", g.Title)
		}
	}
	if strings.Join(titles, ",") != "Planificar,Ejecutar,Medir" {
		t.Fatalf("grupos = %v; se esperaba Planificar, Ejecutar y Medir", titles)
	}
	want := map[string][]string{
		"Planificar": {"/proyectos", "/backlog", "/estimaciones", "/planning-poker", "/sprints"},
		"Ejecutar":   {"/tablero", "/esfuerzo", "/defectos", "/cierre"},
		"Medir":      {"/dashboard", "/estado", "/metricas", "/calidad", "/sprints-anteriores", "/reportes"},
	}
	got := homeGroupPaths(groups)
	for title, paths := range want {
		if strings.Join(got[title], ",") != strings.Join(paths, ",") {
			t.Errorf("%s = %v; se esperaba %v", title, got[title], paths)
		}
	}
	for _, g := range groups {
		for _, it := range g.Items {
			if it.Path == "/" {
				t.Error("Inicio no debe aparecer entre las secciones")
			}
			if it.Name == "" || it.Description == "" || it.Icon == "" {
				t.Errorf("la sección %q debe tener nombre, descripción e ícono: %+v", it.Path, it)
			}
		}
	}

	parcial := homeGroups(homeMenu("/", "/tablero", "/dashboard"))
	if got := homeGroupPaths(parcial); len(parcial) != 2 || len(got["Planificar"]) != 0 ||
		strings.Join(got["Ejecutar"], ",") != "/tablero" || strings.Join(got["Medir"], ",") != "/dashboard" {
		t.Errorf("con solo Tablero y Dashboard registrados se esperaban Ejecutar y Medir; se obtuvo %v", got)
	}
}

// R2: una sección registrada sin grupo no se pierde.
func TestHomeGroups_SeccionSinGrupoVaAOtras(t *testing.T) {
	groups := homeGroups(homeMenu("/", "/backlog", "/libre"))
	got := homeGroupPaths(groups)
	if strings.Join(got["Otras secciones"], ",") != "/libre" {
		t.Errorf("la sección /libre debía ir a \"Otras secciones\"; se obtuvo %v", got)
	}
	if last := groups[len(groups)-1]; last.Title != "Otras secciones" || last.Items[0].Name != "Libre" || !homeIconNames[last.Items[0].Icon] {
		t.Errorf("\"Otras secciones\" debe ir al final, con el nombre del menú y un ícono: %+v", last)
	}
}

// R4: sin secciones (solo Inicio) no hay grupos.
func TestHomeGroups_SinSecciones(t *testing.T) {
	if groups := homeGroups(homeMenu("/")); len(groups) != 0 {
		t.Errorf("sin secciones no debe haber grupos; se obtuvo %+v", groups)
	}
	if groups := homeGroups(nil); len(groups) != 0 {
		t.Errorf("con el menú vacío no debe haber grupos; se obtuvo %+v", groups)
	}
}

// R5: cada sección tiene un ícono conocido; el SVG está en la plantilla (no se marca HTML como seguro
// desde el código, regla de TestEscape_NingunCodigoMarcaDatosComoSeguros).
func TestHomeGroups_IconosSinRecursosExternos(t *testing.T) {
	for _, g := range homeGroups(homeMenu("/", "/proyectos", "/backlog", "/estimaciones", "/planning-poker",
		"/sprints", "/tablero", "/esfuerzo", "/defectos", "/cierre", "/dashboard", "/estado", "/metricas",
		"/calidad", "/sprints-anteriores", "/reportes", "/libre")) {
		for _, it := range g.Items {
			if !homeIconNames[it.Icon] {
				t.Errorf("%s: ícono %q desconocido", it.Path, it.Icon)
			}
		}
	}

	body := homeBody(t, homeServer(t, "/backlog", "/tablero", "/dashboard", "/libre"), nil)
	icons := regexp.MustCompile(`(?s)<svg class="home-icon"[^>]*>.*?</svg>`).FindAllString(body, -1)
	if len(icons) != 4 {
		t.Fatalf("se esperaba un ícono por cada una de las 4 secciones; hay %d", len(icons))
	}
	external := regexp.MustCompile(`(?i)(href|src|url\()|https?:`)
	for _, icon := range icons {
		if !strings.Contains(icon, `aria-hidden="true"`) || !strings.Contains(icon, "<path") && !strings.Contains(icon, "<rect") && !strings.Contains(icon, "<circle") {
			t.Errorf("el ícono debe ser un SVG decorativo con aria-hidden=\"true\": %s", icon)
		}
		if external.MatchString(icon) {
			t.Errorf("el ícono no debe referenciar recursos: %s", icon)
		}
	}
}

func homeServer(t *testing.T, paths ...string) http.Handler {
	t.Helper()
	s := NewServer(quietLogger())
	for _, it := range homeMenu(paths...) {
		page := func(w http.ResponseWriter, r *http.Request) {
			s.Renderer().Page(w, r, "home.html", PageData{Title: "Sección"})
		}
		if err := s.Register(Section{Name: it.Name, Path: it.Path}, Routes{"GET " + it.Path: http.HandlerFunc(page)}); err != nil {
			t.Fatalf("no se pudo registrar %s: %v", it.Path, err)
		}
	}
	return s.Handler()
}

func homeBody(t *testing.T, h http.Handler, ctx context.Context) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d; se esperaba 200", rec.Code)
	}
	return rec.Body.String()
}

// Escenario 7: bienvenida, accesos, "Cómo se trabaja" en orden, grupos con enlaces y pie.
func TestHomePage_BienvenidaPasosYAccesos(t *testing.T) {
	body := homeBody(t, homeServer(t, "/backlog", "/tablero", "/dashboard", "/libre"), nil)

	for _, want := range []string{
		`class="home-hero"`,
		"Scrum · SDD · BDD · TDD",
		"<h1>Bienvenido a Software Metrics &amp; Estimation</h1>",
		`<a class="button" href="/contexto/proyecto">Elegir proyecto</a>`,
		`<a class="button button-secondary" href="/proyectos">Crear un proyecto</a>`,
		"<h2>Cómo se trabaja</h2>",
		"<h2>Secciones</h2>",
		"UTN FRSR · Ingeniería y Calidad de Software · 2026",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("el Inicio debe contener %q", want)
		}
	}

	steps := []string{"Proyecto", "Backlog", "Sprint", "Trabajo diario", "Cierre y métricas"}
	last := -1
	for i, step := range steps {
		marker := `<span class="home-step-number" aria-hidden="true">` + string(rune('1'+i)) + `</span>`
		idx := strings.Index(body, marker)
		title := strings.Index(body, "<h3>"+step+"</h3>")
		if idx < 0 || title < idx || idx < last {
			t.Errorf("el paso %d (%q) debe aparecer en orden, con su número", i+1, step)
		}
		last = title
	}

	order := []int{}
	for _, group := range []string{"Planificar", "Ejecutar", "Medir", "Otras secciones"} {
		idx := strings.Index(body, "<h3>"+group+"</h3>")
		if idx < 0 {
			t.Errorf("debe verse el grupo %q", group)
		}
		order = append(order, idx)
	}
	for i := 1; i < len(order); i++ {
		if order[i] < order[i-1] {
			t.Errorf("los grupos deben verse en el orden Planificar, Ejecutar, Medir y Otras secciones: %v", order)
		}
	}
	for _, link := range []string{`href="/backlog"`, `href="/tablero"`, `href="/dashboard"`, `href="/libre"`} {
		if strings.Count(body, link) < 2 { // menú + tarjeta del Inicio
			t.Errorf("la sección %s debe tener su tarjeta en el Inicio además del menú", link)
		}
	}
}

// Con un proyecto activo, el acceso principal ofrece cambiarlo.
func TestHomePage_ConProyectoActivoOfreceCambiarlo(t *testing.T) {
	ctx := WithActiveContext(context.Background(), &ActiveContext{ProjectID: 7, ProjectName: "Alfa"})
	body := homeBody(t, homeServer(t, "/backlog"), ctx)
	if !strings.Contains(body, `<a class="button" href="/contexto/proyecto">Cambiar proyecto</a>`) {
		t.Error("con proyecto activo el acceso principal debe decir \"Cambiar proyecto\"")
	}
	if strings.Contains(body, `<a class="button" href="/contexto/proyecto">Elegir proyecto</a>`) {
		t.Error("con proyecto activo no debe ofrecerse \"Elegir proyecto\" como acceso principal")
	}
}

// R4: sin secciones se mantiene el mensaje anterior.
func TestHomePage_SinSeccionesMantieneElMensaje(t *testing.T) {
	body := homeBody(t, NewServer(quietLogger()).Handler(), nil)
	if !strings.Contains(body, "Todavía no hay secciones disponibles.") {
		t.Error("sin secciones debe verse \"Todavía no hay secciones disponibles.\"")
	}
	if strings.Contains(body, "<h2>Secciones</h2>") {
		t.Error("sin secciones no debe verse el título Secciones")
	}
}
