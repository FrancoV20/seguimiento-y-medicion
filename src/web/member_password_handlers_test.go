package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
)

// HU-27, T023: pantalla "Identificarme" con contraseña y "Salir", con PostgreSQLContextStore real,
// «Alfa» (Ana y Beto con contraseña, Caro sin) y «Beta» (Dani), y el bloqueo con reloj simulado.

const (
	anaPassword  = "ana-clave-2026"
	betoPassword = "beto-clave-2026"
	daniPassword = "dani-clave-2026"
)

type pwClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *pwClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *pwClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type pwFixture struct {
	alfa, beta            int64
	ana, beto, caro, dani int64
}

func newPwFixture(t *testing.T, db *sql.DB) *pwFixture {
	t.Helper()
	f := &pwFixture{}
	t.Cleanup(func() {
		for _, p := range []int64{f.alfa, f.beta} {
			_, _ = db.Exec(`DELETE FROM project_members WHERE project_id = $1`, p)
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, p)
		}
		for _, m := range []int64{f.ana, f.beto, f.caro, f.dani} {
			_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, m)
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
	member := func(name, password string) int64 {
		var hash any
		if password != "" {
			h, err := auth.HashPasswordIterations(password, 1000)
			if err != nil {
				t.Fatal(err)
			}
			hash = h
		}
		return id(`INSERT INTO members (display_name, password_hash) VALUES ($1, $2) RETURNING id`, name, hash)
	}
	stamp := time.Now().UnixNano()
	f.alfa = id(`INSERT INTO projects (name, start_date, end_date) VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, fmt.Sprintf("Alfa %d", stamp))
	f.beta = id(`INSERT INTO projects (name, start_date, end_date) VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, fmt.Sprintf("Beta %d", stamp))
	f.ana, f.beto, f.caro, f.dani = member("Ana", anaPassword), member("Beto", betoPassword), member("Caro", ""), member("Dani", daniPassword)
	for _, l := range [][2]int64{{f.alfa, f.ana}, {f.alfa, f.beto}, {f.alfa, f.caro}, {f.beta, f.dani}} {
		if _, err := db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, l[0], l[1]); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

type pwApp struct {
	*formsApp
	db    *sql.DB
	fx    *pwFixture
	clock *pwClock
}

func newPwApp(t *testing.T, wrap func(ContextStore) ContextStore) *pwApp {
	t.Helper()
	db := openContextTestDB(t)
	fx := newPwFixture(t, db)
	clock := &pwClock{now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	var store ContextStore = NewPostgreSQLContextStore(db)
	if wrap != nil {
		store = wrap(store)
	}
	app := newFormsApp(t, func(s *Server) { RegisterContextWithLockout(s, store, auth.NewLockout(clock.Now)) }, registerDatos)
	return &pwApp{formsApp: app, db: db, fx: fx, clock: clock}
}

// browserIn devuelve un navegador que eligió el proyecto.
func (a *pwApp) browserIn(t *testing.T, project int64) *http.Client {
	t.Helper()
	c := a.client(false)
	if resp, _ := a.post(t, c, pathSelectProject, url.Values{"proyecto_id": {id64(project)}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: %d", resp.StatusCode)
	}
	return c
}

func (a *pwApp) login(t *testing.T, c *http.Client, member int64, password string) (*http.Response, string) {
	t.Helper()
	return a.post(t, c, pathSelectMember, url.Values{
		"integrante_id": {id64(member)}, FieldPassword: {password}, "volver_a": {"/con-integrante"},
	})
}

func (a *pwApp) cookieOf(c *http.Client, name string) string {
	u, _ := url.Parse(a.ts.URL)
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func wantStatus(t *testing.T, resp *http.Response, status int) {
	t.Helper()
	if resp.StatusCode != status {
		t.Fatalf("status = %d; se esperaba %d", resp.StatusCode, status)
	}
}

func TestMemberPassword_Correcta(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	resp, _ := a.login(t, c, a.fx.ana, anaPassword)
	wantStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/con-integrante" {
		t.Errorf("Location = %q; se esperaba volver a /con-integrante", loc)
	}
	if v := a.cookieOf(c, CookieMember); v != SignedContextValue(CookieMember, a.fx.ana) {
		t.Errorf("cookie del integrante = %q; se esperaba la firmada de Ana", v)
	}
	_, body := a.get(t, c, "/con-integrante")
	if !strings.Contains(body, IdentifiedMessage("Ana")) || !strings.Contains(body, "Integrante: <strong>Ana</strong>") {
		t.Error("no se ve «Te identificaste como Ana» ni el nombre en la estructura común")
	}
}

func TestMemberPassword_Incorrecta(t *testing.T) {
	a := newPwApp(t, nil)
	for _, password := range []string{"otra-clave", strings.Repeat("x", 129)} {
		c := a.browserIn(t, a.fx.alfa)
		resp, body := a.login(t, c, a.fx.ana, password)
		wantStatus(t, resp, http.StatusUnauthorized)
		if !strings.Contains(body, MsgWrongCredentials) {
			t.Errorf("no aparece %q", MsgWrongCredentials)
		}
		if !strings.Contains(body, `value="`+id64(a.fx.ana)+`" checked`) {
			t.Error("Ana no quedó elegida")
		}
		if strings.Contains(body, `name="contrasena" value=`) || strings.Contains(body, password) {
			t.Error("el campo de contraseña no está vacío")
		}
		if a.cookieOf(c, CookieMember) != "" {
			t.Error("no debía quedar identificada")
		}
	}
}

func TestMemberPassword_Vacia(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	resp, body := a.login(t, c, a.fx.beto, "")
	wantStatus(t, resp, http.StatusUnprocessableEntity)
	if !strings.Contains(body, `class="field field-invalid"`) || !strings.Contains(body, `aria-describedby="contrasena-error"`) ||
		!strings.Contains(body, `id="contrasena-error"`) || !strings.Contains(body, MsgPasswordRequired) {
		t.Error("no aparece «Ingresá tu contraseña» junto al campo")
	}
	if a.cookieOf(c, CookieMember) != "" {
		t.Error("no debía quedar identificado")
	}
}

func TestMemberPassword_SinConfigurar(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	resp, body := a.login(t, c, a.fx.caro, "cualquier-cosa")
	wantStatus(t, resp, http.StatusUnprocessableEntity)
	if !strings.Contains(body, NoPasswordMessage("Caro")) || a.cookieOf(c, CookieMember) != "" {
		t.Errorf("se esperaba %q y sin identificación", NoPasswordMessage("Caro"))
	}
}

// Cuentas del bloqueo (spec.md, escenario 5).
func TestMemberPassword_Bloqueo(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	for i := 0; i < 4; i++ {
		resp, _ := a.login(t, c, a.fx.ana, "otra-clave")
		wantStatus(t, resp, http.StatusUnauthorized)
		a.clock.Advance(time.Second)
	}
	// Un rechazo 422 (contraseña vacía) no cuenta.
	resp, _ := a.login(t, c, a.fx.ana, "")
	wantStatus(t, resp, http.StatusUnprocessableEntity)
	resp, _ = a.login(t, c, a.fx.ana, "otra-clave") // 5.º fallo, a los 4 s del 1.º: bloquea 60 s
	wantStatus(t, resp, http.StatusUnauthorized)
	a.clock.Advance(26 * time.Second)
	resp, body := a.login(t, c, a.fx.ana, anaPassword)
	wantStatus(t, resp, http.StatusTooManyRequests)
	if !strings.Contains(body, MsgTooManyAttempts) || a.cookieOf(c, CookieMember) != "" {
		t.Error("durante el bloqueo se esperaba el mensaje y sin identificación")
	}
	a.clock.Advance(33 * time.Second) // 59 s después del 5.º fallo
	resp, _ = a.login(t, c, a.fx.ana, anaPassword)
	wantStatus(t, resp, http.StatusTooManyRequests)
	a.clock.Advance(time.Second) // 60 s después del 5.º fallo
	resp, _ = a.login(t, c, a.fx.ana, anaPassword)
	wantStatus(t, resp, http.StatusSeeOther)
	// Beto nunca estuvo bloqueado.
	c2 := a.browserIn(t, a.fx.alfa)
	resp, _ = a.login(t, c2, a.fx.beto, betoPassword)
	wantStatus(t, resp, http.StatusSeeOther)
}

func TestMemberPassword_RechazoConservaIdentificacionPrevia(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	resp, _ := a.login(t, c, a.fx.beto, betoPassword)
	wantStatus(t, resp, http.StatusSeeOther)
	before := a.cookieOf(c, CookieMember)
	resp, _ = a.login(t, c, a.fx.ana, "otra-clave")
	wantStatus(t, resp, http.StatusUnauthorized)
	if a.cookieOf(c, CookieMember) != before || before == "" {
		t.Error("un rechazo con Ana no debía cambiar la identificación de Beto")
	}
}

func TestMemberPassword_Salir(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	_, page := a.get(t, c, "/con-proyecto")
	if strings.Contains(page, `action="/contexto/salir"`) {
		t.Error("sin integrante identificado no debía verse «Salir»")
	}
	a.login(t, c, a.fx.ana, anaPassword)
	_, page = a.get(t, c, "/con-integrante")
	if !strings.Contains(page, `<form method="post" action="/contexto/salir" class="logout-form"><button type="submit">`+ButtonLogout+`</button></form>`) {
		t.Error("identificada, debía verse el botón «Salir» junto al nombre")
	}
	project := a.cookieOf(c, CookieProject)

	for _, volverA := range []string{"", "https://otro.sitio/", "//otro.sitio"} {
		resp, _ := a.post(t, c, pathLogout, url.Values{"volver_a": {volverA}})
		wantStatus(t, resp, http.StatusSeeOther)
		if loc := resp.Header.Get("Location"); loc != "/" {
			t.Errorf("volver_a=%q: Location = %q; se esperaba /", volverA, loc)
		}
	}
	if a.cookieOf(c, CookieMember) != "" || a.cookieOf(c, CookieProject) != project {
		t.Error("«Salir» debía borrar solo la cookie del integrante")
	}
	resp, _ := a.post(t, c, pathLogout, nil)
	wantStatus(t, resp, http.StatusSeeOther)
	_, body := a.get(t, c, "/con-proyecto")
	if !strings.Contains(body, MsgLoggedOut) || strings.Contains(body, "Integrante: <strong>") {
		t.Error("se esperaba «Cerraste tu identificación» y sin el nombre")
	}
	if resp, _ := a.get(t, c, pathLogout); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET %s = %d; se esperaba 405", pathLogout, resp.StatusCode)
	}
}

func TestMemberPassword_NuncaMuestraLaContrasena(t *testing.T) {
	a := newPwApp(t, nil)
	c := a.browserIn(t, a.fx.alfa)
	check := func(name string, resp *http.Response, body string, password string) {
		t.Helper()
		if strings.Contains(body, password) {
			t.Errorf("%s: el cuerpo contiene la contraseña", name)
		}
		for k, values := range resp.Header {
			for _, v := range values {
				if strings.Contains(v, password) {
					t.Errorf("%s: la cabecera %s contiene la contraseña", name, k)
				}
			}
		}
	}
	resp, body := a.login(t, c, a.fx.ana, "otra-clave-secreta")
	check("incorrecta", resp, body, "otra-clave-secreta")
	resp, body = a.login(t, c, a.fx.caro, "clave-de-caro-secreta")
	check("sin configurar", resp, body, "clave-de-caro-secreta")
	resp, body = a.login(t, c, a.fx.ana, anaPassword)
	check("correcta", resp, body, anaPassword)
	_, body = a.get(t, c, "/con-integrante")
	if strings.Contains(body, anaPassword) {
		t.Error("la página siguiente contiene la contraseña")
	}
}

// failingCredentials falla al leer el integrante.
type failingCredentials struct{ ContextStore }

func (failingCredentials) MemberCredential(context.Context, int64, int64) (MemberCredential, error) {
	return MemberCredential{}, Unavailable(errors.New("pgx: connection refused at 10.0.0.1:5432 select password_hash"))
}

// malformedCredentials devuelve un valor guardado mal formado.
type malformedCredentials struct{ ContextStore }

func (s malformedCredentials) MemberCredential(ctx context.Context, p, m int64) (MemberCredential, error) {
	cred, err := s.ContextStore.MemberCredential(ctx, p, m)
	cred.PasswordHash = "pbkdf2-sha256$0$malo$malo"
	return cred, err
}

func TestMemberPassword_ErrorDeLectura(t *testing.T) {
	a := newPwApp(t, func(s ContextStore) ContextStore { return failingCredentials{s} })
	c := a.browserIn(t, a.fx.alfa)
	resp, body := a.login(t, c, a.fx.ana, anaPassword)
	wantStatus(t, resp, http.StatusServiceUnavailable)
	for _, p := range []string{"sql:", "pgx", "postgres", "select ", "connection", "10.0.0.1", "password_hash"} {
		if strings.Contains(strings.ToLower(body), p) {
			t.Errorf("la página de error expone %q", p)
		}
	}
	if a.cookieOf(c, CookieMember) != "" {
		t.Error("no debía quedar identificada")
	}

	m := newPwApp(t, func(s ContextStore) ContextStore { return malformedCredentials{s} })
	c = m.browserIn(t, m.fx.alfa)
	resp, _ = m.login(t, c, m.fx.ana, anaPassword)
	wantStatus(t, resp, http.StatusInternalServerError)
	if m.cookieOf(c, CookieMember) != "" {
		t.Error("con un valor mal formado no debía quedar identificada")
	}
}
