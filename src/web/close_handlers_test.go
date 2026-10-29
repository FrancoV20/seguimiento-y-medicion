package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

type closeHandlerFixture struct {
	*ctxApp
}

type closeHandlerStore struct {
	db     *sql.DB
	closer *sprint.PostgreSQLSprintCloser
}

func (s *closeHandlerStore) FindSprint(ctx context.Context, projectID, sprintID int64) (sprint.Sprint, error) {
	var found sprint.Sprint
	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, number, status
		FROM sprints
		WHERE project_id = $1 AND id = $2`,
		projectID, sprintID,
	).Scan(&found.ID, &found.ProjectID, &found.Number, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return sprint.Sprint{}, sprint.ErrSprintNotFound
	}
	if err != nil {
		return sprint.Sprint{}, err
	}
	found.Status = sprint.SprintStatus(status)
	return found, nil
}

func (s *closeHandlerStore) CloseSprint(ctx context.Context, sprintID int64) (sprint.CloseResult, error) {
	return s.closer.CloseSprint(ctx, sprintID)
}

type closeHandlerValidator struct {
	store *metrics.PostgreSQLMetricsStore
}

func (v closeHandlerValidator) ValidateSprintClosure(_ context.Context, sprintID int64) error {
	data, err := v.store.LoadSprintData(sprintID)
	if err != nil {
		return err
	}
	return metrics.ValidateSprintClosure(data)
}

type closeStoryFixture struct {
	id     int64
	status sprint.StoryStatus
	points int
}

func newCloseHandlerFixture(t *testing.T) *closeHandlerFixture {
	t.Helper()
	var serviceDB *sql.DB
	app := newCtxApp(t, nil, func(server *Server) {
		serviceDB = openContextTestDB(t)
		store := &closeHandlerStore{
			db:     serviceDB,
			closer: sprint.NewPostgreSQLSprintCloser(serviceDB),
		}
		service := sprint.NewCloseSprintService(
			store,
			closeHandlerValidator{store: metrics.NewPostgreSQLMetricsStore(serviceDB)},
		)
		if err := RegisterClose(server, service, NewPostgreSQLBoardCatalog(serviceDB)); err != nil {
			t.Fatalf("no se pudo registrar la sección de cierre: %v", err)
		}
	})
	return &closeHandlerFixture{ctxApp: app}
}

func newCloseSprintStories(t *testing.T, db *sql.DB, projectID, sprintID int64, statuses ...sprint.StoryStatus) []closeStoryFixture {
	t.Helper()
	stories := make([]closeStoryFixture, 0, len(statuses))
	for i, status := range statuses {
		points := []int{3, 5, 8, 13}[i%4]
		id := newCatStory(t, db, projectID, sprintID,
			fmt.Sprintf("HU15 cierre historia %d", i+1), &points, status)
		stories = append(stories, closeStoryFixture{id: id, status: status, points: points})
	}
	return stories
}

func closeHandlerSelectProject(t *testing.T, app *closeHandlerFixture, client *http.Client, projectID int64) {
	t.Helper()
	resp, _ := app.selectProject(t, client, id64(projectID), "/cierre")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/cierre" {
		t.Errorf("seleccionar el proyecto activo: se esperaba 303 a /cierre, se obtuvo %d a %q",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

func closeHandlerPost(t *testing.T, app *closeHandlerFixture, client *http.Client, sprintID int64) (*http.Response, string) {
	t.Helper()
	return app.post(t, client, "/cierre", url.Values{"sprint_id": {id64(sprintID)}})
}

func TestCloseHandler_CierreExitosoSoloCierraElProyectoActivoYConsumeFlashUnaVez(t *testing.T) {
	// Dado dos proyectos con un Sprint activo cada uno y el Sprint elegido con una historia Hecho
	// y dos historias incompletas.
	app := newCloseHandlerFixture(t)
	projectA := newCatProject(t, app.db, "cierre A")
	projectB := newCatProject(t, app.db, "cierre B")
	activeSprint, number := newCatSprint(t, app.db, projectA, "Activo")
	newCloseSprintStories(t, app.db, projectA, activeSprint,
		sprint.Done, sprint.InProgress, sprint.EnSprint)
	otherSprint, _ := newCatSprint(t, app.db, projectB, "Activo")
	newCloseSprintStories(t, app.db, projectB, otherSprint, sprint.Done)

	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectA)

	// Cuando se solicita el cierre del Sprint activo.
	resp, _ := closeHandlerPost(t, app, client, activeSprint)

	// Entonces responde 303 a /cierre, muestra el flash en el primer GET y lo consume.
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/cierre" {
		t.Errorf("POST /cierre = %d a %q; se esperaba 303 a /cierre",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	first, firstBody := app.get(t, client, "/cierre")
	if first.StatusCode != http.StatusOK {
		t.Errorf("primer GET /cierre = %d; se esperaba 200", first.StatusCode)
	}
	message := fmt.Sprintf("Sprint %d cerrado correctamente. Historias devueltas al Backlog: 2", number)
	if !strings.Contains(firstBody, message) {
		t.Errorf("primer GET /cierre no contiene el flash esperado %q", message)
	}
	second, secondBody := app.get(t, client, "/cierre")
	if second.StatusCode != http.StatusOK {
		t.Errorf("segundo GET /cierre = %d; se esperaba 200", second.StatusCode)
	}
	if strings.Contains(secondBody, message) {
		t.Errorf("el segundo GET /cierre volvió a mostrar el flash %q", message)
	}

	var activeStatus, otherStatus string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, activeSprint).Scan(&activeStatus); err != nil {
		t.Fatalf("no se pudo consultar el Sprint activo: %v", err)
	}
	if activeStatus != string(sprint.SprintFinished) {
		t.Errorf("estado del Sprint cerrado = %q; se esperaba %q", activeStatus, sprint.SprintFinished)
	}
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, otherSprint).Scan(&otherStatus); err != nil {
		t.Fatalf("no se pudo consultar el Sprint del otro proyecto: %v", err)
	}
	if otherStatus != string(sprint.SprintActive) {
		t.Errorf("el Sprint del otro proyecto quedó %q; se esperaba que siguiera %q",
			otherStatus, sprint.SprintActive)
	}
}

func TestCloseHandler_HistoriasHechoConservadasYDevueltasDisponiblesEnBacklog(t *testing.T) {
	// Dado un Sprint activo con una historia Hecho, una En progreso y una En Sprint.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "historias después del cierre")
	sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	stories := newCloseSprintStories(t, app.db, projectID, sprintID,
		sprint.Done, sprint.InProgress, sprint.EnSprint)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se cierra el Sprint desde la sección web.
	resp, _ := closeHandlerPost(t, app, client, sprintID)

	// Entonces la historia Hecho conserva estado, Sprint y Story Points; las otras
	// vuelven al Backlog sin asignación y son ofrecidas como disponibles por HU-04.
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/cierre" {
		t.Errorf("POST /cierre = %d a %q; se esperaba 303 a /cierre",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	for _, story := range stories {
		var status string
		var activeSprint sql.NullInt64
		var points sql.NullInt64
		var assignments int
		err := app.db.QueryRow(`
			SELECT us.status, us.active_sprint_id, us.story_points,
			       (SELECT count(*) FROM sprint_story_assignments a WHERE a.story_id = us.id)
			FROM user_stories us
			WHERE us.id = $1`, story.id,
		).Scan(&status, &activeSprint, &points, &assignments)
		if err != nil {
			t.Fatalf("no se pudo consultar la historia %d: %v", story.id, err)
		}
		if story.status == sprint.Done {
			if status != string(sprint.Done) || !activeSprint.Valid || activeSprint.Int64 != sprintID || assignments != 1 {
				t.Errorf("historia Hecho %d: estado=%q sprint=%v asignaciones=%d",
					story.id, status, activeSprint, assignments)
			}
			if !points.Valid || int(points.Int64) != story.points {
				t.Errorf("Story Points de historia Hecho %d = %v; se esperaban %d",
					story.id, points, story.points)
			}
			continue
		}
		if status != string(sprint.Pendiente) || activeSprint.Valid || assignments != 0 {
			t.Errorf("historia incompleta %d: estado=%q sprint=%v asignaciones=%d; se esperaba Pendiente sin asignación",
				story.id, status, activeSprint, assignments)
		}
	}

	backlogRows, err := NewPostgreSQLStoryCatalog(app.db).ListStories(context.Background(), projectID)
	if err != nil {
		t.Fatalf("HU-02 no pudo leer las historias del Backlog: %v", err)
	}
	options, err := NewPostgreSQLSprintCatalog(app.db).ListStoryOptions(context.Background(), projectID)
	if err != nil {
		t.Fatalf("HU-04 no pudo leer las opciones de asignación: %v", err)
	}
	backlogByID := make(map[int64]StoryRow, len(backlogRows))
	for _, row := range backlogRows {
		backlogByID[row.ID] = row
	}
	optionsByID := make(map[int64]StoryOption, len(options))
	for _, option := range options {
		optionsByID[option.ID] = option
	}
	for _, story := range stories {
		row, foundInBacklog := backlogByID[story.id]
		option, foundAsOption := optionsByID[story.id]
		if !foundInBacklog || !foundAsOption {
			t.Errorf("historia %d no aparece en los catálogos HU-02/HU-04", story.id)
			continue
		}
		if story.status == sprint.Done {
			if row.Status != string(sprint.Done) || option.Available {
				t.Errorf("historia Hecho %d debería conservarse y no estar disponible", story.id)
			}
			continue
		}
		if row.Status != string(sprint.Pendiente) || !option.Available {
			t.Errorf("historia devuelta %d: Backlog status=%q disponible=%v; se esperaba Pendiente y disponible",
				story.id, row.Status, option.Available)
		}
	}
}

func TestCloseHandler_SprintSinHistoriasHechoCierraYHU07CalculaVelocidadCero(t *testing.T) {
	// Dado un Sprint activo con historias incompletas y ninguna historia Hecho.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "sin historias completadas")
	sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	newCloseSprintStories(t, app.db, projectID, sprintID, sprint.InProgress, sprint.EnSprint)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se solicita el cierre del Sprint.
	resp, _ := closeHandlerPost(t, app, client, sprintID)

	// Entonces cierra con 303, devuelve las historias y HU-07 calcula velocidad cero.
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/cierre" {
		t.Errorf("POST /cierre = %d a %q; se esperaba 303 a /cierre",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo consultar el Sprint: %v", err)
	}
	if status != string(sprint.SprintFinished) {
		t.Errorf("estado del Sprint = %q; se esperaba %q", status, sprint.SprintFinished)
	}
	calculated, err := metrics.NewService(metrics.NewPostgreSQLMetricsStore(app.db)).CalculateMetrics(sprintID)
	if err != nil {
		t.Errorf("HU-07 CalculateMetrics() devolvió error: %v", err)
	} else if calculated.Velocity != 0 {
		t.Errorf("HU-07 calculó velocidad %d; se esperaba 0", calculated.Velocity)
	}
}

func TestCloseHandler_ResumenPrevioMuestraSoloDatosDelProyectoActivo(t *testing.T) {
	// Dado dos proyectos con Sprints activos y el proyecto A seleccionado en HU-20b.
	app := newCloseHandlerFixture(t)
	projectA := newCatProject(t, app.db, "resumen activo A")
	projectB := newCatProject(t, app.db, "resumen otro B")
	sprintA, _ := newCatSprint(t, app.db, projectA, "Activo")
	sprintB, _ := newCatSprint(t, app.db, projectB, "Activo")
	newCatStory(t, app.db, projectA, sprintA, "Completa de A", boardPuntos(3), sprint.Done)
	newCatStory(t, app.db, projectA, sprintA, "En progreso de A", boardPuntos(5), sprint.InProgress)
	newCatStory(t, app.db, projectA, sprintA, "En Sprint de A", boardPuntos(8), sprint.EnSprint)
	newCatStory(t, app.db, projectB, sprintB, "Historia privada de B", boardPuntos(13), sprint.Done)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectA)

	// Cuando se abre el formulario de cierre, se mide el tiempo de respuesta.
	started := time.Now()
	resp, body := app.get(t, client, "/cierre")
	elapsed := time.Since(started)

	// Entonces se ve el resumen de A, sus puntos y el botón; no aparecen datos de B
	// y la pantalla responde dentro del umbral SC-007.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /cierre = %d; se esperaba 200", resp.StatusCode)
	}
	for _, expected := range []string{
		"Sprint 1", "Completa de A", "En progreso de A", "En Sprint de A",
		"Hecho", "En progreso", "En Sprint", "3 de 16", "Cerrar Sprint",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("el resumen previo debía contener %q", expected)
		}
	}
	if strings.Contains(body, "Historia privada de B") {
		t.Errorf("el resumen del proyecto activo incluyó una historia del otro proyecto")
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("GET /cierre tardó %v; SC-007 exige menos de 500 ms", elapsed)
	}
}

func TestCloseHandler_SinSprintActivoMuestraMensajeYSinBoton(t *testing.T) {
	// Dado un proyecto activo sin ningún Sprint activo y seleccionado en HU-20b.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "sin Sprint activo")
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se abre la pantalla de cierre.
	resp, body := app.get(t, client, "/cierre")

	// Entonces se informa que no hay Sprint para cerrar y no se muestra el botón.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /cierre sin Sprint activo = %d; se esperaba 200", resp.StatusCode)
	}
	if !strings.Contains(body, "No hay un Sprint activo para cerrar") {
		t.Errorf("la pantalla debía mostrar %q", "No hay un Sprint activo para cerrar")
	}
	if strings.Contains(body, "Cerrar Sprint") || strings.Contains(body, `name="sprint_id"`) {
		t.Errorf("sin Sprint activo no deben mostrarse el botón ni el campo sprint_id")
	}
}

func TestCloseHandler_TituloConScriptSeMuestraComoTexto(t *testing.T) {
	// Dado un Sprint activo cuya historia tiene un título que contiene una etiqueta script.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "título escapado")
	sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	storyID := newCatStory(t, app.db, projectID, sprintID, "<script>alert('x')</script>", boardPuntos(3), sprint.Done)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se carga el resumen previo.
	resp, body := app.get(t, client, "/cierre")

	// Entonces la etiqueta se escapa como texto y no se interpreta como HTML ejecutable.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /cierre = %d; se esperaba 200", resp.StatusCode)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("el título debía mostrarse escapado como texto")
	}
	if strings.Contains(body, "<script>") {
		t.Errorf("la respuesta contiene una etiqueta script sin escapar")
	}
	var persistedTitle string
	if err := app.db.QueryRow(`SELECT title FROM user_stories WHERE id = $1`, storyID).Scan(&persistedTitle); err != nil {
		t.Fatalf("no se pudo verificar el título de la historia: %v", err)
	}
	if persistedTitle != "<script>alert('x')</script>" {
		t.Errorf("el título persistido cambió: %q", persistedTitle)
	}
}
