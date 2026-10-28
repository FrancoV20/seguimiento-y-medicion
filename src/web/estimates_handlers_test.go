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
	hu17MsgInvalid      = "El valor de horas no es válido"
	hu17MsgTooLarge     = "Las horas estimadas no pueden superar 9999,99"
	hu17MsgNotFound     = "La historia no existe"
	hu17MsgLocked       = "La historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar"
	hu17MsgFailed       = "La operación no pudo completarse"
	hu17StoryPointsNone = 3 // Story Points de las historias de prueba: no se confunden con las horas
)

type hu17EstimatesApp struct {
	*ctxApp
	client *http.Client
	stamp  int64
}

func newHU17EstimatesApp(t *testing.T) *hu17EstimatesApp {
	return newHU17EstimatesAppWithStore(t, nil)
}

// newHU17EstimatesAppWithStore permite envolver el almacenamiento real (por ejemplo, para simular
// una falla de guardado).
func newHU17EstimatesAppWithStore(t *testing.T, wrap func(backlog.EstimateStore) backlog.EstimateStore) *hu17EstimatesApp {
	t.Helper()
	app := newCtxApp(t, nil)
	var store backlog.EstimateStore = backlog.NewPostgreSQLEstimateStore(app.db)
	if wrap != nil {
		store = wrap(store)
	}
	if err := RegisterEstimates(app.srv,
		backlog.NewEstimateService(store),
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

// --- US2: rechazos y errores (T020 a T024) ---

// finishSprintWith asigna la historia a un Sprint finalizado del proyecto.
func (a *hu17EstimatesApp) finishSprintWith(t *testing.T, projectID, storyID int64) {
	t.Helper()
	var sprintID int64
	if err := a.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 1, 'Sprint finalizado HU-17', DATE '2026-10-01', DATE '2026-10-15', 'Finalizado')
		RETURNING id`, projectID).Scan(&sprintID); err != nil {
		t.Fatalf("no se pudo crear el Sprint finalizado: %v", err)
	}
	if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, storyID); err != nil {
		t.Fatalf("no se pudo asignar la historia al Sprint finalizado: %v", err)
	}
	if _, err := a.db.Exec(`UPDATE user_stories SET status = 'Hecho', active_sprint_id = $1 WHERE id = $2`, sprintID, storyID); err != nil {
		t.Fatalf("no se pudo marcar la historia: %v", err)
	}
}

var hu17HoursInputPattern = regexp.MustCompile(`<input\b[^>]*\bname="horas_estimadas"[^>]*>`)
var hu17ValueAttrPattern = regexp.MustCompile(`\bvalue="([^"]*)"`)

// hu17InputValue devuelve el valor del campo de horas de la fila (sin escapes HTML).
func hu17InputValue(t *testing.T, row string) string {
	t.Helper()
	input := hu17HoursInputPattern.FindString(row)
	if input == "" {
		t.Fatalf("la fila no tiene el campo horas_estimadas: %s", row)
	}
	match := hu17ValueAttrPattern.FindStringSubmatch(input)
	if match == nil {
		return ""
	}
	return html.UnescapeString(match[1])
}

// hu17AssertRejected verifica el estado, el mensaje una sola vez y la ausencia de éxito.
func hu17AssertRejected(t *testing.T, resp *http.Response, body string, status int, message string) {
	t.Helper()
	if resp.StatusCode != status {
		t.Errorf("estado = %d; se esperaba %d", resp.StatusCode, status)
	}
	if n := strings.Count(body, message); n != 1 {
		t.Errorf("el mensaje %q debía aparecer una vez; apareció %d", message, n)
	}
	for _, success := range []string{hu17MsgRegistered, hu17MsgUpdated} {
		if strings.Contains(body, success) {
			t.Errorf("un rechazo no debía mostrar %q", success)
		}
	}
}

// hu17AssertFieldError verifica el mensaje en la fila de la historia, lo escrito conservado en su
// campo y el valor anterior todavía mostrado.
func hu17AssertFieldError(t *testing.T, body, title, message, written, shownBefore string) {
	t.Helper()
	row := hu17StoryRow(t, body, title)
	if !strings.Contains(row, message) {
		t.Errorf("el mensaje %q debía estar junto al campo de la fila: %s", message, row)
	}
	if got := hu17InputValue(t, row); got != written {
		t.Errorf("el campo debía conservar lo escrito %q; tiene %q", written, got)
	}
	hu17AssertCell(t, row, shownBefore, "las horas estimadas anteriores")
}

// T020
func TestEstimatesHandler_RechazaHorasInvalidasJuntoAlCampo(t *testing.T) {
	app := newHU17EstimatesApp(t)
	previous := 8.0
	storyID, title := app.insertStory(t, app.fx.projectA, "invalidas", &previous)
	other, otherTitle := app.insertStory(t, app.fx.projectA, "vecina", &previous)
	app.chooseProject(t, app.fx.projectA)

	for _, input := range []string{"0", "-1", "abc", "", "1,555"} {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) {
			resp, body := app.save(t, storyID, input)
			hu17AssertRejected(t, resp, body, http.StatusUnprocessableEntity, hu17MsgInvalid)
			hu17AssertFieldError(t, body, title, hu17MsgInvalid, input, "8")
			if strings.Contains(hu17StoryRow(t, body, otherTitle), hu17MsgInvalid) {
				t.Error("el mensaje no debía aparecer en la fila de otra historia")
			}
			if got := app.storedHours(t, storyID); got != "8.00" {
				t.Errorf("el valor anterior cambió a %q", got)
			}
		})
	}
	if got := app.storedHours(t, other); got != "8.00" {
		t.Errorf("la otra historia cambió a %q", got)
	}
}

// T021 (9999,995 tiene más de dos decimales: se informa como valor no válido, ver data-model.md)
func TestEstimatesHandler_RechazaHorasPorEncimaDelMaximo(t *testing.T) {
	app := newHU17EstimatesApp(t)
	previous := 8.0
	storyID, title := app.insertStory(t, app.fx.projectA, "maximo", &previous)
	app.chooseProject(t, app.fx.projectA)

	cases := []struct{ input, message string }{
		{"10000", hu17MsgTooLarge},
		{"12345,5", hu17MsgTooLarge},
		{"9999,995", hu17MsgInvalid},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			resp, body := app.save(t, storyID, c.input)
			hu17AssertRejected(t, resp, body, http.StatusUnprocessableEntity, c.message)
			hu17AssertFieldError(t, body, title, c.message, c.input, "8")
			if got := app.storedHours(t, storyID); got != "8.00" {
				t.Errorf("el valor anterior cambió a %q", got)
			}
		})
	}
}

// T022
func TestEstimatesHandler_RechazaHistoriaDeSprintFinalizado(t *testing.T) {
	app := newHU17EstimatesApp(t)
	previous := 5.0
	storyID, title := app.insertStory(t, app.fx.projectA, "finalizada", &previous)
	app.finishSprintWith(t, app.fx.projectA, storyID)
	app.chooseProject(t, app.fx.projectA)

	// Envío directo, aunque la pantalla no ofrezca el campo editable.
	resp, body := app.save(t, storyID, "9")
	hu17AssertRejected(t, resp, body, http.StatusConflict, hu17MsgLocked)
	row := hu17StoryRow(t, body, title)
	if !strings.Contains(row, hu17MsgLocked) {
		t.Errorf("el motivo debía mostrarse en la fila de la historia: %s", row)
	}
	hu17AssertCell(t, row, "5", "las horas estimadas sin cambios")
	if got := app.storedHours(t, storyID); got != "5.00" {
		t.Errorf("la historia bloqueada cambió a %q", got)
	}
}

// T023
func TestEstimatesHandler_RechazaHistoriaInexistenteOAjena(t *testing.T) {
	app := newHU17EstimatesApp(t)
	foreignHours := 6.0
	foreign, foreignTitle := app.insertStory(t, app.fx.projectB, "ajena", &foreignHours)
	app.chooseProject(t, app.fx.projectA)

	cases := []struct {
		name   string
		values url.Values
	}{
		{"no numérica", url.Values{hu17FieldStoryID: {"abc"}, hu17FieldHours: {"3"}}},
		{"cero", url.Values{hu17FieldStoryID: {"0"}, hu17FieldHours: {"3"}}},
		{"vacía", url.Values{hu17FieldHours: {"3"}}},
		{"inexistente", url.Values{hu17FieldStoryID: {"9223372036854775807"}, hu17FieldHours: {"3"}}},
		{"de otro proyecto", url.Values{hu17FieldStoryID: {id64(foreign)}, hu17FieldHours: {"3"}}},
		{"con campo de proyecto", url.Values{
			hu17FieldStoryID: {id64(foreign)}, hu17FieldHours: {"3"},
			"proyecto_id": {id64(app.fx.projectB)}, "project_id": {id64(app.fx.projectB)},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := app.post(t, app.client, hu17PathEstimates, c.values)
			hu17AssertRejected(t, resp, body, http.StatusUnprocessableEntity, hu17MsgNotFound)
			if strings.Contains(body, foreignTitle) {
				t.Error("la respuesta no debía mostrar historias de otro proyecto")
			}
		})
	}
	if got := app.storedHours(t, foreign); got != "6.00" {
		t.Errorf("la historia del otro proyecto cambió a %q", got)
	}
}

// hu17FailingSaveStore lee con el almacenamiento real y falla al guardar.
type hu17FailingSaveStore struct {
	backlog.EstimateStore
}

func (hu17FailingSaveStore) UpdateEstimate(context.Context, int64, int64, float64) (bool, error) {
	return false, errors.New("simulated database failure: UPDATE user_stories")
}

// T024
func TestEstimatesHandler_FallaDeGuardadoConservaElValorAnterior(t *testing.T) {
	app := newHU17EstimatesAppWithStore(t, func(real backlog.EstimateStore) backlog.EstimateStore {
		return hu17FailingSaveStore{EstimateStore: real}
	})
	previous := 8.0
	storyID, title := app.insertStory(t, app.fx.projectA, "falla", &previous)
	app.chooseProject(t, app.fx.projectA)

	resp, body := app.save(t, storyID, "15")
	hu17AssertRejected(t, resp, body, http.StatusInternalServerError, hu17MsgFailed)
	row := hu17StoryRow(t, body, title)
	hu17AssertCell(t, row, "8", "las horas estimadas anteriores")
	if got := hu17InputValue(t, row); got != "15" {
		t.Errorf("el campo debía conservar lo escrito \"15\"; tiene %q", got)
	}
	if got := app.storedHours(t, storyID); got != "8.00" {
		t.Errorf("el valor anterior cambió a %q", got)
	}
	if strings.Contains(body, "simulated database failure") || strings.Contains(body, "UPDATE") {
		t.Error("la respuesta no debía exponer el detalle interno de la falla")
	}
}
