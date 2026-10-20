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
	return &http.Cookie{Name: nombre, Value: idTexto(id), Path: "/"}
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