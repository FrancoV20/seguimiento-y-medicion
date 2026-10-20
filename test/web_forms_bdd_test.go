package test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-20b, US1: mensajes y formularios (T016).
// Usan solo la API pública del paquete web y el servidor real.

const formTextoExito = "Guardado correctamente"

// formApp es el servidor real con una sección de prueba que guarda un registro.
type formApp struct {
	servidor  *httptest.Server
	guardados atomic.Int32
}

func nuevaFormApp(t *testing.T) *formApp {
	t.Helper()
	app := &formApp{}
	srv := web.NewServer(silencioso())
	pagina := func(titulo string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			srv.Renderer().Page(w, r, "home.html", web.PageData{Title: titulo})
		}
	}
	err := srv.Register(web.Section{Name: "Prueba", Path: "/prueba"}, web.Routes{
		"POST /prueba/guardar": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.PostFormValue("falla") == "1" {
				srv.Renderer().PageStatus(w, r, http.StatusInternalServerError, "home.html", web.PageData{
					Title: "Error al guardar",
					Flash: &web.Flash{Kind: web.FlashError, Text: web.MsgOperationFailed},
				})
				return
			}
			app.guardados.Add(1)
			web.SetFlash(w, r, web.FlashSuccess, formTextoExito)
			web.RedirectSeeOther(w, r, "/prueba/resultado")
		}),
		"GET /prueba/resultado": pagina("Resultado"),
		"GET /prueba/mensaje": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			srv.Renderer().Page(w, r, "home.html", web.PageData{
				Title: "Mensaje",
				Flash: &web.Flash{Kind: web.FlashKind(r.URL.Query().Get("tipo")), Text: "Mensaje de prueba"},
			})
		}),
	})
	if err != nil {
		t.Fatalf("no se pudo registrar la sección de prueba: %v", err)
	}
	app.servidor = httptest.NewServer(srv.Handler())
	t.Cleanup(app.servidor.Close)
	return app
}

// formNavegador simula un navegador con cookies; si seguir es false, no sigue redirecciones.
func formNavegador(seguir bool) *http.Client {
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	if !seguir {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return c
}

func formCuerpo(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return string(b)
}

func (a *formApp) abrir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *formApp) enviar(t *testing.T, c *http.Client, valores url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+"/prueba/guardar", valores)
	if err != nil {
		t.Fatalf("POST /prueba/guardar: %v", err)
	}
	return resp, formCuerpo(t, resp)
}

// US1 - Escenario: Formato de mensajes (FR-001, SC-001).
func TestEscenarioForm_FormatoDeMensajes(t *testing.T) {
	casos := []struct{ tipo, clase, etiqueta string }{
		{"success", "alert alert-success", "Éxito:"},
		{"error", "alert alert-error", "Error:"},
		{"warning", "alert alert-warning", "Advertencia:"},
	}
	for _, c := range casos {
		t.Run(c.tipo, func(t *testing.T) {
			// Dado cualquier pantalla
			app := nuevaFormApp(t)

			// Cuando muestra un mensaje de éxito, error o advertencia
			_, cuerpo := app.abrir(t, formNavegador(true), "/prueba/mensaje?tipo="+c.tipo)

			// Entonces cada tipo tiene su propio formato y una etiqueta de texto además del color
			if !strings.Contains(cuerpo, c.clase) {
				t.Errorf("falta la clase de formato %q", c.clase)
			}
			if !strings.Contains(cuerpo, c.etiqueta) {
				t.Errorf("falta la etiqueta de texto %q", c.etiqueta)
			}
			if !strings.Contains(cuerpo, "Mensaje de prueba") {
				t.Errorf("falta el texto del mensaje")
			}
		})
	}
}

// US1 - Escenario: Error al guardar (FR-003).
func TestEscenarioForm_ErrorAlGuardar(t *testing.T) {
	// Dado un formulario válido
	app := nuevaFormApp(t)

	// Cuando falla el guardado
	resp, cuerpo := app.enviar(t, formNavegador(true), url.Values{"nombre": {"Proyecto X"}, "falla": {"1"}})

	// Entonces veo "La operación no pudo completarse" y no se muestra éxito
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("se esperaba 500, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(cuerpo, "La operación no pudo completarse") {
		t.Errorf("falta el mensaje de fallo")
	}
	if strings.Contains(cuerpo, formTextoExito) || strings.Contains(cuerpo, "alert-success") {
		t.Errorf("no debe mostrarse un mensaje de éxito")
	}
	if app.guardados.Load() != 0 {
		t.Errorf("no debía guardarse nada")
	}
}

// US1 - Escenario: Envío válido sin reenvío (FR-004, SC-003).
func TestEscenarioForm_EnvioValidoSinReenvio(t *testing.T) {
	// Dado que envío un formulario válido
	app := nuevaFormApp(t)
	navegador := formNavegador(false)

	// Cuando el sistema responde
	resp, _ := app.enviar(t, navegador, url.Values{"nombre": {"Proyecto X"}})

	// Entonces me lleva a una página de resultado con el mensaje de éxito
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/prueba/resultado" {
		t.Fatalf("se esperaba 303 a /prueba/resultado, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	_, resultado := app.abrir(t, navegador, "/prueba/resultado")
	if !strings.Contains(resultado, formTextoExito) || !strings.Contains(resultado, "alert-success") {
		t.Errorf("la página de resultado debía mostrar el mensaje de éxito")
	}

	// Y si recargo la página no se reenvía el formulario ni se duplica el registro
	_, recarga := app.abrir(t, navegador, "/prueba/resultado")
	if strings.Contains(recarga, formTextoExito) {
		t.Errorf("al recargar, el mensaje no debía repetirse")
	}
	if app.guardados.Load() != 1 {
		t.Errorf("se esperaba 1 registro, hay %d", app.guardados.Load())
	}
}

// US1 - Escenario: Mensaje de una sola vista (FR-004).
func TestEscenarioForm_MensajeDeUnaSolaVista(t *testing.T) {
	// Dado que se mostró un mensaje de éxito
	app := nuevaFormApp(t)
	navegador := formNavegador(false)
	app.enviar(t, navegador, url.Values{"nombre": {"Proyecto X"}})
	_, primera := app.abrir(t, navegador, "/")
	if !strings.Contains(primera, formTextoExito) {
		t.Fatalf("la primera pantalla debía mostrar el mensaje")
	}

	// Cuando navego a otra pantalla o recargo
	_, segunda := app.abrir(t, navegador, "/prueba/resultado")
	_, tercera := app.abrir(t, navegador, "/")

	// Entonces el mensaje no vuelve a aparecer
	if strings.Contains(segunda, formTextoExito) || strings.Contains(tercera, formTextoExito) {
		t.Errorf("el mensaje no debía volver a aparecer")
	}
}