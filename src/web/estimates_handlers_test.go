package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// Pruebas de handler de la pantalla "Estimaciones" (HU-17, US1: T012 a T014). Usan el servidor
// real con PostgreSQL, el proyecto activo elegido por el flujo de HU-20b y un navegador con cookies.
// Los textos se toman de spec.md.

const (
	hu17PathEstimates   = "/estimaciones"
	hu17FieldStoryID    = "historia_id"
	hu17FieldHours      = "horas_estimadas"
	hu17MsgRegistered   = "Horas estimadas registradas"
	hu17MsgUpdated      = "Horas estimadas actualizadas"
	hu17StoryPointsNone = 3 // Story Points de las historias de prueba: no se confunden con las horas
)

type hu17EstimatesApp struct {
	*ctxApp
	client *http.Client
	stamp  int64
}

func newHU17EstimatesApp(t *testing.T) *hu17EstimatesApp {
	t.Helper()
	app := newCtxApp(t, nil)
	if err := RegisterEstimates(app.srv,
		backlog.NewEstimateService(backlog.NewPostgreSQLEstimateStore(app.db)),
		NewPostgreSQLEstimateCatalog(app.db)); err != nil {
		t.Fatalf("no se pudo registrar la sección Estimaciones: %v", err)
	}
	// Corre antes que la limpieza del fixture de contexto: libera horas e historias.
	t.Cleanup(func() {
		for _, projectID := range []int64{app.fx.projectA, app.fx.projectB} {
			_, _ = app.db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, projectID)
			_, _ = app.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, projectID)
		}
	})
	return &hu17EstimatesApp{ctxApp: app, client: app.client(false), stamp: time.Now().UnixNano()}
}

func (a *hu17EstimatesApp) chooseProject(t *testing.T, projectID int64) {
	t.Helper()
	resp, _ := a.selectProject(t, a.client, id64(projectID), hu17PathEstimates)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != hu17PathEstimates {
		t.Fatalf("elegir proyecto: estado/location = %d/%q; se esperaba 303 %s",
			resp.StatusCode, resp.Header.Get("Location"), hu17PathEstimates)
	}
}

// insertStory crea una historia con horas estimadas opcionales y devuelve su id y título.
func (a *hu17EstimatesApp) insertStory(t *testing.T, projectID int64, name string, hours *float64) (int64, string) {
	t.Helper()
	title := fmt.Sprintf("HU17 %s %d", name, a.stamp)
	var id int64
	if err := a.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, estimated_hours)
		VALUES ($1, $2, 'Historia de prueba HU-17', 'Media', $3, $4) RETURNING id`,
		projectID, title, hu17StoryPointsNone, hours).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear la historia %q: %v", title, err)
	}
	return id, title
}

func (a *hu17EstimatesApp) addEffort(t *testing.T, storyID int64, hours ...string) {
	t.Helper()
	for _, h := range hours {
		if _, err := a.db.Exec(`
			INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
			VALUES ($1, $2, CURRENT_DATE, 'Trabajo HU-17', $3::numeric)`, storyID, a.fx.ana, h); err != nil {
			t.Fatalf("no se pudieron registrar horas reales: %v", err)
		}
	}
}

func (a *hu17EstimatesApp) save(t *testing.T, storyID int64, hours string) (*http.Response, string) {
	t.Helper()
	return a.post(t, a.client, hu17PathEstimates, url.Values{
		hu17FieldStoryID: {id64(storyID)},
		hu17FieldHours:   {hours},
	})
}

func (a *hu17EstimatesApp) page(t *testing.T) (*http.Response, string) {
	t.Helper()
	return a.get(t, a.client, hu17PathEstimates)
}

// storedHours devuelve estimated_hours como texto exacto de NUMERIC ("" si es NULL).
func (a *hu17EstimatesApp) storedHours(t *testing.T, storyID int64) string {
	t.Helper()
	var h sql.NullString
	if err := a.db.QueryRow(`SELECT estimated_hours::text FROM user_stories WHERE id = $1`, storyID).Scan(&h); err != nil {
		t.Fatalf("no se pudieron leer las horas de %d: %v", storyID, err)
	}
	return h.String
}

func (a *hu17EstimatesApp) actualHours(t *testing.T, storyID int64) string {
	t.Helper()
	var total string
	if err := a.db.QueryRow(`SELECT COALESCE(SUM(hours), 0)::text FROM effort_logs WHERE story_id = $1`, storyID).Scan(&total); err != nil {
		t.Fatalf("no se pudieron sumar las horas reales de %d: %v", storyID, err)
	}
	return total
}

// hu17AssertSaved verifica el Post/Redirect/Get: 303 a /estimaciones y el mensaje una sola vez.
func hu17AssertSaved(t *testing.T, app *hu17EstimatesApp, resp *http.Response, message string) string {
	t.Helper()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != hu17PathEstimates {
		t.Fatalf("guardar: estado/location = %d/%q; se esperaba 303 %s",
			resp.StatusCode, resp.Header.Get("Location"), hu17PathEstimates)
	}
	pageResp, body := app.page(t)
	if pageResp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: estado %d; se esperaba 200", hu17PathEstimates, pageResp.StatusCode)
	}
	if n := strings.Count(body, message); n != 1 {
		t.Errorf("el mensaje %q debía aparecer una vez; apareció %d", message, n)
	}
	return body
}

var hu17RowPattern = regexp.MustCompile(`(?s)<tr\b[^>]*>.*?</tr>`)

// hu17StoryRow devuelve la fila de la tabla que contiene el título de la historia.
func hu17StoryRow(t *testing.T, body, title string) string {
	t.Helper()
	for _, row := range hu17RowPattern.FindAllString(body, -1) {
		if strings.Contains(row, title) {
			return row
		}
	}
	t.Fatalf("no se encontró la fila de la historia %q en la pantalla Estimaciones", title)
	return ""
}

// hu17AssertCell verifica que la fila tenga una celda o elemento cuyo texto sea exactamente value.
func hu17AssertCell(t *testing.T, row, value, what string) {
	t.Helper()
	pattern := regexp.MustCompile(`>\s*` + regexp.QuoteMeta(value) + `\s*<`)
	if !pattern.MatchString(row) {
		t.Errorf("la fila debía mostrar %s %q: %s", what, value, row)
	}
}

// T012: registrar horas en una historia sin estimar, solo en el proyecto activo.
func TestEstimatesHandler_RegistraHorasEnElProyectoActivo(t *testing.T) {
	app := newHU17EstimatesApp(t)
	storyA, titleA := app.insertStory(t, app.fx.projectA, "sin estimar A", nil)
	storyB, titleB := app.insertStory(t, app.fx.projectB, "sin estimar B", nil)
	app.chooseProject(t, app.fx.projectA)

	resp, _ := app.save(t, storyA, "8")
	body := hu17AssertSaved(t, app, resp, hu17MsgRegistered)

	if got := app.storedHours(t, storyA); got != "8.00" {
		t.Errorf("estimated_hours de la historia del proyecto activo = %q; se esperaba 8.00", got)
	}
	hu17AssertCell(t, hu17StoryRow(t, body, titleA), "8", "las horas estimadas")
	if got := app.storedHours(t, storyB); got != "" {
		t.Errorf("la historia del otro proyecto no debía cambiar: %q", got)
	}
	if strings.Contains(body, titleB) {
		t.Errorf("la pantalla no debía mostrar historias de otro proyecto (%q)", titleB)
	}
}

// T012: recargar después de guardar no repite el mensaje.
func TestEstimatesHandler_RecargarNoRepiteElMensaje(t *testing.T) {
	app := newHU17EstimatesApp(t)
	storyID, _ := app.insertStory(t, app.fx.projectA, "recarga", nil)
	app.chooseProject(t, app.fx.projectA)

	resp, _ := app.save(t, storyID, "5")
	hu17AssertSaved(t, app, resp, hu17MsgRegistered)
	_, again := app.page(t)
	if strings.Contains(again, hu17MsgRegistered) {
		t.Error("al recargar no debía repetirse el mensaje de éxito")
	}
}

// T013: modificar una estimación existente sin tocar las horas reales.
func TestEstimatesHandler_ModificaEstimacionSinCambiarHorasReales(t *testing.T) {
	app := newHU17EstimatesApp(t)
	previous := 8.0
	storyID, title := app.insertStory(t, app.fx.projectA, "estimada", &previous)
	app.addEffort(t, storyID, "1.5", "2.25")
	app.chooseProject(t, app.fx.projectA)

	resp, _ := app.save(t, storyID, "12,5")
	body := hu17AssertSaved(t, app, resp, hu17MsgUpdated)

	if got := app.storedHours(t, storyID); got != "12.50" {
		t.Errorf("estimated_hours = %q; se esperaba 12.50 (reemplaza a 8)", got)
	}
	if got := app.actualHours(t, storyID); got != "3.75" {
		t.Errorf("las horas reales cambiaron: %q; se esperaba 3.75", got)
	}
	row := hu17StoryRow(t, body, title)
	hu17AssertCell(t, row, "12,5", "las horas estimadas nuevas")
	hu17AssertCell(t, row, "3,75", "las horas reales")
	if regexp.MustCompile(`>\s*8\s*<`).MatchString(row) {
		t.Errorf("la fila no debía seguir mostrando el valor anterior 8: %s", row)
	}
}

// T014: valores válidos con coma y punto, guardados exactos y mostrados con coma.
func TestEstimatesHandler_GuardaHorasConComaYPuntoSinRedondear(t *testing.T) {
	cases := []struct {
		input, stored, shown string
	}{
		{"8", "8.00", "8"},
		{"1,5", "1.50", "1,5"},
		{"1.5", "1.50", "1,5"},
		{"9999,99", "9999.99", "9999,99"},
	}
	app := newHU17EstimatesApp(t)
	app.chooseProject(t, app.fx.projectA)
	for i, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			storyID, title := app.insertStory(t, app.fx.projectA, fmt.Sprintf("formato %d", i), nil)
			resp, _ := app.save(t, storyID, c.input)
			body := hu17AssertSaved(t, app, resp, hu17MsgRegistered)
			if got := app.storedHours(t, storyID); got != c.stored {
				t.Errorf("%q: estimated_hours = %q; se esperaba %q (exacto, sin redondeo)", c.input, got, c.stored)
			}
			hu17AssertCell(t, hu17StoryRow(t, body, title), c.shown, "las horas con coma")
		})
	}
}
