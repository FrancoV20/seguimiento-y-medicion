package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// HU-05a: pruebas de lo que se ve en la pantalla (T047, T048, T064-T068).

var checkedValue = regexp.MustCompile(`name="valor" value="(\d+)" checked`)

// T047 (E4): Ana vota 13 y Beto 3; en la página de Beto no aparece el 13 de ninguna forma.
func TestHU05a_VotosAjenosOcultos(t *testing.T) {
	a := newPokerApp(t)
	ana := a.as(t, a.fx.projectA, a.fx.ana)
	beto := a.as(t, a.fx.projectA, a.fx.beto)
	a.open(t, ana, a.fx.login)
	round := a.roundOf(t, a.fx.login)
	a.vote(t, ana, round, "13")
	a.vote(t, beto, round, "3")
	a.get(t, beto, "/planning-poker") // consume el mensaje del voto

	_, before := a.get(t, beto, "/planning-poker")
	// Se quita el nombre: lleva números al azar que podrían contener un 13.
	anaRow := strings.TrimPrefix(participantRow(t, before, a.fx.ana), a.fx.anaName)
	betoRow := participantRow(t, before, a.fx.beto)
	if !strings.Contains(anaRow, LabelPokerVoted) || strings.Contains(anaRow, "13") {
		t.Errorf("la fila de Ana debía decir «Votó» sin el valor: %s", anaRow)
	}
	if !strings.Contains(betoRow, "Tu voto: 3") {
		t.Errorf("la fila de Beto debía mostrar su voto: %s", betoRow)
	}
	if got := checkedValue.FindAllStringSubmatch(before, -1); len(got) != 1 || got[0][1] != "3" {
		t.Errorf("el único valor marcado debía ser 3, se obtuvo %v", got)
	}

	a.vote(t, ana, round, "21")
	_, after := a.get(t, beto, "/planning-poker")
	if after != before {
		t.Error("la página de Beto cambió cuando Ana cambió su voto: depende del valor ajeno")
	}
}

// T048 (E4): al identificarse como Ana se ve el voto de Ana y no el de Beto.
func TestHU05a_CambioDeIdentificacion(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	a.open(t, c, a.fx.login)
	round := a.roundOf(t, a.fx.login)
	a.vote(t, c, round, "13")
	beto := a.as(t, a.fx.projectA, a.fx.beto)
	a.vote(t, beto, round, "3")

	// El mismo navegador de Beto se identifica como Ana.
	a.post(t, beto, pathSelectMember, url.Values{"integrante_id": {strconv.FormatInt(a.fx.ana, 10)}})
	a.get(t, beto, "/planning-poker")
	_, body := a.get(t, beto, "/planning-poker")
	if !strings.Contains(participantRow(t, body, a.fx.ana), "Tu voto: 13") || strings.Contains(body, "Tu voto: 3") {
		t.Errorf("se esperaba ver solo «Tu voto: 13»")
	}
}

// ---------------------------------------------------------------- Bloque D: pantalla

const pokerOpenForm = `action="/planning-poker/abrir"`

// T064 (E13)
func TestHU05a_PantallaVacia(t *testing.T) {
	a := newPokerApp(t)

	_, empty := a.get(t, a.as(t, a.fx.empty, a.fx.ana), "/planning-poker")
	for _, want := range []string{MsgPokerNoStories, MsgPokerNoOpenRounds} {
		if !strings.Contains(empty, want) {
			t.Errorf("proyecto sin historias: falta «%s»", want)
		}
	}
	if strings.Contains(empty, pokerOpenForm) {
		t.Error("proyecto sin historias: no debía mostrarse el formulario de apertura")
	}

	_, noRounds := a.get(t, a.as(t, a.fx.projectA, a.fx.ana), "/planning-poker")
	if !strings.Contains(noRounds, `id="historia"`) || !strings.Contains(noRounds, MsgPokerNoOpenRounds) {
		t.Error("proyecto con historias y sin rondas: debían verse el selector y «No hay rondas abiertas»")
	}
	if strings.Contains(noRounds, MsgPokerNoStories) {
		t.Errorf("proyecto con historias: no debía decir «%s»", MsgPokerNoStories)
	}
}

// T065
func TestHU05a_TituloEscapado(t *testing.T) {
	a := newPokerApp(t)
	const title, name = `<script>alert("t")</script>`, `<script>alert("n")</script>`
	story := a.fx.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Alta') RETURNING id`,
		a.fx.projectA, title)
	member := a.fx.member(t, name, a.fx.projectA)
	t.Cleanup(func() { _, _ = a.fx.db.Exec(`DELETE FROM members WHERE id = $1`, member) })

	c := a.as(t, a.fx.projectA, a.fx.ana)
	_, before := a.get(t, c, "/planning-poker")
	a.open(t, c, story)
	_, after := a.get(t, c, "/planning-poker")
	for page, body := range map[string]string{"selector": before, "ronda y mensaje": after} {
		if strings.Contains(body, title) || strings.Contains(body, name) {
			t.Errorf("%s: el texto con <script> se insertó sin escapar", page)
		}
	}
	if n := strings.Count(after, `&lt;script&gt;alert(&#34;t&#34;)&lt;/script&gt;`); n < 3 {
		t.Errorf("el título escapado debía verse en el selector, la ronda y el mensaje; apareció %d veces", n)
	}
	if !strings.Contains(after, `&lt;script&gt;alert(&#34;n&#34;)&lt;/script&gt;`) {
		t.Error("el nombre escapado debía verse en la lista de participantes")
	}
}

// failingPokerCatalog falla al leer con un error lleno de detalles internos.
type failingPokerCatalog struct{ err error }

func (f failingPokerCatalog) PokerBoard(context.Context, int64, int64) (PokerBoard, error) {
	return PokerBoard{}, f.err
}

// T066 (FR-015)
func TestHU05a_ErrorAlLeer(t *testing.T) {
	internal := errors.New(`SQL pgx: relation "poker_rounds" does not exist`)
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"falla inesperada", internal, http.StatusInternalServerError},
		{"base sin respuesta", Unavailable(internal), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newPokerApp(t, pokerOption{catalog: func(PokerCatalog) PokerCatalog { return failingPokerCatalog{tc.err} }})
			resp, body := a.get(t, a.as(t, a.fx.projectA, a.fx.ana), "/planning-poker")
			pokerAssertStatus(t, resp, tc.status)
			for _, leak := range []string{"poker_", "SQL", "pgx", "Abrir ronda", `name="valor"`} {
				if strings.Contains(body, leak) {
					t.Errorf("la página de error no debía contener %q", leak)
				}
			}
		})
	}
}

// T067 (E2)
func TestHU05a_RecargaSinReenviar(t *testing.T) {
	a := newPokerApp(t)
	c := a.as(t, a.fx.projectA, a.fx.ana)
	round := a.fx.round(t, a.fx.projectA, a.fx.login, true)
	a.vote(t, c, round, "8")
	_, first := a.get(t, c, "/planning-poker")
	_, second := a.get(t, c, "/planning-poker")
	if !strings.Contains(first, MsgPokerVoteRegistered) {
		t.Errorf("el primer GET debía mostrar «%s»", MsgPokerVoteRegistered)
	}
	if strings.Contains(second, MsgPokerVoteRegistered) {
		t.Error("al recargar, el mensaje no debía repetirse")
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round); n != 1 {
		t.Errorf("recargar no debía registrar votos: hay %d", n)
	}
}

// T068 (SC-007): 50 historias, 20 rondas abiertas y 10 integrantes en menos de 500 ms.
func TestHU05a_Rendimiento(t *testing.T) {
	a := newPokerApp(t)
	members := []int64{a.fx.ana, a.fx.beto}
	for i := len(members); i < 10; i++ {
		id := a.fx.member(t, fmt.Sprintf("Integrante %d Poker %s", i, a.fx.stamp), a.fx.projectA)
		t.Cleanup(func() { _, _ = a.fx.db.Exec(`DELETE FROM members WHERE id = $1`, id) })
		members = append(members, id)
	}
	for i := 2; i < 50; i++ { // login y pagos ya existen
		story := a.fx.id(t, `INSERT INTO user_stories (project_id, title, description, priority) VALUES ($1, $2, 'd', 'Media') RETURNING id`,
			a.fx.projectA, fmt.Sprintf("Historia %d Poker %s", i, a.fx.stamp))
		if i < 22 {
			round := a.fx.round(t, a.fx.projectA, story, true)
			for j, m := range members[:i%10+1] {
				a.fx.vote(t, round, m, poker.Series[j%len(poker.Series)])
			}
		}
	}
	c := a.as(t, a.fx.projectA, a.fx.ana)
	a.get(t, c, "/planning-poker") // calienta conexiones y plantillas
	start := time.Now()
	resp, body := a.get(t, c, "/planning-poker")
	elapsed := time.Since(start)
	pokerAssertStatus(t, resp, http.StatusOK)
	if n := strings.Count(body, `class="poker-round"`); n != 20 {
		t.Errorf("se esperaban 20 rondas abiertas, se vieron %d", n)
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("GET /planning-poker tardó %v (límite 500 ms)", elapsed)
	}
	t.Logf("GET /planning-poker con 50 historias, 20 rondas y 10 integrantes: %v", elapsed)
}
