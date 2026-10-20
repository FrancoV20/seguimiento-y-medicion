package web
package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func newGuardedApp(t *testing.T) *formsApp {
	t.Helper()
	return newFormsApp(t, func(s *Server) { s.Use(CrossOriginGuard(s.Renderer())) })
}

func (a *formsApp) doWithHeaders(t *testing.T, method, path string, v url.Values, headers map[string]string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, a.ts.URL+path, strings.NewReader(v.Encode()))
	if err != nil {
		t.Fatalf("no se pudo armar la solicitud: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, val := range headers {
		req.Header.Set(k, val)
	}
	resp, err := a.client(false).Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp, formsReadBody(t, resp)
}

func TestSeguridad_EnvioDeOtroSitioSeRechazaCon403(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"Sec-Fetch-Site cross-site", map[string]string{"Sec-Fetch-Site": "cross-site"}},
		{"Origin de otro sitio", map[string]string{"Origin": "https://sitio-malicioso.example"}},
		{"ambas cabeceras", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://sitio-malicioso.example"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newGuardedApp(t)

			resp, body := app.doWithHeaders(t, http.MethodPost, "/prueba/form", url.Values{"nombre": {"Proyecto X"}}, tc.headers)

			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("se esperaba 403, se obtuvo %d", resp.StatusCode)
			}
			if !strings.Contains(body, MsgForbidden) {
				t.Errorf("falta el mensaje claro %q", MsgForbidden)
			}
			if !strings.Contains(body, `aria-label="Menú principal"`) {
				t.Errorf("la página 403 debe usar la estructura común")
			}
			if app.saved.Load() != 0 {
				t.Errorf("no debía ejecutarse ninguna acción")
			}
		})
	}
}

func TestSeguridad_EnvioDeLaPropiaAplicacionSeProcesa(t *testing.T) {
	cases := []struct {
		name    string
		headers func(a *formsApp) map[string]string
	}{
		{"sin cabeceras (curl, pruebas)", func(*formsApp) map[string]string { return nil }},
		{"Sec-Fetch-Site same-origin", func(*formsApp) map[string]string { return map[string]string{"Sec-Fetch-Site": "same-origin"} }},
		{"Sec-Fetch-Site none", func(*formsApp) map[string]string { return map[string]string{"Sec-Fetch-Site": "none"} }},
		{"Origin de la aplicación", func(a *formsApp) map[string]string { return map[string]string{"Origin": a.ts.URL} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newGuardedApp(t)

			resp, _ := app.doWithHeaders(t, http.MethodPost, "/prueba/form", url.Values{"nombre": {"Proyecto X"}}, tc.headers(app))

			if resp.StatusCode != http.StatusSeeOther {
				t.Errorf("se esperaba 303, se obtuvo %d", resp.StatusCode)
			}
			if app.saved.Load() != 1 {
				t.Errorf("se esperaba 1 registro, hay %d", app.saved.Load())
			}
		})
	}
}

func TestSeguridad_ConsultasDeOtroSitioNoSeBloquean(t *testing.T) {
	app := newGuardedApp(t)
	req, err := http.NewRequest(http.MethodGet, app.ts.URL+"/libre", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	resp, err := app.client(false).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = formsReadBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("un GET (método seguro) debía pasar, se obtuvo %d", resp.StatusCode)
	}
}