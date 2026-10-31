package web

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

// HU-19: resolver y reabrir defectos desde la pantalla /defectos.

// T025: servidor de prueba con cookiejar, proyecto elegido por el flujo de HU-20b y la sección de
// Defectos registrada con el servicio de resolución. Los Sprints y defectos los arma cada prueba.
type defectResolutionFixture struct {
	app            *ctxApp
	storyA, storyB int64
	stamp          int64
}

func newDefectResolutionFixture(t *testing.T) *defectResolutionFixture {
	t.Helper()
	app := newCtxApp(t, nil, func(server *Server) {
		db := openContextTestDB(t)
		if err := RegisterDefectsWithResolution(
			server,
			defect.NewService(defect.NewPostgreSQLStore(db)),
			defect.NewResolutionService(defect.NewPostgreSQLResolutionStore(db)),
			NewPostgreSQLDefectCatalog(db),
		); err != nil {
			t.Fatalf("no se pudo registrar la sección Defectos con resolución: %v", err)
		}
	})
	f := &defectResolutionFixture{app: app, stamp: time.Now().UnixNano()}
	// Se borra antes que los proyectos del contexto: las historias no caen en cascada.
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM user_stories WHERE project_id IN ($1, $2)`,
			`DELETE FROM sprints WHERE project_id IN ($1, $2)`,
		} {
			if _, err := app.db.Exec(query, app.fx.projectA, app.fx.projectB); err != nil {
				t.Errorf("no se pudieron borrar los datos de HU-19: %v", err)
			}
		}
	})
	f.storyA = f.insertID(t,
		`INSERT INTO user_stories (project_id, title, description, priority, status)
		 VALUES ($1, $2, 'Historia con defectos', 'Media', 'Pendiente') RETURNING id`,
		app.fx.projectA, fmt.Sprintf("HU19 historia A %d", f.stamp))
	f.storyB = f.insertID(t,
		`INSERT INTO user_stories (project_id, title, description, priority, status)
		 VALUES ($1, $2, 'Historia con defectos', 'Media', 'Pendiente') RETURNING id`,
		app.fx.projectB, fmt.Sprintf("HU19 historia B %d", f.stamp))
	return f
}

func (f *defectResolutionFixture) insertID(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.app.db.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo preparar el escenario de HU-19: %v", err)
	}
	return id
}

func (f *defectResolutionFixture) sprint(t *testing.T, projectID int64, number int, status string) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		 VALUES ($1, $2, $3, '2026-01-01', '2026-01-14', $4) RETURNING id`,
		projectID, number, fmt.Sprintf("HU19 handler %d", number), status)
}

func (f *defectResolutionFixture) openDefect(t *testing.T, projectID, storyID, detectionSprintID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id)
		 VALUES ($1, $2, 'Defecto abierto de HU-19', 'Alta', 'Abierto', $3) RETURNING id`,
		projectID, storyID, detectionSprintID)
}

func (f *defectResolutionFixture) resolvedDefect(t *testing.T, projectID, storyID, detectionSprintID, resolutionSprintID int64) int64 {
	t.Helper()
	return f.insertID(t,
		`INSERT INTO defects (project_id, story_id, description, severity, status, detection_sprint_id, resolution_sprint_id)
		 VALUES ($1, $2, 'Defecto resuelto de HU-19', 'Media', 'Resuelto', $3, $4) RETURNING id`,
		projectID, storyID, detectionSprintID, resolutionSprintID)
}

// state devuelve el estado y el Sprint de resolución guardados del defecto.
func (f *defectResolutionFixture) state(t *testing.T, defectID int64) (string, sql.NullInt64) {
	t.Helper()
	var status string
	var resolution sql.NullInt64
	if err := f.app.db.QueryRow(
		`SELECT status, resolution_sprint_id FROM defects WHERE id = $1`, defectID,
	).Scan(&status, &resolution); err != nil {
		t.Fatalf("no se pudo leer el defecto %d: %v", defectID, err)
	}
	return status, resolution
}

func (f *defectResolutionFixture) assertState(t *testing.T, defectID int64, wantStatus string, wantResolution int64) {
	t.Helper()
	status, resolution := f.state(t, defectID)
	if status != wantStatus {
		t.Errorf("estado del defecto %d = %q; se esperaba %q", defectID, status, wantStatus)
	}
	switch {
	case wantResolution == 0 && resolution.Valid:
		t.Errorf("el defecto %d tiene Sprint de resolución %d; se esperaba ninguno", defectID, resolution.Int64)
	case wantResolution != 0 && (!resolution.Valid || resolution.Int64 != wantResolution):
		t.Errorf("Sprint de resolución del defecto %d = %v; se esperaba %d", defectID, resolution, wantResolution)
	}
}

// clientA devuelve un navegador que no sigue redirecciones, con el proyecto A elegido.
func (f *defectResolutionFixture) clientA(t *testing.T) *http.Client {
	t.Helper()
	client := f.app.client(false)
	resp, _ := f.app.selectProject(t, client, id64(f.app.fx.projectA), "/defectos")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto A: %d", resp.StatusCode)
	}
	return client
}

func (f *defectResolutionFixture) resolve(t *testing.T, client *http.Client, defectID string, extra url.Values) (*http.Response, string) {
	t.Helper()
	values := url.Values{"defecto": {defectID}}
	for key, entries := range extra {
		values[key] = entries
	}
	return f.app.post(t, client, "/defectos/resolver", values)
}

// T026 [E1]: resolver un defecto abierto registra el Sprint activo y muestra el éxito una sola vez.
func TestHU19_ResolucionExitosa(t *testing.T) {
	f := newDefectResolutionFixture(t)
	detection := f.sprint(t, f.app.fx.projectA, 1, "Finalizado")
	active := f.sprint(t, f.app.fx.projectA, 2, "Activo")
	f.sprint(t, f.app.fx.projectB, 7, "Activo")
	defectID := f.openDefect(t, f.app.fx.projectA, f.storyA, detection)
	client := f.clientA(t)

	// Un proyecto enviado por formulario se ignora: vale el proyecto activo.
	resp, _ := f.resolve(t, client, id64(defectID), url.Values{"project_id": {id64(f.app.fx.projectB)}})

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/defectos" {
		t.Fatalf("POST /defectos/resolver = %d a %q; se esperaba 303 a /defectos",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	f.assertState(t, defectID, "Resuelto", active)

	resp, body := f.app.get(t, client, "/defectos")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /defectos = %d; se esperaba 200", resp.StatusCode)
	}
	if strings.Count(body, "Defecto resuelto en el Sprint 2") != 1 {
		t.Errorf("la pantalla debía mostrar una vez \"Defecto resuelto en el Sprint 2\"")
	}
	if !strings.Contains(body, "<td>Resuelto</td>") || !strings.Contains(body, "<td>Sprint 2</td>") {
		t.Errorf("la lista debía mostrar el defecto Resuelto con Sprint 2 de resolución")
	}
	if _, body := f.app.get(t, client, "/defectos"); strings.Contains(body, "Defecto resuelto en el Sprint") {
		t.Errorf("el mensaje de éxito no debe repetirse al recargar")
	}
}

// T027 [E2]: sin Sprint activo la resolución se bloquea y el defecto sigue abierto.
func TestHU19_ResolverSinSprintActivo(t *testing.T) {
	f := newDefectResolutionFixture(t)
	finished := f.sprint(t, f.app.fx.projectA, 1, "Finalizado")
	defectID := f.openDefect(t, f.app.fx.projectA, f.storyA, finished)
	client := f.clientA(t)

	resp, body := f.resolve(t, client, id64(defectID), nil)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /defectos/resolver sin Sprint activo = %d; se esperaba 409", resp.StatusCode)
	}
	if !strings.Contains(body, "No hay un Sprint activo para registrar la resolución del defecto") {
		t.Errorf("la respuesta debía informar que no hay Sprint activo para registrar la resolución")
	}
	f.assertState(t, defectID, "Abierto", 0)
}

// T028 [E4]: el Sprint activo (2) es anterior al de detección (3): se rechaza.
func TestHU19_SprintAnteriorAlDeDeteccion(t *testing.T) {
	f := newDefectResolutionFixture(t)
	detection := f.sprint(t, f.app.fx.projectA, 3, "Finalizado")
	f.sprint(t, f.app.fx.projectA, 2, "Activo")
	defectID := f.openDefect(t, f.app.fx.projectA, f.storyA, detection)
	client := f.clientA(t)

	resp, body := f.resolve(t, client, id64(defectID), nil)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /defectos/resolver con Sprint anterior = %d; se esperaba 409", resp.StatusCode)
	}
	if !strings.Contains(body, "El Sprint de resolución no puede ser anterior al Sprint en que se detectó el defecto") {
		t.Errorf("la respuesta debía informar que el Sprint de resolución no puede ser anterior al de detección")
	}
	f.assertState(t, defectID, "Abierto", 0)
}

// T029 [E7]: un defecto ya resuelto en el Sprint 2 no se vuelve a resolver.
func TestHU19_YaResuelto(t *testing.T) {
	f := newDefectResolutionFixture(t)
	sprint2 := f.sprint(t, f.app.fx.projectA, 2, "Finalizado")
	f.sprint(t, f.app.fx.projectA, 3, "Activo")
	defectID := f.resolvedDefect(t, f.app.fx.projectA, f.storyA, sprint2, sprint2)
	client := f.clientA(t)

	resp, body := f.resolve(t, client, id64(defectID), nil)

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("POST /defectos/resolver de un defecto resuelto = %d; se esperaba 409", resp.StatusCode)
	}
	if !strings.Contains(body, "El defecto ya está resuelto") {
		t.Errorf("la respuesta debía informar que el defecto ya está resuelto")
	}
	f.assertState(t, defectID, "Resuelto", sprint2)
}
