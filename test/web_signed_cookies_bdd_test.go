package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-26: cookies de contexto firmadas (T012-T017).
// Usan la API pública de web, el servidor real y PostgreSQL real (misma preparación que
// test/context_bdd_test.go).

type firmaApp struct {
	*seleccionFixture
	servidor    *httptest.Server
	integranteB int64 // segundo integrante del proyecto A
}

// nuevaFirmaApp arma el servidor con el contexto de HU-20b, una sección que requiere proyecto y
// muestra el contexto recibido, y otra que requiere integrante.
func nuevaFirmaApp(t *testing.T) *firmaApp {
	t.Helper()
	f := nuevaSeleccionFixture(t)
	srv := web.NewServer(silencioso())
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(f.db))
	mostrar := func(w http.ResponseWriter, r *http.Request) {
		var p, m int64
		if ac := web.ActiveContextFrom(r.Context()); ac != nil {
			p, m = ac.ProjectID, ac.MemberID
		}
		fmt.Fprintf(w, "contexto:proyecto=%d;integrante=%d", p, m)
	}
	for _, s := range []web.Section{
		{Name: "Datos HU26", Path: "/datos-hu26", RequiresProject: true},
		{Name: "Acción HU26", Path: "/accion-hu26", RequiresProject: true, RequiresMember: true},
	} {
		if err := srv.Register(s, web.Routes{"GET " + s.Path: http.HandlerFunc(mostrar)}); err != nil {
			t.Fatalf("no se pudo registrar %q: %v", s.Name, err)
		}
	}
	app := &firmaApp{seleccionFixture: f, servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	app.integranteB = f.insertar(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, f.nombreIntegranteA+" B")
	t.Cleanup(func() { _, _ = f.db.Exec(`DELETE FROM members WHERE id = $1`, app.integranteB) })
	if _, err := f.db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, f.proyectoA, app.integranteB); err != nil {
		t.Fatalf("no se pudo vincular el segundo integrante: %v", err)
	}
	return app
}

// cambiarNumero reemplaza el número del principio del valor y conserva el resto (la firma, si
// la hay). Sin firma queda un número válido: así la prueba falla si se aceptan cookies sin firmar.
func cambiarNumero(valor string, nuevo int64) string {
	i := 0
	for i < len(valor) && valor[i] >= '0' && valor[i] <= '9' {
		i++
	}
	return idTexto(nuevo) + valor[i:]
}

// identificado elige el proyecto A y se identifica como su integrante por las pantallas de HU-20b.
func (a *firmaApp) identificado(t *testing.T) *http.Client {
	t.Helper()
	c := formNavegador(false)
	a.enviar(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {idTexto(a.proyectoA)}})
	a.enviar(t, c, "/contexto/integrante", identificacionDePrueba(t, a.db, a.integranteA))
	return c
}

func (a *firmaApp) enviar(t *testing.T, c *http.Client, ruta string, v url.Values) *http.Response {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+ruta, v)
	if err != nil {
		t.Fatalf("POST %s: %v", ruta, err)
	}
	formCuerpo(t, resp)
	return resp
}

func (a *firmaApp) pedir(t *testing.T, c *http.Client, ruta string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *firmaApp) cookie(c *http.Client, nombre string) string {
	u, _ := url.Parse(a.servidor.URL)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == nombre {
			return ck.Value
		}
	}
	return ""
}

func (a *firmaApp) poner(t *testing.T, c *http.Client, nombre, valor string) {
	t.Helper()
	u, _ := url.Parse(a.servidor.URL)
	c.Jar.SetCookies(u, []*http.Cookie{{Name: nombre, Value: valor, Path: "/"}})
}

// borra indica si la respuesta pide borrar la cookie.
func borra(resp *http.Response, nombre string) bool {
	for _, ck := range resp.Cookies() {
		if ck.Name == nombre && ck.MaxAge < 0 {
			return true
		}
	}
	return false
}

func (a *firmaApp) contexto(p, m int64) string {
	return fmt.Sprintf("contexto:proyecto=%d;integrante=%d", p, m)
}

// Escenario 1: un contexto válido se conserva.
func TestHU26_ContextoValidoSeConserva(t *testing.T) {
	// Dado que elegí un proyecto y me identifiqué como integrante
	app := nuevaFirmaApp(t)
	c := app.identificado(t)
	if v := app.cookie(c, web.CookieProject); !strings.HasPrefix(v, idTexto(app.proyectoA)+".") {
		t.Fatalf("la cookie de proyecto debía viajar firmada, se obtuvo %q", v)
	}

	// Cuando navego a otra sección
	resp, cuerpo := app.pedir(t, c, "/accion-hu26")

	// Entonces el sistema reconoce mi proyecto y mi integrante sin pedirme nada
	if resp.StatusCode != http.StatusOK || cuerpo != app.contexto(app.proyectoA, app.integranteA) {
		t.Errorf("se esperaba el contexto completo, se obtuvo %d %q", resp.StatusCode, cuerpo)
	}
}

// Escenario 2: cookie de proyecto modificada.
func TestHU26_CookieProyectoModificada(t *testing.T) {
	// Dado que elegí un proyecto y me identifiqué como integrante
	app := nuevaFirmaApp(t)
	c := app.identificado(t)

	// Cuando el valor de la cookie del proyecto se modifica manualmente
	app.poner(t, c, web.CookieProject, cambiarNumero(app.cookie(c, web.CookieProject), app.proyectoB))
	resp, _ := app.pedir(t, c, "/datos-hu26")

	// Entonces no la acepta y la descarta, descarta también mi identificación y me pide elegir un proyecto
	if !borra(resp, web.CookieProject) || !borra(resp, web.CookieMember) {
		t.Error("se esperaba borrar las cookies de proyecto e integrante")
	}
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/proyecto") {
		t.Errorf("se esperaba ir a elegir proyecto, se obtuvo %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// Escenario 3: cookie de integrante modificada.
func TestHU26_CookieIntegranteModificada(t *testing.T) {
	// Dado que elegí un proyecto y me identifiqué como integrante
	app := nuevaFirmaApp(t)
	c := app.identificado(t)

	// Cuando el valor de la cookie del integrante se modifica manualmente
	app.poner(t, c, web.CookieMember, cambiarNumero(app.cookie(c, web.CookieMember), app.integranteB))
	resp, cuerpo := app.pedir(t, c, "/datos-hu26")

	// Entonces no la acepta y la descarta, y conserva mi proyecto
	if !borra(resp, web.CookieMember) || borra(resp, web.CookieProject) {
		t.Error("se esperaba borrar solo la cookie del integrante")
	}
	if cuerpo != app.contexto(app.proyectoA, 0) {
		t.Errorf("se esperaba conservar el proyecto sin integrante, se obtuvo %q", cuerpo)
	}

	// Y me pide identificarme cuando entro a una acción que lo requiere
	resp, _ = app.pedir(t, c, "/accion-hu26")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/integrante") {
		t.Errorf("se esperaba ir a identificarse, se obtuvo %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// Escenario 4: valor firmado copiado a otra cookie.
func TestHU26_ValorCopiadoEntreCookies(t *testing.T) {
	// Dado que existe una cookie de proyecto válida
	app := nuevaFirmaApp(t)
	c := app.identificado(t)

	// Cuando su valor firmado se copia en la cookie del integrante
	app.poner(t, c, web.CookieMember, app.cookie(c, web.CookieProject))
	resp, cuerpo := app.pedir(t, c, "/datos-hu26")

	// Entonces el sistema no acepta esa cookie del integrante
	if !borra(resp, web.CookieMember) || cuerpo != app.contexto(app.proyectoA, 0) {
		t.Errorf("la cookie copiada no debía aceptarse: %q", cuerpo)
	}
}

// Escenario 5: cookies sin firma de una versión anterior.
func TestHU26_CookiesSinFirmaSeDescartan(t *testing.T) {
	// Dado un navegador con cookies de proyecto e integrante sin firma
	app := nuevaFirmaApp(t)
	c := formNavegador(false)
	app.poner(t, c, web.CookieProject, idTexto(app.proyectoA))
	app.poner(t, c, web.CookieMember, idTexto(app.integranteA))

	// Cuando entro a una sección que requiere proyecto
	resp, cuerpo := app.pedir(t, c, "/datos-hu26")

	// Entonces las descarta y me pide elegir el proyecto sin mostrar ningún error técnico
	if !borra(resp, web.CookieProject) || !borra(resp, web.CookieMember) {
		t.Error("se esperaba borrar las cookies sin firma")
	}
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/proyecto") {
		t.Errorf("se esperaba ir a elegir proyecto, se obtuvo %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp, cuerpo = app.pedir(t, c, resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusOK || strings.Contains(cuerpo, web.TitleInternal) {
		t.Errorf("la pantalla de elegir proyecto no debía mostrar errores: %d", resp.StatusCode)
	}
}
