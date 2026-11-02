package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// Pruebas de handler de HU-05a. Servidor real con CrossOriginGuard y el contexto de HU-20b,
// servicio y almacenamiento reales sobre PostgreSQL; solo se reemplazan el almacenamiento o el
// catálogo para simular fallas (T032).

type pokerApp struct {
	ts *httptest.Server
	fx *pokerWebFixture
}

type pokerOption struct {
	store   func(poker.Store) poker.Store
	catalog func(PokerCatalog) PokerCatalog
}

func newPokerApp(t *testing.T, opts ...pokerOption) *pokerApp {
	t.Helper()
	db := openContextTestDB(t)
	fx := newPokerWebFixture(t, db)
	var store poker.Store = poker.NewPostgreSQLStore(db)
	var catalog PokerCatalog = NewPostgreSQLPokerCatalog(db)
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
	if err := RegisterPlanningPoker(srv, poker.NewService(store), catalog); err != nil {
		t.Fatalf("no se pudo registrar Planning Poker: %v", err)
	}
	a := &pokerApp{ts: httptest.NewServer(srv.Handler()), fx: fx}
	t.Cleanup(a.ts.Close)
	return a
}

// failingPokerStore falla al guardar con un error lleno de detalles internos.
type failingPokerStore struct{ poker.Store }

var errPokerStoreInternal = errors.New(`INSERT INTO poker_votes: SQLSTATE 08006 pgx en postgres_store.go`)

func (failingPokerStore) OpenRound(context.Context, int64, int64, int64) (poker.OpenedRound, error) {
	return poker.OpenedRound{}, errPokerStoreInternal
}

func (failingPokerStore) CastVote(context.Context, int64, int64, int64, int) (poker.VoteOutcome, error) {
	return poker.VoteOutcome{}, errPokerStoreInternal
}

func (failingPokerStore) CloseRound(context.Context, int64, int64) (poker.ClosedRound, error) {
	return poker.ClosedRound{}, errPokerStoreInternal
}

func (failingPokerStore) OpenNextRound(context.Context, int64, int64, int64) (poker.OpenedRound, error) { // HU-18
	return poker.OpenedRound{}, errPokerStoreInternal
}

func (failingPokerStore) AgreeEstimate(context.Context, int64, int64, int) (poker.AgreedEstimate, error) { // HU-18
	return poker.AgreedEstimate{}, errPokerStoreInternal
}

// as devuelve un navegador que eligió el proyecto y, si member no es 0, se identificó (HU-20b).
func (a *pokerApp) as(t *testing.T, project, member int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	a.post(t, c, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(project, 10)}})
	if member != 0 {
		a.post(t, c, pathSelectMember, loginAs(t, a.fx.db, member))
	}
	return c
}

func (a *pokerApp) get(t *testing.T, c *http.Client, path string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(a.ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *pokerApp) post(t *testing.T, c *http.Client, path string, v url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := c.PostForm(a.ts.URL+path, v)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp, formsReadBody(t, resp)
}

func (a *pokerApp) open(t *testing.T, c *http.Client, story int64) (*http.Response, string) {
	t.Helper()
	return a.post(t, c, "/planning-poker/abrir", url.Values{"historia": {strconv.FormatInt(story, 10)}})
}

func (a *pokerApp) vote(t *testing.T, c *http.Client, round int64, value string) (*http.Response, string) {
	t.Helper()
	return a.post(t, c, "/planning-poker/votar", url.Values{"ronda": {strconv.FormatInt(round, 10)}, "valor": {value}})
}

// roundOf devuelve la ronda abierta de la historia.
func (a *pokerApp) roundOf(t *testing.T, story int64) int64 {
	t.Helper()
	var id int64
	if err := a.fx.db.QueryRow(`SELECT id FROM poker_rounds WHERE story_id = $1 AND status = 'Abierta'`, story).Scan(&id); err != nil {
		t.Fatalf("no hay ronda abierta para la historia %d: %v", story, err)
	}
	return id
}

func (a *pokerApp) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := a.fx.db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("no se pudo contar: %v", err)
	}
	return n
}

// participantRow devuelve el contenido de la fila de un integrante en la pantalla.
func participantRow(t *testing.T, body string, member int64) string {
	t.Helper()
	m := regexp.MustCompile(fmt.Sprintf(`<li data-integrante="%d">(.*?)</li>`, member)).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no se encontró la fila del integrante %d", member)
	}
	return m[1]
}

func pokerAssertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("se esperaba %d, se obtuvo %d", want, resp.StatusCode)
	}
}

func pokerAssertRedirect(t *testing.T, resp *http.Response) {
	t.Helper()
	pokerAssertStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/planning-poker" {
		t.Fatalf("se esperaba volver a /planning-poker, se obtuvo %q", loc)
	}
}

// ---------------------------------------------------------------- Bloque A: abrir ronda

// T033 (E1)
func TestHU05a_AbrirRonda(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	resp, _ := a.post(t, c, "/planning-poker/abrir", url.Values{
		"historia": {strconv.FormatInt(a.fx.login, 10)}, "integrante": {strconv.FormatInt(a.fx.beto, 10)},
	})
	pokerAssertRedirect(t, resp)

	_, body := a.get(t, c, "/planning-poker")
	if n := strings.Count(body, "Ronda abierta para la historia «"+a.fx.loginTitle+"»"); n != 1 {
		t.Errorf("el mensaje de apertura debía verse una vez, se vio %d", n)
	}
	if !strings.Contains(body, "Ronda 1") {
		t.Error("falta «Ronda 1»")
	}
	for _, m := range []int64{a.fx.ana, a.fx.beto} {
		if row := participantRow(t, body, m); !strings.Contains(row, LabelPokerNotVoted) {
			t.Errorf("el integrante %d debía figurar «%s»: %s", m, LabelPokerNotVoted, row)
		}
	}
	var openedBy int64
	_ = a.fx.db.QueryRow(`SELECT opened_by FROM poker_rounds WHERE story_id = $1`, a.fx.login).Scan(&openedBy)
	if openedBy != a.fx.ana {
		t.Errorf("la ronda debía quedar a nombre del integrante identificado (Ana), quedó %d", openedBy)
	}
}

// T034 (E9)
func TestHU05a_HistoriaConRondaAbierta(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	a.open(t, c, a.fx.login)
	resp, body := a.open(t, c, a.fx.login)
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerRoundAlreadyOpen) {
		t.Errorf("falta «%s»", MsgPokerRoundAlreadyOpen)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, a.fx.login); n != 1 {
		t.Errorf("debía haber una sola ronda, hay %d", n)
	}
}

// T035 (E10)
func TestHU05a_HistoriaYaTuvoRonda(t *testing.T) {
	a := newPokerApp(t)
	a.fx.round(t, a.fx.projectA, a.fx.login, false)
	resp, body := a.open(t, a.as(t, a.fx.projectA, a.fx.ana), a.fx.login)
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerStoryAlreadyRounded) {
		t.Errorf("falta «%s»", MsgPokerStoryAlreadyRounded)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, a.fx.login); n != 1 {
		t.Errorf("no debía crearse otra ronda, hay %d", n)
	}
}

// T036 (E1)
func TestHU05a_SelectorTodasLasHistorias(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	check := func(body string) {
		t.Helper()
		for _, want := range []string{
			fmt.Sprintf(`<option value="%d">%s (Pendiente)</option>`, a.fx.login, a.fx.loginTitle),
			fmt.Sprintf(`<option value="%d">%s (Hecho)</option>`, a.fx.pagos, a.fx.pagosTitle),
		} {
			if !strings.Contains(body, want) {
				t.Errorf("el selector no contiene %s", want)
			}
		}
		if strings.Contains(body, a.fx.bTitle) {
			t.Error("el selector no debía mostrar historias de otro proyecto")
		}
	}
	_, body := a.get(t, c, "/planning-poker")
	check(body)
	a.open(t, c, a.fx.login)
	_, body = a.get(t, c, "/planning-poker")
	check(body)
}

// ---------------------------------------------------------------- Bloque B: votar

// T045 (E2)
func TestHU05a_VotoRegistrado(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	a.open(t, c, a.fx.login)
	round := a.roundOf(t, a.fx.login)
	resp, _ := a.post(t, c, "/planning-poker/votar", url.Values{
		"ronda": {strconv.FormatInt(round, 10)}, "valor": {"5"}, "integrante": {strconv.FormatInt(a.fx.beto, 10)},
	})
	pokerAssertRedirect(t, resp)

	_, body := a.get(t, c, "/planning-poker")
	if n := strings.Count(body, MsgPokerVoteRegistered); n != 1 {
		t.Errorf("el mensaje debía verse una vez, se vio %d", n)
	}
	row := participantRow(t, body, a.fx.ana)
	if !strings.Contains(row, LabelPokerVoted) || !strings.Contains(row, "Tu voto: 5") {
		t.Errorf("la fila propia debía decir «Votó» y «Tu voto: 5»: %s", row)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1 AND member_id = $2 AND value = 5`, round, a.fx.ana); n != 1 {
		t.Errorf("el voto debía quedar a nombre de Ana, hay %d", n)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1 AND member_id = $2`, round, a.fx.beto); n != 0 {
		t.Error("el campo «integrante» enviado no debía usarse")
	}
}

// T046 (E3)
func TestHU05a_VotoActualizado(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	a.open(t, c, a.fx.login)
	round := a.roundOf(t, a.fx.login)
	a.vote(t, c, round, "5")
	a.get(t, c, "/planning-poker") // consume el mensaje del primer voto
	resp, _ := a.vote(t, c, round, "8")
	pokerAssertRedirect(t, resp)
	_, body := a.get(t, c, "/planning-poker")
	if !strings.Contains(body, MsgPokerVoteUpdated) || !strings.Contains(participantRow(t, body, a.fx.ana), "Tu voto: 8") {
		t.Errorf("se esperaba «%s» y «Tu voto: 8»", MsgPokerVoteUpdated)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1 AND member_id = $2`, round, a.fx.ana); n != 1 {
		t.Errorf("debía haber una sola fila de voto, hay %d", n)
	}
}

// ---------------------------------------------------------------- Bloque C: errores y estados

// openRoundAs abre la ronda de «Login» como Ana y devuelve el navegador de Ana y la ronda.
func (a *pokerApp) openRoundAs(t *testing.T) (*http.Client, int64) {
	t.Helper()
	c := a.as(t, a.fx.projectA, a.fx.ana)
	a.open(t, c, a.fx.login)
	return c, a.roundOf(t, a.fx.login)
}

// T054 (E5)
func TestHU05a_ValorFueraDeSerie(t *testing.T) {
	a := newPokerApp(t)
	c, round := a.openRoundAs(t)
	a.vote(t, c, round, "5")
	for _, v := range []string{"4", "0", "90", "abc"} {
		resp, body := a.vote(t, c, round, v)
		pokerAssertStatus(t, resp, http.StatusUnprocessableEntity)
		if !strings.Contains(body, MsgPokerValueNotInSeries) {
			t.Errorf("valor %q: falta «%s»", v, MsgPokerValueNotInSeries)
		}
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1 AND member_id = $2 AND value = 5`, round, a.fx.ana); n != 1 {
		t.Error("el voto anterior de 5 debía conservarse")
	}
}

// T055 (E6)
func TestHU05a_VotoSinValor(t *testing.T) {
	a := newPokerApp(t)
	c, round := a.openRoundAs(t)
	resp, body := a.post(t, c, "/planning-poker/votar", url.Values{"ronda": {strconv.FormatInt(round, 10)}})
	pokerAssertStatus(t, resp, http.StatusUnprocessableEntity)
	if !strings.Contains(body, MsgPokerSelectValue) {
		t.Errorf("falta «%s»", MsgPokerSelectValue)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round); n != 0 {
		t.Errorf("no debía registrarse nada, hay %d votos", n)
	}
}

// T056 (E7)
func TestHU05a_RondaCerrada(t *testing.T) {
	a := newPokerApp(t)
	closed := a.fx.round(t, a.fx.projectA, a.fx.login, false)
	a.fx.vote(t, closed, a.fx.beto, 8)
	resp, body := a.vote(t, a.as(t, a.fx.projectA, a.fx.ana), closed, "5")
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerRoundClosed) {
		t.Errorf("falta «%s»", MsgPokerRoundClosed)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, closed); n != 1 {
		t.Errorf("los votos no debían cambiar, hay %d", n)
	}
}

// T057 (E8a)
func TestHU05a_RondaInexistente(t *testing.T) {
	a := newPokerApp(t)
	c, round := a.openRoundAs(t)
	foreign := a.fx.round(t, a.fx.projectB, a.fx.storyB, true)
	for _, id := range []string{"999999999", "", "abc", "05", strconv.FormatInt(foreign, 10)} {
		resp, body := a.post(t, c, "/planning-poker/votar", url.Values{"ronda": {id}, "valor": {"5"}})
		pokerAssertStatus(t, resp, http.StatusNotFound)
		if !strings.Contains(body, MsgPokerRoundNotFound) {
			t.Errorf("ronda %q: falta «%s»", id, MsgPokerRoundNotFound)
		}
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id IN ($1, $2)`, round, foreign); n != 0 {
		t.Errorf("no debía registrarse nada, hay %d votos", n)
	}
}

// T057 (E8b)
func TestHU05a_HistoriaInexistente(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	for _, id := range []string{"999999999", "", "abc", "05", strconv.FormatInt(a.fx.storyB, 10)} {
		resp, body := a.post(t, c, "/planning-poker/abrir", url.Values{"historia": {id}})
		pokerAssertStatus(t, resp, http.StatusNotFound)
		if !strings.Contains(body, MsgPokerStoryNotFound) {
			t.Errorf("historia %q: falta «%s»", id, MsgPokerStoryNotFound)
		}
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE project_id IN ($1, $2)`, a.fx.projectA, a.fx.projectB); n != 0 {
		t.Errorf("no debía abrirse ninguna ronda, hay %d", n)
	}
}

// T058 (E11)
func TestHU05a_ErrorAlGuardar(t *testing.T) {
	a := newPokerApp(t, pokerOption{store: func(s poker.Store) poker.Store { return failingPokerStore{s} }})
	open := a.fx.round(t, a.fx.projectA, a.fx.login, true)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	for name, send := range map[string]func() (*http.Response, string){
		"votar": func() (*http.Response, string) { return a.vote(t, c, open, "5") },
		"abrir": func() (*http.Response, string) { return a.open(t, c, a.fx.pagos) },
	} {
		resp, body := send()
		pokerAssertStatus(t, resp, http.StatusInternalServerError)
		if !strings.Contains(body, MsgOperationFailed) {
			t.Errorf("%s: falta «%s»", name, MsgOperationFailed)
		}
		for _, leak := range []string{"poker_", "SQL", "pgx", errPokerStoreInternal.Error(), MsgPokerVoteRegistered, "Ronda abierta para"} {
			if strings.Contains(body, leak) {
				t.Errorf("%s: la respuesta no debía contener %q", name, leak)
			}
		}
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, open); n != 0 {
		t.Errorf("no debía registrarse nada, hay %d votos", n)
	}
}

// T059 (E12)
func TestHU05a_SinIdentificarme(t *testing.T) {
	a := newPokerApp(t)
	open := a.fx.round(t, a.fx.projectA, a.fx.login, true)
	c := a.as(t, a.fx.projectA, 0)
	resp, _ := a.vote(t, c, open, "5")
	pokerAssertStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/contexto/integrante?volver_a=%2Fplanning-poker" {
		t.Errorf("se esperaba pedir identificarse, se obtuvo %q", loc)
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, open); n != 0 {
		t.Errorf("no debía registrarse ningún voto, hay %d", n)
	}
	login := loginAs(t, a.fx.db, a.fx.ana)
	login.Set("volver_a", "/planning-poker")
	resp, _ = a.post(t, c, pathSelectMember, login)
	if loc := resp.Header.Get("Location"); loc != "/planning-poker" {
		t.Errorf("al identificarse se esperaba volver a /planning-poker, se obtuvo %q", loc)
	}
}
