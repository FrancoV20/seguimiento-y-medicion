package web

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-16, T011, T018 y T019: "Sprints anteriores" con el servidor, el contexto de HU-20b, el
// catálogo, HU-07 y PostgreSQL reales, y los datos de past_sprint_fixtures_test.go.

type pastConfig struct {
	catalog func(PastSprintCatalog) PastSprintCatalog
	calc    func(PastSprintMetrics) PastSprintMetrics
}

type pastOption func(*pastConfig)

// withPastCatalog reemplaza el catálogo (para contar llamadas o simular fallas).
func withPastCatalog(wrap func(PastSprintCatalog) PastSprintCatalog) pastOption {
	return func(c *pastConfig) { c.catalog = wrap }
}

// withPastMetrics reemplaza HU-07 (para contar llamadas o simular fallas).
func withPastMetrics(wrap func(PastSprintMetrics) PastSprintMetrics) pastOption {
	return func(c *pastConfig) { c.calc = wrap }
}

type pastApp struct {
	*ctxApp
	fx *pastSprintFixture
}

// newPastApp arma el servidor con el contexto de trabajo y "Sprints anteriores" registrados.
func newPastApp(t *testing.T, opts ...pastOption) *pastApp {
	t.Helper()
	var cfg pastConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	db := openContextTestDB(t)
	var catalog PastSprintCatalog = NewPostgreSQLPastSprintCatalog(db)
	if cfg.catalog != nil {
		catalog = cfg.catalog(catalog)
	}
	var calc PastSprintMetrics = metrics.NewService(metrics.NewPostgreSQLMetricsStore(db))
	if cfg.calc != nil {
		calc = cfg.calc(calc)
	}
	app := newCtxApp(t, nil, func(s *Server) {
		if err := RegisterPastSprints(s, catalog, calc); err != nil {
			t.Fatalf("no se pudo registrar Sprints anteriores: %v", err)
		}
	})
	return &pastApp{ctxApp: app, fx: newPastSprintFixture(t, db)}
}

// open devuelve un navegador (que no sigue redirecciones) con el proyecto ya elegido.
func (a *pastApp) open(t *testing.T, projectID int64) *http.Client {
	t.Helper()
	c := a.client(false)
	if resp, _ := a.selectProject(t, c, id64(projectID), "/sprints-anteriores"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("no se pudo elegir el proyecto: se obtuvo %d", resp.StatusCode)
	}
	return c
}

// page elige el proyecto y pide la dirección.
func (a *pastApp) page(t *testing.T, projectID int64, path string) (*http.Response, string) {
	t.Helper()
	return a.get(t, a.open(t, projectID), path)
}

func pastWantText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en la pantalla", want)
		}
	}
}

func pastWantNoText(t *testing.T, body string, unwanted ...string) {
	t.Helper()
	for _, text := range unwanted {
		if strings.Contains(body, text) {
			t.Errorf("la pantalla no debe contener %q", text)
		}
	}
}

func pastWantStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("se esperaba %d, se obtuvo %d", want, resp.StatusCode)
	}
}

// --- T011: lista ---

// Escenarios 1 y 2.
func TestHU16_Listado(t *testing.T) {
	app := newPastApp(t)
	resp, body := app.page(t, app.fx.alfa, "/sprints-anteriores")

	pastWantStatus(t, resp, http.StatusOK)
	pastWantText(t, body,
		"Sprints anteriores",
		"Sprint 2", "Gestionar perfiles y reportes", "Del 01/09/2026 al 14/09/2026",
		"Sprint 1", "Dar de alta usuarios", "Del 18/08/2026 al 31/08/2026",
		fmt.Sprintf(`href="/sprints-anteriores/%d"`, app.fx.sprints["Alfa 2"]),
		fmt.Sprintf(`href="/sprints-anteriores/%d"`, app.fx.sprints["Alfa 1"]),
	)
	if strings.Index(body, "Sprint 2") > strings.Index(body, "Sprint 1") {
		t.Error("el Sprint 2 debía aparecer antes que el Sprint 1")
	}
	pastWantNoText(t, body, "Sprint 3", "Exportar e informar", "Migrar datos", "Auditar accesos", MsgPastSprintsEmpty)
}

// Escenario 6.
func TestHU16_SinSprintsAnteriores(t *testing.T) {
	app := newPastApp(t)
	resp, body := app.page(t, app.fx.beta, "/sprints-anteriores")

	pastWantStatus(t, resp, http.StatusOK)
	pastWantText(t, body, "Aún no hay Sprints anteriores")
}

// T014: con un argumento nil no se registra nada.
func TestRegisterPastSprints_ArgumentosNil(t *testing.T) {
	srv := NewServer(nil)
	calc := metrics.NewService(nil)
	catalog := NewPostgreSQLPastSprintCatalog(nil)
	for name, register := range map[string]func() error{
		"servidor": func() error { return RegisterPastSprints(nil, catalog, calc) },
		"catálogo": func() error { return RegisterPastSprints(srv, nil, calc) },
		"métricas": func() error { return RegisterPastSprints(srv, catalog, nil) },
	} {
		if err := register(); err == nil {
			t.Errorf("sin %s se esperaba un error", name)
		}
	}
}

// FR-013: sin proyecto activo, HU-20b pide elegirlo y vuelve a la dirección pedida.
func TestHU16_SinProyecto(t *testing.T) {
	app := newPastApp(t)
	paths := map[string]string{"/sprints-anteriores": "/contexto/proyecto?volver_a=%2Fsprints-anteriores"}
	for path, want := range paths {
		resp, _ := app.get(t, app.client(false), path)
		pastWantStatus(t, resp, http.StatusSeeOther)
		if loc := resp.Header.Get("Location"); loc != want {
			t.Errorf("%s: se esperaba %q, se obtuvo %q", path, want, loc)
		}
	}
}
