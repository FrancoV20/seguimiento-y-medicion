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

// as devuelve un navegador que eligió el proyecto y, si member no es 0, se identificó (HU-20b).
func (a *pokerApp) as(t *testing.T, project, member int64) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	a.post(t, c, pathSelectProject, url.Values{"proyecto_id": {strconv.FormatInt(project, 10)}})
	if member != 0 {
		a.post(t, c, pathSelectMember, url.Values{"integrante_id": {strconv.FormatInt(member, 10)}})
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
