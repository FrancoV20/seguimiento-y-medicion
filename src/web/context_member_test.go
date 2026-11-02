package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Pruebas de handler de US4 (HU-20b, T040 y T042): identificarse como integrante.
// Reutilizan los auxiliares de context_handlers_test.go (newCtxApp, idCookie, clears...).

func (a *ctxApp) selectMember(t *testing.T, c *http.Client, id, volverA string) (*http.Response, string) {
	t.Helper()
	// HU-27: la identificación pide contraseña.
	if n, err := strconv.ParseInt(id, 10, 64); err == nil && validID(n) {
		setTestPassword(t, a.db, n)
	}
	values := memberLoginValues(id)
	values.Set("volver_a", volverA)
	return a.post(t, c, "/contexto/integrante", values)
}

func datosEsperados(project, member int64) string {
	return "datos:proyecto=" + id64(project) + ";integrante=" + id64(member)
}

// cookieDe devuelve la cookie con ese nombre de una respuesta, o nil.
func cookieDe(resp *http.Response, name string) *http.Cookie {
	for _, ck := range resp.Cookies() {
		if ck.Name == name {
			return ck
		}
	}
	return nil
}

// --- T040: el selector, la identificación y la estructura común ---

func TestIntegrante_SelectorMuestraSoloLosDelProyectoActivo(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	casos := []struct {
		nombre    string
		proyecto  int64
		presentes []string
		ausentes  []string
	}{
		{"proyecto A", fx.projectA, []string{fx.anaName, fx.betoName}, []string{fx.carlaName}},
		{"proyecto B", fx.projectB, []string{fx.carlaName}, []string{fx.anaName, fx.betoName}},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			c := app.client(true)
			app.setCookies(t, c, idCookie(CookieProject, tc.proyecto))

			resp, body := app.get(t, c, "/contexto/integrante")

			if resp.StatusCode != http.StatusOK || !strings.Contains(body, `name="integrante_id"`) {
				t.Fatalf("se esperaba 200 con el selector, se obtuvo %d", resp.StatusCode)
			}
			for _, nombre := range tc.presentes {
				if !strings.Contains(body, nombre) {
					t.Errorf("falta el integrante %q", nombre)
				}
			}
			for _, nombre := range tc.ausentes {
				if strings.Contains(body, nombre) {
					t.Errorf("no debe aparecer el integrante %q de otro proyecto", nombre)
				}
			}
			if !strings.Contains(body, `type="password"`) || !strings.Contains(body, `autocomplete="current-password"`) { // HU-27
				t.Errorf("la identificación no lleva contraseña")
			}
		})
	}
}

func TestIntegrante_ElegirMiNombreLoUsaEnMisAccionesYSeVeEnLaEstructuraComun(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)
	app.selectProject(t, c, id64(fx.projectA), "/libre")

	resp, _ := app.selectMember(t, c, id64(fx.ana), "/libre")

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/libre" {
		t.Fatalf("se esperaba 303 a /libre, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	ck := cookieDe(resp, CookieMember)
	if ck == nil || ck.Value != SignedContextValue(CookieMember, fx.ana) || !ck.HttpOnly || !ck.Expires.IsZero() || ck.MaxAge != 0 {
		t.Errorf("la cookie del integrante debía ser de sesión y HttpOnly: %+v", ck)
	}
	for _, ruta := range []string{"/libre", "/prueba/resultado"} {
		_, body := app.get(t, c, ruta)
		if !strings.Contains(body, fx.anaName) || !strings.Contains(body, "Integrante:") {
			t.Errorf("%s: el integrante debía verse en la estructura común", ruta)
		}
	}
	if _, datos := app.get(t, c, "/datos"); datos != datosEsperados(fx.projectA, fx.ana) {
		t.Errorf("las acciones debían usar a Ana, se obtuvo %q", datos)
	}
}

func TestIntegrante_CambiarDeNombreReemplazaLaIdentificacion(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)
	app.selectProject(t, c, id64(fx.projectA), "/libre")
	app.selectMember(t, c, id64(fx.ana), "/libre")

	app.selectMember(t, c, id64(fx.beto), "/libre")

	if _, datos := app.get(t, c, "/datos"); datos != datosEsperados(fx.projectA, fx.beto) {
		t.Errorf("la identificación debía pasar a Beto, se obtuvo %q", datos)
	}
}

func TestIntegrante_SinProyectoSePideElegirProyectoPrimero(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx

	t.Run("al abrir el selector", func(t *testing.T) {
		resp, _ := app.get(t, app.client(false), "/contexto/integrante")
		if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/proyecto") {
			t.Errorf("se esperaba 303 a la selección de proyecto, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
		}
		_, body := app.get(t, app.client(true), "/contexto/integrante")
		if !strings.Contains(body, MsgChooseProjectFirst) || !strings.Contains(body, `name="proyecto_id"`) {
			t.Errorf("debía verse %q con la lista de proyectos", MsgChooseProjectFirst)
		}
	})
	t.Run("al enviar una identificación", func(t *testing.T) {
		resp, _ := app.selectMember(t, app.client(false), id64(fx.ana), "/libre")
		if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/proyecto") {
			t.Errorf("se esperaba 303 a la selección de proyecto, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
		}
		if cookieDe(resp, CookieMember) != nil {
			t.Errorf("sin proyecto no debe guardarse ningún integrante")
		}
	})
}

// --- T042: integrante ajeno, quitado o manipulado ---

func TestIntegrante_AjenoOInvalidoSeRechazaYLimpiaLaIdentificacion(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	casos := []struct{ nombre, valor string }{
		{"de otro proyecto", id64(fx.carla)},
		{"inexistente", strconv.Itoa(MaxID)},
		{"no numerico", "abc"},
		{"negativo", "-1"},
		{"muy grande", "99999999999"},
		{"vacio", ""},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			c := app.client(false)
			app.selectProject(t, c, id64(fx.projectA), "/libre")
			app.selectMember(t, c, id64(fx.ana), "/libre") // ya estaba identificada

			resp, _ := app.selectMember(t, c, tc.valor, "/libre")

			if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/integrante") {
				t.Fatalf("se esperaba 303 al selector de integrante, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
			}
			if !clears(resp, CookieMember) || app.cookieValue(c, CookieMember) != "" {
				t.Errorf("la identificación debía limpiarse")
			}
			if _, body := app.get(t, c, resp.Header.Get("Location")); !strings.Contains(body, MsgMemberInvalid) {
				t.Errorf("falta el mensaje %q", MsgMemberInvalid)
			}
			if _, datos := app.get(t, c, "/datos"); datos != datosEsperados(fx.projectA, 0) {
				t.Errorf("debía quedar el proyecto sin integrante, se obtuvo %q", datos)
			}
		})
	}
}

func TestIntegrante_CookieDeIntegranteAjenoSeLimpiaConAdvertencia(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)
	app.setCookies(t, c, idCookie(CookieProject, fx.projectA), idCookie(CookieMember, fx.carla)) // Carla es del proyecto B

	resp, primera := app.get(t, c, "/libre")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(primera, MsgMemberGone) || !strings.Contains(primera, "Advertencia:") {
		t.Errorf("debía verse la advertencia %q", MsgMemberGone)
	}
	if !strings.Contains(primera, fx.nameA) || strings.Contains(primera, fx.carlaName) {
		t.Errorf("el proyecto debía seguir activo y sin el integrante ajeno")
	}
	if !clears(resp, CookieMember) || clears(resp, CookieProject) {
		t.Errorf("solo debía borrarse la identificación")
	}
	if _, datos := app.get(t, c, "/datos"); datos != datosEsperados(fx.projectA, 0) {
		t.Errorf("se esperaba el proyecto sin integrante, se obtuvo %q", datos)
	}
	if _, segunda := app.get(t, c, "/libre"); strings.Contains(segunda, MsgMemberGone) {
		t.Errorf("la advertencia no debía repetirse")
	}
}

func TestIntegrante_QuitadoDelProyectoSeLimpiaConAdvertencia(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	nombre := fmt.Sprintf("HU20b Quitado %d", time.Now().UnixNano())
	var id int64
	if err := app.db.QueryRow(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, nombre).Scan(&id); err != nil {
		t.Fatalf("no se pudo crear el integrante: %v", err)
	}
	t.Cleanup(func() { _, _ = app.db.Exec(`DELETE FROM members WHERE id = $1`, id) })
	if _, err := app.db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, fx.projectA, id); err != nil {
		t.Fatalf("no se pudo vincular el integrante: %v", err)
	}
	c := app.client(false)
	app.setCookies(t, c, idCookie(CookieProject, fx.projectA), idCookie(CookieMember, id))
	if _, body := app.get(t, c, "/libre"); !strings.Contains(body, nombre) {
		t.Fatalf("precondición: debía verse al integrante identificado")
	}
	if _, err := app.db.Exec(`DELETE FROM project_members WHERE project_id = $1 AND member_id = $2`, fx.projectA, id); err != nil {
		t.Fatalf("no se pudo quitar al integrante del proyecto: %v", err)
	}

	resp, body := app.get(t, c, "/libre")

	if !strings.Contains(body, MsgMemberGone) || !strings.Contains(body, "Advertencia:") {
		t.Errorf("debía verse la advertencia %q", MsgMemberGone)
	}
	if strings.Contains(body, nombre) || !clears(resp, CookieMember) || app.cookieValue(c, CookieMember) != "" {
		t.Errorf("la identificación debía limpiarse")
	}
}

func TestIntegrante_CookieManipuladaOSinProyectoSeDescartaEnSilencio(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	manipulada := func(valor string) *http.Cookie { return &http.Cookie{Name: CookieMember, Value: valor, Path: "/"} }
	casos := []struct {
		nombre  string
		cookies []*http.Cookie
	}{
		{"valor no numerico", []*http.Cookie{idCookie(CookieProject, fx.projectA), manipulada("abc")}},
		{"negativo", []*http.Cookie{idCookie(CookieProject, fx.projectA), manipulada("-1")}},
		{"muy grande", []*http.Cookie{idCookie(CookieProject, fx.projectA), manipulada("99999999999")}},
		{"integrante sin proyecto", []*http.Cookie{idCookie(CookieMember, fx.ana)}},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			c := app.client(false)
			app.setCookies(t, c, tc.cookies...)

			resp, body := app.get(t, c, "/libre")

			if resp.StatusCode != http.StatusOK {
				t.Errorf("se esperaba 200, se obtuvo %d", resp.StatusCode)
			}
			if !clears(resp, CookieMember) {
				t.Errorf("la cookie debía borrarse")
			}
			if strings.Contains(body, MsgMemberGone) {
				t.Errorf("un valor manipulado se descarta sin advertencia")
			}
		})
	}
}

func TestIntegrante_ProyectoInexistenteLimpiaTambienLaIdentificacion(t *testing.T) {
	app := newCtxApp(t, nil)
	c := app.client(false)
	app.setCookies(t, c, idCookie(CookieProject, MaxID), idCookie(CookieMember, app.fx.ana))

	resp, body := app.get(t, c, "/libre")

	if !clears(resp, CookieProject) || !clears(resp, CookieMember) {
		t.Errorf("debían borrarse el proyecto y la identificación")
	}
	if !strings.Contains(body, MsgProjectGone) {
		t.Errorf("debía verse la advertencia %q", MsgProjectGone)
	}
}
