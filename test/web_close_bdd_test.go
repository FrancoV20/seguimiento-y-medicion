package test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

type cierreBDDStore struct {
	db     *sql.DB
	closer *sprint.PostgreSQLSprintCloser
}

func (s *cierreBDDStore) FindSprint(ctx context.Context, projectID, sprintID int64) (sprint.Sprint, error) {
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

func (s *cierreBDDStore) CloseSprint(
	ctx context.Context,
	sprintID int64,
	check sprint.ClosureCheck,
) (sprint.CloseResult, error) {
	return s.closer.CloseSprint(ctx, sprintID, check)
}

type cierreBDDFallaAlCerrarStore struct {
	*cierreBDDStore
	err error
}

func (s cierreBDDFallaAlCerrarStore) CloseSprint(context.Context, int64, sprint.ClosureCheck) (sprint.CloseResult, error) {
	return sprint.CloseResult{}, s.err
}

type cierreBDDValidator struct{}

// ValidateSprintClosure aplica HU-07 a las historias que el cierre bloqueó, como el adaptador de main.
func (cierreBDDValidator) ValidateSprintClosure(_ context.Context, stories []sprint.ClosingStory) error {
	data := metrics.SprintData{Stories: make([]metrics.CompletedStory, 0, len(stories))}
	for _, story := range stories {
		data.Stories = append(data.Stories, metrics.CompletedStory{
			ID:          story.ID,
			Status:      backlog.StoryStatus(story.Status),
			StoryPoints: story.StoryPoints,
			ActualHours: story.ActualHours,
		})
	}
	return metrics.ValidateSprintClosure(data)
}

type cierreBDDHistoria struct {
	id     int64
	status sprint.StoryStatus
	points int
}

type cierreBDDApp struct {
	*seleccionFixture
	httpServer *httptest.Server
	sprintID   int64
	historias  []cierreBDDHistoria
}

func nuevaCierreBDDApp(t *testing.T, estados ...sprint.StoryStatus) *cierreBDDApp {
	return nuevaCierreBDDAppConSprint(t, true, estados...)
}

func nuevaCierreBDDAppSinSprint(t *testing.T) *cierreBDDApp {
	return nuevaCierreBDDAppConSprint(t, false)
}

func nuevaCierreBDDAppConSprint(t *testing.T, crearSprint bool, estados ...sprint.StoryStatus) *cierreBDDApp {
	return nuevaCierreBDDAppConSprintYStore(t, crearSprint, nil, estados...)
}

func nuevaCierreBDDAppConSprintYStore(
	t *testing.T,
	crearSprint bool,
	nuevoStore func(*sql.DB) sprint.SprintCloseStore,
	estados ...sprint.StoryStatus,
) *cierreBDDApp {
	t.Helper()
	fixture := nuevaSeleccionFixture(t)
	server := web.NewServer(silencioso())
	web.RegisterContext(server, web.NewPostgreSQLContextStore(fixture.db))
	var closeStore sprint.SprintCloseStore = &cierreBDDStore{
		db:     fixture.db,
		closer: sprint.NewPostgreSQLSprintCloser(fixture.db),
	}
	if nuevoStore != nil {
		closeStore = nuevoStore(fixture.db)
	}
	service := sprint.NewCloseSprintService(
		closeStore,
		cierreBDDValidator{},
	)
	if err := web.RegisterClose(server, service, web.NewPostgreSQLBoardCatalog(fixture.db)); err != nil {
		t.Fatalf("no se pudo registrar la sección Cierre de Sprint: %v", err)
	}
	effortService := effort.NewService(
		effort.NewPostgreSQLEffortStore(fixture.db),
		func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) },
	)
	if err := web.RegisterEffort(server,
		web.NewEffortActions(effortService, func() time.Time {
			return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
		}),
		web.NewPostgreSQLEffortCatalog(fixture.db),
		func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) },
	); err != nil {
		t.Fatalf("no se pudo registrar Registro de esfuerzo para HU-03: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)

	app := &cierreBDDApp{
		seleccionFixture: fixture,
		httpServer:       httpServer,
	}
	if crearSprint {
		app.sprintID = fixture.insertar(t, `
			INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, 1, 'Objetivo BDD HU-15', DATE '2026-10-01', DATE '2026-10-15', 'Activo')
			RETURNING id`, fixture.proyectoA)
	}
	t.Cleanup(func() {
		_, _ = fixture.db.Exec(
			`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`,
			fixture.proyectoA,
		)
		_, _ = fixture.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, fixture.proyectoA)
	})
	for i, status := range estados {
		points := []int{3, 5, 8, 13}[i%4]
		id := fixture.insertar(t, `
			INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
			VALUES ($1, $2, 'Historia del escenario BDD de HU-15', 'Media', $3, $4, $5)
			RETURNING id`,
			fixture.proyectoA, fmt.Sprintf("HU15 BDD historia %d", i+1), points, string(status), app.sprintID)
		if _, err := fixture.db.Exec(
			`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`,
			app.sprintID, id,
		); err != nil {
			t.Fatalf("no se pudo asignar la historia BDD: %v", err)
		}
		app.historias = append(app.historias, cierreBDDHistoria{id: id, status: status, points: points})
	}
	return app
}

func (a *cierreBDDApp) elegirProyecto(t *testing.T) *http.Client {
	t.Helper()
	client := formNavegador(false)
	response, err := client.PostForm(a.httpServer.URL+"/contexto/proyecto", url.Values{
		"proyecto_id": {idTexto(a.proyectoA)},
		"volver_a":    {"/cierre"},
	})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto activo: %v", err)
	}
	_ = formCuerpo(t, response)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/cierre" {
		t.Fatalf("elegir proyecto: se esperaba 303 a /cierre, se obtuvo %d a %q",
			response.StatusCode, response.Header.Get("Location"))
	}
	return client
}

func (a *cierreBDDApp) elegirIntegrante(t *testing.T, client *http.Client) {
	t.Helper()
	response, err := client.PostForm(a.httpServer.URL+"/contexto/integrante", url.Values{
		"integrante_id": {idTexto(a.integranteA)},
		"volver_a":      {"/esfuerzo"},
	})
	if err != nil {
		t.Fatalf("no se pudo elegir el integrante: %v", err)
	}
	_ = formCuerpo(t, response)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/esfuerzo" {
		t.Fatalf("elegir integrante: se esperaba 303 a /esfuerzo, se obtuvo %d a %q",
			response.StatusCode, response.Header.Get("Location"))
	}
}

func (a *cierreBDDApp) cerrar(t *testing.T, client *http.Client) (*http.Response, string) {
	t.Helper()
	response, err := client.PostForm(a.httpServer.URL+"/cierre", url.Values{
		"sprint_id": {idTexto(a.sprintID)},
	})
	if err != nil {
		t.Fatalf("POST /cierre: %v", err)
	}
	return response, formCuerpo(t, response)
}

func (a *cierreBDDApp) obtener(t *testing.T, client *http.Client, path string) (*http.Response, string) {
	t.Helper()
	response, err := client.Get(a.httpServer.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return response, formCuerpo(t, response)
}

func TestCloseBDD_ResumenPrevioAntesDeConfirmar(t *testing.T) {
	// Dado un Sprint activo con historias Hecho, En progreso y En Sprint en el proyecto elegido.
	app := nuevaCierreBDDApp(t, sprint.Done, sprint.InProgress, sprint.EnSprint)
	client := app.elegirProyecto(t)

	// Cuando abro por HTTP la pantalla Cierre de Sprint sin confirmar el cierre.
	response, body := app.obtener(t, client, "/cierre")

	// Entonces veo el resumen con historias completadas y a devolver, los puntos y el botón.
	if response.StatusCode != http.StatusOK {
		t.Errorf("GET /cierre = %d; se esperaba 200", response.StatusCode)
	}
	for _, expected := range []string{
		"Sprint 1", "HU15 BDD historia 1", "HU15 BDD historia 2", "HU15 BDD historia 3",
		"Hecho", "En progreso", "En Sprint", "3 de 16", "Cerrar Sprint",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("el resumen previo debía incluir %q", expected)
		}
	}
	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, app.sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo comprobar el estado del Sprint: %v", err)
	}
	if status != string(sprint.SprintActive) {
		t.Errorf("la lectura del resumen cambió el Sprint a %q; debía seguir Activo", status)
	}
}

func TestCloseBDD_SinSprintActivoSinBoton(t *testing.T) {
	// Dado un proyecto elegido en HU-20b que no tiene ningún Sprint activo.
	app := nuevaCierreBDDAppSinSprint(t)
	client := app.elegirProyecto(t)

	// Cuando abro por HTTP la pantalla de Cierre de Sprint.
	response, body := app.obtener(t, client, "/cierre")

	// Entonces la pantalla informa que no hay Sprint activo y no ofrece cerrar uno.
	if response.StatusCode != http.StatusOK {
		t.Errorf("GET /cierre sin Sprint activo = %d; se esperaba 200", response.StatusCode)
	}
	if !strings.Contains(body, "No hay un Sprint activo para cerrar") {
		t.Errorf("la pantalla debía mostrar %q", "No hay un Sprint activo para cerrar")
	}
	if strings.Contains(body, "Cerrar Sprint") || strings.Contains(body, `name="sprint_id"`) {
		t.Errorf("la pantalla sin Sprint activo mostró un botón o formulario de cierre")
	}
}

func (a *cierreBDDApp) exigirRedireccionDeCierre(t *testing.T, response *http.Response) bool {
	t.Helper()
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/cierre" {
		t.Errorf("POST /cierre: se esperaba 303 a /cierre, se obtuvo %d a %q",
			response.StatusCode, response.Header.Get("Location"))
		return false
	}
	return true
}

func TestCloseBDD_CierreExitoso(t *testing.T) {
	// Dado un Sprint activo con una historia Hecho y dos historias no completadas.
	app := nuevaCierreBDDApp(t, sprint.Done, sprint.InProgress, sprint.EnSprint)
	client := app.elegirProyecto(t)

	// Cuando el Scrum Master solicita el cierre por HTTP.
	response, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, response) {
		return
	}

	// Entonces ve el texto exacto de confirmación, el Sprint finaliza y las no completadas
	// vuelven a Pendiente sin asignación.
	_, pantalla := app.obtener(t, client, "/cierre")
	const mensaje = "Sprint 1 cerrado correctamente. Historias devueltas al Backlog: 2"
	if !strings.Contains(pantalla, mensaje) {
		t.Errorf("la pantalla debía mostrar %q", mensaje)
	}
	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, app.sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
	}
	if status != string(sprint.SprintFinished) {
		t.Errorf("estado del Sprint = %q; se esperaba %q", status, sprint.SprintFinished)
	}
	for _, story := range app.historias {
		if story.status == sprint.Done {
			continue
		}
		assertCierreBDDStoryState(t, app.db, story.id, sprint.Pendiente, 0)
	}
}

func TestCloseBDD_HistoriasCompletadasConservadas(t *testing.T) {
	// Dado un Sprint activo con una historia Hecho estimada.
	app := nuevaCierreBDDApp(t, sprint.Done, sprint.InProgress)
	client := app.elegirProyecto(t)

	// Cuando se cierra el Sprint por HTTP.
	response, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, response) {
		return
	}

	// Entonces la historia Hecho mantiene el estado, los Story Points y su asignación
	// al Sprint finalizado.
	completed := app.historias[0]
	var status string
	var sprintID sql.NullInt64
	var points sql.NullInt64
	if err := app.db.QueryRow(`
		SELECT status, active_sprint_id, story_points
		FROM user_stories WHERE id = $1`, completed.id,
	).Scan(&status, &sprintID, &points); err != nil {
		t.Fatalf("no se pudo consultar la historia Hecho: %v", err)
	}
	if status != string(sprint.Done) || !sprintID.Valid || sprintID.Int64 != app.sprintID {
		t.Errorf("la historia Hecho quedó status=%q sprint=%v; se esperaba conservar su Sprint", status, sprintID)
	}
	if !points.Valid || int(points.Int64) != completed.points {
		t.Errorf("Story Points = %v; se esperaban %d", points, completed.points)
	}
	var assignments int
	if err := app.db.QueryRow(
		`SELECT count(*) FROM sprint_story_assignments WHERE sprint_id = $1 AND story_id = $2`,
		app.sprintID, completed.id,
	).Scan(&assignments); err != nil {
		t.Fatalf("no se pudo comprobar la asignación de la historia Hecho: %v", err)
	}
	if assignments != 1 {
		t.Errorf("asignaciones de la historia Hecho = %d; se esperaba 1", assignments)
	}
}

func TestCloseBDD_DatosDelSprintCongeladosTrasReasignar(t *testing.T) {
	// Dado un Sprint activo con una historia completada y otra que será devuelta.
	app := nuevaCierreBDDApp(t, sprint.Done, sprint.InProgress)
	client := app.elegirProyecto(t)

	// Cuando cierro el Sprint por HTTP, creo otro Sprint mediante HU-04 y reasigno
	// la historia devuelta.
	response, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, response) {
		return
	}
	before := readCierreBDDSnapshot(t, app.db, app.sprintID)
	returned := app.historias[1]
	newSprint := createSprintWithReturnedStories(t, app, []cierreBDDHistoria{returned})
	assertCierreBDDStoryState(t, app.db, returned.id, sprint.EnSprint, newSprint.Sprint.ID)

	// Entonces el registro del cierre conserva la misma historia, estado final y Story Points.
	after := readCierreBDDSnapshot(t, app.db, app.sprintID)
	if len(before) != len(after) {
		t.Fatalf("filas del snapshot antes=%d después=%d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("fila del snapshot %d cambió: antes=%+v después=%+v", i, before[i], after[i])
		}
	}
}

func TestCloseBDD_SprintSinHistoriasCompletadasYVelocidadCero(t *testing.T) {
	// Dado un Sprint activo con historias no completadas y ninguna historia Hecho.
	app := nuevaCierreBDDApp(t, sprint.InProgress, sprint.EnSprint)
	client := app.elegirProyecto(t)

	// Cuando solicito el cierre por HTTP.
	response, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, response) {
		return
	}

	// Entonces todas las historias vuelven al Backlog y HU-07 calcula velocidad cero
	// usando el servicio real de métricas.
	for _, story := range app.historias {
		assertCierreBDDStoryState(t, app.db, story.id, sprint.Pendiente, 0)
	}
	result, err := metrics.NewService(metrics.NewPostgreSQLMetricsStore(app.db)).CalculateMetrics(app.sprintID)
	if err != nil {
		t.Fatalf("HU-07 CalculateMetrics() devolvió error: %v", err)
	}
	if result.Velocity != 0 {
		t.Errorf("HU-07 calculó velocidad %d; se esperaba 0", result.Velocity)
	}
}

func TestCloseBDD_HistoriasDevueltasDisponiblesParaUnSprintNuevo(t *testing.T) {
	// Dado un Sprint activo con dos historias que volverán al Backlog.
	app := nuevaCierreBDDApp(t, sprint.InProgress, sprint.EnSprint)
	client := app.elegirProyecto(t)

	// Cuando cierro por HTTP y preparo un Sprint nuevo con el servicio real de HU-04.
	response, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, response) {
		return
	}
	options, err := web.NewPostgreSQLSprintCatalog(app.db).ListStoryOptions(context.Background(), app.proyectoA)
	if err != nil {
		t.Fatalf("HU-04 no pudo consultar las historias disponibles: %v", err)
	}
	available := make(map[int64]bool, len(options))
	for _, option := range options {
		available[option.ID] = option.Available
	}
	returned := append([]cierreBDDHistoria(nil), app.historias...)
	for _, story := range returned {
		if !available[story.id] {
			t.Errorf("historia devuelta %d no aparece disponible en HU-04", story.id)
		}
	}
	created := createSprintWithReturnedStories(t, app, returned)

	// Entonces las historias quedan asignadas al nuevo Sprint.
	for _, story := range returned {
		assertCierreBDDStoryState(t, app.db, story.id, sprint.EnSprint, created.Sprint.ID)
	}
}

func TestCloseBDD_SprintYaCerradoSeRechaza(t *testing.T) {
	// Dado un Sprint que ya se cerró correctamente desde la aplicación.
	app := nuevaCierreBDDApp(t, sprint.Done)
	client := app.elegirProyecto(t)
	first, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, first) {
		return
	}

	// Cuando vuelvo a enviar el mismo formulario de cierre.
	response, body := app.cerrar(t, client)

	// Entonces recibo 409 y el mensaje de Sprint ya cerrado; no se cierra nuevamente.
	if response.StatusCode != http.StatusConflict {
		t.Errorf("cerrar un Sprint ya finalizado = %d; se esperaba 409", response.StatusCode)
	}
	if !strings.Contains(body, web.MsgSprintAlreadyClosed) {
		t.Errorf("la respuesta debía mostrar %q", web.MsgSprintAlreadyClosed)
	}
	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, app.sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
	}
	if status != string(sprint.SprintFinished) {
		t.Errorf("el Sprint quedó %q; se esperaba que siguiera Finalizado", status)
	}
}

func TestCloseBDD_CierreBloqueadoPorValidacionHU07(t *testing.T) {
	// Dado un Sprint activo con una historia Hecho cuyos Story Points son inválidos para HU-07.
	app := nuevaCierreBDDApp(t, sprint.Done, sprint.InProgress)
	invalidStory := app.historias[0]
	if _, err := app.db.Exec(`UPDATE user_stories SET story_points = NULL WHERE id = $1`, invalidStory.id); err != nil {
		t.Fatalf("no se pudieron preparar los datos inválidos para HU-07: %v", err)
	}
	client := app.elegirProyecto(t)

	// Cuando solicito el cierre por HTTP y se ejecuta la validación real de HU-07.
	response, body := app.cerrar(t, client)

	// Entonces recibo 409 con el motivo y la historia a corregir, sin cambios al Sprint ni sus historias.
	if response.StatusCode != http.StatusConflict {
		t.Errorf("cierre bloqueado por HU-07 = %d; se esperaba 409", response.StatusCode)
	}
	want := fmt.Sprintf("Historia %d: Story Points inválidos", invalidStory.id)
	if !strings.Contains(body, want) {
		t.Errorf("la respuesta debía identificar el dato inválido %q", want)
	}
	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, app.sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
	}
	if status != string(sprint.SprintActive) {
		t.Errorf("el Sprint quedó %q; se esperaba Activo", status)
	}
	var results int
	if err := app.db.QueryRow(`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`, app.sprintID).Scan(&results); err != nil {
		t.Fatalf("no se pudo consultar el registro del cierre: %v", err)
	}
	if results != 0 {
		t.Errorf("se registraron %d resultados aunque HU-07 bloqueó el cierre", results)
	}
}

func TestCloseBDD_FallaInternaAlCerrarMuestraMensajeGenerico(t *testing.T) {
	// Dado un Sprint activo y una falla de almacenamiento durante el cierre.
	app := nuevaCierreBDDAppConSprintYStore(t, true, func(db *sql.DB) sprint.SprintCloseStore {
		return cierreBDDFallaAlCerrarStore{
			cierreBDDStore: &cierreBDDStore{db: db, closer: sprint.NewPostgreSQLSprintCloser(db)},
			err:            errors.New("db connection timeout during sprint close"),
		}
	}, sprint.Done, sprint.InProgress)
	client := app.elegirProyecto(t)

	// Cuando solicito el cierre por HTTP.
	response, body := app.cerrar(t, client)

	// Entonces recibo 500 y el mensaje genérico, sin el detalle técnico ni mensaje de éxito.
	if response.StatusCode != http.StatusInternalServerError {
		t.Errorf("falla interna al cerrar = %d; se esperaba 500", response.StatusCode)
	}
	if !strings.Contains(body, web.MsgOperationFailed) {
		t.Errorf("la respuesta debía mostrar %q", web.MsgOperationFailed)
	}
	if strings.Contains(body, "db connection timeout") || strings.Contains(body, "sprint close") {
		t.Errorf("la respuesta expuso detalles internos de persistencia")
	}
	if strings.Contains(body, "cerrado correctamente") {
		t.Errorf("la respuesta comunicó un cierre exitoso")
	}
	var status string
	if err := app.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, app.sprintID).Scan(&status); err != nil {
		t.Fatalf("no se pudo leer el estado del Sprint: %v", err)
	}
	if status != string(sprint.SprintActive) {
		t.Errorf("el Sprint quedó %q; se esperaba que siguiera Activo", status)
	}
	if results := len(readCierreBDDSnapshot(t, app.db, app.sprintID)); results != 0 {
		t.Errorf("se persistieron %d resultados pese a la falla", results)
	}
}

func TestCloseBDD_CargaDeEsfuerzoPosteriorAlCierreSeRechaza(t *testing.T) {
	// Dado un Sprint cerrado y un Sprint siguiente activo para el mismo proyecto.
	app := nuevaCierreBDDApp(t, sprint.Done, sprint.InProgress)
	client := app.elegirProyecto(t)
	response, _ := app.cerrar(t, client)
	if !app.exigirRedireccionDeCierre(t, response) {
		return
	}
	nextSprint := app.insertar(t, `
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, 2, 'Sprint posterior BDD', DATE '2026-10-16', DATE '2026-10-30', 'Activo')
		RETURNING id`, app.proyectoA)
	newStory := app.insertar(t, `
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status, active_sprint_id)
		VALUES ($1, 'Historia del Sprint posterior', 'Fixture BDD', 'Media', 3, 'En Sprint', $2)
		RETURNING id`, app.proyectoA, nextSprint)
	if _, err := app.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`,
		nextSprint, newStory); err != nil {
		t.Fatalf("no se pudo asignar la historia al Sprint posterior: %v", err)
	}
	app.elegirIntegrante(t, client)
	before := countCierreBDDEffort(t, app.db, app.historias[0].id, app.historias[1].id)

	// Cuando intento registrar horas por HTTP sobre la historia Hecho conservada y la devuelta.
	for _, story := range app.historias {
		response, body := app.postEsfuerzo(t, client, story.id)

		// Entonces HU-03 rechaza la historia porque no pertenece al Sprint activo.
		if response.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("registrar esfuerzo sobre historia cerrada %d = %d; se esperaba 422",
				story.id, response.StatusCode)
		}
		if !strings.Contains(body, web.MsgEffortStoryNotInSprint) {
			t.Errorf("la respuesta debía mostrar %q", web.MsgEffortStoryNotInSprint)
		}
		if strings.Contains(body, effort.SuccessMessage) {
			t.Errorf("la respuesta indicó que se registraron horas")
		}
	}
	if after := countCierreBDDEffort(t, app.db, app.historias[0].id, app.historias[1].id); after != before {
		t.Errorf("se guardaron horas después del cierre: antes %d, después %d", before, after)
	}
}

func (a *cierreBDDApp) postEsfuerzo(t *testing.T, client *http.Client, storyID int64) (*http.Response, string) {
	t.Helper()
	response, err := client.PostForm(a.httpServer.URL+"/esfuerzo", url.Values{
		"historia":  {idTexto(storyID)},
		"fecha":     {"2026-10-20"},
		"actividad": {"Trabajo posterior al cierre"},
		"horas":     {"2"},
	})
	if err != nil {
		t.Fatalf("POST /esfuerzo: %v", err)
	}
	return response, formCuerpo(t, response)
}

func countCierreBDDEffort(t *testing.T, db *sql.DB, storyIDs ...int64) int {
	t.Helper()
	var count int
	for _, storyID := range storyIDs {
		var storyCount int
		if err := db.QueryRow(`SELECT count(*) FROM effort_logs WHERE story_id = $1`, storyID).Scan(&storyCount); err != nil {
			t.Fatalf("no se pudo contar el esfuerzo de la historia %d: %v", storyID, err)
		}
		count += storyCount
	}
	return count
}

type cierreBDDSnapshot struct {
	storyID     int64
	finalStatus string
	storyPoints sql.NullInt64
}

func readCierreBDDSnapshot(t *testing.T, db *sql.DB, sprintID int64) []cierreBDDSnapshot {
	t.Helper()
	rows, err := db.Query(`
		SELECT story_id, final_status, story_points
		FROM sprint_story_results WHERE sprint_id = $1 ORDER BY story_id`, sprintID)
	if err != nil {
		t.Fatalf("no se pudo leer el registro del Sprint: %v", err)
	}
	defer rows.Close()
	var snapshot []cierreBDDSnapshot
	for rows.Next() {
		var row cierreBDDSnapshot
		if err := rows.Scan(&row.storyID, &row.finalStatus, &row.storyPoints); err != nil {
			t.Fatalf("no se pudo leer una fila del registro: %v", err)
		}
		snapshot = append(snapshot, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("no se pudo recorrer el registro del Sprint: %v", err)
	}
	return snapshot
}

func createSprintWithReturnedStories(
	t *testing.T,
	app *cierreBDDApp,
	stories []cierreBDDHistoria,
) sprint.SprintCreationResult {
	t.Helper()
	var nextNumber int64
	if err := app.db.QueryRow(
		`SELECT COALESCE(MAX(number), 0) + 1 FROM sprints WHERE project_id = $1`,
		app.proyectoA,
	).Scan(&nextNumber); err != nil {
		t.Fatalf("no se pudo consultar el próximo número de Sprint: %v", err)
	}
	selected := make([]sprint.UserStory, 0, len(stories))
	ids := make([]int64, 0, len(stories))
	for _, story := range stories {
		selected = append(selected, sprint.UserStory{
			ID:        story.id,
			ProjectID: app.proyectoA,
			Status:    sprint.Pendiente,
		})
		ids = append(ids, story.id)
	}
	store, err := sprint.NewPostgreSQLSprintStore(os.Getenv(web.EnvDatabaseURL))
	if err != nil {
		t.Fatalf("no se pudo crear el store PostgreSQL de HU-04: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("no se pudo cerrar el store PostgreSQL de HU-04: %v", err)
		}
	})
	result, err := sprint.NewService(store).CreateSprint(
		context.Background(),
		sprint.SprintCreationRequest{
			ProjectID: app.proyectoA,
			Goal:      "Sprint siguiente BDD",
			StartDate: time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, time.November, 14, 0, 0, 0, 0, time.UTC),
			StoryIDs:  ids,
		},
		sprint.Project{
			ID:               app.proyectoA,
			Status:           "Activo",
			NextSprintNumber: nextNumber,
		},
		selected,
	)
	if err != nil {
		t.Fatalf("HU-04 no pudo crear el nuevo Sprint con las historias devueltas: %v", err)
	}
	return result
}

func assertCierreBDDStoryState(t *testing.T, db *sql.DB, storyID int64, status sprint.StoryStatus, sprintID int64) {
	t.Helper()
	var gotStatus string
	var activeSprint sql.NullInt64
	if err := db.QueryRow(
		`SELECT status, active_sprint_id FROM user_stories WHERE id = $1`,
		storyID,
	).Scan(&gotStatus, &activeSprint); err != nil {
		t.Fatalf("no se pudo leer la historia %d: %v", storyID, err)
	}
	if gotStatus != string(status) {
		t.Errorf("estado de historia %d = %q; se esperaba %q", storyID, gotStatus, status)
	}
	if sprintID == 0 {
		if activeSprint.Valid {
			t.Errorf("historia %d conserva active_sprint_id=%d; se esperaba NULL", storyID, activeSprint.Int64)
		}
		return
	}
	if !activeSprint.Valid || activeSprint.Int64 != sprintID {
		t.Errorf("active_sprint_id de historia %d = %v; se esperaba %d", storyID, activeSprint, sprintID)
	}
}
