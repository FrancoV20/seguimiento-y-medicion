package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// T056: ningún error del contexto de trabajo expone detalles internos (driver, consultas,
// direcciones, trazas). Se prueba con una base que falla con un error técnico real del driver.
func TestContexto_ErroresNoExponenDetallesInternos(t *testing.T) {
	caida := func(ContextStore) ContextStore {
		db := openContextTestDB(t)
		_ = db.Close() // toda consulta falla con "sql: database is closed"
		return NewPostgreSQLContextStore(db)
	}
	app := newCtxApp(t, caida)
	prohibidos := []string{"sql:", "pgx", "postgres", "select ", "connection", "closed",
		"5433", "localhost", "database_url", ".go:", "goroutine", "panic"}

	revisar := func(t *testing.T, resp *http.Response, cuerpo string) {
		t.Helper()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("se esperaba 503, se obtuvo %d", resp.StatusCode)
		}
		if !strings.Contains(cuerpo, MsgUnavailable) {
			t.Errorf("falta el mensaje genérico %q", MsgUnavailable)
		}
		bajo := strings.ToLower(cuerpo)
		for _, p := range prohibidos {
			if strings.Contains(bajo, p) {
				t.Errorf("la página expone un detalle interno: contiene %q", p)
			}
		}
	}

	t.Run("lectura con la selección guardada", func(t *testing.T) {
		c := app.client(false)
		app.setCookies(t, c, idCookie(CookieProject, app.fx.projectA))
		resp, cuerpo := app.get(t, c, "/libre")
		revisar(t, resp, cuerpo)
	})
	t.Run("lista de proyectos", func(t *testing.T) {
		resp, cuerpo := app.get(t, app.client(false), "/contexto/proyecto")
		revisar(t, resp, cuerpo)
	})
	t.Run("elegir un proyecto", func(t *testing.T) {
		resp, cuerpo := app.post(t, app.client(false), "/contexto/proyecto",
			url.Values{"proyecto_id": {id64(app.fx.projectA)}})
		revisar(t, resp, cuerpo)
	})
}
