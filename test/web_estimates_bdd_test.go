package test

import (
	"context"
	"database/sql"
	"errors"
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
	return newHU17BDDAppWithStore(t, nil)
}

// newHU17BDDAppWithStore permite envolver el almacenamiento real para simular fallas.
func newHU17BDDAppWithStore(t *testing.T, wrap func(backlog.EstimateStore) backlog.EstimateStore) *hu17BDDApp {
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
	var store backlog.EstimateStore = backlog.NewPostgreSQLEstimateStore(db)
	if wrap != nil {
		store = wrap(store)
	}
	if err := web.RegisterEstimates(server,
		backlog.NewEstimateService(store),
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

// --- US2: rechazos y errores (T025) ---

// whenSubmit envía el formulario de la fila y devuelve la respuesta tal como llega (sin seguir
// redirecciones).
func (a *hu17BDDApp) whenSubmit(t *testing.T, storyID int64, hours string) (*http.Response, string) {
	t.Helper()
	return a.do(t, http.MethodPost, "/estimaciones", url.Values{
		"historia_id":     {fmt.Sprintf("%d", storyID)},
		"horas_estimadas": {hours},
	})
}

func (a *hu17BDDApp) givenFinishedSprintWith(t *testing.T, storyID int64) {
	t.Helper()
	var sprintID int64
	if err := a.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Sprint BDD HU-17', DATE '2026-10-01', DATE '2026-10-15', 'Finalizado') RETURNING id`,
		a.projectID).Scan(&sprintID); err != nil {
		t.Fatalf("crear el Sprint finalizado: %v", err)
	}
	if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID); err != nil {
		t.Fatalf("asignar la historia al Sprint finalizado: %v", err)
	}
}

var hu17BDDHoursInput = regexp.MustCompile(`<input\b[^>]*\bname="horas_estimadas"[^>]*\bvalue="([^"]*)"`)

// hu17BDDThenRejectedInRow verifica estado, mensaje en la fila, lo escrito conservado y sin éxito.
func hu17BDDThenRejectedInRow(t *testing.T, resp *http.Response, body, title string, status int, message, written string) {
	t.Helper()
	if resp.StatusCode != status {
		t.Errorf("estado = %d; se esperaba %d", resp.StatusCode, status)
	}
	if strings.Contains(body, "Horas estimadas registradas") || strings.Contains(body, "Horas estimadas actualizadas") {
		t.Error("un rechazo no debía mostrar un mensaje de éxito")
	}
	for _, row := range hu17BDDRow.FindAllString(body, -1) {
		if !strings.Contains(row, title) {
			continue
		}
		if !strings.Contains(row, message) {
			t.Errorf("la fila debía mostrar %q: %s", message, row)
		}
		if written != "" {
			if m := hu17BDDHoursInput.FindStringSubmatch(row); m == nil || m[1] != written {
				t.Errorf("el campo debía conservar lo escrito %q: %s", written, row)
			}
		}
		return
	}
	t.Errorf("no se encontró la historia %q en la respuesta", title)
}

// Escenario: Horas inválidas.
func TestWebEstimatesBDD_HorasInvalidas(t *testing.T) {
	for _, input := range []string{"-1", "0", "abc", "1,555"} {
		t.Run(input, func(t *testing.T) {
			// Given una historia con horas estimadas
			app := newHU17BDDApp(t)
			previous := 8.0
			storyID, title := app.givenStory(t, "invalida", &previous)

			// When ingreso un valor negativo, cero, no numérico o con más de dos decimales
			resp, body := app.whenSubmit(t, storyID, input)

			// Then veo "El valor de horas no es válido" junto al campo, el valor anterior no cambia
			// y lo que escribí se conserva
			hu17BDDThenRejectedInRow(t, resp, body, title, http.StatusUnprocessableEntity, "El valor de horas no es válido", input)
			if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "8.00" {
				t.Errorf("el valor anterior cambió a %q", got)
			}
		})
	}
}

// Escenario: Horas por encima del máximo.
func TestWebEstimatesBDD_HorasPorEncimaDelMaximo(t *testing.T) {
	// Given una historia
	app := newHU17BDDApp(t)
	previous := 8.0
	storyID, title := app.givenStory(t, "maximo", &previous)

	// When ingreso un valor mayor que 9999,99
	resp, body := app.whenSubmit(t, storyID, "10000")

	// Then veo "Las horas estimadas no pueden superar 9999,99" junto al campo y el valor anterior no cambia
	hu17BDDThenRejectedInRow(t, resp, body, title, http.StatusUnprocessableEntity, "Las horas estimadas no pueden superar 9999,99", "10000")
	if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "8.00" {
		t.Errorf("el valor anterior cambió a %q", got)
	}
}

// Escenario: Historia de un Sprint finalizado.
func TestWebEstimatesBDD_HistoriaDeSprintFinalizado(t *testing.T) {
	// Given una historia de un Sprint finalizado
	app := newHU17BDDApp(t)
	previous := 5.0
	storyID, title := app.givenStory(t, "finalizada", &previous)
	app.givenFinishedSprintWith(t, storyID)

	// When se envía el cambio de sus horas estimadas
	resp, body := app.whenSubmit(t, storyID, "9")

	// Then se bloquea con el motivo y el valor no cambia
	hu17BDDThenRejectedInRow(t, resp, body, title, http.StatusConflict,
		"La historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar", "")
	if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "5.00" {
		t.Errorf("la historia bloqueada cambió a %q", got)
	}
}

type hu17BDDFailingSaveStore struct {
	backlog.EstimateStore
}

func (hu17BDDFailingSaveStore) UpdateEstimate(context.Context, int64, int64, float64) (bool, error) {
	return false, errors.New("simulated database failure")
}

// Escenario: Error al guardar.
func TestWebEstimatesBDD_ErrorAlGuardar(t *testing.T) {
	// Given un valor válido y un almacenamiento que falla al guardar
	app := newHU17BDDAppWithStore(t, func(real backlog.EstimateStore) backlog.EstimateStore {
		return hu17BDDFailingSaveStore{EstimateStore: real}
	})
	previous := 8.0
	storyID, title := app.givenStory(t, "falla", &previous)

	// When falla el guardado
	resp, body := app.whenSubmit(t, storyID, "15")

	// Then veo "La operación no pudo completarse", el valor anterior se conserva y no veo éxito
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("estado = %d; se esperaba 500", resp.StatusCode)
	}
	if n := strings.Count(body, "La operación no pudo completarse"); n != 1 {
		t.Errorf("el mensaje debía aparecer una vez; apareció %d", n)
	}
	hu17BDDThenRejectedInRow(t, resp, body, title, http.StatusInternalServerError, "", "15")
	hu17BDDThenRowShows(t, body, title, "8")
	if got := app.stored(t, hu17BDDHoursSQL, storyID); got != "8.00" {
		t.Errorf("el valor anterior cambió a %q", got)
	}
}

// --- US3: ver las estimaciones (T033) ---

func (a *hu17BDDApp) givenStoryWith(t *testing.T, name string, points *int, hours *float64, status string, effort ...string) (int64, string) {
	t.Helper()
	title := fmt.Sprintf("HU17 BDD %s %d", name, a.stamp)
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours, status)
		VALUES ($1, $2, 'Historia BDD de HU-17', 'Media', $3, $4, $5) RETURNING id`,
		a.projectID, title, points, hours, status).Scan(&id); err != nil {
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

func (a *hu17BDDApp) whenOpenEstimates(t *testing.T) string {
	t.Helper()
	resp, body := a.do(t, http.MethodGet, "/estimaciones", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("abrir Estimaciones: estado %d; se esperaba 200", resp.StatusCode)
	}
	return body
}

func hu17BDDRowOf(t *testing.T, body, title string) string {
	t.Helper()
	for _, row := range hu17BDDRow.FindAllString(body, -1) {
		if strings.Contains(row, title) {
			return row
		}
	}
	t.Fatalf("no se encontró la historia %q en la pantalla Estimaciones", title)
	return ""
}

// Escenario: Lista de estimaciones.
func TestWebEstimatesBDD_ListaDeEstimaciones(t *testing.T) {
	// Given un proyecto con historias
	app := newHU17BDDApp(t)
	five, eight := 5, 8.0
	_, title := app.givenStoryWith(t, "lista", &five, &eight, "En progreso", "1.5", "0.5")

	// When abro "Estimaciones"
	body := app.whenOpenEstimates(t)

	// Then veo cada historia con su título, estado, Story Points, horas estimadas y horas reales
	hu17BDDThenRowShows(t, body, title, "En progreso", "5", "8", "2")
	if strings.Contains(body, "Sin estimar") {
		t.Error("una historia estimada no debía mostrarse como \"Sin estimar\"")
	}
}

// Escenario: Historia sin estimar.
func TestWebEstimatesBDD_HistoriaSinEstimar(t *testing.T) {
	// Given una historia sin horas estimadas
	app := newHU17BDDApp(t)
	five := 5
	_, title := app.givenStoryWith(t, "sin estimar", &five, nil, "Pendiente")

	// When abro "Estimaciones"
	body := app.whenOpenEstimates(t)

	// Then la veo como "Sin estimar" y no como cero (las horas reales sí se muestran como 0)
	row := hu17BDDRowOf(t, body, title)
	hu17BDDThenRowShows(t, body, title, "Sin estimar", "5", "0")
	if n := len(regexp.MustCompile(`>\s*0\s*<`).FindAllString(row, -1)); n != 1 {
		t.Errorf("solo las horas reales debían mostrarse como 0; apareció %d veces: %s", n, row)
	}
}

// Escenario: Historia bloqueada en la lista.
func TestWebEstimatesBDD_HistoriaBloqueadaEnLaLista(t *testing.T) {
	// Given una historia de un Sprint finalizado
	app := newHU17BDDApp(t)
	five := 5.0
	storyID, title := app.givenStoryWith(t, "bloqueada", nil, &five, "Hecho")
	app.givenFinishedSprintWith(t, storyID)

	// When abro "Estimaciones"
	body := app.whenOpenEstimates(t)

	// Then su campo de horas aparece deshabilitado con el motivo "Sprint finalizado"
	row := hu17BDDRowOf(t, body, title)
	if !regexp.MustCompile(`<input\b[^>]*\bname="horas_estimadas"[^>]*\bdisabled\b`).MatchString(row) {
		t.Errorf("el campo de horas debía estar deshabilitado: %s", row)
	}
	if !strings.Contains(row, "Sprint finalizado") {
		t.Errorf("la fila debía mostrar el motivo \"Sprint finalizado\": %s", row)
	}
}

// Escenario: Sin historias.
func TestWebEstimatesBDD_SinHistorias(t *testing.T) {
	// Given un proyecto sin historias
	app := newHU17BDDApp(t)

	// When abro "Estimaciones"
	body := app.whenOpenEstimates(t)

	// Then veo "Aún no hay historias"
	if n := strings.Count(body, "Aún no hay historias"); n != 1 {
		t.Errorf("el estado vacío debía aparecer una vez; apareció %d", n)
	}
	if strings.Contains(body, `name="horas_estimadas"`) {
		t.Error("sin historias no debían mostrarse campos de horas")
	}
}
