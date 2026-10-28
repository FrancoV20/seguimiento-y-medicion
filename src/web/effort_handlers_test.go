package web

import (
	"context"
	"errors"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// Pruebas de handler de HU-24 (T013-T017, T023-T027, T031-T035, T038, T043-T045). Usan el
// servidor real, el flujo de HU-20b para elegir proyecto e integrante, el servicio real de HU-03
// y PostgreSQL real. Solo se reemplazan el store o el catálogo para simular fallas.

type effortApp struct {
	ts *httptest.Server
	fx *effortFixture
}

type effortAppOption struct {
	store   func(effort.EffortStore) effort.EffortStore
	catalog func(EffortCatalog) EffortCatalog
}

func newEffortApp(t *testing.T, opts ...effortAppOption) *effortApp {
	t.Helper()
	db := openContextTestDB(t)
	fx := newEffortFixture(t, db)

	var store effort.EffortStore = effort.NewPostgreSQLEffortStore(db)
	var catalog EffortCatalog = NewPostgreSQLEffortCatalog(db)
	for _, o := range opts {
		if o.store != nil {
			store = o.store(store)
		}
		if o.catalog != nil {
			catalog = o.catalog(catalog)
		}
	}

	srv := NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.Use(CrossOriginGuard(srv.Renderer()))
	RegisterContext(srv, NewPostgreSQLContextStore(db))
	actions := NewEffortActions(effort.NewService(store, effortClock), effortClock)
	if err := RegisterEffort(srv, actions, catalog, effortClock); err != nil {
		t.Fatalf("no se pudo registrar la sección: %v", err)
	}
	a := &effortApp{ts: httptest.NewServer(srv.Handler()), fx: fx}
	t.Cleanup(a.ts.Close)
	return a
}

// browser es un navegador con cookies que no sigue redirecciones.
func (a *effortApp) browser() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// login elige el proyecto y, si member no es 0, se identifica, por el flujo de HU-20b.
func (a *effortApp) login(t *testing.T, project, member int64) *http.Client {
	t.Helper()
	c := a.browser()
	resp, _ := a.postTo(t, c, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(project, 10)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: se esperaba 303, se obtuvo %d", resp.StatusCode)
	}
	if member != 0 {
		resp, _ = a.postTo(t, c, pathSelectMember, url.Values{"integrante_id": {strconv.FormatInt(member, 10)}})
		if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/") {
			t.Fatalf("identificarse: se esperaba 303, se obtuvo %d", resp.StatusCode)
		}
	}
	return c
}

func (a *effortApp) get(t *testing.T, c *http.Client, path string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *effortApp) postTo(t *testing.T, c *http.Client, path string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.ts.URL+path, v)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *effortApp) submit(t *testing.T, c *http.Client, v url.Values) (*http.Response, string) {
	t.Helper()
	return a.postTo(t, c, pathEffort, v)
}

func effortForm(story int64, date, activity, hours string) url.Values {
	return url.Values{
		EffortFieldStory: {strconv.FormatInt(story, 10)}, EffortFieldDate: {date},
		EffortFieldActivity: {activity}, EffortFieldHours: {hours},
	}
}

func idQuery(id int64) string {
	return pathEffort + "?" + EffortFieldStory + "=" + strconv.FormatInt(id, 10)
}

// fieldErrorHTML es el error que se muestra junto al campo name.
func fieldErrorHTML(name, msg string) string {
	return `id="` + name + `-error"><span class="field-error-label">Error:</span> ` + html.EscapeString(msg) + `</p>`
}

func inputValueHTML(name, value string) string {
	return `name="` + name + `" value="` + html.EscapeString(value) + `"`
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("se esperaba %d, se obtuvo %d", want, resp.StatusCode)
	}
}

func assertContains(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(body, p) {
			t.Errorf("la página no contiene %q", p)
		}
	}
}

func assertNotContains(t *testing.T, body string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if strings.Contains(body, p) {
			t.Errorf("la página no debía contener %q", p)
		}
	}
}

const effortFormHTML = `<form method="post" action="/esfuerzo">`

// effortRow lee la carga más reciente de la historia.
type effortRow struct {
	member   int64
	date     string
	activity string
	hours    int64
}

func (f *effortFixture) lastEffort(t *testing.T, story int64) effortRow {
	t.Helper()
	var r effortRow
	err := f.db.QueryRow(`SELECT member_id, to_char(work_date, 'YYYY-MM-DD'), activity, (hours * 100)::bigint
		FROM effort_logs WHERE story_id = $1 ORDER BY id DESC LIMIT 1`, story).Scan(&r.member, &r.date, &r.activity, &r.hours)
	if err != nil {
		t.Fatalf("no se pudo leer la carga: %v", err)
	}
	return r
}

// ---------------------------------------------------------------- US1

// T013 y T043: carga exitosa, mensaje de una sola vista y recarga sin reenviar.
func TestEffort_CargaExitosa_RegistraAlIntegranteIdentificadoYMuestraAcumulado(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	v := effortForm(a.fx.enSprint, "2026-10-20", "Maquetado", "2,5")
	v.Set("integrante", strconv.FormatInt(a.fx.beto, 10)) // se ignora
	resp, _ := a.submit(t, c, v)
	assertStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != idQuery(a.fx.enSprint) {
		t.Fatalf("se esperaba redirigir a %s, se obtuvo %q", idQuery(a.fx.enSprint), loc)
	}

	got := a.fx.lastEffort(t, a.fx.enSprint)
	want := effortRow{member: a.fx.ana, date: "2026-10-20", activity: "Maquetado", hours: 250}
	if got != want || a.fx.countEffort(t, a.fx.enSprint) != 1 {
		t.Fatalf("carga guardada: se esperaba %+v (una sola), se obtuvo %+v", want, got)
	}

	resp, body := a.get(t, c, idQuery(a.fx.enSprint))
	assertStatus(t, resp, http.StatusOK)
	if n := strings.Count(body, MsgEffortRecorded); n != 1 {
		t.Errorf("el mensaje de éxito debe verse una vez, se vio %d", n)
	}
	assertContains(t, body, "<td>2,5</td>", "Maquetado", a.fx.anaName, "Total: <strong>2,5</strong>")

	// T043: recargar no reenvía ni repite el mensaje.
	_, body = a.get(t, c, idQuery(a.fx.enSprint))
	assertNotContains(t, body, MsgEffortRecorded)
	if n := a.fx.countEffort(t, a.fx.enSprint); n != 1 {
		t.Errorf("recargar no debe registrar otra carga, hay %d", n)
	}
}

// T014: fecha de hoy por defecto (reloj fijo) y horas con coma y con punto.
func TestEffort_FechaDeHoyPorDefectoYHorasConComaOPunto(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	_, body := a.get(t, c, pathEffort)
	assertContains(t, body, inputValueHTML(EffortFieldDate, "2026-10-20"))

	for _, hours := range []string{"1,5", "1.25"} {
		resp, _ := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Carga "+hours, hours))
		assertStatus(t, resp, http.StatusSeeOther)
	}
	_, body = a.get(t, c, idQuery(a.fx.enSprint))
	assertContains(t, body, "Total: <strong>2,75</strong>", "<td>1,5</td>", "<td>1,25</td>")
}

// T015: el selector y la lista tienen solo las historias del Sprint activo del proyecto activo.
func TestEffort_SelectorYListaSoloDelSprintActivoDelProyecto(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	resp, body := a.get(t, c, pathEffort)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, effortFormHTML, "Historias del Sprint 2",
		a.fx.enSprintTitle, a.fx.enProgresoTitle, a.fx.hechoTitle, "<td>En Sprint</td>", "<td>Hecho</td>", "<td>6,75</td>")
	for _, id := range []int64{a.fx.enSprint, a.fx.enProgreso, a.fx.hecho} {
		assertContains(t, body, `<option value="`+strconv.FormatInt(id, 10)+`"`)
	}
	assertNotContains(t, body, a.fx.pendienteTitle, a.fx.finalizadaTitle, a.fx.ajenaTitle)
	for _, id := range []int64{a.fx.pendiente, a.fx.finalizada, a.fx.ajena} {
		assertNotContains(t, body, `<option value="`+strconv.FormatInt(id, 10)+`"`)
	}
}

// T016: sin Sprint activo no hay formulario y un envío responde 409 sin registrar.
func TestEffort_SinSprintActivo(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectC, a.fx.ana)
	before := a.fx.countAllEffort(t)

	resp, body := a.get(t, c, pathEffort)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, MsgEffortNoSprint)
	assertNotContains(t, body, effortFormHTML)

	resp, body = a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
	assertStatus(t, resp, http.StatusConflict)
	assertContains(t, body, MsgEffortNoSprint)
	assertNotContains(t, body, effortFormHTML, MsgEffortRecorded)
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// T017: sin integrante identificado se pide identificarse y se vuelve a /esfuerzo.
func TestEffort_SinIntegranteIdentificado(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, 0)
	before := a.fx.countAllEffort(t)

	resp, _ := a.get(t, c, pathEffort)
	assertStatus(t, resp, http.StatusSeeOther)
	wantLoc := pathSelectMember + "?volver_a=" + url.QueryEscape(pathEffort)
	if loc := resp.Header.Get("Location"); loc != wantLoc {
		t.Fatalf("se esperaba redirigir a %s, se obtuvo %q", wantLoc, loc)
	}

	// Un POST directo sin integrante tampoco registra: HU-20b lo desvía antes de HU-03.
	resp, _ = a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
	assertStatus(t, resp, http.StatusSeeOther)
	if !strings.HasPrefix(resp.Header.Get("Location"), pathSelectMember) {
		t.Errorf("se esperaba ir a identificarse, se obtuvo %q", resp.Header.Get("Location"))
	}
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}

	resp, _ = a.postTo(t, c, pathSelectMember, url.Values{
		"integrante_id": {strconv.FormatInt(a.fx.ana, 10)}, "volver_a": {pathEffort},
	})
	assertStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != pathEffort {
		t.Fatalf("después de identificarse se esperaba volver a %s, se obtuvo %q", pathEffort, loc)
	}
	resp, body := a.get(t, c, pathEffort)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, effortFormHTML, a.fx.anaName)
}

// ---------------------------------------------------------------- US2

// T023: todos los errores juntos, junto a su campo, con los mensajes de HU-03.
func TestEffort_VariosErroresJuntosConLosMensajesDeHU03(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	v := url.Values{EffortFieldStory: {""}, EffortFieldDate: {""}, EffortFieldActivity: {"   "}, EffortFieldHours: {"0"}}
	resp, body := a.submit(t, c, v)
	assertStatus(t, resp, http.StatusUnprocessableEntity)
	assertContains(t, body,
		fieldErrorHTML(EffortFieldStory, MsgEffortSelectStory),
		fieldErrorHTML(EffortFieldDate, effort.ErrDateRequired.Error()),
		fieldErrorHTML(EffortFieldActivity, effort.ErrActivityRequired.Error()),
		fieldErrorHTML(EffortFieldHours, effort.ErrInvalidHours.Error()),
		inputValueHTML(EffortFieldHours, "0"), inputValueHTML(EffortFieldActivity, "   "))
	assertNotContains(t, body, MsgEffortRecorded)
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// T024: horas inválidas y fecha con formato inválido o futura, con datos conservados.
func TestEffort_HorasYFechaInvalidas(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	for _, hours := range []string{"-1", "0", "25", "24,5", "abc", "", "1e2"} {
		t.Run("horas "+hours, func(t *testing.T) {
			resp, body := a.submit(t, c, effortForm(a.fx.enProgreso, "2026-10-20", "Diseño", hours))
			assertStatus(t, resp, http.StatusUnprocessableEntity)
			assertContains(t, body, fieldErrorHTML(EffortFieldHours, effort.ErrInvalidHours.Error()),
				inputValueHTML(EffortFieldHours, hours), inputValueHTML(EffortFieldActivity, "Diseño"),
				`<option value="`+strconv.FormatInt(a.fx.enProgreso, 10)+`" selected>`)
		})
	}
	dates := map[string]string{
		"20/10/2026": MsgEffortInvalidDate,
		"2026-13-01": MsgEffortInvalidDate,
		"2026-10-21": effort.ErrDateInFuture.Error(),
	}
	for date, msg := range dates {
		t.Run("fecha "+date, func(t *testing.T) {
			resp, body := a.submit(t, c, effortForm(a.fx.enProgreso, date, "Diseño", "2"))
			assertStatus(t, resp, http.StatusUnprocessableEntity)
			assertContains(t, body, fieldErrorHTML(EffortFieldDate, msg), inputValueHTML(EffortFieldDate, date))
		})
	}
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// T025: historia que no es del Sprint activo del proyecto activo.
func TestEffort_HistoriaNoDisponible(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	cases := map[string]struct{ story, msg string }{
		"de otro proyecto":        {strconv.FormatInt(a.fx.ajena, 10), MsgEffortStoryNotInSprint},
		"pendiente":               {strconv.FormatInt(a.fx.pendiente, 10), MsgEffortStoryNotInSprint},
		"de un Sprint finalizado": {strconv.FormatInt(a.fx.finalizada, 10), MsgEffortStoryNotInSprint},
		"inexistente":             {"2147483000", MsgEffortStoryNotInSprint},
		"no numérica":             {"abc", MsgEffortSelectStory},
		"no seleccionada":         {"", MsgEffortSelectStory},
	}
	for name, cs := range cases {
		t.Run(name, func(t *testing.T) {
			v := url.Values{EffortFieldStory: {cs.story}, EffortFieldDate: {"2026-10-19"},
				EffortFieldActivity: {"Análisis"}, EffortFieldHours: {"3"}}
			resp, body := a.submit(t, c, v)
			assertStatus(t, resp, http.StatusUnprocessableEntity)
			assertContains(t, body, fieldErrorHTML(EffortFieldStory, cs.msg),
				inputValueHTML(EffortFieldActivity, "Análisis"), inputValueHTML(EffortFieldHours, "3"),
				inputValueHTML(EffortFieldDate, "2026-10-19"))
		})
	}
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// failingSaveStore es el store real salvo que el guardado falla con un error con detalles internos.
type failingSaveStore struct{ effort.EffortStore }

func (failingSaveStore) SaveEntry(context.Context, effort.EffortEntry) (effort.EffortEntry, effort.Hours, error) {
	return effort.EffortEntry{}, 0, errors.New(`ERROR: insert into effort_logs failed (SQLSTATE 23514) at postgres_effort_store.go`)
}

// T026 y T031: falla de guardado sin éxito, datos conservados y sin detalles internos.
func TestEffort_FallaDeGuardado(t *testing.T) {
	a := newEffortApp(t, effortAppOption{store: func(s effort.EffortStore) effort.EffortStore { return failingSaveStore{s} }})
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	resp, body := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Documentación", "1,5"))
	assertStatus(t, resp, http.StatusInternalServerError)
	assertContains(t, body, MsgOperationFailed, inputValueHTML(EffortFieldActivity, "Documentación"),
		inputValueHTML(EffortFieldHours, "1,5"), inputValueHTML(EffortFieldDate, "2026-10-20"))
	assertNotContains(t, body, MsgEffortRecorded)
	assertNoInternals(t, body)
	for _, ck := range resp.Cookies() {
		if ck.Name == CookieFlash && ck.MaxAge >= 0 {
			t.Errorf("no debe guardarse un mensaje de éxito: %v", ck)
		}
	}
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// T027: el Sprint se cierra entre abrir el formulario y enviarlo: HU-03 rechaza la carga.
func TestEffort_SprintCerradoEntreAbrirYEnviar(t *testing.T) {
	a := newEffortApp(t, effortAppOption{catalog: func(real EffortCatalog) EffortCatalog {
		return &lazyStale{real: real}
	}})
	c := a.login(t, a.fx.projectA, a.fx.ana)
	_, body := a.get(t, c, pathEffort) // se abre el formulario con el Sprint activo
	assertContains(t, body, effortFormHTML)

	a.fx.exec(t, `UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, a.fx.sprintA)
	resp, body := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
	assertStatus(t, resp, http.StatusUnprocessableEntity)
	assertContains(t, body, fieldErrorHTML(EffortFieldStory, effort.ErrStoryNotInSprint.Error()), inputValueHTML(EffortFieldActivity, "Diseño"))
	if n := a.fx.countEffort(t, a.fx.enSprint); n != 0 {
		t.Errorf("no debía registrarse nada, hay %d cargas", n)
	}
}

// lazyStale guarda el Sprint de la primera lectura y lo repite después, como si el formulario
// se hubiera abierto antes de que el Sprint se cerrara.
type lazyStale struct {
	real   EffortCatalog
	cached *EffortSprint
	read   bool
}

func (l *lazyStale) ActiveSprintStories(ctx context.Context, projectID int64) (*EffortSprint, error) {
	if !l.read {
		s, err := l.real.ActiveSprintStories(ctx, projectID)
		if err != nil {
			return nil, err
		}
		l.cached, l.read = s, true
	}
	return l.cached, nil
}

func (l *lazyStale) StoryHistory(ctx context.Context, projectID, storyID int64) (*EffortHistory, error) {
	return l.real.StoryHistory(ctx, projectID, storyID)
}

// assertNoInternals verifica que la respuesta no exponga consultas, tablas, rutas ni el driver.
func assertNoInternals(t *testing.T, body string) {
	t.Helper()
	for _, s := range []string{"SELECT", "INSERT", "effort_logs", "SQLSTATE", ".go", "pgx", "postgres", "DATABASE_URL"} {
		if strings.Contains(body, s) {
			t.Errorf("la respuesta expone un detalle interno: %q", s)
		}
	}
}

// ---------------------------------------------------------------- US3

// T032: historial con las cargas de dos integrantes, más recientes primero, y total.
func TestEffort_HistorialDeLaHistoriaYTiempoDeRespuesta(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	start := time.Now()
	resp, body := a.get(t, c, idQuery(a.fx.enProgreso))
	elapsed := time.Since(start)
	assertStatus(t, resp, http.StatusOK)
	if elapsed > 500*time.Millisecond {
		t.Errorf("SC-006: la pantalla tardó %v (máximo 500 ms)", elapsed)
	}
	assertContains(t, body, "Historial de #"+strconv.FormatInt(a.fx.enProgreso, 10), a.fx.anaName, a.fx.betoName,
		"<td>12/10/2026</td>", "<td>10/10/2026</td>", "Total: <strong>6,75</strong>", "<td>6,75</td>",
		`<option value="`+strconv.FormatInt(a.fx.enProgreso, 10)+`" selected>`)
	iCode, iTests, iDesign := strings.Index(body, "<td>Código</td>"), strings.Index(body, "<td>Pruebas</td>"), strings.Index(body, "<td>Diseño</td>")
	if iCode < 0 || iTests < 0 || iDesign < 0 || !(iCode < iTests && iTests < iDesign) {
		t.Errorf("las cargas deben verse de la más reciente a la más antigua (Código, Pruebas, Diseño): %d %d %d", iCode, iTests, iDesign)
	}
	assertNotContains(t, body, "Cierre", "Ajena", a.fx.carlaName)
}

// T033: historia sin cargas.
func TestEffort_HistoriaSinRegistros(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	resp, body := a.get(t, c, idQuery(a.fx.enSprint))
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, MsgEffortNoEntries, "Total: <strong>0</strong>")
}

// T034: un ?historia inválido o ajeno se ignora y no se muestra historial.
func TestEffort_HistoriaInvalidaEnLaDireccionSeIgnora(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	for _, q := range []string{"abc", "-1", "2147483000", strconv.FormatInt(a.fx.ajena, 10),
		strconv.FormatInt(a.fx.pendiente, 10), strconv.FormatInt(a.fx.finalizada, 10)} {
		resp, body := a.get(t, c, pathEffort+"?historia="+url.QueryEscape(q))
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, effortFormHTML)
		assertNotContains(t, body, "Historial de", MsgEffortNoEntries)
	}
}

// T035: una actividad con HTML se muestra como texto.
func TestEffort_ActividadConScriptSeMuestraComoTexto(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)

	resp, _ := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", `<script>alert("x")</script>`, "1"))
	assertStatus(t, resp, http.StatusSeeOther)
	_, body := a.get(t, c, idQuery(a.fx.enSprint))
	assertContains(t, body, "&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;")
	assertNotContains(t, body, `<script>alert("x")</script>`)
}

// failingCatalog falla al leer las historias o el historial.
type failingCatalog struct {
	EffortCatalog
	failSprint, failHistory bool
}

var errCatalogInternal = errors.New(`SELECT us.id FROM user_stories: conexión rechazada en postgres_effort_catalog.go`)

func (f failingCatalog) ActiveSprintStories(ctx context.Context, projectID int64) (*EffortSprint, error) {
	if f.failSprint {
		return nil, errCatalogInternal
	}
	return f.EffortCatalog.ActiveSprintStories(ctx, projectID)
}

func (f failingCatalog) StoryHistory(ctx context.Context, projectID, storyID int64) (*EffortHistory, error) {
	if f.failHistory {
		return nil, errCatalogInternal
	}
	return f.EffortCatalog.StoryHistory(ctx, projectID, storyID)
}

// T038: falla al leer el historial.
func TestEffort_FallaAlLeerElHistorial(t *testing.T) {
	a := newEffortApp(t, effortAppOption{catalog: func(c EffortCatalog) EffortCatalog {
		return failingCatalog{EffortCatalog: c, failHistory: true}
	}})
	c := a.login(t, a.fx.projectA, a.fx.ana)

	resp, body := a.get(t, c, idQuery(a.fx.enProgreso))
	assertStatus(t, resp, http.StatusInternalServerError)
	assertContains(t, body, TitleInternal, MsgInternal)
	assertNotContains(t, body, effortFormHTML)
	assertNoInternals(t, body)
}

// ---------------------------------------------------------------- Polish

// T044: sin proyecto activo válido se pide elegir proyecto y no se registra nada.
func TestEffort_SinProyectoActivoValido(t *testing.T) {
	a := newEffortApp(t)
	before := a.fx.countAllEffort(t)

	t.Run("sin selección", func(t *testing.T) {
		c := a.browser()
		for _, send := range []func() *http.Response{
			func() *http.Response { r, _ := a.get(t, c, pathEffort); return r },
			func() *http.Response {
				r, _ := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
				return r
			},
		} {
			resp := send()
			assertStatus(t, resp, http.StatusSeeOther)
			if !strings.HasPrefix(resp.Header.Get("Location"), pathSelectProject) {
				t.Errorf("se esperaba ir a elegir proyecto, se obtuvo %q", resp.Header.Get("Location"))
			}
		}
	})

	t.Run("cookie inválida", func(t *testing.T) {
		c := a.browser()
		u, _ := url.Parse(a.ts.URL)
		c.Jar.SetCookies(u, []*http.Cookie{
			{Name: CookieProject, Value: "abc", Path: "/"},
			{Name: CookieMember, Value: strconv.FormatInt(a.fx.ana, 10), Path: "/"},
		})
		resp, _ := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
		assertStatus(t, resp, http.StatusSeeOther)
		if !strings.HasPrefix(resp.Header.Get("Location"), pathSelectProject) {
			t.Errorf("se esperaba ir a elegir proyecto, se obtuvo %q", resp.Header.Get("Location"))
		}
	})

	t.Run("proyecto eliminado", func(t *testing.T) {
		gone := a.fx.id(t, `INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, "HU24 eliminado "+a.fx.stamp)
		a.fx.exec(t, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, gone, a.fx.ana)
		c := a.login(t, gone, a.fx.ana)
		a.fx.exec(t, `DELETE FROM projects WHERE id = $1`, gone)

		resp, _ := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
		assertStatus(t, resp, http.StatusSeeOther)
		if !strings.HasPrefix(resp.Header.Get("Location"), pathSelectProject) {
			t.Errorf("se esperaba ir a elegir proyecto, se obtuvo %q", resp.Header.Get("Location"))
		}
	})

	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// T045: falla al leer las historias: página de error y no una pantalla incompleta.
func TestEffort_FallaAlLeerLasHistorias(t *testing.T) {
	a := newEffortApp(t, effortAppOption{catalog: func(c EffortCatalog) EffortCatalog {
		return failingCatalog{EffortCatalog: c, failSprint: true}
	}})
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	resp, body := a.get(t, c, pathEffort)
	assertStatus(t, resp, http.StatusInternalServerError)
	assertContains(t, body, TitleInternal)
	assertNotContains(t, body, effortFormHTML)
	assertNoInternals(t, body)

	resp, body = a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2"))
	assertStatus(t, resp, http.StatusInternalServerError)
	assertNoInternals(t, body)
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// Seguridad: un envío originado en otro sitio se rechaza con 403 sin registrar (HU-20b).
func TestEffort_EnvioDeOtroOrigenSeRechaza(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	req, _ := http.NewRequest(http.MethodPost, a.ts.URL+pathEffort,
		strings.NewReader(effortForm(a.fx.enSprint, "2026-10-20", "Diseño", "2").Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "https://otro-sitio.example")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	_ = formsReadBody(t, resp)
	assertStatus(t, resp, http.StatusForbidden)
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}

// El handler no confía en un integrante enviado ni en un proyecto ajeno: Carla (proyecto B)
// no puede cargar horas en historias del proyecto A aunque mande su id.
func TestEffort_NoSePuedeCargarEnHistoriasDeOtroProyecto(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectB, a.fx.carla)

	v := effortForm(a.fx.enProgreso, "2026-10-20", "Intrusión", "2")
	v.Set("proyecto_id", strconv.FormatInt(a.fx.projectA, 10))
	resp, body := a.submit(t, c, v)
	assertStatus(t, resp, http.StatusUnprocessableEntity)
	assertContains(t, body, fieldErrorHTML(EffortFieldStory, MsgEffortStoryNotInSprint))
	if n := a.fx.countEffort(t, a.fx.enProgreso); n != 3 {
		t.Errorf("la historia del proyecto A debía seguir con 3 cargas, tiene %d", n)
	}
}

// Seguridad: un envío más grande que el límite no registra nada.
func TestEffort_EnvioDemasiadoGrandeNoRegistra(t *testing.T) {
	a := newEffortApp(t)
	c := a.login(t, a.fx.projectA, a.fx.ana)
	before := a.fx.countAllEffort(t)

	resp, body := a.submit(t, c, effortForm(a.fx.enSprint, "2026-10-20", strings.Repeat("a", effortMaxBody), "1"))
	if resp.StatusCode == http.StatusSeeOther {
		t.Fatalf("un envío de más de %d bytes no debía aceptarse", effortMaxBody)
	}
	assertNotContains(t, body, MsgEffortRecorded)
	if after := a.fx.countAllEffort(t); after != before {
		t.Errorf("no debía registrarse nada: antes %d, después %d", before, after)
	}
}
