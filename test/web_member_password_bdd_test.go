package test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// HU-27: escenarios BDD de la identificación con contraseña, con la aplicación real (contexto de
// HU-20b, Planning Poker como sección que exige integrante) contra PostgreSQL. Datos del Issue:
// «Alfa» con Ana ("ana-clave-2026"), Beto ("beto-clave-2026") y Caro (sin contraseña); «Beta» con Dani.

const (
	claveAna  = "ana-clave-2026"
	claveBeto = "beto-clave-2026"
	claveDani = "dani-clave-2026"
)

type relojHU27 struct {
	mu    sync.Mutex
	ahora time.Time
}

func (r *relojHU27) Ahora() time.Time { r.mu.Lock(); defer r.mu.Unlock(); return r.ahora }
func (r *relojHU27) Avanzar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ahora = r.ahora.Add(d)
}

type hu27Mundo struct {
	db                    *sql.DB
	alfa, beta            int64
	ana, beto, caro, dani int64
}

func nuevoMundoHU27(t *testing.T) *hu27Mundo {
	t.Helper()
	raw := os.Getenv(web.EnvDatabaseURL)
	if raw == "" {
		t.Skip("DATABASE_URL no definida: se omite la integración BDD de HU-27")
	}
	db, err := sql.Open("pgx", raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	m := &hu27Mundo{db: db}
	t.Cleanup(func() {
		for _, p := range []int64{m.alfa, m.beta} {
			_, _ = db.Exec(`DELETE FROM project_members WHERE project_id = $1`, p)
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, p)
		}
		for _, i := range []int64{m.ana, m.beto, m.caro, m.dani} {
			_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, i)
		}
	})
	id := func(q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(q, args...).Scan(&v); err != nil {
			t.Fatalf("no se pudo preparar el dato de prueba: %v", err)
		}
		return v
	}
	sello := time.Now().UnixNano()
	m.alfa = id(`INSERT INTO projects (name, start_date, end_date) VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, fmt.Sprintf("Alfa HU27 %d", sello))
	m.beta = id(`INSERT INTO projects (name, start_date, end_date) VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, fmt.Sprintf("Beta HU27 %d", sello))
	for _, p := range []struct {
		dst      *int64
		nombre   string
		clave    string
		proyecto int64
	}{{&m.ana, "Ana", claveAna, m.alfa}, {&m.beto, "Beto", claveBeto, m.alfa}, {&m.caro, "Caro", "", m.alfa}, {&m.dani, "Dani", claveDani, m.beta}} {
		*p.dst = id(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, p.nombre)
		if p.clave != "" {
			configurarClave(t, db, *p.dst, p.clave)
		}
		if _, err := db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, p.proyecto, *p.dst); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

type hu27App struct {
	servidor *httptest.Server
	reloj    *relojHU27
	registro *bytes.Buffer
}

// laAplicacion arma el servidor; envolver permite simular una falla del almacén.
func (m *hu27Mundo) laAplicacion(t *testing.T, envolver func(web.ContextStore) web.ContextStore) *hu27App {
	t.Helper()
	registro := &bytes.Buffer{}
	srv := web.NewServer(slog.New(slog.NewTextHandler(registro, &slog.HandlerOptions{Level: slog.LevelDebug})))
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	var store web.ContextStore = web.NewPostgreSQLContextStore(m.db)
	if envolver != nil {
		store = envolver(store)
	}
	reloj := &relojHU27{ahora: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	web.RegisterContextWithLockout(srv, store, auth.NewLockout(reloj.Ahora))
	if err := web.RegisterPlanningPoker(srv, poker.NewService(poker.NewPostgreSQLStore(m.db)), web.NewPostgreSQLPokerCatalog(m.db)); err != nil {
		t.Fatalf("no se pudo registrar Planning Poker: %v", err)
	}
	app := &hu27App{servidor: httptest.NewServer(srv.Handler()), reloj: reloj, registro: registro}
	t.Cleanup(app.servidor.Close)
	return app
}

func (a *hu27App) navegador() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func (a *hu27App) pedir(t *testing.T, c *http.Client, metodo, ruta string, valores url.Values) (*http.Response, string) {
	t.Helper()
	var cuerpo *strings.Reader
	if valores != nil {
		cuerpo = strings.NewReader(valores.Encode())
	} else {
		cuerpo = strings.NewReader("")
	}
	req, err := http.NewRequest(metodo, a.servidor.URL+ruta, cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	if metodo == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", a.servidor.URL)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", metodo, ruta, err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	return resp, b.String()
}

// enProyecto devuelve un navegador que eligió el proyecto.
func (a *hu27App) enProyecto(t *testing.T, proyecto int64) *http.Client {
	t.Helper()
	c := a.navegador()
	a.elegirProyecto(t, c, proyecto)
	return c
}

func (a *hu27App) elegirProyecto(t *testing.T, c *http.Client, proyecto int64) {
	t.Helper()
	if resp, _ := a.pedir(t, c, http.MethodPost, "/contexto/proyecto", url.Values{"proyecto_id": {strconv.FormatInt(proyecto, 10)}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: %d", resp.StatusCode)
	}
}

func (a *hu27App) identificarme(t *testing.T, c *http.Client, integrante int64, clave string) (*http.Response, string) {
	t.Helper()
	valores := formIdentificacion(strconv.FormatInt(integrante, 10), clave)
	valores.Set("volver_a", "/planning-poker")
	return a.pedir(t, c, http.MethodPost, "/contexto/integrante", valores)
}

// identificado indica si el navegador entra a Planning Poker sin que se le pida identificarse.
func (a *hu27App) identificado(t *testing.T, c *http.Client) bool {
	t.Helper()
	resp, _ := a.pedir(t, c, http.MethodGet, "/planning-poker", nil)
	return resp.StatusCode == http.StatusOK
}

func esperarEstado(t *testing.T, resp *http.Response, estado int) {
	t.Helper()
	if resp.StatusCode != estado {
		t.Fatalf("estado = %d; se esperaba %d", resp.StatusCode, estado)
	}
}

func TestHU27_IdentificacionCorrecta(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	// Dado el proyecto «Alfa» elegido
	c := a.enProyecto(t, m.alfa)
	// Cuando elijo a Ana, ingreso "ana-clave-2026" y presiono "Identificarme"
	resp, _ := a.identificarme(t, c, m.ana, claveAna)
	// Entonces vuelvo a la pantalla que había pedido, veo "Te identificaste como Ana" y su nombre
	esperarEstado(t, resp, http.StatusSeeOther)
	if resp.Header.Get("Location") != "/planning-poker" {
		t.Errorf("Location = %q; se esperaba /planning-poker", resp.Header.Get("Location"))
	}
	_, pagina := a.pedir(t, c, http.MethodGet, "/planning-poker", nil)
	if !strings.Contains(pagina, "Te identificaste como Ana") || !strings.Contains(pagina, "Integrante: <strong>Ana</strong>") {
		t.Error("no se ve el mensaje o el nombre de Ana en la estructura común")
	}
}

func TestHU27_ContrasenaIncorrecta(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	// Cuando elijo a Ana e ingreso "otra-clave"
	resp, cuerpo := a.identificarme(t, c, m.ana, "otra-clave")
	// Entonces veo "Integrante o contraseña incorrectos", Ana sigue elegida y el campo está vacío
	esperarEstado(t, resp, http.StatusUnauthorized)
	if !strings.Contains(cuerpo, "Integrante o contraseña incorrectos") ||
		!strings.Contains(cuerpo, `value="`+strconv.FormatInt(m.ana, 10)+`" checked`) || strings.Contains(cuerpo, "otra-clave") {
		t.Error("se esperaba el mensaje, Ana elegida y el campo de contraseña vacío")
	}
	// y no quedo identificado
	if a.identificado(t, c) {
		t.Error("no debía quedar identificada")
	}
}

func TestHU27_ContrasenaVacia(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	// Cuando elijo a Beto y presiono "Identificarme" sin ingresar contraseña
	resp, cuerpo := a.identificarme(t, c, m.beto, "")
	// Entonces veo "Ingresá tu contraseña" junto al campo y no quedo identificado
	esperarEstado(t, resp, http.StatusUnprocessableEntity)
	if !strings.Contains(cuerpo, `id="contrasena-error"`) || !strings.Contains(cuerpo, "Ingresá tu contraseña") || a.identificado(t, c) {
		t.Error("se esperaba «Ingresá tu contraseña» junto al campo y sin identificación")
	}
}

func TestHU27_IntegranteSinContrasena(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	// Dado que Caro todavía no tiene contraseña; cuando la elijo e ingreso cualquier contraseña
	resp, cuerpo := a.identificarme(t, c, m.caro, "cualquier-clave")
	// Entonces veo el aviso y no quedo identificado
	esperarEstado(t, resp, http.StatusUnprocessableEntity)
	if !strings.Contains(cuerpo, "Caro todavía no tiene contraseña configurada. Pedile al Scrum Master que la configure") || a.identificado(t, c) {
		t.Error("se esperaba el aviso de Caro sin contraseña y sin identificación")
	}
}

func TestHU27_DemasiadosIntentos(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	// Dado que se ingresó 5 veces seguidas una contraseña incorrecta para Ana
	for i := 0; i < 5; i++ {
		resp, _ := a.identificarme(t, c, m.ana, "otra-clave")
		esperarEstado(t, resp, http.StatusUnauthorized)
		a.reloj.Avanzar(time.Second)
	}
	// Cuando ingreso su contraseña correcta antes de que pase un minuto
	a.reloj.Avanzar(25 * time.Second)
	resp, cuerpo := a.identificarme(t, c, m.ana, claveAna)
	// Entonces veo "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar" y no quedo identificado
	esperarEstado(t, resp, http.StatusTooManyRequests)
	if !strings.Contains(cuerpo, "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar") || a.identificado(t, c) {
		t.Error("durante el bloqueo se esperaba el mensaje y sin identificación")
	}
	// pasado el minuto, con la contraseña correcta me identifico
	a.reloj.Avanzar(time.Minute)
	resp, _ = a.identificarme(t, c, m.ana, claveAna)
	esperarEstado(t, resp, http.StatusSeeOther)
	if !a.identificado(t, c) {
		t.Error("pasado el minuto debía poder identificarse")
	}
}

func TestHU27_IntegranteDeOtroProyecto(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	// Cuando se envía la identificación de Dani, de «Beta», con su contraseña
	resp, _ := a.identificarme(t, c, m.dani, claveDani)
	// Entonces se rechaza con el mensaje de HU-20b y no quedo identificado
	esperarEstado(t, resp, http.StatusSeeOther)
	_, selector := a.pedir(t, c, http.MethodGet, resp.Header.Get("Location"), nil)
	if !strings.Contains(selector, web.MsgMemberInvalid) || a.identificado(t, c) {
		t.Errorf("se esperaba %q y sin identificación", web.MsgMemberInvalid)
	}
}

func TestHU27_Salir(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	// Dado que estoy identificado como Ana
	c := a.enProyecto(t, m.alfa)
	a.identificarme(t, c, m.ana, claveAna)
	// Cuando presiono "Salir"
	resp, _ := a.pedir(t, c, http.MethodPost, "/contexto/salir", url.Values{})
	esperarEstado(t, resp, http.StatusSeeOther)
	// Entonces veo "Cerraste tu identificación" y mi nombre deja de verse
	_, inicio := a.pedir(t, c, http.MethodGet, resp.Header.Get("Location"), nil)
	if !strings.Contains(inicio, "Cerraste tu identificación") || strings.Contains(inicio, "Integrante: <strong>Ana</strong>") {
		t.Error("se esperaba el mensaje y sin el nombre de Ana")
	}
	// y, si intento votar, se me pide identificarme con contraseña
	resp, _ = a.pedir(t, c, http.MethodGet, "/planning-poker", nil)
	esperarEstado(t, resp, http.StatusSeeOther)
	_, identificarme := a.pedir(t, c, http.MethodGet, resp.Header.Get("Location"), nil)
	if !strings.Contains(identificarme, `type="password"`) {
		t.Error("la pantalla para identificarse debía pedir la contraseña")
	}
}

func TestHU27_CambioDeProyecto(t *testing.T) {
	m := nuevoMundoHU27(t)
	a := m.laAplicacion(t, nil)
	// Dado que estoy identificado como Ana en «Alfa»
	c := a.enProyecto(t, m.alfa)
	a.identificarme(t, c, m.ana, claveAna)
	// Cuando cambio a otro proyecto y vuelvo a «Alfa»
	a.elegirProyecto(t, c, m.beta)
	a.elegirProyecto(t, c, m.alfa)
	// Entonces la identificación se limpió y para volver a identificarme tengo que ingresar la contraseña
	if a.identificado(t, c) {
		t.Fatal("la identificación debía limpiarse al cambiar de proyecto")
	}
	if resp, _ := a.identificarme(t, c, m.ana, ""); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("sin contraseña = %d; se esperaba 422", resp.StatusCode)
	}
	resp, _ := a.identificarme(t, c, m.ana, claveAna)
	esperarEstado(t, resp, http.StatusSeeOther)
}

func TestHU27_ContrasenaProtegida(t *testing.T) {
	m := nuevoMundoHU27(t)
	// Dado que Ana y Beto tienen contraseñas configuradas (en esta prueba, la misma)
	configurarClave(t, m.db, m.beto, claveAna)
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	var paginas []string
	for _, intento := range []struct {
		integrante int64
		clave      string
	}{{m.ana, "otra-clave"}, {m.beto, claveAna}, {m.ana, claveAna}} {
		_, cuerpo := a.identificarme(t, c, intento.integrante, intento.clave)
		paginas = append(paginas, cuerpo)
	}
	_, pagina := a.pedir(t, c, http.MethodGet, "/planning-poker", nil)
	paginas = append(paginas, pagina)
	// Cuando se revisa la base de datos, la página y el registro del servidor
	var valorAna, valorBeto string
	if err := m.db.QueryRow(`SELECT password_hash FROM members WHERE id = $1`, m.ana).Scan(&valorAna); err != nil {
		t.Fatal(err)
	}
	if err := m.db.QueryRow(`SELECT password_hash FROM members WHERE id = $1`, m.beto).Scan(&valorBeto); err != nil {
		t.Fatal(err)
	}
	// Entonces ninguna contraseña aparece en texto plano
	for nombre, texto := range map[string]string{"base (Ana)": valorAna, "base (Beto)": valorBeto,
		"páginas": strings.Join(paginas, "\n"), "registro del servidor": a.registro.String()} {
		for _, clave := range []string{claveAna, "otra-clave"} {
			if strings.Contains(texto, clave) {
				t.Errorf("%s contiene la contraseña %q en texto plano", nombre, clave)
			}
		}
	}
	// y dos integrantes con la misma contraseña tienen valores guardados distintos
	if valorAna == valorBeto {
		t.Error("Ana y Beto con la misma contraseña tienen el mismo valor guardado")
	}
}

func TestHU27_ConfigurarContrasena(t *testing.T) {
	m := nuevoMundoHU27(t)
	// Dado que Caro no tiene contraseña; cuando el Scrum Master la configura (como el comando)
	hash, err := auth.HashPassword("caro-clave-2026")
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.NewPostgreSQLPasswordStore(m.db).SetPasswordHash(context.Background(), m.caro, hash); err != nil {
		t.Fatal(err)
	}
	// Entonces Caro puede identificarse con esa contraseña
	a := m.laAplicacion(t, nil)
	c := a.enProyecto(t, m.alfa)
	resp, _ := a.identificarme(t, c, m.caro, "caro-clave-2026")
	esperarEstado(t, resp, http.StatusSeeOther)
	if !a.identificado(t, c) {
		t.Error("Caro debía quedar identificada")
	}
}

// almacenHU27QueFalla falla al leer los datos del integrante.
type almacenHU27QueFalla struct{ web.ContextStore }

func (almacenHU27QueFalla) MemberCredential(context.Context, int64, int64) (web.MemberCredential, error) {
	return web.MemberCredential{}, web.Unavailable(errors.New("pgx: dial tcp 10.0.0.1:5432: connection refused"))
}

func TestHU27_ErrorAlVerificar(t *testing.T) {
	m := nuevoMundoHU27(t)
	// Dado un fallo al leer los datos del integrante
	a := m.laAplicacion(t, func(s web.ContextStore) web.ContextStore { return almacenHU27QueFalla{s} })
	c := a.enProyecto(t, m.alfa)
	// Cuando intento identificarme
	resp, cuerpo := a.identificarme(t, c, m.ana, claveAna)
	// Entonces veo la página de error de la aplicación, sin detalles internos, y no quedo identificado
	esperarEstado(t, resp, http.StatusServiceUnavailable)
	for _, detalle := range []string{"pgx", "10.0.0.1", "connection refused", "dial tcp"} {
		if strings.Contains(cuerpo, detalle) {
			t.Errorf("la página de error muestra %q", detalle)
		}
	}
	if a.identificado(t, c) {
		t.Error("no debía quedar identificada")
	}
}
