package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T044 y T045): comportamientos heredados de HU-20b aplicados al Tablero.

// FR-001 (patrón de HU-20b): recargar después de un cambio no lo repite ni repite el mensaje.
func TestTablero_RecargarDespuesDeUnCambioNoLoRepiteNiRepiteElMensaje(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "recarga")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia En Sprint", boardPuntos(3), sprint.EnSprint)
	c := app.openBoard(t, projectID)

	resp, _ := app.changeStory(t, c, story, sprint.ActionStart, sprint.EnSprint)

	wantSeeOtherToBoard(t, resp)
	if _, primera := app.get(t, c, "/tablero"); !strings.Contains(primera, tableroComenzado) {
		t.Fatalf("la primera pantalla debía mostrar %q", tableroComenzado)
	}
	for i := 0; i < 3; i++ { // recargar varias veces (F5)
		_, recarga := app.get(t, c, "/tablero")
		if strings.Contains(recarga, tableroComenzado) {
			t.Errorf("recarga %d: el mensaje no debía repetirse", i+1)
		}
		if got := app.storyStatus(t, story); got != sprint.InProgress {
			t.Errorf("recarga %d: estado = %q, se esperaba %q", i+1, got, sprint.InProgress)
		}
	}
}

// FR-014: sin proyecto activo válido se redirige a elegirlo y no se cambia nada.
func TestTablero_SinProyectoActivoValidoRedirigeYNoCambiaNada(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "sin proyecto")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia En Sprint", boardPuntos(3), sprint.EnSprint)
	casos := []struct {
		nombre  string
		cookies []*http.Cookie
	}{
		{"sin selección", nil},
		{"cookie inválida", []*http.Cookie{{Name: CookieProject, Value: "abc", Path: "/"}}},
		{"proyecto inexistente", []*http.Cookie{idCookie(CookieProject, MaxID)}},
	}
	want := pathSelectProject + "?" + volverAQuery("/tablero")
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			c := app.client(false)
			if tc.cookies != nil {
				app.setCookies(t, c, tc.cookies...)
			}

			post, _ := app.changeStory(t, c, story, sprint.ActionStart, sprint.EnSprint)
			get, _ := app.get(t, c, "/tablero")

			for nombre, resp := range map[string]*http.Response{"envío": post, "consulta": get} {
				if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
					t.Errorf("%s: se esperaba 303 a %q, se obtuvo %d a %q",
						nombre, want, resp.StatusCode, resp.Header.Get("Location"))
				}
			}
			if got := app.storyStatus(t, story); got != sprint.EnSprint {
				t.Errorf("el estado cambió a %q", got)
			}
		})
	}
}
