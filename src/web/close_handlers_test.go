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

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
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

type failingCloseHandlerStore struct {
	*closeHandlerStore
	err error
}

func (s failingCloseHandlerStore) CloseSprint(context.Context, int64) (sprint.CloseResult, error) {
	return sprint.CloseResult{}, s.err
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

func newCloseHandlerFixtureWithStore(
	t *testing.T,
	newStore func(*sql.DB) sprint.SprintCloseStore,
) *closeHandlerFixture {
	t.Helper()
	if newStore == nil {
		t.Fatal("se requiere un store de prueba")
	}
	var serviceDB *sql.DB
	app := newCtxApp(t, nil, func(server *Server) {
		serviceDB = openContextTestDB(t)
		service := sprint.NewCloseSprintService(
			newStore(serviceDB),
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

func TestCloseHandler_RecargaDespuesDelCierreNoRepiteOperacionNiMensaje(t *testing.T) {
	// Dado un Sprint que ya se cerró y cuyo mensaje de éxito se mostró una sola vez.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "recarga después del cierre")
	sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	newCloseSprintStories(t, app.db, projectID, sprintID, sprint.Done, sprint.InProgress)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	firstPost, _ := closeHandlerPost(t, app, client, sprintID)
	firstPost.Body.Close()
	if firstPost.StatusCode != http.StatusSeeOther {
		t.Fatalf("primer POST /cierre = %d; se esperaba 303", firstPost.StatusCode)
	}
	firstGet, firstBody := app.get(t, client, "/cierre")
	defer firstGet.Body.Close()
	successMessage := fmt.Sprintf(MsgSprintClosed, 1, 1)
	if firstGet.StatusCode != http.StatusOK || !strings.Contains(firstBody, successMessage) {
		t.Fatalf("el primer GET debía mostrar %q; status=%d", successMessage, firstGet.StatusCode)
	}

	// Cuando el navegador recarga y reenvía el POST ya procesado.
	repeatedPost, repeatedBody := closeHandlerPost(t, app, client, sprintID)
	defer repeatedPost.Body.Close()

	// Entonces el segundo cierre se rechaza, no duplica el registro y no repite el mensaje de éxito.
	if repeatedPost.StatusCode != http.StatusConflict {
		t.Errorf("reenvío de POST /cierre = %d; se esperaba 409", repeatedPost.StatusCode)
	}
	if strings.Contains(repeatedBody, successMessage) {
		t.Errorf("la respuesta del reenvío repitió el mensaje de éxito %q", successMessage)
	}
	if !strings.Contains(repeatedBody, MsgSprintAlreadyClosed) {
		t.Errorf("la respuesta del reenvío debía mostrar %q", MsgSprintAlreadyClosed)
	}
	secondGet, secondBody := app.get(t, client, "/cierre")
	defer secondGet.Body.Close()
	if strings.Contains(secondBody, successMessage) {
		t.Errorf("el GET posterior al reenvío volvió a mostrar %q", successMessage)
	}

	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
	}
	if status != string(sprint.SprintFinished) {
		t.Errorf("el Sprint quedó %q; se esperaba %q", status, sprint.SprintFinished)
	}
	var resultCount int
	if err := app.db.QueryRow(
		`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`, sprintID,
	).Scan(&resultCount); err != nil {
		t.Fatalf("no se pudo contar el registro del cierre: %v", err)
	}
	if resultCount != 2 {
		t.Errorf("el reenvío dejó %d resultados; se esperaban los 2 del primer cierre", resultCount)
	}
}

func TestCloseHandler_SinContextoDeProyectoRedirigeAlSelectorSinCerrar(t *testing.T) {
	// Dado un Sprint activo que no debe cerrarse sin un proyecto válido en el contexto de HU-20b.
	cases := []struct {
		name        string
		contextType string
	}{
		{name: "sin selección", contextType: "none"},
		{name: "proyecto eliminado", contextType: "deleted"},
		{name: "cookie inválida", contextType: "invalid"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app := newCloseHandlerFixture(t)
			projectID := newCatProject(t, app.db, "contexto inválido "+tc.name)
			sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
			stories := newCloseSprintStories(t, app.db, projectID, sprintID, sprint.Done, sprint.InProgress)
			client := app.client(false)

			switch tc.contextType {
			case "deleted":
				deletedProject := newCatProject(t, app.db, "contexto eliminado "+tc.name)
				app.setCookies(t, client, idCookie(CookieProject, deletedProject))
				if _, err := app.db.Exec(`DELETE FROM projects WHERE id = $1`, deletedProject); err != nil {
					t.Fatalf("no se pudo eliminar el proyecto del contexto: %v", err)
				}
			case "invalid":
				app.setCookies(t, client, &http.Cookie{
					Name: CookieProject, Value: "invalid-signature", Path: "/",
				})
			}

			// Cuando se intenta enviar el cierre sin selección válida o con una cookie obsoleta.
			resp, _ := closeHandlerPost(t, app, client, sprintID)
			defer resp.Body.Close()

			// Entonces HU-20b redirige al selector y la operación no modifica el Sprint ni las historias.
			location, err := url.Parse(resp.Header.Get("Location"))
			if resp.StatusCode != http.StatusSeeOther || err != nil ||
				location.Path != pathSelectProject || location.Query().Get("volver_a") != closePath {
				t.Errorf("POST /cierre = %d a %q; se esperaba 303 al selector de proyectos para %s",
					resp.StatusCode, resp.Header.Get("Location"), closePath)
			}
			var status string
			if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&status); err != nil {
				t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
			}
			if status != string(sprint.SprintActive) {
				t.Errorf("el Sprint quedó %q; se esperaba que siguiera Activo", status)
			}
			var resultCount int
			if err := app.db.QueryRow(
				`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`, sprintID,
			).Scan(&resultCount); err != nil {
				t.Fatalf("no se pudo consultar el registro del Sprint: %v", err)
			}
			if resultCount != 0 {
				t.Errorf("se registraron %d resultados sin un contexto de proyecto válido", resultCount)
			}
			for _, story := range stories {
				var storyStatus string
				var activeSprint sql.NullInt64
				if err := app.db.QueryRow(
					`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`, story.id,
				).Scan(&storyStatus, &activeSprint); err != nil {
					t.Fatalf("no se pudo leer la historia %d: %v", story.id, err)
				}
				if storyStatus != string(story.status) || !activeSprint.Valid || activeSprint.Int64 != sprintID {
					t.Errorf("la historia %d cambió sin contexto válido: estado=%q sprint=%v",
						story.id, storyStatus, activeSprint)
				}
			}
		})
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

type failingCloseBoardCatalog struct{}

func (failingCloseBoardCatalog) ActiveBoard(context.Context, int64) (BoardView, bool, error) {
	return BoardView{}, false, errors.New(
		`pq: SELECT id FROM sprints failed at C:\internal\close_catalog.go:42`,
	)
}

func TestCloseHandler_FallaDeLecturaMuestraPaginaDeError(t *testing.T) {
	// Dado un proyecto válido cuyo catálogo de tablero falla al leer el Sprint activo.
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		store := &closeHandlerStore{db: db, closer: sprint.NewPostgreSQLSprintCloser(db)}
		service := sprint.NewCloseSprintService(
			store,
			closeHandlerValidator{store: metrics.NewPostgreSQLMetricsStore(db)},
		)
		if err := RegisterClose(server, service, failingCloseBoardCatalog{}); err != nil {
			t.Fatalf("no se pudo registrar la sección de cierre: %v", err)
		}
	})
	projectID := newCatProject(t, app.db, "falla de lectura del cierre")
	client := app.client(false)
	closeHandlerSelectProject(t, &closeHandlerFixture{ctxApp: app}, client, projectID)

	// Cuando se solicita la vista previa y el catálogo devuelve un error de lectura.
	resp, body := app.get(t, client, closePath)
	defer resp.Body.Close()

	// Entonces se muestra la página de error común, no una pantalla incompleta ni el detalle interno.
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("GET /cierre ante falla de lectura = %d; se esperaba 503", resp.StatusCode)
	}
	for _, expected := range []string{TitleUnavailable, MsgUnavailable, LinkBackHome} {
		if !strings.Contains(body, expected) {
			t.Errorf("la página de error debía contener %q", expected)
		}
	}
	for _, unexpected := range []string{
		MsgNoSprintToClose, "name=\"sprint_id\"", "Cerrar Sprint",
		"SELECT id FROM sprints", "pq:", "close_catalog.go", `C:\internal`,
	} {
		if strings.Contains(body, unexpected) {
			t.Errorf("la respuesta contiene contenido incompleto o interno %q", unexpected)
		}
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

func TestCloseHandler_SprintFinalizadoRechazadoSinCerrarElSprintActivo(t *testing.T) {
	// Dado un Sprint finalizado y un Sprint nuevo activo en el mismo proyecto.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "Sprint finalizado con reemplazo activo")
	closedSprintID, _ := newCatSprint(t, app.db, projectID, "Finalizado")
	activeSprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se intenta cerrar el Sprint finalizado enviando su identificador anterior.
	resp, body := closeHandlerPost(t, app, client, closedSprintID)
	defer resp.Body.Close()

	// Entonces se rechaza con 409 y el mensaje de dominio; el Sprint activo no cambia.
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /cierre para Sprint finalizado = %d; se esperaba 409", resp.StatusCode)
	}
	if !strings.Contains(body, MsgSprintAlreadyClosed) {
		t.Errorf("la respuesta debía mostrar %q", MsgSprintAlreadyClosed)
	}
	assertSprintStatus := func(id int64, want string) {
		t.Helper()
		var got string
		if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, id).Scan(&got); err != nil {
			t.Fatalf("no se pudo leer el estado del Sprint %d: %v", id, err)
		}
		if got != want {
			t.Errorf("estado del Sprint %d = %q; se esperaba %q", id, got, want)
		}
	}
	assertSprintStatus(closedSprintID, "Finalizado")
	assertSprintStatus(activeSprintID, "Activo")
}

func TestCloseHandler_SprintIDInvalidoInexistenteODeOtroProyectoSeRechaza(t *testing.T) {
	// Dado dos proyectos con Sprints activos y el proyecto A seleccionado como contexto.
	app := newCloseHandlerFixture(t)
	projectA := newCatProject(t, app.db, "id de cierre A")
	projectB := newCatProject(t, app.db, "id de cierre B")
	activeSprintA, _ := newCatSprint(t, app.db, projectA, "Activo")
	activeSprintB, _ := newCatSprint(t, app.db, projectB, "Activo")
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectA)

	// Cuando se envía un identificador malformado, negativo, inexistente o perteneciente
	// a otro proyecto (aunque el formulario incluya un project_id no confiable).
	cases := []struct {
		name      string
		sprintID  string
		projectID string
	}{
		{name: "texto", sprintID: "abc"},
		{name: "decimal", sprintID: "1.5"},
		{name: "exponente", sprintID: "1e2"},
		{name: "negativo", sprintID: "-1"},
		{name: "vacío", sprintID: ""},
		{name: "inexistente", sprintID: "9223372036854775807"},
		{name: "sprint de otro proyecto", sprintID: id64(activeSprintB), projectID: id64(projectB)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{"sprint_id": {tc.sprintID}}
			if tc.projectID != "" {
				form.Set("project_id", tc.projectID)
			}
			resp, body := app.post(t, client, "/cierre", form)
			defer resp.Body.Close()

			// Entonces responde 422 con "El Sprint no existe" y ambos Sprints siguen activos.
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("POST /cierre con sprint_id=%q = %d; se esperaba 422", tc.sprintID, resp.StatusCode)
			}
			if !strings.Contains(body, MsgSprintNotFound) {
				t.Errorf("la respuesta debía mostrar %q", MsgSprintNotFound)
			}
			for _, id := range []int64{activeSprintA, activeSprintB} {
				var status string
				if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, id).Scan(&status); err != nil {
					t.Fatalf("no se pudo leer el estado del Sprint %d: %v", id, err)
				}
				if status != "Activo" {
					t.Errorf("el Sprint %d quedó %q; se esperaba que siguiera Activo", id, status)
				}
			}
		})
	}
}

func TestCloseHandler_RechazaCierrePorBloqueoHU07(t *testing.T) {
	// Dado un Sprint activo con historias Hecho que HU-07 no puede validar.
	app := newCloseHandlerFixture(t)
	projectID := newCatProject(t, app.db, "bloqueo de validación HU-07")
	sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	invalidStoryIDs := []int64{
		newCatStory(t, app.db, projectID, sprintID, "Hecho sin puntos 1", nil, sprint.Done),
		newCatStory(t, app.db, projectID, sprintID, "Hecho sin puntos 2", nil, sprint.Done),
	}
	incompleteStoryID := newCatStory(
		t, app.db, projectID, sprintID, "Historia incompleta", boardPuntos(5), sprint.InProgress,
	)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se solicita cerrar el Sprint, el validador real de HU-07 rechaza los datos inválidos.
	resp, body := closeHandlerPost(t, app, client, sprintID)
	defer resp.Body.Close()

	// Entonces responde 409, enumera las historias que deben corregirse y no modifica el Sprint,
	// las asignaciones ni el registro del cierre.
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /cierre con datos inválidos para HU-07 = %d; se esperaba 409", resp.StatusCode)
	}
	for _, storyID := range invalidStoryIDs {
		want := fmt.Sprintf("Historia %d: Story Points inválidos", storyID)
		if !strings.Contains(body, want) {
			t.Errorf("la respuesta no identifica el dato que debe corregirse %q", want)
		}
	}

	var sprintStatus string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&sprintStatus); err != nil {
		t.Fatalf("no se pudo consultar el Sprint después del rechazo: %v", err)
	}
	if sprintStatus != string(sprint.SprintActive) {
		t.Errorf("estado del Sprint = %q; se esperaba %q", sprintStatus, sprint.SprintActive)
	}

	for _, storyID := range append(invalidStoryIDs, incompleteStoryID) {
		var status string
		var activeSprintID sql.NullInt64
		var assignments, results int
		err := app.db.QueryRow(`
			SELECT us.status, us.active_sprint_id,
			       (SELECT count(*) FROM sprint_story_assignments a WHERE a.story_id = us.id),
			       (SELECT count(*) FROM sprint_story_results r WHERE r.sprint_id = $2 AND r.story_id = us.id)
			FROM user_stories us
			WHERE us.id = $1`, storyID, sprintID,
		).Scan(&status, &activeSprintID, &assignments, &results)
		if err != nil {
			t.Fatalf("no se pudo consultar la historia %d después del rechazo: %v", storyID, err)
		}
		if !activeSprintID.Valid || activeSprintID.Int64 != sprintID || assignments != 1 || results != 0 {
			t.Errorf("historia %d cambió tras el rechazo: active_sprint_id=%v asignaciones=%d registros=%d",
				storyID, activeSprintID, assignments, results)
		}
		if storyID == incompleteStoryID && status != string(sprint.InProgress) {
			t.Errorf("estado de la historia incompleta = %q; se esperaba %q", status, sprint.InProgress)
		}
		if storyID != incompleteStoryID && status != string(sprint.Done) {
			t.Errorf("estado de la historia inválida = %q; se esperaba %q", status, sprint.Done)
		}
	}
}

func TestCloseHandler_FallaDeAlmacenamientoNoAplicaCambiosParciales(t *testing.T) {
	// Dado un Sprint activo y un almacenamiento de cierre que devuelve un error de persistencia.
	app := newCloseHandlerFixtureWithStore(t, func(db *sql.DB) sprint.SprintCloseStore {
		return failingCloseHandlerStore{
			closeHandlerStore: &closeHandlerStore{db: db, closer: sprint.NewPostgreSQLSprintCloser(db)},
			err:               errors.New("db connection timeout during sprint_story_results insert"),
		}
	})
	projectID := newCatProject(t, app.db, "falla de persistencia al cerrar")
	sprintID, _ := newCatSprint(t, app.db, projectID, "Activo")
	points := 3
	doneStoryID := newCatStory(t, app.db, projectID, sprintID, "Hecho válido", &points, sprint.Done)
	incompleteStoryID := newCatStory(t, app.db, projectID, sprintID, "Historia a devolver", boardPuntos(5), sprint.InProgress)
	client := app.client(false)
	closeHandlerSelectProject(t, app, client, projectID)

	// Cuando se solicita el cierre, el puerto de almacenamiento falla.
	resp, body := closeHandlerPost(t, app, client, sprintID)
	defer resp.Body.Close()

	// Entonces responde 500 con el mensaje genérico, sin exponer el detalle técnico ni confirmar éxito.
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("POST /cierre ante falla de almacenamiento = %d; se esperaba 500", resp.StatusCode)
	}
	if !strings.Contains(body, MsgOperationFailed) {
		t.Errorf("la respuesta debía mostrar %q", MsgOperationFailed)
	}
	if strings.Contains(body, "db connection timeout") || strings.Contains(body, "sprint_story_results insert") {
		t.Errorf("la respuesta expuso detalles internos del almacenamiento")
	}
	if strings.Contains(body, MsgSprintClosed) {
		t.Errorf("la respuesta indicó éxito pese a la falla de almacenamiento")
	}

	var sprintStatus string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&sprintStatus); err != nil {
		t.Fatalf("no se pudo consultar el Sprint después del fallo: %v", err)
	}
	if sprintStatus != string(sprint.SprintActive) {
		t.Errorf("estado del Sprint = %q; se esperaba que siguiera %q", sprintStatus, sprint.SprintActive)
	}
	for _, storyID := range []int64{doneStoryID, incompleteStoryID} {
		var status string
		var activeSprintID sql.NullInt64
		var assignments, results int
		err := app.db.QueryRow(`
				SELECT us.status, us.active_sprint_id,
				       (SELECT count(*) FROM sprint_story_assignments a WHERE a.story_id = us.id),
				       (SELECT count(*) FROM sprint_story_results r WHERE r.sprint_id = $2 AND r.story_id = us.id)
				FROM user_stories us
				WHERE us.id = $1`, storyID, sprintID,
		).Scan(&status, &activeSprintID, &assignments, &results)
		if err != nil {
			t.Fatalf("no se pudo consultar la historia %d después del fallo: %v", storyID, err)
		}
		if !activeSprintID.Valid || activeSprintID.Int64 != sprintID || assignments != 1 || results != 0 {
			t.Errorf("historia %d cambió tras el fallo: estado=%q active_sprint_id=%v asignaciones=%d registros=%d",
				storyID, status, activeSprintID, assignments, results)
		}
	}
}

func TestCloseHandler_RechazaCargaDeEsfuerzoDespuesDelCierre(t *testing.T) {
	// Dado un Sprint que se cierra con una historia Hecho y otra que vuelve al Backlog.
	app := newEffortApp(t)
	points := 3
	if _, err := app.fx.db.Exec(`UPDATE user_stories SET story_points = $1 WHERE id = $2`, points, app.fx.hecho); err != nil {
		t.Fatalf("no se pudieron preparar los Story Points para HU-07: %v", err)
	}
	store := &closeHandlerStore{
		db:     app.fx.db,
		closer: sprint.NewPostgreSQLSprintCloser(app.fx.db),
	}
	service := sprint.NewCloseSprintService(
		store,
		closeHandlerValidator{store: metrics.NewPostgreSQLMetricsStore(app.fx.db)},
	)
	if _, err := service.CloseSprint(context.Background(), app.fx.projectA, app.fx.sprintA); err != nil {
		t.Fatalf("no se pudo cerrar el Sprint con el servicio real: %v", err)
	}
	activeSprint, _ := newCatSprint(t, app.fx.db, app.fx.projectA, string(sprint.SprintActive))
	newCatStory(t, app.fx.db, app.fx.projectA, activeSprint, "Historia del Sprint siguiente",
		boardPuntos(5), sprint.EnSprint)
	client := app.login(t, app.fx.projectA, app.fx.ana)
	before := app.fx.countAllEffort(t)

	// Cuando HU-03 recibe horas para la historia que quedó en el Sprint finalizado o fue devuelta.
	for _, storyID := range []int64{app.fx.hecho, app.fx.enSprint} {
		result := effort.NewService(effort.NewPostgreSQLEffortStore(app.fx.db), effortClock).LogEffort(
			context.Background(),
			effort.LogEffortRequest{
				StoryID:  id64(storyID),
				MemberID: id64(app.fx.ana),
				WorkDate: effortClock(),
				Activity: "Trabajo posterior al cierre",
				Hours:    "2",
			},
		)
		if result.Succeeded() || result.StorageError != nil {
			t.Errorf("HU-03 debía rechazar la carga para la historia %d sin guardarla; resultado=%+v", storyID, result)
		}
		var rejectsStory bool
		for _, validationErr := range result.ValidationErrors {
			if validationErr.Field == effort.FieldStory && errors.Is(validationErr.Err, effort.ErrStoryNotInSprint) {
				rejectsStory = true
			}
		}
		if !rejectsStory {
			t.Errorf("HU-03 no devolvió ErrStoryNotInSprint para la historia %d: %+v", storyID, result.ValidationErrors)
		}

		resp, body := app.submit(t, client, effortForm(storyID, "2026-10-20", "Trabajo posterior al cierre", "2"))

		// Entonces el handler rechaza cada carga y no se agrega ningún registro de horas.
		assertStatus(t, resp, http.StatusUnprocessableEntity)
		assertContains(t, body, fieldErrorHTML(EffortFieldStory, MsgEffortStoryNotInSprint))
		assertNotContains(t, body, MsgEffortRecorded)
	}
	if after := app.fx.countAllEffort(t); after != before {
		t.Errorf("HU-03 no debía registrar horas después del cierre: antes %d, después %d", before, after)
	}
}
