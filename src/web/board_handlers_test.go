package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T014 a T017): handlers del Tablero con el servidor real, el servicio real y PostgreSQL
// real. Reutilizan ctxApp (HU-20b) y las fixtures del catálogo (T009).

// Textos de éxito de la spec; se definen en board_form.go.
const (
	tableroComenzado        = "La historia pasó a En progreso"
	tableroCompletado       = "La historia pasó a Hecho"
	tableroCompletadoSinHrs = "La historia pasó a Hecho. Advertencia: no tiene horas reales registradas"
	tableroReabierto        = "La historia volvió a En progreso"
)

type boardConfig struct {
	store   func(sprint.StoryStatusStore) sprint.StoryStatusStore
	catalog func(BoardCatalog) BoardCatalog
}

type boardOption func(*boardConfig)

// withStatusStore reemplaza el almacenamiento del cambio de estado (para simular fallas en US2).
func withStatusStore(wrap func(sprint.StoryStatusStore) sprint.StoryStatusStore) boardOption {
	return func(c *boardConfig) { c.store = wrap }
}

// withBoardCatalog reemplaza el catálogo de lectura (para simular fallas en US3).
func withBoardCatalog(wrap func(BoardCatalog) BoardCatalog) boardOption {
	return func(c *boardConfig) { c.catalog = wrap }
}

type boardApp struct{ *ctxApp }

// newBoardApp arma el servidor con el contexto de trabajo y el Tablero registrados.
func newBoardApp(t *testing.T, opts ...boardOption) *boardApp {
	t.Helper()
	var cfg boardConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	db := openContextTestDB(t)
	var store sprint.StoryStatusStore = sprint.NewPostgreSQLStoryStatusStore(db)
	if cfg.store != nil {
		store = cfg.store(store)
	}
	var catalog BoardCatalog = NewPostgreSQLBoardCatalog(db)
	if cfg.catalog != nil {
		catalog = cfg.catalog(catalog)
	}
	app := newCtxApp(t, nil, func(s *Server) {
		if err := RegisterBoard(s, sprint.NewStatusService(store), catalog); err != nil {
			t.Fatalf("no se pudo registrar el Tablero: %v", err)
		}
	})
	return &boardApp{ctxApp: app}
}

// newBoardProject crea un proyecto con un Sprint activo.
func (a *boardApp) newBoardProject(t *testing.T, label string) (projectID, sprintID int64) {
	t.Helper()
	projectID = newCatProject(t, a.db, label)
	sprintID, _ = newCatSprint(t, a.db, projectID, "Activo")
	return projectID, sprintID
}

// storyStatus lee el estado guardado de la historia.
func (a *boardApp) storyStatus(t *testing.T, storyID int64) sprint.StoryStatus {
	t.Helper()
	var status string
	if err := a.db.QueryRow(`SELECT status FROM user_stories WHERE id = $1`, storyID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado de la historia %d: %v", storyID, err)
	}
	return sprint.StoryStatus(status)
}

// openBoard devuelve un navegador (que no sigue redirecciones) con el proyecto ya elegido.
func (a *boardApp) openBoard(t *testing.T, projectID int64) *http.Client {
	t.Helper()
	c := a.client(false)
	if resp, _ := a.selectProject(t, c, id64(projectID), "/tablero"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto: se obtuvo %d", resp.StatusCode)
	}
	return c
}

// changeStory envía una acción del Tablero. expected es el estado que la persona veía ("" si no se envía).
func (a *boardApp) changeStory(t *testing.T, c *http.Client, storyID int64, action sprint.Action, expected sprint.StoryStatus) (*http.Response, string) {
	t.Helper()
	form := url.Values{"historia_id": {id64(storyID)}, "accion": {string(action)}}
	if expected != "" {
		form.Set("estado_actual", string(expected))
	}
	return a.post(t, c, "/tablero/estado", form)
}

// wantSeeOtherToBoard verifica la redirección 303 a /tablero que sigue a un cambio válido.
func wantSeeOtherToBoard(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/tablero" {
		t.Fatalf("se esperaba 303 a /tablero, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// FR-004 (SC-001): comenzar pasa la historia a En progreso, avisa una sola vez y no toca otros proyectos.
func TestTablero_ComenzarPasaLaHistoriaAEnProgreso(t *testing.T) {
	app := newBoardApp(t)
	projectA, sprintA := app.newBoardProject(t, "A")
	projectB, sprintB := app.newBoardProject(t, "B")
	storyA := newCatStory(t, app.db, projectA, sprintA, "Historia de A", boardPuntos(3), sprint.EnSprint)
	storyB := newCatStory(t, app.db, projectB, sprintB, "Historia de B", boardPuntos(5), sprint.EnSprint)
	c := app.openBoard(t, projectA)

	resp, _ := app.changeStory(t, c, storyA, sprint.ActionStart, sprint.EnSprint)

	wantSeeOtherToBoard(t, resp)
	if got := app.storyStatus(t, storyA); got != sprint.InProgress {
		t.Errorf("estado de la historia = %q, se esperaba %q", got, sprint.InProgress)
	}
	if got := app.storyStatus(t, storyB); got != sprint.EnSprint {
		t.Errorf("la historia del otro proyecto cambió a %q", got)
	}
	_, primera := app.get(t, c, "/tablero")
	if !strings.Contains(primera, tableroComenzado) || !strings.Contains(primera, "alert-success") {
		t.Errorf("debía verse el mensaje de éxito %q", tableroComenzado)
	}
	if _, segunda := app.get(t, c, "/tablero"); strings.Contains(segunda, tableroComenzado) {
		t.Errorf("el mensaje no debía repetirse")
	}
}

// FR-005, FR-017 (SC-001, SC-003): completar con horas pasa a Hecho y suma los Story Points.
func TestTablero_CompletarConHorasPasaAHechoYSumaLosPuntos(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "completar")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia con horas", boardPuntos(8), sprint.InProgress)
	addCatEffort(t, app.db, story)
	newCatStory(t, app.db, projectID, sprintID, "Otra historia", boardPuntos(5), sprint.EnSprint)
	c := app.openBoard(t, projectID)
	if _, antes := app.get(t, c, "/tablero"); !strings.Contains(antes, "0 de 13") {
		t.Fatalf("precondición: antes de completar, los Story Points completados son 0 de 13")
	}

	resp, _ := app.changeStory(t, c, story, sprint.ActionComplete, sprint.InProgress)

	wantSeeOtherToBoard(t, resp)
	if got := app.storyStatus(t, story); got != sprint.Done {
		t.Errorf("estado de la historia = %q, se esperaba %q", got, sprint.Done)
	}
	_, despues := app.get(t, c, "/tablero")
	if !strings.Contains(despues, tableroCompletado) {
		t.Errorf("debía verse el mensaje %q", tableroCompletado)
	}
	if strings.Contains(despues, "no tiene horas reales") {
		t.Errorf("con horas registradas no debe haber advertencia")
	}
	if !strings.Contains(despues, "8 de 13") {
		t.Errorf("los Story Points completados debían pasar a 8 de 13")
	}
}

// FR-006 (SC-005): completar sin horas se permite y la advertencia viene en el mismo mensaje.
func TestTablero_CompletarSinHorasAdvierteEnUnSoloMensaje(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "sin horas")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia sin horas", boardPuntos(3), sprint.InProgress)
	c := app.openBoard(t, projectID)

	resp, _ := app.changeStory(t, c, story, sprint.ActionComplete, sprint.InProgress)

	wantSeeOtherToBoard(t, resp)
	if got := app.storyStatus(t, story); got != sprint.Done {
		t.Errorf("estado de la historia = %q, se esperaba %q", got, sprint.Done)
	}
	_, body := app.get(t, c, "/tablero")
	if !strings.Contains(body, tableroCompletadoSinHrs) {
		t.Errorf("debía verse el mensaje %q", tableroCompletadoSinHrs)
	}
	if n := strings.Count(body, `class="alert `); n != 1 {
		t.Errorf("la advertencia debía venir en un solo mensaje, hay %d", n)
	}
	if !strings.Contains(body, "alert-warning") || strings.Contains(body, "alert-success") {
		t.Errorf("completar sin horas debe verse como advertencia, no como éxito")
	}
}

// FR-007, FR-017 (SC-003): reabrir vuelve a En progreso y los Story Points dejan de contarse.
func TestTablero_ReabrirVuelveAEnProgresoYRestaLosPuntos(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "reabrir")
	story := newCatStory(t, app.db, projectID, sprintID, "Historia hecha", boardPuntos(8), sprint.Done)
	newCatStory(t, app.db, projectID, sprintID, "Otra historia", boardPuntos(5), sprint.EnSprint)
	c := app.openBoard(t, projectID)
	if _, antes := app.get(t, c, "/tablero"); !strings.Contains(antes, "8 de 13") {
		t.Fatalf("precondición: antes de reabrir, los Story Points completados son 8 de 13")
	}

	resp, _ := app.changeStory(t, c, story, sprint.ActionReopen, sprint.Done)

	wantSeeOtherToBoard(t, resp)
	if got := app.storyStatus(t, story); got != sprint.InProgress {
		t.Errorf("estado de la historia = %q, se esperaba %q", got, sprint.InProgress)
	}
	_, despues := app.get(t, c, "/tablero")
	if !strings.Contains(despues, tableroReabierto) {
		t.Errorf("debía verse el mensaje %q", tableroReabierto)
	}
	if !strings.Contains(despues, "0 de 13") {
		t.Errorf("los Story Points completados debían volver a 0 de 13")
	}
}

// El tipo del mensaje al completar: éxito con horas y advertencia sin horas.
func TestTablero_CompletarSinHorasSeMuestraComoAdvertenciaYConHorasComoExito(t *testing.T) {
	app := newBoardApp(t)
	projectID, sprintID := app.newBoardProject(t, "tipo del mensaje")
	conHoras := newCatStory(t, app.db, projectID, sprintID, "Historia con horas", boardPuntos(3), sprint.InProgress)
	addCatEffort(t, app.db, conHoras)
	sinHoras := newCatStory(t, app.db, projectID, sprintID, "Historia sin horas", boardPuntos(5), sprint.InProgress)
	c := app.openBoard(t, projectID)

	app.changeStory(t, c, conHoras, sprint.ActionComplete, sprint.InProgress)
	_, conHorasBody := app.get(t, c, "/tablero")
	app.changeStory(t, c, sinHoras, sprint.ActionComplete, sprint.InProgress)
	_, sinHorasBody := app.get(t, c, "/tablero")

	if !strings.Contains(conHorasBody, "alert-success") || strings.Contains(conHorasBody, "alert-warning") {
		t.Errorf("completar con horas debía mostrarse como éxito")
	}
	if !strings.Contains(sinHorasBody, "alert-warning") || strings.Contains(sinHorasBody, "alert-success") {
		t.Errorf("completar sin horas debía mostrarse como advertencia")
	}
}
