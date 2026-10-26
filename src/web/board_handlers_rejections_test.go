package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T023 a T027 y T031): rechazos y errores del Tablero. Servidor, servicio y PostgreSQL
// reales; solo se reemplaza el guardado para simular una falla de almacenamiento.

// Textos de los escenarios del Issue; se definen en board_form.go.
const (
	tableroFueraDeSprint  = "La historia no está asignada a un Sprint activo"
	tableroNoExiste       = "La historia no existe"
	tableroAccionInvalida = "La acción no es válida"
	tableroYaEnHecho      = "La historia ya está en Hecho"
	tableroNoPermitida    = "No se puede pasar de En Sprint a Hecho"
	tableroDesactualizado = "El estado de la historia cambió; revise el estado actual y vuelva a intentarlo"
)

// failingStatusStore lee con el almacenamiento real pero falla al guardar, con un error técnico
// que no debe llegar a la pantalla.
type failingStatusStore struct{ sprint.StoryStatusStore }

func (failingStatusStore) UpdateStatus(context.Context, int64, sprint.StoryStatus, sprint.StoryStatus) (bool, error) {
	return false, errors.New(`pq: connection reset by peer; SELECT us.status FROM user_stories us at C:\internal\store.go:42`)
}

func withFailingSave() boardOption {
	return withStatusStore(func(real sprint.StoryStatusStore) sprint.StoryStatusStore {
		return failingStatusStore{StoryStatusStore: real}
	})
}

// wantRejected verifica una respuesta de rechazo: código, mensaje de error y ningún éxito.
func wantRejected(t *testing.T, resp *http.Response, body string, status int, message string) {
	t.Helper()
	if resp.StatusCode != status {
		t.Errorf("se esperaba el código %d, se obtuvo %d", status, resp.StatusCode)
	}
	if !strings.Contains(body, message) || !strings.Contains(body, "alert-error") {
		t.Errorf("debía verse el mensaje de error %q", message)
	}
	if strings.Contains(body, "alert-success") {
		t.Errorf("no debe mostrarse un mensaje de éxito")
	}
}

// FR-008 (SC-002): una historia sin Sprint o de un Sprint finalizado se rechaza sin cambios.
func TestTablero_HistoriaFueraDeUnSprintActivoSeRechaza(t *testing.T) {
	app := newBoardApp(t)
	projectID, _ := app.newBoardProject(t, "fuera de Sprint")
	finished, _ := newCatSprint(t, app.db, projectID, "Finalizado")
	casos := []struct {
		nombre string
		story  int64
		action sprint.Action
	}{
		{"Pendiente sin Sprint", newCatStory(t, app.db, projectID, 0, "Sin Sprint", boardPuntos(2), sprint.Pendiente), sprint.ActionStart},
		{"de un Sprint finalizado", newCatStory(t, app.db, projectID, finished, "De un Sprint finalizado", boardPuntos(3), sprint.Done), sprint.ActionReopen},
	}
	c := app.openBoard(t, projectID)
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			before := app.storyStatus(t, tc.story)

			resp, body := app.changeStory(t, c, tc.story, tc.action, "")

			wantRejected(t, resp, body, http.StatusConflict, tableroFueraDeSprint)
			if got := app.storyStatus(t, tc.story); got != before {
				t.Errorf("el estado cambió de %q a %q", before, got)
			}
		})
	}
}

// FR-014: una historia inexistente, no numérica o de otro proyecto "no existe".
func TestTablero_HistoriaInexistenteONoDelProyectoNoExiste(t *testing.T) {
	app := newBoardApp(t)
	projectA, sprintA := app.newBoardProject(t, "A")
	projectB, sprintB := app.newBoardProject(t, "B")
	newCatStory(t, app.db, projectA, sprintA, "Historia de A", boardPuntos(3), sprint.EnSprint)
	storyB := newCatStory(t, app.db, projectB, sprintB, "Historia de B", boardPuntos(5), sprint.EnSprint)
	c := app.openBoard(t, projectA)
	casos := []struct{ nombre, historia string }{
		{"inexistente", id64(MaxID)},
		{"no numérica", "abc"},
		{"vacía", ""},
		{"negativa", "-1"},
		{"de otro proyecto", id64(storyB)},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			resp, body := app.post(t, c, "/tablero/estado",
				url.Values{"historia_id": {tc.historia}, "accion": {string(sprint.ActionStart)}})

			wantRejected(t, resp, body, http.StatusUnprocessableEntity, tableroNoExiste)
		})
	}
	if got := app.storyStatus(t, storyB); got != sprint.EnSprint {
		t.Errorf("la historia del otro proyecto cambió a %q", got)
	}
}

// FR-014: el proyecto sale del contexto del servidor; un campo de proyecto en el formulario se ignora.
func TestTablero_ElProyectoDelFormularioSeIgnora(t *testing.T) {
	app := newBoardApp(t)
	projectA, sprintA := app.newBoardProject(t, "A")
	projectB, sprintB := app.newBoardProject(t, "B")
	storyA := newCatStory(t, app.db, projectA, sprintA, "Historia de A", boardPuntos(3), sprint.EnSprint)
	storyB := newCatStory(t, app.db, projectB, sprintB, "Historia de B", boardPuntos(5), sprint.EnSprint)
	c := app.openBoard(t, projectA)

	resp, _ := app.post(t, c, "/tablero/estado", url.Values{
		"historia_id": {id64(storyA)}, "accion": {string(sprint.ActionStart)}, "proyecto_id": {id64(projectB)},
	})

	wantSeeOtherToBoard(t, resp)
	if got := app.storyStatus(t, storyA); got != sprint.InProgress {
		t.Errorf("la historia del proyecto activo debía pasar a %q, quedó %q", sprint.InProgress, got)
	}
	if got := app.storyStatus(t, storyB); got != sprint.EnSprint {
		t.Errorf("la historia del otro proyecto cambió a %q", got)
	}
}

// FR-009 (SC-003): marcar dos veces no cambia nada ni duplica los Story Points.
func TestTablero_MarcarDosVecesNoCambiaNiDuplicaPuntos(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "dos veces")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia hecha", boardPuntos(8), sprint.Done)
	newCatStory(t, app.db, projectID, sprintID, "Otra historia", boardPuntos(5), sprint.EnSprint)
	c := app.openBoard(t, projectID)
	casos := []struct {
		nombre   string
		expected sprint.StoryStatus
	}{
		{"con el estado que se veía", sprint.InProgress},
		{"sin estado visto", ""},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			resp, _ := app.changeStory(t, c, story, sprint.ActionComplete, tc.expected)

			wantSeeOtherToBoard(t, resp)
			_, body := app.get(t, c, "/tablero")
			if !strings.Contains(body, tableroYaEnHecho) || !strings.Contains(body, "alert-warning") {
				t.Errorf("debía verse la advertencia %q", tableroYaEnHecho)
			}
			if strings.Contains(body, "alert-success") {
				t.Errorf("no debe mostrarse un mensaje de éxito")
			}
			if !strings.Contains(body, "8 de 13") {
				t.Errorf("los Story Points completados no debían duplicarse (8 de 13)")
			}
			if got := app.storyStatus(t, story); got != sprint.Done {
				t.Errorf("estado = %q, se esperaba %q", got, sprint.Done)
			}
		})
	}
}

// FR-010, FR-015: una transición no permitida o una acción desconocida se rechazan sin cambios.
func TestTablero_TransicionNoPermitidaYAccionDesconocida(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "transiciones")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia En Sprint", boardPuntos(3), sprint.EnSprint)
	c := app.openBoard(t, projectID)

	t.Run("completar una historia En Sprint", func(t *testing.T) {
		resp, body := app.changeStory(t, c, story, sprint.ActionComplete, sprint.EnSprint)

		wantRejected(t, resp, body, http.StatusConflict, tableroNoPermitida)
	})
	for _, accion := range []string{"borrar", "", "COMENZAR"} {
		t.Run("acción "+accion, func(t *testing.T) {
			resp, body := app.post(t, c, "/tablero/estado",
				url.Values{"historia_id": {id64(story)}, "accion": {accion}})

			wantRejected(t, resp, body, http.StatusUnprocessableEntity, tableroAccionInvalida)
		})
	}
	if got := app.storyStatus(t, story); got != sprint.EnSprint {
		t.Errorf("el estado cambió a %q", got)
	}
}

// FR-011 (SC-002): si el estado cambió por otro camino, no se cambia nada.
func TestTablero_EstadoDesactualizadoNoCambiaNada(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "desactualizado")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia En Sprint", boardPuntos(3), sprint.EnSprint)
	c := app.openBoard(t, projectID)
	// Otra persona la lleva a Hecho mientras el Tablero seguía abierto.
	if _, err := app.db.Exec(`UPDATE user_stories SET status = 'Hecho' WHERE id = $1`, story); err != nil {
		t.Fatalf("no se pudo simular el cambio de otra persona: %v", err)
	}

	resp, body := app.changeStory(t, c, story, sprint.ActionStart, sprint.EnSprint)

	wantRejected(t, resp, body, http.StatusConflict, tableroDesactualizado)
	if got := app.storyStatus(t, story); got != sprint.Done {
		t.Errorf("estado = %q, se esperaba %q (el de la otra persona)", got, sprint.Done)
	}
}

// FR-011: la acción define su estado de origen aunque no se envíe el estado visto.
func TestTablero_LaAccionDebeCorresponderAlEstadoRealSinEstadoVisto(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "origen de la acción")
	c := app.openBoard(t, projectID)
	casos := []struct {
		nombre string
		real   sprint.StoryStatus
		action sprint.Action
	}{
		{"comenzar sobre una historia Hecho", sprint.Done, sprint.ActionStart},
		{"reabrir sobre una historia En Sprint", sprint.EnSprint, sprint.ActionReopen},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			story := newCatStory(t, app.db, projectID, sprintID, tc.nombre, boardPuntos(3), tc.real)

			resp, body := app.changeStory(t, c, story, tc.action, "")

			wantRejected(t, resp, body, http.StatusConflict, tableroDesactualizado)
			if got := app.storyStatus(t, story); got != tc.real {
				t.Errorf("el estado cambió de %q a %q", tc.real, got)
			}
		})
	}
}

// FR-012 (SC-004): una falla de guardado no muestra éxito y no cambia el estado.
func TestTablero_FallaDeGuardadoNoMuestraExitoNiCambiaElEstado(t *testing.T) {
	app := newBoardApp(t, withFailingSave())
	projectID, sprintID := app.newBoardProject(t, "falla de guardado")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia En progreso", boardPuntos(3), sprint.InProgress)
	c := app.openBoard(t, projectID)

	resp, body := app.changeStory(t, c, story, sprint.ActionComplete, sprint.InProgress)

	wantRejected(t, resp, body, http.StatusInternalServerError, MsgOperationFailed)
	if got := app.storyStatus(t, story); got != sprint.InProgress {
		t.Errorf("estado = %q, se esperaba %q", got, sprint.InProgress)
	}
}

// FR-018: ninguna respuesta de error expone consultas, rutas ni detalles del almacenamiento.
func TestTablero_LasRespuestasDeErrorNoExponenDetallesInternos(t *testing.T) {
	app := newBoardApp(t, withFailingSave())
	projectID, sprintID := app.newBoardProject(t, "sin detalles")
	enProgreso := newCatStory(t, app.db, projectID, sprintID, "Historia En progreso", boardPuntos(3), sprint.InProgress)
	enSprint := newCatStory(t, app.db, projectID, sprintID, "Historia En Sprint", boardPuntos(2), sprint.EnSprint)
	c := app.openBoard(t, projectID)

	respuestas := map[string]string{}
	_, respuestas["falla de guardado"] = app.changeStory(t, c, enProgreso, sprint.ActionComplete, sprint.InProgress)
	_, respuestas["historia inexistente"] = app.post(t, c, "/tablero/estado",
		url.Values{"historia_id": {id64(MaxID)}, "accion": {string(sprint.ActionStart)}})
	_, respuestas["transición no permitida"] = app.changeStory(t, c, enSprint, sprint.ActionComplete, sprint.EnSprint)

	prohibidos := []string{"select ", "user_stories", "pq:", "pgx", "sql:", ".go:", `c:\`, "internal", "connection reset"}
	for nombre, body := range respuestas {
		bajo := strings.ToLower(body)
		for _, p := range prohibidos {
			if strings.Contains(bajo, p) {
				t.Errorf("%s: la página expone un detalle interno (contiene %q)", nombre, p)
			}
		}
	}
}
