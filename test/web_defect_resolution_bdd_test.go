package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// HU-19: escenarios BDD de resolver y reabrir defectos desde la pantalla web.

type resolucionBDDApp struct {
	*seleccionFixture
	httpServer *httptest.Server
	historia   int64
}

func nuevaResolucionBDDApp(t *testing.T) *resolucionBDDApp {
	t.Helper()
	fixture := nuevaSeleccionFixture(t)
	server := web.NewServer(silencioso())
	web.RegisterContext(server, web.NewPostgreSQLContextStore(fixture.db))
	if err := web.RegisterDefectsWithResolution(
		server,
		defect.NewService(defect.NewPostgreSQLStore(fixture.db)),
		defect.NewResolutionService(defect.NewPostgreSQLResolutionStore(fixture.db)),
		web.NewPostgreSQLDefectCatalog(fixture.db),
	); err != nil {
		t.Fatalf("no se pudo registrar la sección Defectos con resolución: %v", err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	t.Cleanup(func() {
		_, _ = fixture.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, fixture.proyectoA)
		_, _ = fixture.db.Exec(`DELETE FROM sprints WHERE project_id = $1`, fixture.proyectoA)
	})
	app := &resolucionBDDApp{seleccionFixture: fixture, httpServer: httpServer}
	app.historia = fixture.insertar(t, `
		INSERT INTO user_stories (project_id, title, description, priority, status)
		VALUES ($1, 'HU19 BDD historia', 'Historia del escenario BDD de HU-19', 'Media', 'Pendiente')
		RETURNING id`, fixture.proyectoA)
	return app
}

func (a *resolucionBDDApp) sprint(t *testing.T, number int, status string) int64 {
	t.Helper()
	return a.insertar(t, `
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, DATE '2026-10-01', DATE '2026-10-15', $4)
		RETURNING id`, a.proyectoA, number, fmt.Sprintf("Objetivo BDD HU-19 %d", number), status)
}

func (a *resolucionBDDApp) defectoAbierto(t *testing.T, detectionSprintID int64) int64 {
	t.Helper()
	return a.insertar(t, `
		INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		VALUES ($1, $2, 'Defecto del escenario BDD', 'Alta', 'Abierto', $3)
		RETURNING id`, a.proyectoA, a.historia, detectionSprintID)
}

func (a *resolucionBDDApp) elegirProyecto(t *testing.T) *http.Client {
	t.Helper()
	client := formNavegador(false)
	response, err := client.PostForm(a.httpServer.URL+"/contexto/proyecto", url.Values{
		"proyecto_id": {idTexto(a.proyectoA)},
		"volver_a":    {"/defectos"},
	})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto activo: %v", err)
	}
	_ = formCuerpo(t, response)
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: se esperaba 303, se obtuvo %d", response.StatusCode)
	}
	return client
}

func (a *resolucionBDDApp) enviar(t *testing.T, client *http.Client, path string, defectID int64) (*http.Response, string) {
	t.Helper()
	response, err := client.PostForm(a.httpServer.URL+path, url.Values{"defecto": {idTexto(defectID)}})
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return response, formCuerpo(t, response)
}

func (a *resolucionBDDApp) obtener(t *testing.T, client *http.Client, path string) (*http.Response, string) {
	t.Helper()
	response, err := client.Get(a.httpServer.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return response, formCuerpo(t, response)
}

func (a *resolucionBDDApp) estado(t *testing.T, defectID int64) (string, *int64) {
	t.Helper()
	var status string
	var resolution *int64
	if err := a.db.QueryRow(
		`SELECT status, resolution_sprint_id FROM defects WHERE id = $1`, defectID,
	).Scan(&status, &resolution); err != nil {
		t.Fatalf("no se pudo leer el defecto %d: %v", defectID, err)
	}
	return status, resolution
}

// Escenario 1: Resolución exitosa
//
//	Dado un defecto "Abierto" del proyecto
//	Y un Sprint activo en el proyecto
//	Cuando resuelvo el defecto
//	Entonces el defecto pasa a "Resuelto"
//	Y el Sprint activo queda registrado como "Sprint de resolución"
//	Y el sistema muestra un mensaje de éxito
func TestEscenarioHU19_ResolucionExitosa(t *testing.T) {
	app := nuevaResolucionBDDApp(t)
	activo := app.sprint(t, 1, "Activo")
	defecto := app.defectoAbierto(t, activo)
	client := app.elegirProyecto(t)

	response, _ := app.enviar(t, client, "/defectos/resolver", defecto)

	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("resolver: se esperaba 303, se obtuvo %d", response.StatusCode)
	}
	status, resolution := app.estado(t, defecto)
	if status != "Resuelto" || resolution == nil || *resolution != activo {
		t.Errorf("el defecto quedó %q con Sprint de resolución %v; se esperaba Resuelto en el Sprint activo", status, resolution)
	}
	if _, body := app.obtener(t, client, "/defectos"); !strings.Contains(body, "Defecto resuelto en el Sprint 1") {
		t.Errorf("la pantalla debía mostrar el mensaje de éxito con el Sprint de resolución")
	}
}

// Escenario 2: Resolver sin Sprint activo
//
//	Dado un defecto "Abierto" del proyecto
//	Y que el proyecto no tiene un Sprint activo
//	Cuando intento resolver el defecto
//	Entonces el sistema bloquea la resolución
//	Y informa que no hay un Sprint activo para registrar la resolución
//	Y el defecto sigue "Abierto"
func TestEscenarioHU19_ResolverSinSprintActivo(t *testing.T) {
	app := nuevaResolucionBDDApp(t)
	finalizado := app.sprint(t, 1, "Finalizado")
	defecto := app.defectoAbierto(t, finalizado)
	client := app.elegirProyecto(t)

	response, body := app.enviar(t, client, "/defectos/resolver", defecto)

	if response.StatusCode != http.StatusConflict {
		t.Errorf("resolver sin Sprint activo: se esperaba 409, se obtuvo %d", response.StatusCode)
	}
	if !strings.Contains(body, "No hay un Sprint activo para registrar la resolución del defecto") {
		t.Errorf("la respuesta debía informar que no hay Sprint activo para registrar la resolución")
	}
	if status, resolution := app.estado(t, defecto); status != "Abierto" || resolution != nil {
		t.Errorf("el defecto quedó %q con Sprint de resolución %v; se esperaba Abierto sin Sprint", status, resolution)
	}
}
