package test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/pdf"
	"github.com/FrancoV20/seguimiento-y-medicion/src/report"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de "Reportes" (HU-09, T030 y T041), con el servidor, report.Service, HU-07, HU-10 y
// PostgreSQL reales. Cada escenario crea su propio proyecto, integrantes, Sprints, historias, esfuerzo
// y defectos, como los dejan HU-04, HU-06, HU-14 y HU-15.

var hu09BDDNow = time.Date(2026, 10, 28, 10, 30, 0, 0, time.UTC)

type hu09BDDApp struct {
	db       *sql.DB
	server   *httptest.Server
	client   *http.Client
	stamp    int64
	projects []int64
	members  []int64
	sprints  map[string]int64
	stories  map[string]int64
}

func newHU09BDDApp(t *testing.T) *hu09BDDApp {
	t.Helper()
	databaseURL := os.Getenv(web.EnvDatabaseURL)
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite la integración BDD de HU-09")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("abrir la base BDD de HU-09: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("base BDD de HU-09 no disponible: %v", err)
	}
	app := &hu09BDDApp{db: db, stamp: time.Now().UnixNano(), sprints: map[string]int64{}, stories: map[string]int64{}}
	t.Cleanup(app.cleanup)

	server := web.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.Use(web.CrossOriginGuard(server.Renderer()))
	web.RegisterContext(server, web.NewPostgreSQLContextStore(db))
	service := report.NewService(report.NewPostgreSQLStore(db),
		metrics.NewService(metrics.NewPostgreSQLMetricsStore(db)),
		metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db)),
		func() time.Time { return hu09BDDNow })
	if err := web.RegisterReports(server, web.NewPostgreSQLReportCatalog(db), service); err != nil {
		t.Fatalf("registrar la sección Reportes: %v", err)
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
	return app
}

func (a *hu09BDDApp) cleanup() {
	const sprints = `SELECT id FROM sprints WHERE project_id = $1`
	for _, p := range a.projects {
		_, _ = a.db.Exec(`DELETE FROM defects WHERE project_id = $1`, p)
		_, _ = a.db.Exec(`DELETE FROM sprint_story_results WHERE sprint_id IN (`+sprints+`)`, p)
		_, _ = a.db.Exec(`DELETE FROM sprint_story_assignments WHERE sprint_id IN (`+sprints+`)`, p)
		_, _ = a.db.Exec(`DELETE FROM sprint_metrics WHERE sprint_id IN (`+sprints+`)`, p)
		_, _ = a.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, p)
		_, _ = a.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, p)
		_, _ = a.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, p)
		_, _ = a.db.Exec(`DELETE FROM projects WHERE id = $1`, p)
	}
	for _, m := range a.members {
		_, _ = a.db.Exec(`DELETE FROM members WHERE id = $1`, m)
	}
}

func (a *hu09BDDApp) id(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := a.db.QueryRow(q, args...).Scan(&id); err != nil {
		t.Fatalf("preparar los datos BDD de HU-09: %v", err)
	}
	return id
}

func (a *hu09BDDApp) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := a.db.Exec(q, args...); err != nil {
		t.Fatalf("preparar los datos BDD de HU-09: %v", err)
	}
}

// givenProject crea un proyecto con sus integrantes y devuelve su id.
func (a *hu09BDDApp) givenProject(t *testing.T, name string, members ...string) int64 {
	t.Helper()
	p := a.id(t, `INSERT INTO projects (name, start_date, end_date) VALUES ($1, '2026-08-18', '2026-11-30') RETURNING id`, name)
	a.projects = append(a.projects, p)
	for _, m := range members {
		id := a.id(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, m)
		a.members = append(a.members, id)
		a.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, p, id)
	}
	return p
}

func (a *hu09BDDApp) givenStory(t *testing.T, project int64, title string, points int, estimated string) {
	t.Helper()
	var sp, est any
	if points != 0 {
		sp = points
	}
	if estimated != "" {
		est = estimated
	}
	a.stories[title] = a.id(t, `INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours)
		VALUES ($1, $2, 'd', 'Media', $3, $4) RETURNING id`, project, title, sp, est)
}

func (a *hu09BDDApp) givenSprint(t *testing.T, project int64, key string, number int, goal, start, end, status string) int64 {
	t.Helper()
	id := a.id(t, `INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, project, number, goal, start, end, status)
	a.sprints[key] = id
	return id
}

func (a *hu09BDDApp) givenAssigned(t *testing.T, sprintID int64, title, status string) {
	t.Helper()
	a.exec(t, `INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, a.stories[title])
	a.exec(t, `UPDATE user_stories SET status = $1, active_sprint_id = $2 WHERE id = $3`, status, sprintID, a.stories[title])
}

// givenClosed deja la historia como la deja el cierre de HU-15.
func (a *hu09BDDApp) givenClosed(t *testing.T, sprintID int64, title, finalStatus string, points int) {
	t.Helper()
	a.exec(t, `INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points) VALUES ($1, $2, $3, $4)`,
		sprintID, a.stories[title], finalStatus, points)
	if finalStatus == "Hecho" {
		a.givenAssigned(t, sprintID, title, finalStatus)
	}
}

func (a *hu09BDDApp) givenEffort(t *testing.T, project int64, title, member, hours string) {
	t.Helper()
	a.exec(t, `INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		SELECT $1::bigint, m.id, '2026-09-10', 'Desarrollo', $4::numeric
		FROM members m JOIN project_members pm ON pm.member_id = m.id
		WHERE pm.project_id = $2 AND m.display_name = $3`, a.stories[title], project, member, hours)
}

func (a *hu09BDDApp) givenDefect(t *testing.T, project int64, title, description, severity string, detection, resolution int64) {
	t.Helper()
	status, res := "Abierto", any(nil)
	if resolution != 0 {
		status, res = "Resuelto", resolution
	}
	a.exec(t, `INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, project, a.stories[title], description, severity, status, detection, res)
}

// givenAlfa arma el proyecto de "Datos de ejemplo": Sprint 1 y 2 finalizados y 3 en curso.
func (a *hu09BDDApp) givenAlfa(t *testing.T) (int64, string) {
	t.Helper()
	name := fmt.Sprintf("Alfa BDD %d", a.stamp)
	p := a.givenProject(t, name, "Ana", "Beto")
	a.givenStory(t, p, "Alta", 3, "4")
	a.givenStory(t, p, "Login", 5, "10")
	a.givenStory(t, p, "Tablero", 3, "6")
	a.givenStory(t, p, "Perfil", 2, "")
	a.givenStory(t, p, "Reportes", 8, "12")
	a.givenStory(t, p, "Exportar", 5, "6")
	a.givenStory(t, p, "Ayuda", 0, "")

	s1 := a.givenSprint(t, p, "1", 1, "Dar de alta usuarios", "2026-08-18", "2026-08-31", "Finalizado")
	a.givenClosed(t, s1, "Alta", "Hecho", 3)
	a.givenEffort(t, p, "Alta", "Ana", "4")

	s2 := a.givenSprint(t, p, "2", 2, "Gestionar perfiles y reportes", "2026-09-01", "2026-09-14", "Finalizado")
	a.givenClosed(t, s2, "Login", "Hecho", 5)
	a.givenClosed(t, s2, "Tablero", "Hecho", 3)
	a.givenClosed(t, s2, "Perfil", "Hecho", 2)
	a.givenClosed(t, s2, "Reportes", "En progreso", 8)
	for _, e := range [][3]string{{"Login", "Ana", "8"}, {"Login", "Beto", "3"}, {"Tablero", "Ana", "6"},
		{"Perfil", "Ana", "3"}, {"Perfil", "Beto", "1"}, {"Reportes", "Beto", "3"}} {
		a.givenEffort(t, p, e[0], e[1], e[2])
	}

	s3 := a.givenSprint(t, p, "3", 3, "Exportar e informar", "2026-09-15", "2026-09-28", "Activo")
	a.givenAssigned(t, s3, "Exportar", "En progreso")
	a.givenAssigned(t, s3, "Ayuda", "En Sprint")
	a.givenEffort(t, p, "Exportar", "Ana", "2")

	a.givenDefect(t, p, "Login", "El login no valida la contraseña", "Alta", s2, s2)
	a.givenDefect(t, p, "Perfil", "La foto de perfil no se guarda", "Media", s2, 0)
	a.givenDefect(t, p, "Exportar", "El PDF sale en blanco", "Crítica", s3, 0)
	return p, name
}

// selectProject elige el proyecto de trabajo (HU-20b).
func (a *hu09BDDApp) selectProject(t *testing.T, project int64) {
	t.Helper()
	form := url.Values{"proyecto_id": {fmt.Sprint(project)}, "volver_a": {"/reportes"}}
	req, err := http.NewRequest(http.MethodPost, a.server.URL+"/contexto/proyecto", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", a.server.URL)
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatalf("elegir el proyecto: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir el proyecto: estado %d; se esperaba 303", resp.StatusCode)
	}
}

func (a *hu09BDDApp) get(t *testing.T, path string) (*http.Response, string) {
	t.Helper()
	resp, err := a.client.Get(a.server.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer GET %s: %v", path, err)
	}
	return resp, string(body)
}

// whenExport presiona "Exportar PDF" en el Sprint y devuelve la respuesta.
func (a *hu09BDDApp) whenExport(t *testing.T, key, query string) (*http.Response, string) {
	t.Helper()
	return a.get(t, fmt.Sprintf("/reportes/sprints/%d/pdf%s", a.sprints[key], query))
}

func thenPDF(t *testing.T, resp *http.Response, body, fileName string) string {
	t.Helper()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("se esperaba un PDF: estado %d, Content-Type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="`+fileName+`"` {
		t.Errorf("Content-Disposition = %q, se esperaba el archivo %s", cd, fileName)
	}
	text, err := pdf.ExtractText([]byte(body))
	if err != nil {
		t.Fatalf("leer el PDF: %v", err)
	}
	return text
}

func thenContains(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("falta %q", want)
		}
	}
}

// Escenario 1: Exportación exitosa de un Sprint finalizado.
func TestHU09BDD_ExportacionExitosaDeUnSprintFinalizado(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el Sprint 2 finalizado de «Alfa»
	project, name := app.givenAlfa(t)
	app.selectProject(t, project)
	// Cuando en "Reportes" presiono "Exportar PDF" en el Sprint 2
	_, list := app.get(t, "/reportes")
	thenContains(t, list, fmt.Sprintf(`href="/reportes/sprints/%d/pdf"`, app.sprints["2"]))
	resp, body := app.whenExport(t, "2", "")
	// Entonces se descarga "reporte-sprint-2.pdf" con el proyecto, el Sprint, su objetivo, sus fechas y la
	// fecha de generación
	text := thenPDF(t, resp, body, "reporte-sprint-2.pdf")
	thenContains(t, text, "Proyecto: "+name, "Sprint 2", "Objetivo: Gestionar perfiles y reportes",
		"Fechas: 01/09/2026 al 14/09/2026", "Generado: 28/10/2026 10:30")
}

// Escenario 2: Historias planificadas y completadas.
func TestHU09BDD_HistoriasPlanificadasYCompletadas(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el Sprint 2 finalizado de «Alfa»
	project, _ := app.givenAlfa(t)
	app.selectProject(t, project)
	// Cuando exporto su reporte
	resp, body := app.whenExport(t, "2", "")
	// Entonces el PDF lista las 4 historias con su estado al cerrar, indica que «Reportes» volvió al
	// Backlog y muestra "3 de 4 historias completadas (75,00 %)"
	text := thenPDF(t, resp, body, "reporte-sprint-2.pdf")
	thenContains(t, text, "Login Hecho", "Tablero Hecho", "Perfil Hecho",
		"Reportes En progreso – volvió al Backlog", "3 de 4 historias completadas (75,00 %)")
}

// Escenario 5: Métricas del Sprint.
func TestHU09BDD_MetricasDelSprint(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el Sprint 2 finalizado de «Alfa»
	project, _ := app.givenAlfa(t)
	app.selectProject(t, project)
	// Cuando exporto su reporte
	resp, body := app.whenExport(t, "2", "")
	// Entonces el PDF muestra la velocidad en 10 Story Points, la desviación en "+6,25 %" y el aviso de
	// cálculo parcial
	text := thenPDF(t, resp, body, "reporte-sprint-2.pdf")
	thenContains(t, text, "Velocidad: 10 Story Points", "Desviación de esfuerzo: +6,25 %",
		"Cálculo parcial: basado en 2 de 3 historias")
}

// Escenario 8: Advertencia al exportar un Sprint en curso.
func TestHU09BDD_AdvertenciaAlExportarUnSprintEnCurso(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado que el Sprint 3 de «Alfa» está en curso
	project, _ := app.givenAlfa(t)
	app.selectProject(t, project)
	// Cuando presiono "Exportar PDF" en el Sprint 3
	resp, body := app.whenExport(t, "3", "")
	// Entonces veo la advertencia con "Generar igual" y "Cancelar", y todavía no se genera el PDF
	if resp.StatusCode != http.StatusOK || strings.Contains(resp.Header.Get("Content-Type"), "pdf") || strings.Contains(body, "%PDF") {
		t.Fatalf("se esperaba la advertencia sin PDF: estado %d, Content-Type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	thenContains(t, body, "El Sprint 3 sigue en curso: el reporte puede estar incompleto",
		fmt.Sprintf(`href="/reportes/sprints/%d/pdf?confirmar=si"`, app.sprints["3"]), "Generar igual",
		`href="/reportes"`, "Cancelar")
}

// Escenario 11: Sin Sprints.
func TestHU09BDD_SinSprints(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el proyecto «Beta», sin Sprints
	project := app.givenProject(t, fmt.Sprintf("Beta BDD %d", app.stamp), "Ana")
	app.selectProject(t, project)
	// Cuando abro "Reportes"
	resp, body := app.get(t, "/reportes")
	// Entonces veo "Aún no hay Sprints para exportar" y el botón "Exportar reporte del proyecto"
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	thenContains(t, body, "Aún no hay Sprints para exportar", `href="/reportes/proyecto/pdf"`, "Exportar reporte del proyecto")
}

// Escenario 14: Advertencia al exportar un proyecto con un Sprint en curso.
func TestHU09BDD_AdvertenciaAlExportarUnProyectoConUnSprintEnCurso(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el proyecto «Alfa», con el Sprint 3 en curso
	project, _ := app.givenAlfa(t)
	app.selectProject(t, project)
	// Cuando en "Reportes" presiono "Exportar reporte del proyecto"
	resp, body := app.get(t, "/reportes/proyecto/pdf")
	// Entonces veo la advertencia con "Generar igual" y "Cancelar", y todavía no se genera el PDF
	if resp.StatusCode != http.StatusOK || strings.Contains(resp.Header.Get("Content-Type"), "pdf") || strings.Contains(body, "%PDF") {
		t.Fatalf("se esperaba la advertencia sin PDF: estado %d, Content-Type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	thenContains(t, body, "El proyecto tiene un Sprint en curso (Sprint 3): el reporte puede estar incompleto",
		`href="/reportes/proyecto/pdf?confirmar=si"`, "Generar igual", `href="/reportes"`, "Cancelar")
}

// Escenario 16: Avance y Sprints del proyecto.
func TestHU09BDD_AvanceYSprintsDelProyecto(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el reporte del proyecto «Alfa»
	project, _ := app.givenAlfa(t)
	app.selectProject(t, project)
	// Cuando lo genero igual y lo abro
	resp, body := app.get(t, "/reportes/proyecto/pdf?confirmar=si")
	// Entonces veo las historias por estado, el avance, una fila por Sprint y la velocidad acumulada
	text := thenPDF(t, resp, body, fmt.Sprintf("reporte-proyecto-%d.pdf", project))
	thenContains(t, text,
		"Reporte parcial: hay un Sprint en curso",
		"Pendiente 1", "En Sprint 1", "En progreso 1", "Hecho 4",
		"4 de 7 historias completadas (57,14 %)",
		"Sprint 1 Finalizado 1 de 1 3 de 3 3 0,00 % 0 detectados y 0 resueltos",
		"Sprint 2 Finalizado 3 de 4 10 de 18 10 +6,25 % (cálculo parcial) 2 detectados y 1 resuelto",
		"Sprint 3 En curso: se calcula al cerrar el Sprint",
		"Velocidad acumulada: 13 Story Points",
		"Promedio: 6,50 Story Points por Sprint",
	)
}

// Escenario 18: Proyecto sin Sprints ni historias.
func TestHU09BDD_ProyectoSinSprintsNiHistorias(t *testing.T) {
	app := newHU09BDDApp(t)
	// Dado el proyecto «Beta», sin Sprints ni historias
	project := app.givenProject(t, fmt.Sprintf("Beta BDD %d", app.stamp), "Ana")
	app.selectProject(t, project)
	// Cuando presiono "Exportar reporte del proyecto"
	resp, body := app.get(t, "/reportes/proyecto/pdf")
	// Entonces se descarga el PDF sin advertencia, con "Aún no hay Sprints", "0 de 0 historias completadas
	// (0,00 %)", "0 horas registradas en total" y "No se registraron defectos en este proyecto"
	text := thenPDF(t, resp, body, fmt.Sprintf("reporte-proyecto-%d.pdf", project))
	thenContains(t, text, "Aún no hay Sprints", "0 de 0 historias completadas (0,00 %)",
		"0 horas registradas en total", "No se registraron defectos en este proyecto")
	if strings.Contains(text, "Reporte parcial") {
		t.Error("sin Sprint en curso el reporte no es parcial")
	}
}
