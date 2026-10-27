package test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"sync/atomic"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-20b, US3: selección del proyecto activo (T035).
// Usan la API pública de web, el servidor real y PostgreSQL real.

type seleccionFixture struct {
	db                *sql.DB
	proyectoA         int64
	proyectoB         int64
	nombreA           string
	nombreB           string
	integranteA       int64
	nombreIntegranteA string
}

func nuevaSeleccionFixture(t *testing.T) *seleccionFixture {
	t.Helper()
	dsn := os.Getenv(web.EnvDatabaseURL)
	if dsn == "" {
		t.Skip("DATABASE_URL no definida: se omite el escenario con PostgreSQL")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sello := time.Now().UnixNano()
	f := &seleccionFixture{
		db:                db,
		nombreA:           fmt.Sprintf("BDD Proyecto A %d", sello),
		nombreB:           fmt.Sprintf("BDD Proyecto B %d", sello),
		nombreIntegranteA: fmt.Sprintf("BDD Integrante %d", sello),
	}
	// Se registra antes de insertar: limpia aunque la preparación falle a mitad.
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1 OR id = $2`, f.proyectoA, f.proyectoB)
		_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, f.integranteA)
	})
	f.proyectoA = f.insertar(t, `INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, f.nombreA)
	f.proyectoB = f.insertar(t, `INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, f.nombreB)
	f.integranteA = f.insertar(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, f.nombreIntegranteA)
	if _, err := db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, f.proyectoA, f.integranteA); err != nil {
		t.Fatalf("no se pudo vincular el integrante: %v", err)
	}
	return f
}

func (f *seleccionFixture) insertar(t *testing.T, consulta string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := f.db.QueryRow(consulta, args...).Scan(&id); err != nil {
		t.Fatalf("no se pudo insertar el dato de prueba: %v", err)
	}
	return id
}

// crearEfimero crea un proyecto que la prueba puede eliminar después.
func (f *seleccionFixture) crearEfimero(t *testing.T) int64 {
	t.Helper()
	id := f.insertar(t, `INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`,
		fmt.Sprintf("BDD Efimero %d", time.Now().UnixNano()))
	t.Cleanup(func() { _, _ = f.db.Exec(`DELETE FROM projects WHERE id = $1`, id) })
	return id
}

// listaFija delega todo en el store real salvo la lista de proyectos.
type listaFija struct {
	web.ContextStore
	proyectos []web.ProjectSummary
}

func (l listaFija) ListProjects(context.Context) ([]web.ProjectSummary, error) {
	return l.proyectos, nil
}

type seleccionApp struct {
	*seleccionFixture
	servidor *httptest.Server
}

// nuevaSeleccionApp arma el servidor real con el contexto de trabajo. envolver permite
// cambiar la lista de proyectos; conProyectos registra la sección Proyectos.
func nuevaSeleccionApp(t *testing.T, envolver func(web.ContextStore, *seleccionFixture) web.ContextStore, conProyectos bool) *seleccionApp {
	t.Helper()
	f := nuevaSeleccionFixture(t)
	var store web.ContextStore = web.NewPostgreSQLContextStore(f.db)
	if envolver != nil {
		store = envolver(store, f)
	}
	srv := web.NewServer(silencioso())
	web.RegisterContext(srv, store)
	registrar := func(s web.Section, ruta string, h http.HandlerFunc) {
		t.Helper()
		if err := srv.Register(s, web.Routes{"GET " + ruta: h}); err != nil {
			t.Fatalf("no se pudo registrar %q: %v", s.Name, err)
		}
	}
	if conProyectos {
		registrar(web.Section{Name: "Proyectos", Path: "/proyectos"}, "/proyectos", func(w http.ResponseWriter, r *http.Request) {
			srv.Renderer().Page(w, r, "home.html", web.PageData{Title: "Proyectos"})
		})
	}
	// Muestra el proyecto activo que reciben las secciones.
	registrar(web.Section{Name: "Datos", Path: "/datos"}, "/datos", func(w http.ResponseWriter, r *http.Request) {
		var id int64
		if ac := web.ActiveContextFrom(r.Context()); ac != nil {
			id = ac.ProjectID
		}
		fmt.Fprintf(w, "datos:proyecto=%d", id)
	})
	app := &seleccionApp{seleccionFixture: f, servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	return app
}

func (a *seleccionApp) abrir(t *testing.T, c *http.Client, ruta string) string {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return formCuerpo(t, resp)
}

func (a *seleccionApp) elegir(t *testing.T, c *http.Client, id, volverA string) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+"/contexto/proyecto", url.Values{"proyecto_id": {id}, "volver_a": {volverA}})
	if err != nil {
		t.Fatalf("POST /contexto/proyecto: %v", err)
	}
	return resp, formCuerpo(t, resp)
}

func (a *seleccionApp) poner(t *testing.T, c *http.Client, cookies ...*http.Cookie) {
	t.Helper()
	u, err := url.Parse(a.servidor.URL)
	if err != nil {
		t.Fatal(err)
	}
	c.Jar.SetCookies(u, cookies)
}

func (a *seleccionApp) valorCookie(c *http.Client, nombre string) string {
	u, _ := url.Parse(a.servidor.URL)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == nombre {
			return ck.Value
		}
	}
	return ""
}

func idTexto(id int64) string { return strconv.FormatInt(id, 10) }

func cookieID(nombre string, id int64) *http.Cookie {
	return &http.Cookie{Name: nombre, Value: web.SignedContextValue(nombre, id), Path: "/"} // firmada (HU-26)
}

// US3 - Escenario: Selección de proyecto que se mantiene (FR-007, FR-008, FR-009, SC-006).
func TestEscenarioContexto_SeleccionQueSeMantiene(t *testing.T) {
	// Dado que existen proyectos
	app := nuevaSeleccionApp(t, nil, false)
	navegador := formNavegador(false)

	// Cuando selecciono uno y navego entre secciones
	resp, _ := app.elegir(t, navegador, idTexto(app.proyectoA), "/datos")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("se esperaba 303, se obtuvo %d", resp.StatusCode)
	}
	datos := app.abrir(t, navegador, "/datos")
	inicio := app.abrir(t, navegador, "/")

	// Entonces las secciones muestran únicamente los datos de ese proyecto sin volver a elegirlo,
	// y su nombre queda visible en la estructura común
	if datos != "datos:proyecto="+idTexto(app.proyectoA) {
		t.Errorf("la sección debía recibir solo el proyecto A, recibió %q", datos)
	}
	if !strings.Contains(inicio, app.nombreA) {
		t.Errorf("el nombre del proyecto debía verse en la estructura común")
	}
	if strings.Contains(inicio, app.nombreB) {
		t.Errorf("no debía verse el otro proyecto")
	}
}

// US3 - Escenario: Sin proyectos (FR-010).
func TestEscenarioContexto_SinProyectos(t *testing.T) {
	vacio := func(real web.ContextStore, _ *seleccionFixture) web.ContextStore { return listaFija{ContextStore: real} }

	t.Run("sin la sección Proyectos", func(t *testing.T) {
		// Dado que no existen proyectos
		app := nuevaSeleccionApp(t, vacio, false)

		// Cuando abro la aplicación
		cuerpo := app.abrir(t, formNavegador(true), "/contexto/proyecto")

		// Entonces veo "Aún no hay proyectos" y todavía no se ofrece crearlo
		if !strings.Contains(cuerpo, "Aún no hay proyectos") {
			t.Errorf("falta el mensaje \"Aún no hay proyectos\"")
		}
		if strings.Contains(cuerpo, web.LinkCreateFirstProject) {
			t.Errorf("no debe ofrecerse crear proyectos si la sección Proyectos no existe")
		}
	})
	t.Run("con la sección Proyectos", func(t *testing.T) {
		app := nuevaSeleccionApp(t, vacio, true)

		cuerpo := app.abrir(t, formNavegador(true), "/contexto/proyecto")

		// Entonces el acceso para crear el primero aparece cuando exista la sección de Proyectos
		if !strings.Contains(cuerpo, "Aún no hay proyectos") || !strings.Contains(cuerpo, web.LinkCreateFirstProject) {
			t.Errorf("debía verse el mensaje con el acceso para crear el primer proyecto")
		}
	})
}

// US3 - Escenario: Cambio de proyecto (FR-012).
func TestEscenarioContexto_CambioDeProyectoLimpiaElIntegrante(t *testing.T) {
	// Dado un proyecto seleccionado con un integrante identificado
	app := nuevaSeleccionApp(t, nil, false)
	navegador := formNavegador(false)
	app.poner(t, navegador, cookieID(web.CookieProject, app.proyectoA), cookieID(web.CookieMember, app.integranteA))
	if antes := app.abrir(t, navegador, "/"); !strings.Contains(antes, app.nombreIntegranteA) {
		t.Fatalf("precondición: debía verse al integrante identificado")
	}

	// Cuando cambio a otro proyecto
	app.elegir(t, navegador, idTexto(app.proyectoB), "/")

	// Entonces la identificación del integrante se limpia
	despues := app.abrir(t, navegador, "/")
	if strings.Contains(despues, app.nombreIntegranteA) || app.valorCookie(navegador, web.CookieMember) != "" {
		t.Errorf("la identificación debía limpiarse")
	}
	if !strings.Contains(despues, app.nombreB) {
		t.Errorf("debía verse el proyecto B")
	}
}

// US3 - Escenario: Mismo proyecto (FR-012).
func TestEscenarioContexto_MismoProyectoConservaElIntegrante(t *testing.T) {
	// Dado un proyecto seleccionado con un integrante identificado
	app := nuevaSeleccionApp(t, nil, false)
	navegador := formNavegador(false)
	app.poner(t, navegador, cookieID(web.CookieProject, app.proyectoA), cookieID(web.CookieMember, app.integranteA))

	// Cuando vuelvo a seleccionar el mismo proyecto
	app.elegir(t, navegador, idTexto(app.proyectoA), "/")

	// Entonces la identificación del integrante se mantiene
	if despues := app.abrir(t, navegador, "/"); !strings.Contains(despues, app.nombreIntegranteA) {
		t.Errorf("la identificación debía mantenerse")
	}
}

// US3 - Escenario: Identificador inválido (FR-013, SC-008).
func TestEscenarioContexto_IdentificadorInvalidoConservaLaSeleccion(t *testing.T) {
	for _, invalido := range []string{"abc", "-1", "99999999999", strconv.Itoa(web.MaxID)} {
		t.Run(invalido, func(t *testing.T) {
			// Dado un identificador de proyecto inexistente o inválido
			app := nuevaSeleccionApp(t, nil, false)
			navegador := formNavegador(true)
			app.elegir(t, navegador, idTexto(app.proyectoA), "/")

			// Cuando intento seleccionarlo
			resp, cuerpo := app.elegir(t, navegador, invalido, "/")

			// Entonces veo un mensaje claro, la aplicación no falla y se conserva la selección anterior
			if resp.StatusCode != http.StatusOK {
				t.Errorf("la aplicación no debía fallar: se obtuvo %d", resp.StatusCode)
			}
			if !strings.Contains(cuerpo, web.MsgProjectInvalid) {
				t.Errorf("falta el mensaje claro %q", web.MsgProjectInvalid)
			}
			if datos := app.abrir(t, navegador, "/datos"); datos != "datos:proyecto="+idTexto(app.proyectoA) {
				t.Errorf("la selección anterior debía conservarse, la sección recibió %q", datos)
			}
		})
	}
}

// US3 - Escenario: Selección guardada que ya no es válida (FR-014).
func TestEscenarioContexto_ProyectoEliminadoSeLimpiaConAdvertencia(t *testing.T) {
	// Dado que el proyecto que tenía seleccionado ya no existe en el sistema
	app := nuevaSeleccionApp(t, nil, false)
	id := app.crearEfimero(t)
	navegador := formNavegador(false)
	app.poner(t, navegador, cookieID(web.CookieProject, id))
	if _, err := app.db.Exec(`DELETE FROM projects WHERE id = $1`, id); err != nil {
		t.Fatalf("no se pudo eliminar el proyecto: %v", err)
	}

	// Cuando vuelvo a usar la aplicación
	cuerpo := app.abrir(t, navegador, "/")

	// Entonces la selección se limpia, veo una advertencia y se me pide elegir otro proyecto
	if !strings.Contains(cuerpo, "Advertencia:") || !strings.Contains(cuerpo, web.MsgProjectGone) {
		t.Errorf("debía verse la advertencia %q", web.MsgProjectGone)
	}
	if !strings.Contains(cuerpo, web.MsgNoProjectSelected) || app.valorCookie(navegador, web.CookieProject) != "" {
		t.Errorf("la selección debía limpiarse")
	}
}

// US3 - Escenario: Un solo proyecto (FR-011).
func TestEscenarioContexto_UnSoloProyectoNoSeAutoselecciona(t *testing.T) {
	// Dado que existe un solo proyecto
	solo := func(real web.ContextStore, f *seleccionFixture) web.ContextStore {
		return listaFija{ContextStore: real, proyectos: []web.ProjectSummary{{ID: f.proyectoA, Name: f.nombreA}}}
	}
	app := nuevaSeleccionApp(t, solo, false)
	navegador := formNavegador(false)

	// Cuando abro la aplicación
	inicio := app.abrir(t, navegador, "/")

	// Entonces no se selecciona automáticamente y se me pide elegirlo
	if !strings.Contains(inicio, web.MsgNoProjectSelected) || app.valorCookie(navegador, web.CookieProject) != "" {
		t.Errorf("no debía seleccionarse ningún proyecto")
	}
	if datos := app.abrir(t, navegador, "/datos"); datos != "datos:proyecto=0" {
		t.Errorf("las secciones no debían recibir proyecto, recibieron %q", datos)
	}
	lista := app.abrir(t, navegador, "/contexto/proyecto")
	if !strings.Contains(lista, app.nombreA) || !strings.Contains(lista, `name="proyecto_id"`) {
		t.Errorf("debía pedirse elegir el proyecto de la lista")
	}
}

// US3 - Escenario: Sesión nueva (FR-009).
func TestEscenarioContexto_SesionNuevaPideElegirDeNuevo(t *testing.T) {
	// Dado un proyecto seleccionado
	app := nuevaSeleccionApp(t, nil, false)
	primero := formNavegador(false)
	app.elegir(t, primero, idTexto(app.proyectoA), "/")
	if !strings.Contains(app.abrir(t, primero, "/"), app.nombreA) {
		t.Fatalf("precondición: el primer navegador debía tener el proyecto")
	}

	// Cuando cierro el navegador y vuelvo a abrir la aplicación (navegador nuevo, sin cookies)
	cuerpo := app.abrir(t, formNavegador(false), "/")

	// Entonces se me pide seleccionar el proyecto nuevamente
	if !strings.Contains(cuerpo, web.MsgNoProjectSelected) || strings.Contains(cuerpo, app.nombreA) {
		t.Errorf("en una sesión nueva debía pedirse elegir el proyecto")
	}
}
// --- US4 y US5: identificación del integrante y secciones que requieren contexto (T043, T050) ---

// seccionesApp es el servidor real con secciones de prueba: Libre (sin requisitos), Con proyecto
// (RequiresProject), Con integrante (RequiresMember, con una acción POST) y Datos (muestra el
// contexto que recibe la sección).
type seccionesApp struct {
	*seleccionApp
	acciones atomic.Int32
}

func nuevaSeccionesApp(t *testing.T) *seccionesApp {
	t.Helper()
	f := nuevaSeleccionFixture(t)
	app := &seccionesApp{}
	srv := web.NewServer(silencioso())
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(f.db))
	pagina := func(titulo string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			srv.Renderer().Page(w, r, "home.html", web.PageData{Title: titulo})
		}
	}
	registrar := func(s web.Section, rutas web.Routes) {
		t.Helper()
		if err := srv.Register(s, rutas); err != nil {
			t.Fatalf("no se pudo registrar la sección %q: %v", s.Name, err)
		}
	}
	registrar(web.Section{Name: "Libre", Path: "/libre"}, web.Routes{"GET /libre": pagina("Libre")})
	registrar(web.Section{Name: "Con proyecto", Path: "/con-proyecto", RequiresProject: true},
		web.Routes{"GET /con-proyecto": pagina("Con proyecto")})
	registrar(web.Section{Name: "Con integrante", Path: "/con-integrante", RequiresMember: true}, web.Routes{
		"GET /con-integrante": pagina("Con integrante"),
		"POST /con-integrante/guardar": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			app.acciones.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}),
	})
	registrar(web.Section{Name: "Datos", Path: "/datos"}, web.Routes{
		"GET /datos": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var proyecto, integrante int64
			if ac := web.ActiveContextFrom(r.Context()); ac != nil {
				proyecto, integrante = ac.ProjectID, ac.MemberID
			}
			fmt.Fprintf(w, "datos:proyecto=%d;integrante=%d", proyecto, integrante)
		}),
	})
	app.seleccionApp = &seleccionApp{seleccionFixture: f, servidor: httptest.NewServer(srv.Handler())}
	t.Cleanup(app.servidor.Close)
	return app
}

// integranteEn crea un integrante vinculado a un proyecto; se elimina al terminar la prueba.
func (a *seleccionApp) integranteEn(t *testing.T, proyecto int64, etiqueta string) (int64, string) {
	t.Helper()
	nombre := fmt.Sprintf("BDD %s %d", etiqueta, time.Now().UnixNano())
	id := a.insertar(t, `INSERT INTO members (display_name) VALUES ($1) RETURNING id`, nombre)
	t.Cleanup(func() { _, _ = a.db.Exec(`DELETE FROM members WHERE id = $1`, id) })
	if _, err := a.db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, proyecto, id); err != nil {
		t.Fatalf("no se pudo vincular el integrante: %v", err)
	}
	return id, nombre
}

func (a *seleccionApp) elegirIntegrante(t *testing.T, c *http.Client, id, volverA string) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+"/contexto/integrante", url.Values{"integrante_id": {id}, "volver_a": {volverA}})
	if err != nil {
		t.Fatalf("POST /contexto/integrante: %v", err)
	}
	return resp, formCuerpo(t, resp)
}

// pedir hace un GET y devuelve la respuesta (con el cuerpo ya leído); no sigue redirecciones
// si el navegador simulado no las sigue.
func (a *seleccionApp) pedir(t *testing.T, c *http.Client, ruta string) *http.Response {
	t.Helper()
	resp, err := c.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	_ = formCuerpo(t, resp)
	return resp
}

// enviarAccion envía la acción POST de la sección que requiere integrante.
func (a *seccionesApp) enviarAccion(t *testing.T, c *http.Client) *http.Response {
	t.Helper()
	resp, err := c.PostForm(a.servidor.URL+"/con-integrante/guardar", url.Values{"dato": {"1"}})
	if err != nil {
		t.Fatalf("POST /con-integrante/guardar: %v", err)
	}
	_ = formCuerpo(t, resp)
	return resp
}

// esPagina indica si el cuerpo es la página con ese título (el título va en <title>).
func esPagina(cuerpo, titulo string) bool {
	return strings.Contains(cuerpo, "<title>"+titulo+" · ")
}

// volverADe extrae el destino de regreso de una dirección de redirección.
func volverADe(t *testing.T, ubicacion string) string {
	t.Helper()
	u, err := url.Parse(ubicacion)
	if err != nil {
		t.Fatalf("dirección de redirección inválida %q: %v", ubicacion, err)
	}
	return u.Query().Get("volver_a")
}

func datosDe(proyecto, integrante int64) string {
	return "datos:proyecto=" + idTexto(proyecto) + ";integrante=" + idTexto(integrante)
}

// US4 - Escenario: Identificación (FR-015, SC-011).
func TestEscenarioIntegrante_Identificacion(t *testing.T) {
	// Dado un proyecto seleccionado
	app := nuevaSeccionesApp(t)
	_, nombreCarla := app.integranteEn(t, app.proyectoB, "Carla")
	navegador := formNavegador(false)
	app.elegir(t, navegador, idTexto(app.proyectoA), "/")

	// Cuando abro el selector de integrante y elijo mi nombre
	selector := app.abrir(t, navegador, "/contexto/integrante")
	resp, _ := app.elegirIntegrante(t, navegador, idTexto(app.integranteA), "/datos")

	// Entonces en el selector aparecen solo los integrantes de ese proyecto, sin contraseña
	if !strings.Contains(selector, app.nombreIntegranteA) {
		t.Errorf("el selector debía mostrar al integrante del proyecto")
	}
	if strings.Contains(selector, nombreCarla) {
		t.Errorf("el selector no debe mostrar integrantes de otro proyecto")
	}
	if strings.Contains(selector, `type="password"`) {
		t.Errorf("la identificación no lleva contraseña")
	}
	// el sistema usa mi nombre en mis acciones hasta que lo cambie
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/datos" {
		t.Fatalf("se esperaba 303 a /datos, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if datos := app.abrir(t, navegador, "/datos"); datos != datosDe(app.proyectoA, app.integranteA) {
		t.Errorf("las acciones debían usar mi nombre, la sección recibió %q", datos)
	}
	// y mi nombre queda visible en la estructura común mientras navego
	for _, ruta := range []string{"/", "/libre"} {
		cuerpo := app.abrir(t, navegador, ruta)
		if !strings.Contains(cuerpo, app.nombreIntegranteA) || !strings.Contains(cuerpo, "Integrante:") {
			t.Errorf("%s: el integrante debía verse en la estructura común", ruta)
		}
	}
}

// US4 - Escenario: Acción sin identificarme (FR-016).
func TestEscenarioIntegrante_AccionSinIdentificarse(t *testing.T) {
	// Dado que no me identifiqué
	app := nuevaSeccionesApp(t)
	navegador := formNavegador(false)
	app.elegir(t, navegador, idTexto(app.proyectoA), "/")

	// Cuando intento una acción que requiere integrante
	resp := app.enviarAccion(t, navegador)

	// Entonces se me pide identificarme
	ubicacion := resp.Header.Get("Location")
	if want := "/contexto/integrante?volver_a=" + url.QueryEscape("/con-integrante"); resp.StatusCode != http.StatusSeeOther || ubicacion != want {
		t.Fatalf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, ubicacion)
	}
	if app.acciones.Load() != 0 {
		t.Errorf("la acción no debía ejecutarse sin identificarse")
	}
	pantalla := app.abrir(t, navegador, ubicacion)
	if !strings.Contains(pantalla, web.MsgMemberRequired) || !strings.Contains(pantalla, app.nombreIntegranteA) {
		t.Errorf("debía verse %q con los integrantes del proyecto", web.MsgMemberRequired)
	}
	// y luego vuelvo a donde estaba
	vuelta, _ := app.elegirIntegrante(t, navegador, idTexto(app.integranteA), volverADe(t, ubicacion))
	if vuelta.StatusCode != http.StatusSeeOther || vuelta.Header.Get("Location") != "/con-integrante" {
		t.Fatalf("se esperaba volver a /con-integrante, se obtuvo %d a %q", vuelta.StatusCode, vuelta.Header.Get("Location"))
	}
	if cuerpo := app.abrir(t, navegador, "/con-integrante"); !esPagina(cuerpo, "Con integrante") {
		t.Errorf("la sección debía mostrarse tras identificarme")
	}
}

// US4 - Escenario: Integrante ajeno al proyecto (FR-017, SC-009).
func TestEscenarioIntegrante_IntegranteAjenoAlProyecto(t *testing.T) {
	app := nuevaSeccionesApp(t)
	idCarla, _ := app.integranteEn(t, app.proyectoB, "Carla")

	t.Run("al elegirlo en el selector", func(t *testing.T) {
		// Dado un integrante que no pertenece al proyecto activo
		navegador := formNavegador(false)
		app.elegir(t, navegador, idTexto(app.proyectoA), "/")
		app.elegirIntegrante(t, navegador, idTexto(app.integranteA), "/")

		// Cuando se envía una acción en su nombre
		resp, _ := app.elegirIntegrante(t, navegador, idTexto(idCarla), "/")

		// Entonces se rechaza, se limpia la identificación y se me pide elegir un integrante válido
		ubicacion := resp.Header.Get("Location")
		if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(ubicacion, "/contexto/integrante") {
			t.Fatalf("se esperaba 303 al selector de integrante, se obtuvo %d a %q", resp.StatusCode, ubicacion)
		}
		if app.valorCookie(navegador, web.CookieMember) != "" {
			t.Errorf("la identificación debía limpiarse")
		}
		if pantalla := app.abrir(t, navegador, ubicacion); !strings.Contains(pantalla, web.MsgMemberInvalid) {
			t.Errorf("falta el mensaje %q", web.MsgMemberInvalid)
		}
		if datos := app.abrir(t, navegador, "/datos"); datos != datosDe(app.proyectoA, 0) {
			t.Errorf("debía quedar el proyecto sin integrante, se obtuvo %q", datos)
		}
	})
	t.Run("con una identificación forjada", func(t *testing.T) {
		// Dado un navegador con la identificación de un integrante de otro proyecto
		navegador := formNavegador(false)
		app.poner(t, navegador, cookieID(web.CookieProject, app.proyectoA), cookieID(web.CookieMember, idCarla))
		antes := app.acciones.Load()

		// Cuando se envía una acción en su nombre
		resp := app.enviarAccion(t, navegador)

		// Entonces se rechaza, se limpia la identificación y se me pide elegir un integrante válido
		if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/integrante") {
			t.Fatalf("se esperaba 303 al selector de integrante, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
		}
		if app.acciones.Load() != antes {
			t.Errorf("la acción no debía ejecutarse")
		}
		if app.valorCookie(navegador, web.CookieMember) != "" {
			t.Errorf("la identificación debía limpiarse")
		}
	})
}

// US4 - Escenario: Integrante que dejó de pertenecer al proyecto (FR-018).
func TestEscenarioIntegrante_DejoDePertenecerAlProyecto(t *testing.T) {
	// Dado que mi integrante identificado ya no pertenece al proyecto
	app := nuevaSeccionesApp(t)
	idExtra, nombreExtra := app.integranteEn(t, app.proyectoA, "Quitado")
	navegador := formNavegador(false)
	app.poner(t, navegador, cookieID(web.CookieProject, app.proyectoA), cookieID(web.CookieMember, idExtra))
	if inicio := app.abrir(t, navegador, "/"); !strings.Contains(inicio, nombreExtra) {
		t.Fatalf("precondición: debía verse al integrante identificado")
	}
	if _, err := app.db.Exec(`DELETE FROM project_members WHERE project_id = $1 AND member_id = $2`, app.proyectoA, idExtra); err != nil {
		t.Fatalf("no se pudo quitar al integrante del proyecto: %v", err)
	}

	// Cuando navego por la aplicación
	cuerpo := app.abrir(t, navegador, "/")

	// Entonces la identificación se limpia y veo una advertencia
	if !strings.Contains(cuerpo, "Advertencia:") || !strings.Contains(cuerpo, web.MsgMemberGone) {
		t.Errorf("debía verse la advertencia %q", web.MsgMemberGone)
	}
	if strings.Contains(cuerpo, nombreExtra) || app.valorCookie(navegador, web.CookieMember) != "" {
		t.Errorf("la identificación debía limpiarse")
	}
	// y se me pide elegir otro integrante
	resp := app.enviarAccion(t, navegador)
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/contexto/integrante") {
		t.Errorf("se esperaba pedir identificarme, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if app.acciones.Load() != 0 {
		t.Errorf("la acción no debía ejecutarse")
	}
}

// US5 - Escenario: Sección sin proyecto (FR-019, SC-007).
func TestEscenarioSeccion_SinProyectoRedirigeYVuelve(t *testing.T) {
	// Dado que no hay proyecto seleccionado
	app := nuevaSeccionesApp(t)
	navegador := formNavegador(false)

	// Cuando entro a una sección declarada como que requiere proyecto
	resp := app.pedir(t, navegador, "/con-proyecto")

	// Entonces se me redirige a la selección de proyecto con un mensaje
	ubicacion := resp.Header.Get("Location")
	if want := "/contexto/proyecto?volver_a=" + url.QueryEscape("/con-proyecto"); resp.StatusCode != http.StatusSeeOther || ubicacion != want {
		t.Fatalf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, ubicacion)
	}
	pantalla := app.abrir(t, navegador, ubicacion)
	if !strings.Contains(pantalla, web.MsgProjectRequired) || !strings.Contains(pantalla, `name="proyecto_id"`) {
		t.Errorf("debía verse %q con la lista de proyectos", web.MsgProjectRequired)
	}
	if !strings.Contains(pantalla, `name="volver_a" value="/con-proyecto"`) {
		t.Errorf("el formulario debía llevar el destino de regreso")
	}
	// y, al elegir uno, vuelvo a la sección que había pedido
	vuelta, _ := app.elegir(t, navegador, idTexto(app.proyectoA), volverADe(t, ubicacion))
	if vuelta.StatusCode != http.StatusSeeOther || vuelta.Header.Get("Location") != "/con-proyecto" {
		t.Fatalf("se esperaba volver a /con-proyecto, se obtuvo %d a %q", vuelta.StatusCode, vuelta.Header.Get("Location"))
	}
	if cuerpo := app.abrir(t, navegador, "/con-proyecto"); !esPagina(cuerpo, "Con proyecto") || !strings.Contains(cuerpo, app.nombreA) {
		t.Errorf("la sección debía mostrarse con el proyecto activo")
	}
}

// US5 - Escenario: nunca una pantalla vacía ni un error interno (SC-007).
func TestEscenarioSeccion_SinContextoNuncaTerminaEnPantallaVaciaNiError(t *testing.T) {
	app := nuevaSeccionesApp(t)
	casos := []struct {
		nombre, ruta string
		conProyecto  bool
		destino      string
	}{
		{"requiere proyecto y no hay proyecto", "/con-proyecto", false, "/contexto/proyecto"},
		{"requiere integrante y no hay proyecto", "/con-integrante", false, "/contexto/proyecto"},
		{"requiere integrante y no me identifiqué", "/con-integrante", true, "/contexto/integrante"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			navegador := formNavegador(false)
			if tc.conProyecto {
				app.elegir(t, navegador, idTexto(app.proyectoA), "/")
			}

			resp := app.pedir(t, navegador, tc.ruta)

			if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), tc.destino) {
				t.Fatalf("se esperaba 303 a %s, se obtuvo %d a %q", tc.destino, resp.StatusCode, resp.Header.Get("Location"))
			}
			pantalla := app.abrir(t, navegador, resp.Header.Get("Location"))
			if !strings.Contains(pantalla, "alert-warning") {
				t.Errorf("la pantalla de selección debía explicar por qué se llegó a ella")
			}
		})
	}
}

// US5 - Escenario: Destino de regreso inválido (FR-020, SC-010).
func TestEscenarioSeccion_DestinoDeRegresoInvalidoSeIgnora(t *testing.T) {
	app := nuevaSeccionesApp(t)
	for _, destino := range []string{"//otro.com", "https://otro.com", `\\otro.com`, "/no-existe", "javascript:alert(1)"} {
		t.Run(destino, func(t *testing.T) {
			// Dado un destino de regreso externo o malformado
			navegador := formNavegador(false)

			// Cuando elijo un proyecto o un integrante
			proyecto, _ := app.elegir(t, navegador, idTexto(app.proyectoA), destino)
			integrante, _ := app.elegirIntegrante(t, navegador, idTexto(app.integranteA), destino)

			// Entonces el destino se ignora y voy a la página de inicio
			if proyecto.Header.Get("Location") != "/" {
				t.Errorf("al elegir proyecto: se esperaba ir a /, se obtuvo %q", proyecto.Header.Get("Location"))
			}
			if integrante.Header.Get("Location") != "/" {
				t.Errorf("al identificarme: se esperaba ir a /, se obtuvo %q", integrante.Header.Get("Location"))
			}
		})
	}
}

// US5 - Escenario: Sección sin requisitos (FR-021).
func TestEscenarioSeccion_SinRequisitosSeMuestraNormalmente(t *testing.T) {
	// Dado una sección que no requiere contexto
	app := nuevaSeccionesApp(t)
	navegador := formNavegador(false)

	// Cuando entro sin proyecto seleccionado
	resp := app.pedir(t, navegador, "/libre")
	cuerpo := app.abrir(t, navegador, "/libre")

	// Entonces se muestra normalmente
	if resp.StatusCode != http.StatusOK || !esPagina(cuerpo, "Libre") {
		t.Errorf("la sección debía mostrarse normalmente (estado %d)", resp.StatusCode)
	}
	if !strings.Contains(cuerpo, web.MsgNoProjectSelected) {
		t.Errorf("la estructura común debía indicar que no hay proyecto")
	}
}