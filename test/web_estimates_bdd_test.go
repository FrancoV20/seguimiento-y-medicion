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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de la pantalla "Estimaciones" (HU-17, US1: T015), con el servidor, el servicio y
// PostgreSQL reales. Cada escenario crea su propio proyecto, historias, integrante y horas.

type hu17BDDApp struct {
	db        *sql.DB
	server    *httptest.Server
	client    *http.Client
	projectID int64
	memberID  int64
	stamp     int64
}

func newHU17BDDApp(t *testing.T) *hu17BDDApp {
	t.Helper()
	databaseURL := os.Getenv(web.EnvDatabaseURL)
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite la integración BDD de HU-17")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("abrir la base BDD de HU-17: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("base BDD de HU-17 no disponible: %v", err)
	}

	app := &hu17BDDApp{db: db, stamp: time.Now().UnixNano()}
	if err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, CURRENT_DATE, CURRENT_DATE + 30) RETURNING id`,
		fmt.Sprintf("HU17 BDD %d", app.stamp)).Scan(&app.projectID); err != nil {
		t.Fatalf("crear el proyecto BDD de HU-17: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("HU17 BDD integrante %d", app.stamp)).Scan(&app.memberID); err != nil {
		t.Fatalf("crear el integrante BDD de HU-17: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, app.memberID)
		_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, app.projectID)
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, app.projectID)
		_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, app.memberID)
	})

	server := web.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.Use(web.CrossOriginGuard(server.Renderer()))
	web.RegisterContext(server, web.NewPostgreSQLContextStore(db))
	if err := web.RegisterEstimates(server,
		backlog.NewEstimateService(backlog.NewPostgreSQLEstimateStore(db)),
		web.NewPostgreSQLEstimateCatalog(db)); err != nil {
		t.Fatalf("registrar la sección Estimaciones: %v", err)
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

	resp, _ := app.do(t, http.MethodPost, "/contexto/proyecto", url.Values{
		"proyecto_id": {fmt.Sprintf("%d", app.projectID)},
		"volver_a":    {"/estimaciones"},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/estimaciones" {
		t.Fatalf("elegir el proyecto: estado/location = %d/%q; se esperaba 303 /estimaciones",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	return app
}

func (a *hu17BDDApp) do(t *testing.T, method, path string, values url.Values) (*http.Response, string) {
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
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer %s %s: %v", method, path, err)
	}
	return resp, string(data)
}

func (a *hu17BDDApp) givenStory(t *testing.T, name string, hours *float64, effort ...string) (int64, string) {
	t.Helper()
	title := fmt.Sprintf("HU17 BDD %s %d", name, a.stamp)
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours)
		VALUES ($1, $2, 'Historia BDD de HU-17', 'Media', 3, $3) RETURNING id`,
		a.projectID, title, hours).Scan(&id); err != nil {
		t.Fatalf("crear la historia %q: %v", title, err)
	}
	for _, h := range effort {
		if _, err := a.db.Exec(`
			INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
			VALUES ($1, $2, CURRENT_DATE, 'Trabajo BDD', $3::numeric)`, id, a.memberID, h); err != nil {
			t.Fatalf("registrar horas reales: %v", err)
		}
	}
	return id, title
}

// whenSave envía el formulario de la fila y, si el guardado redirige, abre la pantalla resultante.
func (a *hu17BDDApp) whenSave(t *testing.T, storyID int64, hours string) (*http.Response, string) {
	t.Helper()
	resp, _ := a.do(t, http.MethodPost, "/estimaciones", url.Values{
		"historia_id":     {fmt.Sprintf("%d", storyID)},
		"horas_estimadas": {hours},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/estimaciones" {
		t.Fatalf("Guardar: estado/location = %d/%q; se esperaba 303 /estimaciones",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	page, body := a.do(t, http.MethodGet, "/estimaciones", nil)
	if page.StatusCode != http.StatusOK {
		t.Fatalf("GET /estimaciones: estado %d; se esperaba 200", page.StatusCode)
	}
	return page, body
}

func (a *hu17BDDApp) stored(t *testing.T, query string, storyID int64) string {
	t.Helper()
	var value sql.NullString
	if err := a.db.QueryRow(query, storyID).Scan(&value); err != nil {
		t.Fatalf("leer la historia %d: %v", storyID, err)
	}
	return value.String
}

var hu17BDDRow = regexp.MustCompile(`(?s)<tr\b[^>]*>.*?</tr>`)

func hu17BDDThenRowShows(t *testing.T, body, title string, values ...string) {
	t.Helper()
	for _, row := range hu17BDDRow.FindAllString(body, -1) {
		if !strings.Contains(row, title) {
			continue
		}
		for _, v := range values {
			if !regexp.MustCompile(`>\s*` + regexp.QuoteMeta(v) + `\s*<`).MatchString(row) {
				t.Errorf("la historia %q debía mostrar %q: %s", title, v, row)
			}
		}
		return
	}
	t.Errorf("no se encontró la historia %q en la pantalla Estimaciones", title)
}

const (
	hu17BDDHoursSQL  = `SELECT estimated_hours::text FROM user_stories WHERE id = $1`
	hu17BDDActualSQL = `SELECT COALESCE(SUM(hours), 0)::text FROM effort_logs WHERE story_id = $1`
)

// Escenario: Estimación registrada.
func TestWebEstimatesBDD_EstimacionRegistrada(t *testing.T) {
	// Given una historia sin horas estimadas
	app := newHU17BDDApp(t)
	storyID, title := app.givenStory(t, "sin estimar", nil)

	// When ingreso "8" horas estimadas y presiono "Guardar"
	_, body := app.whenSave(t, storyID, "8")

	// Then veo "Horas estimadas registradas" y la historia muestra 8 horas estimadas
	if n := strings.Count(body, "Horas estimadas registradas"); n != 1 {
		t.Errorf("el mensaje debía aparecer una vez; apareció %d", n)
	}
	hu17BDDThenRowShows(t, body, title, "8")
	if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "8.00" {
		t.Errorf("PostgreSQL guardó %q; se esperaba 8.00", got)
	}
}

// Escenario: Modificar la estimación.
func TestWebEstimatesBDD_ModificarLaEstimacion(t *testing.T) {
	// Given una historia con horas estimadas y horas reales registradas
	app := newHU17BDDApp(t)
	previous := 8.0
	storyID, title := app.givenStory(t, "estimada", &previous, "2", "1.25")

	// When ingreso un valor nuevo válido y presiono "Guardar"
	_, body := app.whenSave(t, storyID, "10")

	// Then veo "Horas estimadas actualizadas", el valor reemplaza al anterior y las horas reales no cambian
	if n := strings.Count(body, "Horas estimadas actualizadas"); n != 1 {
		t.Errorf("el mensaje debía aparecer una vez; apareció %d", n)
	}
	hu17BDDThenRowShows(t, body, title, "10", "3,25")
	if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "10.00" {
		t.Errorf("PostgreSQL guardó %q; se esperaba 10.00", got)
	}
	if got := app.stored(t, hu17BDDActualSQL, storyID); got != "3.25" {
		t.Errorf("las horas reales cambiaron a %q; se esperaba 3.25", got)
	}
}

// Escenario: Horas con coma decimal (y con punto).
func TestWebEstimatesBDD_HorasConComaDecimal(t *testing.T) {
	for _, input := range []string{"1,5", "1.5"} {
		t.Run(input, func(t *testing.T) {
			// Given una historia
			app := newHU17BDDApp(t)
			storyID, title := app.givenStory(t, "decimal", nil)

			// When ingreso "1,5" o "1.5" horas estimadas y presiono "Guardar"
			_, body := app.whenSave(t, storyID, input)

			// Then se guarda 1,5 exacto, sin redondeo, y se muestra con coma
			if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "1.50" {
				t.Errorf("PostgreSQL guardó %q; se esperaba 1.50", got)
			}
			hu17BDDThenRowShows(t, body, title, "1,5")
			if regexp.MustCompile(`>\s*1\.5\s*<`).MatchString(body) {
				t.Error("las horas no debían mostrarse con punto")
			}
		})
	}
}
