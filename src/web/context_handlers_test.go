package web

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Pruebas de handler de US3 (HU-20b, T030-T034): servidor real, PostgreSQLContextStore real
// y un navegador simulado con cookies.

type ctxApp struct {
	*formsApp
	db *sql.DB
	fx *contextFixture
}

// newCtxApp arma el servidor con el contexto de trabajo registrado. wrap permite reemplazar
// el store (por ejemplo, para simular una lista vacía o una base caída).
func newCtxApp(t *testing.T, wrap func(ContextStore) ContextStore, setup ...func(*Server)) *ctxApp {
	t.Helper()
	db := openContextTestDB(t)
	fx := newContextFixture(t, db)
	var store ContextStore = NewPostgreSQLContextStore(db)
	if wrap != nil {
		store = wrap(store)
	}
	all := []func(*Server){func(s *Server) { RegisterContext(s, store) }, registerDatos}
	all = append(all, setup...)
	return &ctxApp{formsApp: newFormsApp(t, all...), db: db, fx: fx}
}

// registerDatos agrega una sección que muestra el contexto activo que recibe el handler.
func registerDatos(s *Server) {
	_ = s.Register(Section{Name: "Datos", Path: "/datos"}, Routes{
		"GET /datos": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var project, member int64
			if ac := ActiveContextFrom(r.Context()); ac != nil {
				project, member = ac.ProjectID, ac.MemberID
			}
			fmt.Fprintf(w, "datos:proyecto=%d;integrante=%d", project, member)
		}),
	})
}

func idCookie(name string, id int64) *http.Cookie {
	return &http.Cookie{Name: name, Value: SignedContextValue(name, id), Path: "/"} // firmada (HU-26)
}

func (a *ctxApp) setCookies(t *testing.T, c *http.Client, cookies ...*http.Cookie) {
	t.Helper()
	u, err := url.Parse(a.ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.Jar.SetCookies(u, cookies)
}

func (a *ctxApp) cookieValue(c *http.Client, name string) string {
	u, _ := url.Parse(a.ts.URL)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func (a *ctxApp) selectProject(t *testing.T, c *http.Client, id, volverA string) (*http.Response, string) {
	t.Helper()
	return a.post(t, c, "/contexto/proyecto", url.Values{"proyecto_id": {id}, "volver_a": {volverA}})
}

func id64(id int64) string { return strconv.FormatInt(id, 10) }

// clears indica si la respuesta pide borrar la cookie.
func clears(resp *http.Response, name string) bool {
	for _, ck := range resp.Cookies() {
		if ck.Name == name && ck.MaxAge < 0 {
			return true
		}
	}
	return false
}

// touches indica si la respuesta escribe o borra la cookie.
func touches(resp *http.Response, name string) bool {
	for _, ck := range resp.Cookies() {
		if ck.Name == name {
			return true
		}
	}
	return false
}

// --- T030: selección y permanencia ---

func TestContexto_SinSeleccionNoSeAutoselecciona(t *testing.T) {
	// Con proyectos existentes (el fixture) y sin elegir, nunca se selecciona uno solo.
	app := newCtxApp(t, nil)
	c := app.client(true)

	resp, body := app.get(t, c, "/libre")

	if resp.StatusCode != http.StatusOK || !strings.Contains(body, MsgNoProjectSelected) {
		t.Errorf("se esperaba 200 con %q, se obtuvo %d", MsgNoProjectSelected, resp.StatusCode)
	}
	if touches(resp, CookieProject) {
		t.Errorf("no debe escribirse la cookie del proyecto sin que la persona elija")
	}
	if _, datos := app.get(t, c, "/datos"); datos != "datos:proyecto=0;integrante=0" {
		t.Errorf("sin selección se esperaba contexto vacío, se obtuvo %q", datos)
	}
}

func TestContexto_ListaDeProyectosPermiteElegir(t *testing.T) {
	app := newCtxApp(t, nil)

	resp, body := app.get(t, app.client(true), "/contexto/proyecto")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	for _, want := range []string{app.fx.nameA, app.fx.nameB, `name="proyecto_id"`,
		`value="` + id64(app.fx.projectA) + `"`, `value="` + id64(app.fx.projectB) + `"`} {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q en la lista de selección", want)
		}
	}
}

func TestContexto_ElegirProyectoMuestraSuNombreYSusDatos(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)

	resp, _ := app.selectProject(t, c, id64(fx.projectA), "/libre")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/libre" {
		t.Fatalf("se esperaba 303 a /libre, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	// El nombre queda visible en la estructura común y se mantiene al navegar.
	for _, ruta := range []string{"/libre", "/prueba/resultado"} {
		if _, body := app.get(t, c, ruta); !strings.Contains(body, fx.nameA) {
			t.Errorf("%s: falta el nombre del proyecto activo", ruta)
		}
	}
	// Las secciones reciben únicamente el proyecto elegido.
	if _, datos := app.get(t, c, "/datos"); datos != "datos:proyecto="+id64(fx.projectA)+";integrante=0" {
		t.Errorf("se esperaba el proyecto A, se obtuvo %q", datos)
	}

	// Cambiar de proyecto cambia los datos que ven las secciones.
	app.selectProject(t, c, id64(fx.projectB), "/libre")
	if _, datos := app.get(t, c, "/datos"); datos != "datos:proyecto="+id64(fx.projectB)+";integrante=0" {
		t.Errorf("se esperaba el proyecto B, se obtuvo %q", datos)
	}
	if _, body := app.get(t, c, "/libre"); !strings.Contains(body, fx.nameB) || strings.Contains(body, fx.nameA) {
		t.Errorf("la estructura común debía mostrar solo el proyecto B")
	}
}

func TestContexto_ElegirSinDestinoVuelveAlInicio(t *testing.T) {
	app := newCtxApp(t, nil)

	resp, _ := app.selectProject(t, app.client(false), id64(app.fx.projectA), "")

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Errorf("se esperaba 303 a /, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// --- T031: cambio de proyecto, mismo proyecto y sesión nueva ---

func TestContexto_CambiarDeProyectoLimpiaElIntegrante(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)
	app.setCookies(t, c, idCookie(CookieProject, fx.projectA), idCookie(CookieMember, fx.ana))
	if _, body := app.get(t, c, "/libre"); !strings.Contains(body, fx.anaName) {
		t.Fatalf("precondición: debía verse al integrante identificado")
	}

	resp, _ := app.selectProject(t, c, id64(fx.projectB), "/libre")

	if !clears(resp, CookieMember) {
		t.Errorf("al cambiar de proyecto debía borrarse la cookie del integrante")
	}
	if v := app.cookieValue(c, CookieMember); v != "" {
		t.Errorf("la identificación debía limpiarse, quedó %q", v)
	}
	if _, datos := app.get(t, c, "/datos"); datos != "datos:proyecto="+id64(fx.projectB)+";integrante=0" {
		t.Errorf("se esperaba el proyecto B sin integrante, se obtuvo %q", datos)
	}
}

func TestContexto_VolverAElegirElMismoProyectoConservaElIntegrante(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)
	app.setCookies(t, c, idCookie(CookieProject, fx.projectA), idCookie(CookieMember, fx.ana))

	resp, _ := app.selectProject(t, c, id64(fx.projectA), "/libre")

	if clears(resp, CookieMember) {
		t.Errorf("al elegir el mismo proyecto no debe borrarse la identificación")
	}
	want := "datos:proyecto=" + id64(fx.projectA) + ";integrante=" + id64(fx.ana)
	if _, datos := app.get(t, c, "/datos"); datos != want {
		t.Errorf("se esperaba %q, se obtuvo %q", want, datos)
	}
}

func TestContexto_SesionNuevaPideElegirDeNuevo(t *testing.T) {
	app := newCtxApp(t, nil)
	primero := app.client(false)
	app.selectProject(t, primero, id64(app.fx.projectA), "/libre")
	if _, body := app.get(t, primero, "/libre"); !strings.Contains(body, app.fx.nameA) {
		t.Fatalf("precondición: el primer navegador debía tener el proyecto")
	}

	// Cerrar y volver a abrir el navegador equivale a un navegador sin cookies.
	_, body := app.get(t, app.client(false), "/libre")

	if !strings.Contains(body, MsgNoProjectSelected) || strings.Contains(body, app.fx.nameA) {
		t.Errorf("en una sesión nueva debía pedirse elegir el proyecto")
	}
}

// --- T032: identificadores inválidos, cookies manipuladas y proyecto eliminado ---

func TestContexto_IdentificadorInvalidoConservaLaSeleccionAnterior(t *testing.T) {
	app := newCtxApp(t, nil)
	casos := []struct{ nombre, valor string }{
		{"no numerico", "abc"},
		{"vacio", ""},
		{"negativo", "-1"},
		{"cero", "0"},
		{"muy grande", "99999999999"},
		{"inexistente", strconv.Itoa(MaxID)},
		{"html", "<script>"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			c := app.client(true)
			app.selectProject(t, c, id64(app.fx.projectA), "/libre")

			resp, body := app.selectProject(t, c, tc.valor, "/libre")

			if resp.StatusCode != http.StatusOK {
				t.Errorf("la aplicación no debe fallar: se obtuvo %d", resp.StatusCode)
			}
			if !strings.Contains(body, MsgProjectInvalid) {
				t.Errorf("falta el mensaje claro %q", MsgProjectInvalid)
			}
			want := "datos:proyecto=" + id64(app.fx.projectA) + ";integrante=0"
			if _, datos := app.get(t, c, "/datos"); datos != want {
				t.Errorf("la selección anterior debía conservarse: se obtuvo %q", datos)
			}
		})
	}
}

func TestContexto_CookieManipuladaSeTrataComoAusencia(t *testing.T) {
	app := newCtxApp(t, nil)
	for _, valor := range []string{"abc", "-1", "0", "99999999999", "<script>"} {
		t.Run(valor, func(t *testing.T) {
			c := app.client(false)
			app.setCookies(t, c, &http.Cookie{Name: CookieProject, Value: valor, Path: "/"})

			resp, body := app.get(t, c, "/libre")

			if resp.StatusCode != http.StatusOK || !strings.Contains(body, MsgNoProjectSelected) {
				t.Errorf("se esperaba 200 sin selección, se obtuvo %d", resp.StatusCode)
			}
			if !clears(resp, CookieProject) {
				t.Errorf("la cookie manipulada debía borrarse")
			}
		})
	}
}

func TestContexto_ProyectoEliminadoLimpiaLaSeleccionConAdvertencia(t *testing.T) {
	app := newCtxApp(t, nil)
	c := app.client(false)
	var id int64
	err := app.db.QueryRow(`INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`,
		fmt.Sprintf("HU20b Efimero %d", time.Now().UnixNano())).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba: %v", err)
	}
	t.Cleanup(func() { _, _ = app.db.Exec(`DELETE FROM projects WHERE id = $1`, id) })
	app.setCookies(t, c, idCookie(CookieProject, id))
	if _, body := app.get(t, c, "/libre"); strings.Contains(body, MsgProjectGone) {
		t.Fatalf("precondición: con el proyecto existente no debe haber advertencia")
	}
	if _, err := app.db.Exec(`DELETE FROM projects WHERE id = $1`, id); err != nil {
		t.Fatalf("no se pudo eliminar el proyecto: %v", err)
	}

	resp, primera := app.get(t, c, "/libre")

	if resp.StatusCode != http.StatusOK {
		t.Errorf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(primera, MsgProjectGone) || !strings.Contains(primera, "Advertencia:") {
		t.Errorf("debía mostrarse la advertencia %q", MsgProjectGone)
	}
	if !strings.Contains(primera, MsgNoProjectSelected) {
		t.Errorf("la estructura común debía mostrar que no hay proyecto")
	}
	if v := app.cookieValue(c, CookieProject); v != "" {
		t.Errorf("la selección debía limpiarse, quedó %q", v)
	}
	if _, segunda := app.get(t, c, "/libre"); strings.Contains(segunda, MsgProjectGone) {
		t.Errorf("la advertencia no debía repetirse")
	}
}

// --- T033: sin proyectos ---

// sinProyectos delega todo en el store real salvo la lista de proyectos, que devuelve vacía.
// Se usa para no vaciar la base compartida del equipo.
type sinProyectos struct{ ContextStore }

func (sinProyectos) ListProjects(context.Context) ([]ProjectSummary, error) { return nil, nil }

func TestContexto_SinProyectosMuestraMensajeYAccesoSoloSiExisteLaSeccion(t *testing.T) {
	vacio := func(real ContextStore) ContextStore { return sinProyectos{ContextStore: real} }

	t.Run("sin la sección Proyectos", func(t *testing.T) {
		app := newCtxApp(t, vacio)

		_, body := app.get(t, app.client(true), "/contexto/proyecto")

		if !strings.Contains(body, MsgNoProjects) {
			t.Errorf("falta el mensaje %q", MsgNoProjects)
		}
		if strings.Contains(body, LinkCreateFirstProject) {
			t.Errorf("no debe ofrecerse crear proyectos si la sección no existe")
		}
		if strings.Contains(body, `name="proyecto_id"`) {
			t.Errorf("no debe mostrarse un selector vacío")
		}
	})
	t.Run("con la sección Proyectos", func(t *testing.T) {
		app := newCtxApp(t, vacio, func(s *Server) {
			_ = s.Register(Section{Name: "Proyectos", Path: "/proyectos"}, Routes{
				"GET /proyectos": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") }),
			})
		})

		_, body := app.get(t, app.client(true), "/contexto/proyecto")

		if !strings.Contains(body, MsgNoProjects) || !strings.Contains(body, LinkCreateFirstProject) {
			t.Errorf("debía verse %q con el acceso %q", MsgNoProjects, LinkCreateFirstProject)
		}
	})
}

// --- T034: base de datos caída ---

func TestContexto_BaseCaidaNoLimpiaLaSeleccionYResponde503(t *testing.T) {
	caida := func(ContextStore) ContextStore {
		db := openContextTestDB(t)
		_ = db.Close() // toda consulta falla como "no disponible"
		return NewPostgreSQLContextStore(db)
	}
	app := newCtxApp(t, caida)
	c := app.client(false)
	app.setCookies(t, c, idCookie(CookieProject, app.fx.projectA))

	resp, body := app.get(t, c, "/libre")

	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(body, MsgUnavailable) {
		t.Errorf("se esperaba 503 con %q, se obtuvo %d", MsgUnavailable, resp.StatusCode)
	}
	if touches(resp, CookieProject) {
		t.Errorf("una caída transitoria no debe borrar la selección")
	}
	if v := app.cookieValue(c, CookieProject); v != SignedContextValue(CookieProject, app.fx.projectA) {
		t.Errorf("la selección debía conservarse, quedó %q", v)
	}
	// Sin cookies de contexto no hace falta la base: la página se muestra normalmente.
	if r, _ := app.get(t, app.client(false), "/libre"); r.StatusCode != http.StatusOK {
		t.Errorf("sin selección la página no depende de la base: se obtuvo %d", r.StatusCode)
	}
}
