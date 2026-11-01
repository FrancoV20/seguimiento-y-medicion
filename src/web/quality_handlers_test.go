package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// Pruebas de pantalla de "Calidad y avance" (HU-10, T022 y T023). Usan el servidor real con
// PostgreSQL, el servicio y el catálogo reales, el proyecto activo elegido por el flujo de HU-20b y
// un navegador con cookies. Los textos se toman de spec.md.
//
// Contrato de la pantalla que verifican (T029): cada Sprint de la lista es un enlace
// <a href="/calidad?sprint=ID">Sprint N</a>, el elegido con aria-current="page"; los números se leen
// como texto, cada uno después de su etiqueta ("Story Points planificados", "Story Points
// completados", "Defectos detectados" y "Defectos resueltos").

const (
	hu10PathQuality   = "/calidad"
	hu10MsgNoSprints  = "Aún no hay Sprints finalizados"
	hu10MsgWarningOne = "1 historia sin Story Points no se suma a los planificados"
)

func hu10MsgActiveHint(number int) string {
	return fmt.Sprintf("El Sprint %d sigue en curso; sus métricas se calculan al cerrarlo", number)
}

type hu10QualityApp struct {
	*ctxApp
	client *http.Client
	stamp  int64
}

func newHU10QualityApp(t *testing.T) *hu10QualityApp {
	return newHU10QualityAppWith(t, nil, nil)
}

// newHU10QualityAppWith permite reemplazar el cálculo o el catálogo (por ejemplo, para simular una
// falla de lectura).
func newHU10QualityAppWith(t *testing.T, readerFor func(*sql.DB) QualityReader, catalogFor func(*sql.DB) QualityCatalog) *hu10QualityApp {
	t.Helper()
	app := newCtxApp(t, nil)
	var reader QualityReader = metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(app.db))
	if readerFor != nil {
		reader = readerFor(app.db)
	}
	var catalog QualityCatalog = NewPostgreSQLQualityCatalog(app.db)
	if catalogFor != nil {
		catalog = catalogFor(app.db)
	}
	if err := RegisterQuality(app.srv, reader, catalog); err != nil {
		t.Fatalf("no se pudo registrar la sección Calidad y avance: %v", err)
	}
	// Corre antes que la limpieza del fixture de contexto, que borra los proyectos. Orden: defectos,
	// registro de cierre, asignaciones, historias y Sprints.
	t.Cleanup(func() {
		for _, projectID := range []int64{app.fx.projectA, app.fx.projectB} {
			const sprints = `SELECT id FROM sprints WHERE project_id = $1`
			_, _ = app.db.Exec(`DELETE FROM defects WHERE project_id = $1 OR detection_sprint_id IN (`+sprints+`)`, projectID)
			_, _ = app.db.Exec(`DELETE FROM sprint_story_results WHERE sprint_id IN (`+sprints+`)`, projectID)
			_, _ = app.db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id IN (`+sprints+`)`, projectID)
			_, _ = app.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, projectID)
			_, _ = app.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, projectID)
		}
	})
	return &hu10QualityApp{ctxApp: app, client: app.client(false), stamp: time.Now().UnixNano()}
}

func (a *hu10QualityApp) chooseProject(t *testing.T, projectID int64) {
	t.Helper()
	resp, _ := a.selectProject(t, a.client, id64(projectID), hu10PathQuality)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != hu10PathQuality {
		t.Fatalf("elegir proyecto: estado/location = %d/%q; se esperaba 303 %s",
			resp.StatusCode, resp.Header.Get("Location"), hu10PathQuality)
	}
}

// page abre la pantalla; con raw distinto de nil agrega ?sprint=raw.
func (a *hu10QualityApp) page(t *testing.T, raw *string) (*http.Response, string) {
	t.Helper()
	path := hu10PathQuality
	if raw != nil {
		path += "?" + url.Values{"sprint": {*raw}}.Encode()
	}
	return a.get(t, a.client, path)
}

func (a *hu10QualityApp) pageOf(t *testing.T, sprintID int64) string {
	t.Helper()
	raw := id64(sprintID)
	resp, body := a.page(t, &raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s?sprint=%s: estado %d; se esperaba 200", hu10PathQuality, raw, resp.StatusCode)
	}
	return body
}

func (a *hu10QualityApp) sprint(t *testing.T, projectID int64, number int, status string) int64 {
	t.Helper()
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, 'Sprint HU-10', DATE '2026-10-01', DATE '2026-10-15', $3) RETURNING id`,
		projectID, number, status).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear el Sprint %d: %v", number, err)
	}
	return id
}

// closedStory crea una historia como la deja el cierre de HU-15: una fila en sprint_story_results
// con su estado final; las "Hecho" conservan la asignación y las demás vuelven al Backlog.
func (a *hu10QualityApp) closedStory(t *testing.T, projectID, sprintID int64, points *int, finalStatus string) int64 {
	t.Helper()
	status, active := "Pendiente", sql.NullInt64{}
	if finalStatus == "Hecho" {
		status, active = "Hecho", sql.NullInt64{Int64: sprintID, Valid: true}
	}
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
		VALUES ($1, $2, 'Historia de prueba HU-10', 'Media', $3, $4, $5) RETURNING id`,
		projectID, fmt.Sprintf("HU10 pantalla %d", a.stamp), points, status, active).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear la historia: %v", err)
	}
	if finalStatus == "Hecho" {
		if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id); err != nil {
			t.Fatalf("no se pudo asignar la historia: %v", err)
		}
	}
	if _, err := a.db.Exec(`
		INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		VALUES ($1, $2, $3, $4)`, sprintID, id, finalStatus, points); err != nil {
		t.Fatalf("no se pudo registrar el cierre de la historia: %v", err)
	}
	return id
}

// defect crea un defecto detectado en detection; con resolution distinto de 0 queda 'Resuelto' en él.
func (a *hu10QualityApp) defect(t *testing.T, projectID, storyID, detection, resolution int64) {
	t.Helper()
	status, resolved := "Abierto", sql.NullInt64{}
	if resolution != 0 {
		status, resolved = "Resuelto", sql.NullInt64{Int64: resolution, Valid: true}
	}
	if _, err := a.db.Exec(`
		INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		VALUES ($1, $2, 'Defecto de prueba HU-10', 'Media', $3, $4, $5)`,
		projectID, storyID, status, detection, resolved); err != nil {
		t.Fatalf("no se pudo crear el defecto: %v", err)
	}
}

// threeSprints crea los Sprints 1, 2 y 3 finalizados del proyecto, con números distintos en cada uno:
// Sprint 1 → 1 de 1 ("100,00 %"), Sprint 2 → 0 de 2 ("0,00 %") y Sprint 3 → 1 de 4 ("25,00 %").
func (a *hu10QualityApp) threeSprints(t *testing.T, projectID int64) (s1, s2, s3 int64) {
	t.Helper()
	sp := func(n int) *int { return &n }
	s1 = a.sprint(t, projectID, 1, "Finalizado")
	a.closedStory(t, projectID, s1, sp(5), "Hecho")
	s2 = a.sprint(t, projectID, 2, "Finalizado")
	a.closedStory(t, projectID, s2, sp(3), "En progreso")
	a.closedStory(t, projectID, s2, sp(2), "En Sprint")
	s3 = a.sprint(t, projectID, 3, "Finalizado")
	a.closedStory(t, projectID, s3, sp(8), "Hecho")
	a.closedStory(t, projectID, s3, sp(1), "En progreso")
	a.closedStory(t, projectID, s3, sp(1), "En progreso")
	a.closedStory(t, projectID, s3, sp(1), "En Sprint")
	return s1, s2, s3
}

var (
	hu10SprintLinkPattern = regexp.MustCompile(`<a\b[^>]*\bhref="/calidad\?sprint=(\d+)"[^>]*>`)
	hu10TagPattern        = regexp.MustCompile(`(?s)<[^>]*>`)
	hu10PercentPattern    = regexp.MustCompile(`\d+,\d\d %`)
)

// hu10SprintLinks devuelve los ids de la lista de Sprints, en el orden de la página.
func hu10SprintLinks(body string) []string {
	var ids []string
	for _, m := range hu10SprintLinkPattern.FindAllStringSubmatch(body, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

// hu10CurrentSprint devuelve el id del enlace con aria-current="page", o "" si no hay uno solo.
func hu10CurrentSprint(body string) string {
	var current []string
	for _, m := range hu10SprintLinkPattern.FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[0], `aria-current="page"`) {
			current = append(current, m[1])
		}
	}
	if len(current) != 1 {
		return ""
	}
	return current[0]
}

// hu10Text devuelve el texto visible del contenido principal, sin etiquetas y con los espacios
// normalizados.
func hu10Text(body string) string {
	if start := strings.Index(body, "<main"); start >= 0 {
		body = body[start:]
		if end := strings.Index(body, "</main>"); end >= 0 {
			body = body[:end]
		}
	}
	text := html.UnescapeString(hu10TagPattern.ReplaceAllString(body, " "))
	return strings.Join(strings.Fields(text), " ")
}

func hu10AssertText(t *testing.T, body string, want ...string) {
	t.Helper()
	text := hu10Text(body)
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("la pantalla debía mostrar %q; texto: %s", w, text)
		}
	}
}

// hu10AssertValue verifica que el número aparezca después de su etiqueta (con ":" opcional).
func hu10AssertValue(t *testing.T, body, label string, value int) {
	t.Helper()
	pattern := regexp.MustCompile(regexp.QuoteMeta(label) + `:?\s*` + fmt.Sprint(value) + `\b`)
	if text := hu10Text(body); !pattern.MatchString(text) {
		t.Errorf("la pantalla debía mostrar %q con %d; texto: %s", label, value, text)
	}
}

func hu10AssertNumbers(t *testing.T, body string, planned, completed, detected, resolved int) {
	t.Helper()
	hu10AssertValue(t, body, "Story Points planificados", planned)
	hu10AssertValue(t, body, "Story Points completados", completed)
	hu10AssertValue(t, body, "Defectos detectados", detected)
	hu10AssertValue(t, body, "Defectos resueltos", resolved)
}

// hu10AssertNoInvalidValues verifica que no haya "NaN" ni un "%" que no sea parte de un porcentaje
// con formato ("75,00 %").
func hu10AssertNoInvalidValues(t *testing.T, body string) {
	t.Helper()
	text := hu10Text(body)
	if strings.Contains(text, "NaN") || strings.Contains(text, "Inf") {
		t.Errorf("la pantalla muestra un valor inválido: %s", text)
	}
	if got, want := strings.Count(text, "%"), len(hu10PercentPattern.FindAllString(text, -1)); got != want {
		t.Errorf("la pantalla muestra un %% suelto (%d %% y %d porcentajes con formato): %s", got, want, text)
	}
}

func hu10AssertList(t *testing.T, body string, want []int64, current int64) {
	t.Helper()
	var wantIDs []string
	for _, id := range want {
		wantIDs = append(wantIDs, id64(id))
	}
	if got := hu10SprintLinks(body); strings.Join(got, ",") != strings.Join(wantIDs, ",") {
		t.Errorf("lista de Sprints = %v; se esperaba %v (del más reciente al más antiguo)", got, wantIDs)
	}
	if got := hu10CurrentSprint(body); got != id64(current) {
		t.Errorf("Sprint elegido (aria-current) = %q; se esperaba %d", got, current)
	}
}

// hu10AssertNoInternalDetails verifica que una respuesta de error no exponga consultas, errores de
// la base, rutas internas ni configuración.
func hu10AssertNoInternalDetails(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{
		"simulated database failure",
		"SELECT ", "FROM ", "sprint_story_results", "defects", "sprints WHERE",
		"SQLSTATE", "pgx", "sql:", "postgres://", "DATABASE_URL",
		"load sprint quality", "sprint not found", "sprint is not finished",
		".go:", "/src/web/", "/src/metrics/", "src\\web\\",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("la respuesta de error expuso un detalle interno: %q", forbidden)
		}
	}
}

func hu10AssertErrorPage(t *testing.T, resp *http.Response, body string) {
	t.Helper()
	if resp.StatusCode < http.StatusInternalServerError {
		t.Errorf("estado = %d; se esperaba la página de error (500 o 503)", resp.StatusCode)
	}
	if !strings.Contains(body, TitleInternal) && !strings.Contains(body, TitleUnavailable) {
		t.Errorf("debía verse la página de error de la aplicación: %s", body)
	}
	text := hu10Text(body)
	for _, partial := range []string{"historias completadas", "Story Points planificados", "Defectos detectados", " %"} {
		if strings.Contains(text, partial) {
			t.Errorf("la página de error no debía mostrar números parciales (%q): %s", partial, text)
		}
	}
	if links := hu10SprintLinks(body); len(links) != 0 {
		t.Errorf("la página de error no debía mostrar la lista de Sprints: %v", links)
	}
	hu10AssertNoInternalDetails(t, body)
}

func hu10Str(s string) *string { return &s }

// --- T022: pantalla ---

// Escenario 10a: la lista va del Sprint más reciente al más antiguo y se muestra el más reciente.
func TestHU10_ListaYSprintPorDefecto(t *testing.T) {
	app := newHU10QualityApp(t)
	s1, s2, s3 := app.threeSprints(t, app.fx.projectA)
	app.chooseProject(t, app.fx.projectA)

	resp, body := app.page(t, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("estado %d; se esperaba 200", resp.StatusCode)
	}
	if !strings.Contains(body, "<h1>"+TitleQuality+"</h1>") {
		t.Errorf("debía verse el título %q", TitleQuality)
	}
	hu10AssertList(t, body, []int64{s3, s2, s1}, s3)
	hu10AssertText(t, body, "Sprint 3", "Sprint 2", "Sprint 1", "1 de 4 historias completadas", "25,00 %")
	hu10AssertNumbers(t, body, 11, 8, 0, 0)
	if text := hu10Text(body); strings.Index(text, "Sprint 3") > strings.Index(text, "Sprint 2") ||
		strings.Index(text, "Sprint 2") > strings.Index(text, "Sprint 1") {
		t.Errorf("los Sprints debían verse en el orden 3, 2, 1: %s", text)
	}
	if strings.Contains(hu10Text(body), hu10MsgNoSprints) {
		t.Errorf("con Sprints finalizados no debía verse %q", hu10MsgNoSprints)
	}
}

// Escenario 10b: elegir otro Sprint muestra sus datos y lo marca en la lista.
func TestHU10_ElegirOtroSprint(t *testing.T) {
	app := newHU10QualityApp(t)
	s1, s2, s3 := app.threeSprints(t, app.fx.projectA)
	app.chooseProject(t, app.fx.projectA)

	body := app.pageOf(t, s1)
	hu10AssertList(t, body, []int64{s3, s2, s1}, s1)
	hu10AssertText(t, body, "1 de 1 historias completadas", "100,00 %")
	hu10AssertNumbers(t, body, 5, 5, 0, 0)
	if strings.Contains(hu10Text(body), "25,00 %") {
		t.Error("no debían verse los datos del Sprint 3")
	}

	body = app.pageOf(t, s2)
	hu10AssertList(t, body, []int64{s3, s2, s1}, s2)
	hu10AssertText(t, body, "0 de 2 historias completadas", "0,00 %")
	hu10AssertNumbers(t, body, 5, 0, 0, 0)
}

// Escenario 10c: el Sprint activo no figura en la lista y se avisa que sigue en curso.
func TestHU10_SprintActivoNoFigura(t *testing.T) {
	app := newHU10QualityApp(t)
	s1, s2, s3 := app.threeSprints(t, app.fx.projectA)
	s4 := app.sprint(t, app.fx.projectA, 4, "Activo")
	app.chooseProject(t, app.fx.projectA)

	_, body := app.page(t, nil)
	hu10AssertList(t, body, []int64{s3, s2, s1}, s3)
	hu10AssertText(t, body, hu10MsgActiveHint(4))
	if strings.Contains(body, `href="/calidad?sprint=`+id64(s4)+`"`) {
		t.Error("el Sprint 4 activo no debía figurar en la lista")
	}

	// Pedirlo por la dirección tampoco lo muestra: se ve el más reciente de los finalizados.
	body = app.pageOf(t, s4)
	hu10AssertList(t, body, []int64{s3, s2, s1}, s3)
	hu10AssertText(t, body, hu10MsgActiveHint(4), "1 de 4 historias completadas")
}

// Escenario 11: sin Sprints finalizados se ve el mensaje, sin error ni pantalla vacía.
func TestHU10_SinSprintsFinalizados(t *testing.T) {
	t.Run("con un Sprint activo", func(t *testing.T) {
		app := newHU10QualityApp(t)
		app.sprint(t, app.fx.projectA, 1, "Activo")
		app.chooseProject(t, app.fx.projectA)

		resp, body := app.page(t, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("estado %d; se esperaba 200", resp.StatusCode)
		}
		hu10AssertText(t, body, TitleQuality, hu10MsgNoSprints, hu10MsgActiveHint(1))
		if links := hu10SprintLinks(body); len(links) != 0 {
			t.Errorf("no debía haber Sprints en la lista: %v", links)
		}
		if strings.Contains(body, TitleInternal) || strings.Contains(body, TitleUnavailable) {
			t.Error("no debía verse la página de error")
		}
	})

	t.Run("sin ningún Sprint", func(t *testing.T) {
		app := newHU10QualityApp(t)
		app.chooseProject(t, app.fx.projectA)

		resp, body := app.page(t, hu10Str("5"))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("estado %d; se esperaba 200", resp.StatusCode)
		}
		hu10AssertText(t, body, TitleQuality, hu10MsgNoSprints)
		if text := hu10Text(body); strings.Contains(text, "sigue en curso") || strings.Contains(text, "historias completadas") {
			t.Errorf("sin Sprints no debía verse el aviso del activo ni números: %s", text)
		}
	})
}

// Casos límite: un sprint inválido, inexistente o ajeno se ignora y se ve el más reciente, sin error.
func TestHU10_SprintInvalidoSeIgnora(t *testing.T) {
	app := newHU10QualityApp(t)
	s1, s2, s3 := app.threeSprints(t, app.fx.projectA)
	foreign := app.sprint(t, app.fx.projectB, 1, "Finalizado")
	app.chooseProject(t, app.fx.projectA)

	for _, raw := range []string{
		"", "abc", "0", "-1", "0" + id64(s1), " " + id64(s1), id64(s1) + " ", "1e3", "99999999999999999999",
		"9223372036854775807", id64(foreign),
	} {
		t.Run(fmt.Sprintf("%q", raw), func(t *testing.T) {
			resp, body := app.page(t, &raw)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("estado %d; se esperaba 200", resp.StatusCode)
			}
			hu10AssertList(t, body, []int64{s3, s2, s1}, s3)
			hu10AssertText(t, body, "1 de 4 historias completadas")
			if strings.Contains(body, "field-error") || strings.Contains(body, "alert-error") {
				t.Error("un sprint inválido no debía mostrar un mensaje de error")
			}
		})
	}
}

// Escenarios 1, 3, 4, 5 y 6 con dos Sprints como los de T011 y uno sin registro de cierre.
func TestHU10_NumerosDelSprint(t *testing.T) {
	app := newHU10QualityApp(t)
	sp := func(n int) *int { return &n }
	project := app.fx.projectA

	s1 := app.sprint(t, project, 1, "Finalizado")
	s1Done3 := app.closedStory(t, project, s1, sp(3), "Hecho")
	app.closedStory(t, project, s1, sp(5), "Hecho")
	app.closedStory(t, project, s1, sp(8), "En progreso")
	app.closedStory(t, project, s1, sp(2), "En Sprint")

	s2 := app.sprint(t, project, 2, "Finalizado")
	s2Done := app.closedStory(t, project, s2, sp(1), "Hecho")
	app.closedStory(t, project, s2, sp(1), "Hecho")
	app.closedStory(t, project, s2, sp(1), "Hecho")
	app.closedStory(t, project, s2, nil, "En Sprint")

	// Sprint 2: tres detectados en él (uno resuelto en él) y uno detectado en el 1 y resuelto en el 2.
	app.defect(t, project, s2Done, s2, s2)
	app.defect(t, project, s2Done, s2, 0)
	app.defect(t, project, s2Done, s2, 0)
	app.defect(t, project, s1Done3, s1, s2)

	s3 := app.sprint(t, project, 3, "Finalizado") // sin registro de cierre
	app.chooseProject(t, project)

	t.Run("3 de 4 completadas, 75,00 %, advertencia y defectos", func(t *testing.T) {
		body := app.pageOf(t, s2)
		hu10AssertText(t, body, "Sprint 2", "3 de 4 historias completadas", "75,00 %", hu10MsgWarningOne)
		hu10AssertNumbers(t, body, 3, 3, 3, 2)
		hu10AssertNoInvalidValues(t, body)
	})

	t.Run("3, 5, 8 y 2 Story Points: 2 de 4, 50,00 %, 18 y 8", func(t *testing.T) {
		body := app.pageOf(t, s1)
		hu10AssertText(t, body, "2 de 4 historias completadas", "50,00 %")
		hu10AssertNumbers(t, body, 18, 8, 1, 0)
		if strings.Contains(hu10Text(body), "sin Story Points") {
			t.Error("sin historias sin estimar no debía verse la advertencia")
		}
		hu10AssertNoInvalidValues(t, body)
	})

	t.Run("sin registro de cierre: No disponible, sin NaN ni % suelto", func(t *testing.T) {
		body := app.pageOf(t, s3)
		hu10AssertText(t, body, "Sprint 3", MsgQualityNotAvailable)
		hu10AssertNumbers(t, body, 0, 0, 0, 0)
		hu10AssertNoInvalidValues(t, body)
		if text := hu10Text(body); strings.Contains(text, MsgQualityNotAvailable+" %") || hu10PercentPattern.MatchString(text) {
			t.Errorf("sin historias el porcentaje debía ser solo %q: %s", MsgQualityNotAvailable, text)
		}
	})
}

// Escenario 9: con el proyecto A activo nunca se ven Sprints, historias ni defectos de B.
func TestHU10_OtroProyecto(t *testing.T) {
	app := newHU10QualityApp(t)
	sp := func(n int) *int { return &n }

	a1 := app.sprint(t, app.fx.projectA, 1, "Finalizado")
	aStory := app.closedStory(t, app.fx.projectA, a1, sp(2), "Hecho")
	app.closedStory(t, app.fx.projectA, a1, sp(3), "En progreso")
	app.defect(t, app.fx.projectA, aStory, a1, 0)

	b1 := app.sprint(t, app.fx.projectB, 1, "Finalizado")
	bStory := app.closedStory(t, app.fx.projectB, b1, sp(13), "Hecho")
	app.closedStory(t, app.fx.projectB, b1, sp(21), "Hecho")
	app.closedStory(t, app.fx.projectB, b1, sp(34), "Hecho")
	// Un defecto de B cuyos Sprints son el Sprint 1 de A (la clave foránea lo permite).
	app.defect(t, app.fx.projectB, bStory, a1, a1)
	app.chooseProject(t, app.fx.projectA)

	for name, raw := range map[string]*string{"sin elegir": nil, "pidiendo el Sprint de B": hu10Str(id64(b1))} {
		t.Run(name, func(t *testing.T) {
			resp, body := app.page(t, raw)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("estado %d; se esperaba 200", resp.StatusCode)
			}
			hu10AssertList(t, body, []int64{a1}, a1)
			hu10AssertText(t, body, "1 de 2 historias completadas", "50,00 %")
			hu10AssertNumbers(t, body, 5, 2, 1, 0)
			text := hu10Text(body)
			for _, foreign := range []string{"3 de 3 historias completadas", "100,00 %", "planificados 68", "completados 68"} {
				if strings.Contains(text, foreign) {
					t.Errorf("se vio un dato del proyecto B (%q): %s", foreign, text)
				}
			}
		})
	}
}

// --- T023: errores y bordes ---

// hu10FailingCatalog simula una falla al leer la lista de Sprints.
type hu10FailingCatalog struct{}

func (hu10FailingCatalog) QualitySprints(context.Context, int64) (QualitySprintList, error) {
	return QualitySprintList{Finished: []QualitySprintOption{{ID: 1, Number: 1}}},
		errors.New("simulated database failure: SELECT id, number, status FROM sprints")
}

// hu10FailingReader simula una falla al calcular, con números parciales que no deben verse.
type hu10FailingReader struct{ err error }

func (r hu10FailingReader) SprintQuality(context.Context, int64, int64) (metrics.SprintQuality, error) {
	return metrics.SprintQuality{StoriesPlanned: 4, StoriesCompleted: 3, CompletionPercentage: "75.00", PointsPlanned: 18}, r.err
}

// Escenario 12: si falla la lectura, se ve la página de error de HU-20a y ningún número.
func TestHU10_ErrorAlLeer(t *testing.T) {
	readError := fmt.Errorf("load sprint quality 1: count defects: %w",
		errors.New("simulated database failure: SELECT COUNT(*) FROM defects"))
	cases := []struct {
		name       string
		readerFor  func(*sql.DB) QualityReader
		catalogFor func(*sql.DB) QualityCatalog
	}{
		{name: "falla el catálogo", catalogFor: func(*sql.DB) QualityCatalog { return hu10FailingCatalog{} }},
		{name: "falla el cálculo", readerFor: func(*sql.DB) QualityReader { return hu10FailingReader{err: readError} }},
		{name: "el Sprint dejó de estar finalizado", readerFor: func(*sql.DB) QualityReader {
			return hu10FailingReader{err: fmt.Errorf("%w: 1", metrics.ErrSprintNotFinished)}
		}},
		{name: "el Sprint dejó de existir", readerFor: func(*sql.DB) QualityReader {
			return hu10FailingReader{err: metrics.ErrSprintNotFound}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := newHU10QualityAppWith(t, c.readerFor, c.catalogFor)
			app.threeSprints(t, app.fx.projectA)
			app.chooseProject(t, app.fx.projectA)

			resp, body := app.page(t, nil)
			hu10AssertErrorPage(t, resp, body)
		})
	}
}

// HU-20b: sin proyecto activo válido se redirige a elegirlo, con regreso a "Calidad y avance".
func TestHU10_SinProyecto(t *testing.T) {
	app := newHU10QualityApp(t)
	app.threeSprints(t, app.fx.projectA)
	const wantLocation = "/contexto/proyecto?volver_a=%2Fcalidad"

	cases := []struct {
		name    string
		cookies []*http.Cookie
	}{
		{name: "sin selección"},
		{name: "cookie inválida", cookies: []*http.Cookie{{Name: CookieProject, Value: "invalida", Path: "/"}}},
		{name: "proyecto eliminado", cookies: []*http.Cookie{idCookie(CookieProject, 9223372036854775807)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := app.ctxApp.client(false)
			if len(c.cookies) > 0 {
				app.setCookies(t, client, c.cookies...)
			}
			resp, body := app.get(t, client, hu10PathQuality)
			if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != wantLocation {
				t.Errorf("estado/location = %d/%q; se esperaba 303 %q", resp.StatusCode, resp.Header.Get("Location"), wantLocation)
			}
			if strings.Contains(body, "historias completadas") {
				t.Error("sin proyecto no debían verse números")
			}
		})
	}
}

// HU-20a: la pantalla es de solo lectura; cualquier método distinto de GET da 405.
func TestHU10_PostNoPermitido(t *testing.T) {
	app := newHU10QualityApp(t)
	s1, _, _ := app.threeSprints(t, app.fx.projectA)
	app.chooseProject(t, app.fx.projectA)

	resp, body := app.post(t, app.client, hu10PathQuality, url.Values{"sprint": {id64(s1)}})
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST: estado %d; se esperaba 405", resp.StatusCode)
	}
	if !strings.Contains(body, TitleMethodNotAllowed) {
		t.Errorf("POST: debía verse la página %q", TitleMethodNotAllowed)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req, err := http.NewRequest(method, app.ts.URL+hu10PathQuality, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := app.client.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		_ = got.Body.Close()
		if got.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s: estado %d; se esperaba 405", method, got.StatusCode)
		}
	}
}

// SC-005: con 20 Sprints, 50 historias por Sprint y 100 defectos, la pantalla responde en menos de
// 500 ms.
func TestHU10_Rendimiento(t *testing.T) {
	app := newHU10QualityApp(t)
	project := app.fx.projectA
	var sprintIDs []int64
	for number := 1; number <= 20; number++ {
		id := app.sprint(t, project, number, "Finalizado")
		sprintIDs = append(sprintIDs, id)
		if _, err := app.db.Exec(`
			WITH stories AS (
				INSERT INTO user_stories (project_id, title, description, priority, story_points, status)
				SELECT $1, 'HU10 rendimiento ' || $2::bigint || '-' || g, 'Historia de prueba HU-10', 'Media',
				       (ARRAY[1, 2, 3, 5, 8])[1 + g % 5], 'Pendiente'
				FROM generate_series(1, 50) AS g
				RETURNING id, story_points
			)
			INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
			SELECT $2, id, CASE WHEN id % 3 = 0 THEN 'En progreso' ELSE 'Hecho' END, story_points
			FROM stories`, project, id); err != nil {
			t.Fatalf("no se pudieron crear las historias del Sprint %d: %v", number, err)
		}
	}
	if _, err := app.db.Exec(`
		INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		SELECT $1::integer, (SELECT MIN(id) FROM user_stories WHERE project_id = $1::bigint), 'Defecto de prueba HU-10', 'Media',
		       CASE WHEN g % 2 = 0 THEN 'Resuelto' ELSE 'Abierto' END,
		       ($2::bigint[])[1 + g % 20],
		       CASE WHEN g % 2 = 0 THEN ($2::bigint[])[1 + g % 20] END
		FROM generate_series(1, 100) AS g`, project, sprintIDs); err != nil {
		t.Fatalf("no se pudieron crear los defectos: %v", err)
	}
	app.chooseProject(t, project)

	for _, raw := range []*string{nil, hu10Str(id64(sprintIDs[0])), hu10Str(id64(sprintIDs[10]))} {
		start := time.Now()
		resp, body := app.page(t, raw)
		elapsed := time.Since(start)
		if resp.StatusCode != http.StatusOK || !strings.Contains(hu10Text(body), "historias completadas") {
			t.Fatalf("estado %d; se esperaba 200 con los datos del Sprint", resp.StatusCode)
		}
		if len(hu10SprintLinks(body)) != 20 {
			t.Errorf("la lista debía tener los 20 Sprints; tiene %d", len(hu10SprintLinks(body)))
		}
		if elapsed >= 500*time.Millisecond {
			t.Errorf("la pantalla tardó %v; se esperaba menos de 500 ms", elapsed)
		}
	}
}
