package web

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	xssScript        = `<script>alert(1)</script>`
	xssScriptEscaped = `&lt;script&gt;alert(1)&lt;/script&gt;`
	xssAttr          = `"><img src=x onerror=alert(1)>`
)

func TestEscape_ValorDelFormularioSeMuestraComoTexto(t *testing.T) {
	app := newFormsApp(t)
	client := app.client(true)

	_, body := app.post(t, client, "/prueba/form", url.Values{"nombre": {""}, "detalle": {xssScript}})
	if strings.Contains(body, xssScript) {
		t.Errorf("el script se escribió sin escapar en la página")
	}
	if !strings.Contains(body, xssScriptEscaped) {
		t.Errorf("el script debía verse como texto escapado")
	}

	_, body = app.post(t, client, "/prueba/form", url.Values{"nombre": {""}, "detalle": {xssAttr}})
	if strings.Contains(body, "<img src=x") {
		t.Errorf("una comilla en un valor no debe permitir cerrar el atributo e inyectar una etiqueta")
	}
}

func TestEscape_MensajeSeMuestraComoTexto(t *testing.T) {
	app := newFormsApp(t)

	_, body := app.get(t, app.client(true), "/prueba/alerta?tipo=warning&texto="+url.QueryEscape(xssScript))

	if strings.Contains(body, xssScript) || !strings.Contains(body, xssScriptEscaped) {
		t.Errorf("el texto del mensaje debía mostrarse escapado")
	}
}

func TestEscape_CookieFlashManipuladaSeMuestraComoTexto(t *testing.T) {
	app := newFormsApp(t)
	req, err := http.NewRequest(http.MethodGet, app.ts.URL+"/libre", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: CookieFlash, Value: url.Values{"k": {"error"}, "t": {xssScript}}.Encode()})

	resp, err := app.client(true).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := formsReadBody(t, resp)

	if strings.Contains(body, xssScript) || !strings.Contains(body, xssScriptEscaped) {
		t.Errorf("un flash forjado por el usuario debía mostrarse escapado")
	}
}

func TestEscape_ProyectoEIntegranteActivosSeMuestranComoTexto(t *testing.T) {
	app := newFormsApp(t, func(s *Server) {
		s.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ac := &ActiveContext{ProjectID: 1, ProjectName: xssScript, MemberID: 2, MemberName: xssScript}
				next.ServeHTTP(w, r.WithContext(WithActiveContext(r.Context(), ac)))
			})
		})
	})

	_, body := app.get(t, app.client(true), "/libre")

	if strings.Contains(body, xssScript) || !strings.Contains(body, xssScriptEscaped) {
		t.Errorf("el nombre del proyecto y del integrante debían mostrarse escapados en la estructura común")
	}
}

// T026: ningún código de la aplicación debe marcar datos como HTML, JS, URL o CSS "seguro".
func TestEscape_NingunCodigoMarcaDatosComoSeguros(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{"template.HTML(", "template.JS(", "template.URL(", "template.CSS(", "template.HTMLAttr("}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range banned {
			if strings.Contains(string(src), b) {
				t.Errorf("%s usa %s: los datos de usuarios no deben marcarse como seguros", f, b)
			}
		}
	}
}