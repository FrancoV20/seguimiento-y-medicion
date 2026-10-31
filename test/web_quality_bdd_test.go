package test

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de la pantalla "Calidad y avance" (HU-10, T024), con el servidor, el servicio, el
// catálogo y PostgreSQL reales. Cada escenario crea su propio proyecto, Sprints, registro de cierre y
// defectos, como los dejan HU-15 y HU-06.

type hu10BDDApp struct {
	db        *sql.DB
	server    *httptest.Server
	client    *http.Client
	projectID int64
	stamp     int64
}

func newHU10BDDApp(t *testing.T) *hu10BDDApp {
	t.Helper()
	databaseURL := os.Getenv(web.EnvDatabaseURL)
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite la integración BDD de HU-10")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("abrir la base BDD de HU-10: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("base BDD de HU-10 no disponible: %v", err)
	}

	app := &hu10BDDApp{db: db, stamp: time.Now().UnixNano()}
	if err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, CURRENT_DATE, CURRENT_DATE + 30) RETURNING id`,
		fmt.Sprintf("HU10 BDD %d", app.stamp)).Scan(&app.projectID); err != nil {
		t.Fatalf("crear el proyecto BDD de HU-10: %v", err)
	}
	// Orden: defectos, registro de cierre, asignaciones, historias, Sprints y proyecto.
	t.Cleanup(func() {
		const sprints = `SELECT id FROM sprints WHERE project_id = $1`
		_, _ = db.Exec(`DELETE FROM defects WHERE project_id = $1`, app.projectID)
		_, _ = db.Exec(`DELETE FROM sprint_story_results WHERE sprint_id IN (`+sprints+`)`, app.projectID)
		_, _ = db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id IN (`+sprints+`)`, app.projectID)
		_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, app.projectID)
		_, _ = db.Exec(`DELETE FROM sprints WHERE project_id = $1`, app.projectID)
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, app.projectID)
	})

	server := web.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.Use(web.CrossOriginGuard(server.Renderer()))
	web.RegisterContext(server, web.NewPostgreSQLContextStore(db))
	if err := web.RegisterQuality(server,
		metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db)),
		web.NewPostgreSQLQualityCatalog(db)); err != nil {
		t.Fatalf("registrar la sección Calidad y avance: %v", err)
	}
	app.server = httptest.NewServer(server.Handler())
	t.Cleanup(app.server.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("crear el cookie jar: %v", err)
	}
	app.client = &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp := app.request(t, http.MethodPost, "/contexto/proyecto", url.Values{
		"proyecto_id": {fmt.Sprintf("%d", app.projectID)},
		"volver_a":    {"/calidad"},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/calidad" {
		t.Fatalf("elegir el proyecto: estado/location = %d/%q; se esperaba 303 /calidad",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	return app
}

func (a *hu10BDDApp) request(t *testing.T, method, path string, values url.Values) *http.Response {
	t.Helper()
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequest(method, a.server.URL+path, body)
	if err != nil {
		t.Fatalf("armar %s %s: %v", method, path, err)
	}
	if values != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if method == http.MethodPost {
		req.Header.Set("Origin", a.server.URL)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// whenOpen abre "Calidad y avance"; con sprintID distinto de 0 elige ese Sprint.
func (a *hu10BDDApp) whenOpen(t *testing.T, sprintID int64) string {
	t.Helper()
	path := "/calidad"
	if sprintID != 0 {
		path += fmt.Sprintf("?sprint=%d", sprintID)
	}
	resp := a.request(t, http.MethodGet, path, nil)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer GET %s: %v", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: estado %d; se esperaba 200", path, resp.StatusCode)
	}
	return string(data)
}

func (a *hu10BDDApp) givenSprint(t *testing.T, number int, status string) int64 {
	t.Helper()
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, 'Sprint BDD de HU-10', DATE '2026-10-01', DATE '2026-10-15', $3) RETURNING id`,
		a.projectID, number, status).Scan(&id); err != nil {
		t.Fatalf("crear el Sprint %d: %v", number, err)
	}
	return id
}

// givenClosedStory deja una historia como la cierra HU-15: las "Hecho" conservan su Sprint y las
// demás vuelven al Backlog; todas quedan en el registro de cierre con su estado final.
func (a *hu10BDDApp) givenClosedStory(t *testing.T, sprintID int64, points int, finalStatus string) int64 {
	t.Helper()
	status, active := "Pendiente", sql.NullInt64{}
	if finalStatus == "Hecho" {
		status, active = "Hecho", sql.NullInt64{Int64: sprintID, Valid: true}
	}
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
		VALUES ($1, $2, 'Historia BDD de HU-10', 'Media', $3, $4, $5) RETURNING id`,
		a.projectID, fmt.Sprintf("HU10 BDD %d", a.stamp), points, status, active).Scan(&id); err != nil {
		t.Fatalf("crear la historia: %v", err)
	}
	if finalStatus == "Hecho" {
		if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id); err != nil {
			t.Fatalf("asignar la historia: %v", err)
		}
	}
	if _, err := a.db.Exec(`
		INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
		VALUES ($1, $2, $3, $4)`, sprintID, id, finalStatus, points); err != nil {
		t.Fatalf("registrar el cierre de la historia: %v", err)
	}
	return id
}

func (a *hu10BDDApp) givenDefect(t *testing.T, storyID, detection, resolution int64) {
	t.Helper()
	status, resolved := "Abierto", sql.NullInt64{}
	if resolution != 0 {
		status, resolved = "Resuelto", sql.NullInt64{Int64: resolution, Valid: true}
	}
	if _, err := a.db.Exec(`
		INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		VALUES ($1, $2, 'Defecto BDD de HU-10', 'Media', $3, $4, $5)`,
		a.projectID, storyID, status, detection, resolved); err != nil {
		t.Fatalf("crear el defecto: %v", err)
	}
}

var (
	hu10BDDTag  = regexp.MustCompile(`(?s)<[^>]*>`)
	hu10BDDLink = regexp.MustCompile(`<a\b[^>]*\bhref="/calidad\?sprint=(\d+)"[^>]*>`)
)

// hu10BDDText devuelve el texto visible del contenido principal.
func hu10BDDText(body string) string {
	if start := strings.Index(body, "<main"); start >= 0 {
		body = body[start:]
		if end := strings.Index(body, "</main>"); end >= 0 {
			body = body[:end]
		}
	}
	return strings.Join(strings.Fields(html.UnescapeString(hu10BDDTag.ReplaceAllString(body, " "))), " ")
}

func hu10BDDThenSee(t *testing.T, body string, want ...string) {
	t.Helper()
	text := hu10BDDText(body)
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("debía verse %q; texto: %s", w, text)
		}
	}
}

func hu10BDDThenValue(t *testing.T, body, label string, value int) {
	t.Helper()
	if !regexp.MustCompile(regexp.QuoteMeta(label) + `:?\s*` + fmt.Sprint(value) + `\b`).MatchString(hu10BDDText(body)) {
		t.Errorf("debía verse %q con %d; texto: %s", label, value, hu10BDDText(body))
	}
}

// Escenario 1: Porcentaje de avance.
func TestWebQualityBDD_PorcentajeDeAvance(t *testing.T) {
	// Given un Sprint finalizado que tenía 4 historias y cerró con 3 en "Hecho" (la otra volvió al Backlog)
	app := newHU10BDDApp(t)
	sprint := app.givenSprint(t, 1, "Finalizado")
	for i := 0; i < 3; i++ {
		app.givenClosedStory(t, sprint, 2, "Hecho")
	}
	app.givenClosedStory(t, sprint, 3, "En progreso")

	// When abro la calidad y el avance de ese Sprint
	body := app.whenOpen(t, sprint)

	// Then veo 3 de 4 historias completadas y "75,00 %"
	hu10BDDThenSee(t, body, "3 de 4 historias completadas", "75,00 %")
}

// Escenario 6: Defectos detectados y resueltos en el Sprint.
func TestWebQualityBDD_DefectosDetectadosYResueltos(t *testing.T) {
	// Given el Sprint 2 finalizado con 3 defectos detectados en él, uno de ellos resuelto en él, y un
	// defecto detectado en el Sprint 1 y resuelto en el Sprint 2
	app := newHU10BDDApp(t)
	sprint1 := app.givenSprint(t, 1, "Finalizado")
	story1 := app.givenClosedStory(t, sprint1, 3, "Hecho")
	sprint2 := app.givenSprint(t, 2, "Finalizado")
	story2 := app.givenClosedStory(t, sprint2, 5, "Hecho")
	app.givenDefect(t, story2, sprint2, sprint2)
	app.givenDefect(t, story2, sprint2, 0)
	app.givenDefect(t, story2, sprint2, 0)
	app.givenDefect(t, story1, sprint1, sprint2)

	// When abro su calidad y avance
	body := app.whenOpen(t, sprint2)

	// Then veo 3 defectos detectados y 2 resueltos
	hu10BDDThenValue(t, body, "Defectos detectados", 3)
	hu10BDDThenValue(t, body, "Defectos resueltos", 2)
}

// Escenario 10a: Lista de Sprints y Sprint por defecto.
func TestWebQualityBDD_ListaDeSprintsYSprintPorDefecto(t *testing.T) {
	// Given un proyecto con los Sprints 1, 2 y 3 finalizados
	app := newHU10BDDApp(t)
	sprint1 := app.givenSprint(t, 1, "Finalizado")
	app.givenClosedStory(t, sprint1, 1, "En progreso")
	sprint2 := app.givenSprint(t, 2, "Finalizado")
	app.givenClosedStory(t, sprint2, 1, "Hecho")
	app.givenClosedStory(t, sprint2, 1, "En progreso")
	sprint3 := app.givenSprint(t, 3, "Finalizado")
	app.givenClosedStory(t, sprint3, 8, "Hecho")

	// When abro "Calidad y avance"
	body := app.whenOpen(t, 0)

	// Then veo la lista "Sprint 3", "Sprint 2", "Sprint 1", en ese orden, y los datos del Sprint 3
	var links []string
	current := ""
	for _, m := range hu10BDDLink.FindAllStringSubmatch(body, -1) {
		links = append(links, m[1])
		if strings.Contains(m[0], `aria-current="page"`) {
			current = m[1]
		}
	}
	want := []string{fmt.Sprint(sprint3), fmt.Sprint(sprint2), fmt.Sprint(sprint1)}
	if strings.Join(links, ",") != strings.Join(want, ",") {
		t.Errorf("lista de Sprints = %v; se esperaba %v (Sprint 3, Sprint 2, Sprint 1)", links, want)
	}
	if current != fmt.Sprint(sprint3) {
		t.Errorf("Sprint elegido = %q; se esperaba el Sprint 3 (%d)", current, sprint3)
	}
	text := hu10BDDText(body)
	if i3, i2, i1 := strings.Index(text, "Sprint 3"), strings.Index(text, "Sprint 2"), strings.Index(text, "Sprint 1"); i3 < 0 || i2 < 0 || i1 < 0 || i3 > i2 || i2 > i1 {
		t.Errorf("debían verse \"Sprint 3\", \"Sprint 2\" y \"Sprint 1\", en ese orden: %s", text)
	}
	hu10BDDThenSee(t, body, "1 de 1 historias completadas", "100,00 %")
	hu10BDDThenValue(t, body, "Story Points completados", 8)
}

// Escenario 11: Sin Sprints finalizados.
func TestWebQualityBDD_SinSprintsFinalizados(t *testing.T) {
	// Given un proyecto sin Sprints finalizados
	app := newHU10BDDApp(t)
	app.givenSprint(t, 1, "Activo")

	// When abro "Calidad y avance"
	body := app.whenOpen(t, 0)

	// Then veo "Aún no hay Sprints finalizados" y no veo un error ni una pantalla vacía
	hu10BDDThenSee(t, body, "Calidad y avance", "Aún no hay Sprints finalizados")
	if strings.Contains(body, "Error inesperado") || strings.Contains(body, "Servicio no disponible") {
		t.Error("no debía verse la página de error")
	}
}
