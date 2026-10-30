package web

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// HU-05b, T018: POST /planning-poker/cerrar con el servidor real (CrossOriginGuard y contexto de
// HU-20b), el servicio real y PostgreSQL; failingPokerStore simula la falla al guardar.

const pokerCloseAction = `action="/planning-poker/cerrar"`

var pokerCloseForm = regexp.MustCompile(`(?s)<form[^>]*action="/planning-poker/cerrar"[^>]*>(.*?)</form>`)

func (a *pokerApp) closeRound(t *testing.T, c *http.Client, round int64) (*http.Response, string) {
	t.Helper()
	return a.post(t, c, "/planning-poker/cerrar", url.Values{"ronda": {strconv.FormatInt(round, 10)}})
}

// closedCount cuenta las rondas cerradas de «Alfa».
func (f *alfaWebFixture) closedCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM poker_rounds WHERE project_id = $1 AND status = 'Cerrada'`, f.alfa).Scan(&n); err != nil {
		t.Fatalf("no se pudo contar: %v", err)
	}
	return n
}

// T018 (E1, FR-001)
func TestHU05b_BotonCerrarRonda(t *testing.T) {
	a, f := newAlfaApp(t)
	f.closeAt(t, f.rounds["Login"], alfaBase)
	body := a.page(t, f, f.ana)
	for _, title := range []string{"Pagos", "Reportes", "Perfil", "Ayuda"} {
		forms := pokerCloseForm.FindAllStringSubmatch(roundSection(body, f.rounds[title]), -1)
		if len(forms) != 1 {
			t.Errorf("«%s»: se esperaba un formulario «Cerrar ronda», hay %d", title, len(forms))
			continue
		}
		inner := forms[0][1]
		if !strings.Contains(inner, fmt.Sprintf(`<input type="hidden" name="ronda" value="%d">`, f.rounds[title])) || !strings.Contains(inner, "Cerrar ronda") {
			t.Errorf("«%s»: el formulario debía enviar su ronda y tener el botón «Cerrar ronda»:\n%s", title, inner)
		}
	}
	if section := closedSection(body, f.rounds["Login"]); section == "" || strings.Contains(section, pokerCloseAction) {
		t.Errorf("«Login» cerrada no debía ofrecer «Cerrar ronda»:\n%s", section)
	}
	if n := strings.Count(body, pokerCloseAction); n != 4 {
		t.Errorf("se esperaban 4 formularios «Cerrar ronda» (uno por ronda abierta), hay %d", n)
	}
}

// T018 (E1, FR-002, FR-016): Ana cierra «Login»; los campos integrante y proyecto enviados se ignoran.
func TestHU05b_CerrarRonda(t *testing.T) {
	a, f := newAlfaApp(t)
	c := a.as(t, f.alfa, f.ana)
	login := f.rounds["Login"]
	resp, _ := a.post(t, c, "/planning-poker/cerrar", url.Values{
		"ronda":      {strconv.FormatInt(login, 10)},
		"integrante": {strconv.FormatInt(f.dani, 10)},
		"proyecto":   {strconv.FormatInt(f.beta, 10)},
	})
	pokerAssertRedirect(t, resp)

	_, body := a.get(t, c, "/planning-poker")
	if n := strings.Count(body, "Ronda cerrada para la historia «Login»"); n != 1 {
		t.Errorf("el mensaje de cierre debía verse una vez, se vio %d", n)
	}
	if roundSection(body, login) != "" {
		t.Error("«Login» ya no debía figurar entre las rondas abiertas")
	}
	for member, want := range map[int64]string{f.ana: "Ana: 2", f.beto: "Beto: 13", f.caro: "Caro: 5"} {
		if row := closedRow(t, body, login, member); row != want {
			t.Errorf("en «Rondas cerradas» se esperaba %q, se obtuvo %q", want, row)
		}
	}
	if s := f.status(t, login); s != "Cerrada" || !f.closedAt(t, login).Valid {
		t.Errorf("«Login» debía quedar 'Cerrada' con fecha de cierre, quedó %q", s)
	}
	if s := f.status(t, f.rounds["Tablero"]); s != "Abierta" {
		t.Errorf("el campo «proyecto» enviado no debía usarse: «Tablero» quedó %q", s)
	}
}

// T018 (E5)
func TestHU05b_CerrarSinTodosLosVotos(t *testing.T) {
	a, f := newAlfaApp(t)
	c := a.as(t, f.alfa, f.ana)
	perfil := f.rounds["Perfil"]
	resp, _ := a.closeRound(t, c, perfil)
	pokerAssertRedirect(t, resp)
	_, body := a.get(t, c, "/planning-poker")
	for member, want := range map[int64]string{f.ana: "Ana: 5", f.beto: "Beto: 5", f.caro: "Caro: No votó"} {
		if row := closedRow(t, body, perfil, member); row != want {
			t.Errorf("se esperaba %q, se obtuvo %q", want, row)
		}
	}
	if !strings.Contains(closedSection(body, perfil), "Sin divergencia: los votos van de 5 a 5") {
		t.Error("falta «Sin divergencia: los votos van de 5 a 5»")
	}
}

// T018 (E6, FR-005)
func TestHU05b_RondaSinVotos(t *testing.T) {
	a, f := newAlfaApp(t)
	ayuda := f.rounds["Ayuda"]
	resp, body := a.closeRound(t, a.as(t, f.alfa, f.ana), ayuda)
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, "La ronda no tiene votos") {
		t.Error("falta «La ronda no tiene votos»")
	}
	if s := f.status(t, ayuda); s != "Abierta" || f.closedAt(t, ayuda).Valid {
		t.Errorf("«Ayuda» debía seguir abierta sin fecha de cierre, quedó %q", s)
	}
	if roundSection(body, ayuda) == "" {
		t.Error("«Ayuda» debía seguir entre las rondas abiertas")
	}
}

// T018 (E8, FR-012)
func TestHU05b_VotarEnRondaCerrada(t *testing.T) {
	a, f := newAlfaApp(t)
	login := f.rounds["Login"]
	pokerAssertRedirect(t, pokerResp(a.closeRound(t, a.as(t, f.alfa, f.ana), login)))
	resp, body := a.vote(t, a.as(t, f.alfa, f.caro), login, "8")
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerRoundClosed) {
		t.Errorf("falta «%s»", MsgPokerRoundClosed)
	}
	if got := f.vote(t, login, f.caro); got != 5 {
		t.Errorf("el voto de Caro debía seguir en 5, quedó %d", got)
	}
}

// T018 (E9, FR-004)
func TestHU05b_CerrarDosVeces(t *testing.T) {
	a, f := newAlfaApp(t)
	c := a.as(t, f.alfa, f.ana)
	login := f.rounds["Login"]
	pokerAssertRedirect(t, pokerResp(a.closeRound(t, c, login)))
	closedAt := f.closedAt(t, login)
	resp, body := a.closeRound(t, c, login)
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerRoundClosed) {
		t.Errorf("falta «%s»", MsgPokerRoundClosed)
	}
	if again := f.closedAt(t, login); !again.Valid || !again.Time.Equal(closedAt.Time) {
		t.Errorf("la fecha de cierre no debía cambiar: %v → %v", closedAt.Time, again.Time)
	}
}

// T018 (E10, FR-003): inexistente, inválido o de otro proyecto (abierta o cerrada) → 404.
func TestHU05b_RondaInexistente(t *testing.T) {
	a, f := newAlfaApp(t)
	c := a.as(t, f.alfa, f.ana)
	agendaClosed := f.closedAt(t, f.rounds["Agenda"])
	for _, id := range []string{"2147483647", "", "abc", "05", "-3", "99999999999999999999",
		strconv.FormatInt(f.rounds["Tablero"], 10), strconv.FormatInt(f.rounds["Agenda"], 10)} {
		resp, body := a.post(t, c, "/planning-poker/cerrar", url.Values{"ronda": {id}})
		pokerAssertStatus(t, resp, http.StatusNotFound)
		if !strings.Contains(body, MsgPokerRoundNotFound) {
			t.Errorf("ronda %q: falta «%s»", id, MsgPokerRoundNotFound)
		}
	}
	if n := f.closedCount(t); n != 0 {
		t.Errorf("no debía cerrarse ninguna ronda de «Alfa», hay %d", n)
	}
	if s := f.status(t, f.rounds["Tablero"]); s != "Abierta" {
		t.Errorf("«Tablero» debía seguir abierta, quedó %q", s)
	}
	if at := f.closedAt(t, f.rounds["Agenda"]); !at.Time.Equal(agendaClosed.Time) {
		t.Error("la fecha de cierre de «Agenda» no debía cambiar")
	}
}

// T018 (E11, FR-015)
func TestHU05b_SinIdentificarme(t *testing.T) {
	a, f := newAlfaApp(t)
	login := f.rounds["Login"]
	resp, _ := a.closeRound(t, a.as(t, f.alfa, 0), login)
	pokerAssertStatus(t, resp, http.StatusSeeOther)
	if loc := resp.Header.Get("Location"); loc != "/contexto/integrante?volver_a=%2Fplanning-poker" {
		t.Errorf("se esperaba pedir identificarse, se obtuvo %q", loc)
	}
	if s := f.status(t, login); s != "Abierta" {
		t.Errorf("«Login» debía seguir abierta, quedó %q", s)
	}
}

// T018 (E12, FR-013, FR-019)
func TestHU05b_ErrorAlCerrar(t *testing.T) {
	a, f := newAlfaApp(t, pokerOption{store: func(s poker.Store) poker.Store { return failingPokerStore{s} }})
	login := f.rounds["Login"]
	resp, body := a.closeRound(t, a.as(t, f.alfa, f.ana), login)
	pokerAssertStatus(t, resp, http.StatusInternalServerError)
	if !strings.Contains(body, MsgOperationFailed) {
		t.Errorf("falta «%s»", MsgOperationFailed)
	}
	for _, leak := range []string{"Ronda cerrada para", "poker_", "SQL", "pgx", errPokerStoreInternal.Error()} {
		if strings.Contains(body, leak) {
			t.Errorf("la respuesta no debía contener %q", leak)
		}
	}
	if s := f.status(t, login); s != "Abierta" {
		t.Errorf("«Login» debía seguir abierta, quedó %q", s)
	}
	if !strings.Contains(body, "Aún no hay rondas cerradas") {
		t.Error("no debía haber rondas cerradas: falta «Aún no hay rondas cerradas»")
	}
	if row := roundRow(t, body, login, f.beto); !strings.Contains(row, LabelPokerVoted) || strings.Contains(row, "13") {
		t.Errorf("los votos de «Login» debían seguir ocultos: %q", row)
	}
}

// T018 (FR-002): recargar después de cerrar no repite el mensaje ni reenvía el cierre.
func TestHU05b_RecargaSinReenviar(t *testing.T) {
	a, f := newAlfaApp(t)
	c := a.as(t, f.alfa, f.ana)
	login := f.rounds["Login"]
	pokerAssertRedirect(t, pokerResp(a.closeRound(t, c, login)))
	closedAt := f.closedAt(t, login)
	_, firstGet := a.get(t, c, "/planning-poker")
	_, second := a.get(t, c, "/planning-poker")
	if n := strings.Count(firstGet, "Ronda cerrada para la historia «Login»"); n != 1 {
		t.Errorf("el primer GET debía mostrar el mensaje una vez, lo mostró %d", n)
	}
	if strings.Contains(second, "Ronda cerrada para la historia") {
		t.Error("al recargar, el mensaje no debía repetirse")
	}
	if at := f.closedAt(t, login); !at.Time.Equal(closedAt.Time) {
		t.Error("recargar no debía volver a cerrar la ronda")
	}
}

// pokerResp devuelve solo la respuesta de un par (respuesta, cuerpo).
func pokerResp(resp *http.Response, _ string) *http.Response { return resp }
