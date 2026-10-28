package web

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T032 a T036): la pantalla del Tablero. Servidor, servicio, catálogo y PostgreSQL reales;
// solo se reemplaza el catálogo para simular una falla de lectura.

// Textos de los escenarios del Issue; se definen en board_form.go.
const (
	tableroSinSprintActivo    = "No hay un Sprint activo"
	tableroSprintSinHistorias = "El Sprint no tiene historias asignadas"
)

// boardActionForm encuentra, en cada formulario de acción, la historia y la acción que ofrece.
var boardActionForm = regexp.MustCompile(`name="historia_id" value="(\d+)">\s*<input type="hidden" name="accion" value="(\w+)">`)

// failingBoardCatalog simula una falla de lectura con un error técnico que no debe llegar a la pantalla.
type failingBoardCatalog struct{}

func (failingBoardCatalog) ActiveBoard(context.Context, int64) (BoardView, bool, error) {
	return BoardView{}, false, errors.New(`pq: connection refused; SELECT id FROM sprints at C:\internal\catalog.go:7`)
}

// FR-002 (SC-007): el Sprint activo del proyecto activo, con sus historias agrupadas por estado.
func TestTablero_MuestraElSprintActivoConSusHistoriasPorEstado(t *testing.T) {
	app := newBoardApp(t)
	projectA, sprintA := app.newBoardProject(t, "vista A")
	finished, _ := newCatSprint(t, app.db, projectA, "Finalizado")
	newCatStory(t, app.db, projectA, finished, "Historia de un Sprint finalizado", boardPuntos(21), sprint.Done)
	newCatStory(t, app.db, projectA, 0, "Historia pendiente del Backlog", boardPuntos(2), sprint.Pendiente)
	projectB, sprintB := app.newBoardProject(t, "vista B")
	newCatStory(t, app.db, projectB, sprintB, "Historia de otro proyecto", boardPuntos(13), sprint.Done)
	newCatStory(t, app.db, projectA, sprintA, "Historia En Sprint", boardPuntos(3), sprint.EnSprint)
	newCatStory(t, app.db, projectA, sprintA, "Historia En progreso", boardPuntos(5), sprint.InProgress)
	newCatStory(t, app.db, projectA, sprintA, "Historia Hecho", boardPuntos(8), sprint.Done)
	newCatStory(t, app.db, projectA, sprintA, "Historia sin estimar", nil, sprint.Done)
	c := app.openBoard(t, projectA)

	resp, body := app.get(t, c, "/tablero")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	for _, want := range []string{
		"Sprint 1", "Objetivo del Tablero",
		`En Sprint <span class="board-count">(1)</span>`,
		`En progreso <span class="board-count">(1)</span>`,
		`Hecho <span class="board-count">(2)</span>`,
		"Historia En Sprint", "Historia En progreso", "Historia Hecho", "Historia sin estimar",
		"8 de 16", // completados de planificados; la historia sin estimar cuenta 0
	} {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en el Tablero", want)
		}
	}
	for _, unwanted := range []string{
		"Historia de un Sprint finalizado", "Historia pendiente del Backlog", "Historia de otro proyecto",
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("el Tablero no debe mostrar %q", unwanted)
		}
	}
	if n := strings.Count(body, "Sin estimar"); n != 1 {
		t.Errorf("la historia sin Story Points debía verse como \"Sin estimar\" una vez, aparece %d", n)
	}
}

// Un título con HTML o scripts se ve como texto.
func TestTablero_TituloConHTMLSeVeComoTexto(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "html")
	newCatStory(t, app.db, projectID, sprintID, "<script>alert(1)</script>", boardPuntos(3), sprint.EnSprint)
	c := app.openBoard(t, projectID)

	_, body := app.get(t, c, "/tablero")

	if strings.Contains(body, "<script>alert(1)</script>") || !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("el título con HTML debía verse como texto escapado")
	}
}

// SC-007: el Tablero responde en menos de 500 ms con un Sprint de menos de 50 historias.
func TestTablero_RespondeEnMenosDe500ms(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "tiempo")
	estados := []sprint.StoryStatus{sprint.EnSprint, sprint.InProgress, sprint.Done}
	for i := 0; i < 40; i++ {
		newCatStory(t, app.db, projectID, sprintID, fmt.Sprintf("Historia %d", i), boardPuntos(3), estados[i%3])
	}
	c := app.openBoard(t, projectID)
	app.get(t, c, "/tablero") // calienta la conexión y las plantillas

	const runs = 5
	start := time.Now()
	for i := 0; i < runs; i++ {
		app.get(t, c, "/tablero")
	}

	if avg := time.Since(start) / runs; avg >= 500*time.Millisecond {
		t.Errorf("SC-007: el promedio fue %v, debía ser menor a 500 ms", avg)
	}
}

// FR-003: cada historia ofrece solo la acción válida de su estado.
func TestTablero_CadaHistoriaOfreceSoloLaAccionDeSuEstado(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "acciones")
	enSprint := newCatStory(t, app.db, projectID, sprintID, "Historia En Sprint", boardPuntos(3), sprint.EnSprint)
	enProgreso := newCatStory(t, app.db, projectID, sprintID, "Historia En progreso", boardPuntos(5), sprint.InProgress)
	hecha := newCatStory(t, app.db, projectID, sprintID, "Historia Hecho", boardPuntos(8), sprint.Done)
	c := app.openBoard(t, projectID)

	_, body := app.get(t, c, "/tablero")

	got := map[string]string{}
	matches := boardActionForm.FindAllStringSubmatch(body, -1)
	for _, m := range matches {
		got[m[1]] = m[2]
	}
	want := map[string]string{
		id64(enSprint):   string(sprint.ActionStart),
		id64(enProgreso): string(sprint.ActionComplete),
		id64(hecha):      string(sprint.ActionReopen),
	}
	if len(matches) != 3 || !maps.Equal(got, want) {
		t.Errorf("acciones ofrecidas = %v (%d formularios), se esperaba %v", got, len(matches), want)
	}
	for _, label := range []string{">Comenzar</button>", ">Completar</button>", ">Reabrir</button>"} {
		if n := strings.Count(body, label); n != 1 {
			t.Errorf("el botón %q debía aparecer una vez, aparece %d", label, n)
		}
	}
}

// FR-013: sin Sprint activo se muestra el mensaje y ninguna acción.
func TestTablero_SinSprintActivoMuestraElMensajeYNingunaAccion(t *testing.T) {
	app := newBoardApp(t)
	sinSprints := newCatProject(t, app.db, "sin Sprints")
	soloFinalizado := newCatProject(t, app.db, "solo finalizado")
	finished, _ := newCatSprint(t, app.db, soloFinalizado, "Finalizado")
	newCatStory(t, app.db, soloFinalizado, finished, "Historia del Sprint finalizado", boardPuntos(8), sprint.Done)
	casos := []struct {
		nombre     string
		proyectoID int64
	}{
		{"proyecto sin Sprints", sinSprints},
		{"proyecto con solo un Sprint finalizado", soloFinalizado},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			c := app.openBoard(t, tc.proyectoID)

			resp, body := app.get(t, c, "/tablero")

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
			}
			if !strings.Contains(body, tableroSinSprintActivo) {
				t.Errorf("debía verse %q", tableroSinSprintActivo)
			}
			if strings.Contains(body, `name="accion"`) || strings.Contains(body, "Story Points completados") {
				t.Errorf("sin Sprint activo no debe haber acciones ni puntos")
			}
		})
	}
}

// FR-013: un Sprint activo sin historias muestra el Sprint y el mensaje, sin acciones.
func TestTablero_SprintSinHistoriasMuestraElMensaje(t *testing.T) {
	app := newBoardApp(t)
	projectID, _ := app.newBoardProject(t, "sin historias")
	c := app.openBoard(t, projectID)

	resp, body := app.get(t, c, "/tablero")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	for _, want := range []string{"Sprint 1", "Objetivo del Tablero", tableroSprintSinHistorias} {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en el Tablero", want)
		}
	}
	if strings.Contains(body, `name="accion"`) {
		t.Errorf("un Sprint sin historias no ofrece acciones")
	}
}

// FR-016: si falla la lectura se muestra la página de error, no un Tablero incompleto, y sin
// detalles internos.
func TestTablero_FallaDeLecturaMuestraLaPaginaDeError(t *testing.T) {
	app := newBoardApp(t, withBoardCatalog(func(BoardCatalog) BoardCatalog { return failingBoardCatalog{} }))
	projectID, _ := app.newBoardProject(t, "falla de lectura")
	c := app.openBoard(t, projectID)

	resp, body := app.get(t, c, "/tablero")

	if resp.StatusCode < http.StatusInternalServerError {
		t.Errorf("se esperaba la página de error (5xx), se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, LinkBackHome) {
		t.Errorf("debía ser la página de error de la aplicación, con el enlace %q", LinkBackHome)
	}
	if strings.Contains(body, "Story Points completados") || strings.Contains(body, `class="board-columns"`) {
		t.Errorf("no debe mostrarse un Tablero incompleto")
	}
	bajo := strings.ToLower(body)
	for _, p := range []string{"select ", "pq:", "sql:", ".go:", `c:\`, "internal", "connection refused"} {
		if strings.Contains(bajo, p) {
			t.Errorf("la página expone un detalle interno (contiene %q)", p)
		}
	}
}
