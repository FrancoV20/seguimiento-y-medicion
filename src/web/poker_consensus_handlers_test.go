package web

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// HU-18, T019: "Nueva ronda" y "Estimación acordada" con el servidor real (CrossOriginGuard y
// contexto de HU-20b), el servicio real, PostgreSQL y el Backlog registrado para ver los Story
// Points; failingPokerStore simula la falla al guardar.

const (
	pokerNextRoundAction = `action="/planning-poker/nueva-ronda"`
	pokerEstimateAction  = `action="/planning-poker/estimacion"`
)

var pokerEstimateForm = regexp.MustCompile(`(?s)<form[^>]*action="/planning-poker/estimacion"[^>]*>(.*?)</form>`)

// T019 (FR-001)
func TestHU18_AccionesSoloEnLaUltimaRonda(t *testing.T) {
	a, f := newConsensusApp(t)
	body := a.consensusPage(t, a.as(t, f.alfa, f.ana))
	for _, title := range []string{"Login", "Pagos", "Alta"} {
		section := closedSection(body, f.rounds[title])
		hidden := fmt.Sprintf(`<input type="hidden" name="ronda" value="%d">`, f.rounds[title])
		for _, want := range []string{pokerNextRoundAction, pokerEstimateAction, hidden, "Nueva ronda", "Registrar estimación"} {
			if !strings.Contains(section, want) {
				t.Errorf("«%s»: faltaba %q en la última ronda cerrada:\n%s", title, want, section)
			}
		}
		if n := strings.Count(section, `name="valor"`); n != 10 {
			t.Errorf("«%s»: se esperaban diez radios «valor», hay %d", title, n)
		}
		if strings.Contains(section, "checked") {
			t.Errorf("«%s»: ningún valor de la estimación debía salir marcado", title)
		}
	}
	reportes := closedSection(body, f.rounds["Reportes"])
	if reportes == "" || strings.Contains(reportes, pokerNextRoundAction) || strings.Contains(reportes, pokerEstimateAction) {
		t.Errorf("la ronda 1 de «Reportes» no es la última y no debía ofrecer acciones:\n%s", reportes)
	}
}

// T019 (E1)
func TestHU18_NuevaRonda(t *testing.T) {
	a, f := newConsensusApp(t)
	beto := a.as(t, f.alfa, f.beto)
	resp, _ := a.nextRound(t, beto, id64(f.rounds["Login"]), "integrante", id64(f.ana))
	pokerAssertRedirect(t, resp)

	_, body := a.get(t, beto, "/planning-poker")
	if n := strings.Count(body, "Ronda 2 abierta para la historia «Login»"); n != 1 {
		t.Errorf("el mensaje debía aparecer una vez, aparece %d", n)
	}
	second := f.roundNumbered(t, "Login", 2)
	for member, name := range map[int64]string{f.ana: "Ana", f.beto: "Beto", f.caro: "Caro"} {
		if row := roundRow(t, body, second, member); row != name+": "+LabelPokerNotVoted {
			t.Errorf("en la ronda 2 se esperaba «%s: Sin votar», se obtuvo %q", name, row)
		}
	}
	for member, want := range map[int64]string{f.ana: "Ana: 2", f.beto: "Beto: 13", f.caro: "Caro: 5"} {
		if row := closedRow(t, body, f.rounds["Login"], member); row != want {
			t.Errorf("la ronda 1 debía seguir en «Rondas cerradas» con %q, se obtuvo %q", want, row)
		}
	}
	if n := a.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE id = $1 AND opened_by = $2`, second, f.beto); n != 1 {
		t.Error("la ronda 2 debía quedar abierta por Beto: el campo «integrante» enviado se ignora")
	}
}

// T019 (E2)
func TestHU18_VotosOcultosEnLaNuevaRonda(t *testing.T) {
	a, f := newConsensusApp(t)
	ana, beto := a.as(t, f.alfa, f.ana), a.as(t, f.alfa, f.beto)
	a.nextRound(t, ana, id64(f.rounds["Login"]))
	second := f.roundNumbered(t, "Login", 2)
	a.vote(t, ana, second, "8")

	before := a.consensusPage(t, beto)
	if row := roundRow(t, before, second, f.ana); row != "Ana: "+LabelPokerVoted {
		t.Errorf("Beto debía ver que Ana votó, sin el valor: %q", row)
	}
	a.vote(t, ana, second, "21")
	if after := a.consensusPage(t, beto); after != before {
		t.Error("la página de Beto cambió cuando Ana cambió su voto: depende del valor ajeno")
	}
}

// T019 (E3)
func TestHU18_RegistrarEstimacion(t *testing.T) {
	a, f := newConsensusApp(t)
	ana, beto, caro := a.as(t, f.alfa, f.ana), a.as(t, f.alfa, f.beto), a.as(t, f.alfa, f.caro)
	a.nextRound(t, ana, id64(f.rounds["Login"]))
	second := f.roundNumbered(t, "Login", 2)
	a.vote(t, ana, second, "8")
	a.vote(t, beto, second, "8")
	a.vote(t, caro, second, "5")
	a.closeRound(t, ana, second)

	resp, _ := a.agree(t, ana, id64(second), "8")
	pokerAssertRedirect(t, resp)
	_, body := a.get(t, ana, "/planning-poker")
	if n := strings.Count(body, "Estimación registrada: «Login» queda con 8 Story Points"); n != 1 {
		t.Errorf("el mensaje debía aparecer una vez, aparece %d", n)
	}
	_, list := a.get(t, ana, "/backlog")
	if got := backlogPoints(list, "Login"); got != "8" {
		t.Errorf("el Backlog debía mostrar «Login» con 8 Story Points, muestra %q", got)
	}
}

// T019 (E4)
func TestHU18_ConsensoDirecto(t *testing.T) {
	a, f := newConsensusApp(t)
	ana := a.as(t, f.alfa, f.ana)
	resp, _ := a.agree(t, ana, id64(f.rounds["Pagos"]), "5")
	pokerAssertRedirect(t, resp)
	_, body := a.get(t, ana, "/planning-poker")
	if !strings.Contains(body, "Estimación registrada: «Pagos» queda con 5 Story Points") {
		t.Error("faltaba el mensaje de la estimación registrada")
	}
	if sp := f.storyPoints(t, "Pagos"); sp != 5 || f.roundCount(t, "Pagos") != 1 {
		t.Errorf("«Pagos» debía quedar con 5 Story Points sin otra ronda: %d SP, %d rondas", sp, f.roundCount(t, "Pagos"))
	}
}

// T019 (E5)
func TestHU18_RecordatorioDeDivergencia(t *testing.T) {
	a, f := newConsensusApp(t)
	ana := a.as(t, f.alfa, f.ana)
	body := a.consensusPage(t, ana)
	login := pokerEstimateForm.FindStringSubmatch(closedSection(body, f.rounds["Login"]))
	if login == nil || !strings.Contains(login[1], pokerWarningAlert) || !strings.Contains(login[1], MsgPokerDivergenceReminder) {
		t.Errorf("el formulario de «Login» debía tener el recordatorio con «Advertencia:»:\n%v", login)
	}
	for _, title := range []string{"Pagos", "Alta"} {
		if strings.Contains(closedSection(body, f.rounds[title]), MsgPokerDivergenceReminder) {
			t.Errorf("«%s» no tuvo divergencia: no debía mostrar el recordatorio", title)
		}
	}
	resp, _ := a.agree(t, ana, id64(f.rounds["Login"]), "5")
	pokerAssertRedirect(t, resp)
	if sp := f.storyPoints(t, "Login"); sp != 5 {
		t.Errorf("«Login» debía quedar con 5 Story Points, tiene %d", sp)
	}
}

// T019 (E6)
func TestHU18_ValorFueraDeSerie(t *testing.T) {
	a, f := newConsensusApp(t)
	ana := a.as(t, f.alfa, f.ana)
	for _, v := range []string{"4", "abc", "05"} {
		resp, body := a.agree(t, ana, id64(f.rounds["Pagos"]), v)
		pokerAssertStatus(t, resp, http.StatusUnprocessableEntity)
		if !strings.Contains(body, MsgPokerValueNotInSeries) {
			t.Errorf("valor %q: faltaba «%s»", v, MsgPokerValueNotInSeries)
		}
	}
	for _, round := range []string{id64(f.rounds["Pagos"]), "999999999"} {
		resp, body := a.agree(t, ana, round, "")
		pokerAssertStatus(t, resp, http.StatusUnprocessableEntity)
		if !strings.Contains(body, MsgPokerSelectValue) {
			t.Errorf("ronda %s sin valor: faltaba «%s»", round, MsgPokerSelectValue)
		}
	}
	if sp := f.storyPoints(t, "Pagos"); sp != 0 {
		t.Errorf("«Pagos» no debía cambiar, tiene %d Story Points", sp)
	}
}

// T019 (E7)
func TestHU18_RondaSinCerrar(t *testing.T) {
	a, f := newConsensusApp(t)
	resp, body := a.agree(t, a.as(t, f.alfa, f.ana), id64(f.rounds["Reportes 2"]), "5")
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerRoundStillOpen) {
		t.Errorf("faltaba «%s»", MsgPokerRoundStillOpen)
	}
	if sp := f.storyPoints(t, "Reportes"); sp != 3 {
		t.Errorf("«Reportes» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

// T019 (E8)
func TestHU18_NuevaRondaConRondaAbierta(t *testing.T) {
	a, f := newConsensusApp(t)
	resp, body := a.nextRound(t, a.as(t, f.alfa, f.ana), id64(f.rounds["Reportes"]))
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerRoundAlreadyOpen) {
		t.Errorf("faltaba «%s»", MsgPokerRoundAlreadyOpen)
	}
	if n := f.roundCount(t, "Reportes"); n != 2 {
		t.Errorf("«Reportes» debía seguir con dos rondas, tiene %d", n)
	}
}

// T019 (E9)
func TestHU18_RondaQueNoEsLaUltima(t *testing.T) {
	a, f := newConsensusApp(t)
	resp, body := a.agree(t, a.as(t, f.alfa, f.ana), id64(f.rounds["Reportes"]), "5")
	pokerAssertStatus(t, resp, http.StatusConflict)
	if !strings.Contains(body, MsgPokerNotLatestRound) {
		t.Errorf("faltaba «%s»", MsgPokerNotLatestRound)
	}
	if sp := f.storyPoints(t, "Reportes"); sp != 3 {
		t.Errorf("«Reportes» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

// T019 (E10)
func TestHU18_SprintFinalizado(t *testing.T) {
	a, f := newConsensusApp(t)
	ana := a.as(t, f.alfa, f.ana)
	section := closedSection(a.consensusPage(t, ana), f.rounds["Alta"])
	for _, want := range []string{
		`<button type="submit" disabled>Nueva ronda</button>`,
		`<fieldset class="poker-estimate" disabled>`,
		`<button type="submit" disabled>Registrar estimación</button>`,
		LabelPokerFinishedSprint,
	} {
		if !strings.Contains(section, want) {
			t.Errorf("«Alta» debía mostrar %q:\n%s", want, section)
		}
	}
	for name, send := range map[string]func() (*http.Response, string){
		"estimación":  func() (*http.Response, string) { return a.agree(t, ana, id64(f.rounds["Alta"]), "5") },
		"nueva ronda": func() (*http.Response, string) { return a.nextRound(t, ana, id64(f.rounds["Alta"])) },
	} {
		resp, body := send()
		pokerAssertStatus(t, resp, http.StatusConflict)
		if !strings.Contains(body, MsgPokerStoryInFinishedSprint) {
			t.Errorf("%s: faltaba «%s»", name, MsgPokerStoryInFinishedSprint)
		}
	}
	if sp, n := f.storyPoints(t, "Alta"), f.roundCount(t, "Alta"); sp != 3 || n != 1 {
		t.Errorf("«Alta» debía seguir con 3 Story Points y una ronda: %d SP, %d rondas", sp, n)
	}
}

// T019 (E11)
func TestHU18_RondaInexistente(t *testing.T) {
	a, f := newConsensusApp(t)
	ana := a.as(t, f.alfa, f.ana)
	for _, round := range []string{"999999999", "", "abc", "05", id64(f.rounds["Tablero"])} {
		for name, send := range map[string]func() (*http.Response, string){
			"estimación":  func() (*http.Response, string) { return a.agree(t, ana, round, "5") },
			"nueva ronda": func() (*http.Response, string) { return a.nextRound(t, ana, round) },
		} {
			resp, body := send()
			pokerAssertStatus(t, resp, http.StatusNotFound)
			if !strings.Contains(body, MsgPokerRoundNotFound) {
				t.Errorf("%s desde %q: faltaba «%s»", name, round, MsgPokerRoundNotFound)
			}
		}
	}
	if sp, n := f.storyPoints(t, "Tablero"), f.roundCount(t, "Tablero"); sp != 0 || n != 1 {
		t.Errorf("«Tablero» no debía cambiar: %d SP, %d rondas", sp, n)
	}
}

// T019 (E12)
func TestHU18_ErrorAlGuardar(t *testing.T) {
	a, f := newConsensusApp(t, pokerOption{store: func(s poker.Store) poker.Store { return failingPokerStore{s} }})
	ana := a.as(t, f.alfa, f.ana)
	for name, send := range map[string]func() (*http.Response, string){
		"estimación":  func() (*http.Response, string) { return a.agree(t, ana, id64(f.rounds["Pagos"]), "5") },
		"nueva ronda": func() (*http.Response, string) { return a.nextRound(t, ana, id64(f.rounds["Login"])) },
	} {
		resp, body := send()
		pokerAssertStatus(t, resp, http.StatusInternalServerError)
		if !strings.Contains(body, MsgOperationFailed) {
			t.Errorf("%s: faltaba «%s»", name, MsgOperationFailed)
		}
		for _, leak := range []string{"poker_", "SQL", "pgx", errPokerStoreInternal.Error(), "Estimación registrada", "Ronda 2 abierta"} {
			if strings.Contains(body, leak) {
				t.Errorf("%s: la respuesta no debía contener %q", name, leak)
			}
		}
	}
	if sp, n := f.storyPoints(t, "Pagos"), f.roundCount(t, "Login"); sp != 0 || n != 1 {
		t.Errorf("no debía cambiar nada: «Pagos» %d SP, «Login» %d rondas", sp, n)
	}
}

// T019 (FR-005)
func TestHU18_SinIdentificarme(t *testing.T) {
	a, f := newConsensusApp(t)
	c := a.as(t, f.alfa, 0)
	for name, send := range map[string]func() (*http.Response, string){
		"estimación":  func() (*http.Response, string) { return a.agree(t, c, id64(f.rounds["Pagos"]), "5") },
		"nueva ronda": func() (*http.Response, string) { return a.nextRound(t, c, id64(f.rounds["Login"])) },
	} {
		resp, _ := send()
		pokerAssertStatus(t, resp, http.StatusSeeOther)
		if loc := resp.Header.Get("Location"); loc != "/contexto/integrante?volver_a=%2Fplanning-poker" {
			t.Errorf("%s: se esperaba pedir identificarse, se obtuvo %q", name, loc)
		}
	}
	if sp, n := f.storyPoints(t, "Pagos"), f.roundCount(t, "Login"); sp != 0 || n != 1 {
		t.Errorf("no debía cambiar nada: «Pagos» %d SP, «Login» %d rondas", sp, n)
	}
}

// T019 (FR-007)
func TestHU18_RecargaSinReenviar(t *testing.T) {
	a, f := newConsensusApp(t)
	ana := a.as(t, f.alfa, f.ana)
	a.agree(t, ana, id64(f.rounds["Pagos"]), "5")
	a.nextRound(t, ana, id64(f.rounds["Login"]))
	_, first := a.get(t, ana, "/planning-poker")
	_, second := a.get(t, ana, "/planning-poker")
	if !strings.Contains(first, "Ronda 2 abierta para la historia «Login»") {
		t.Error("el primer GET debía mostrar el mensaje de la ronda abierta")
	}
	for _, msg := range []string{"Ronda 2 abierta para la historia «Login»", "Estimación registrada"} {
		if strings.Contains(second, msg) {
			t.Errorf("al recargar, «%s» no debía repetirse", msg)
		}
	}
	if sp, n := f.storyPoints(t, "Pagos"), f.roundCount(t, "Login"); sp != 5 || n != 2 {
		t.Errorf("recargar no debía repetir las acciones: «Pagos» %d SP, «Login» %d rondas", sp, n)
	}
}

// T019 (FR-019)
func TestHU18_TituloEscapado(t *testing.T) {
	a, f := newConsensusApp(t)
	const title = `<script>alert("x")</script>`
	f.story(t, f.alfa, title, "Pendiente", 0)
	round := f.closedRound(t, f.alfa, title, 1, consensusBase.Add(time.Hour), map[int64]int{f.ana: 3})
	ana := a.as(t, f.alfa, f.ana)
	const escaped = `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`

	a.agree(t, ana, id64(round), "3")
	_, estimate := a.get(t, ana, "/planning-poker")
	a.nextRound(t, ana, id64(round))
	_, next := a.get(t, ana, "/planning-poker")
	for msg, body := range map[string]string{
		"Estimación registrada: «" + escaped + "» queda con 3 Story Points": estimate,
		"Ronda 2 abierta para la historia «" + escaped + "»":                next,
	} {
		if strings.Contains(body, title) {
			t.Error("el título con <script> se insertó sin escapar")
		}
		if !strings.Contains(body, msg) {
			t.Errorf("faltaba el mensaje escapado %q", msg)
		}
	}
}

// T019 (SC-005): 20 rondas abiertas, 50 cerradas y 10 integrantes en menos de 500 ms.
func TestHU18_Rendimiento(t *testing.T) {
	a, f := newConsensusApp(t)
	members := []int64{f.ana, f.beto, f.caro}
	for i := len(members); i < 10; i++ {
		members = append(members, f.addMember(t, fmt.Sprintf("Integrante %d", i)))
	}
	for i := 0; i < 19; i++ { // más «Reportes 2» = 20 abiertas
		title := fmt.Sprintf("Abierta %d", i)
		f.story(t, f.alfa, title, "Pendiente", 0)
		round := f.openRound(t, f.alfa, title, 1, f.ana)
		for j, m := range members[:i%10+1] {
			f.setVote(t, round, m, []int{1, 2, 3, 5, 8}[j%5])
		}
	}
	for i := 0; i < 46; i++ { // más Login, Pagos, Reportes 1 y Alta = 50 cerradas
		title := fmt.Sprintf("Cerrada %d", i)
		f.story(t, f.alfa, title, "Pendiente", 0)
		votes := map[int64]int{}
		for j, m := range members {
			votes[m] = []int{1, 3, 8, 21}[(i+j)%4]
		}
		f.closedRound(t, f.alfa, title, 1, consensusBase.Add(time.Duration(i)*time.Second), votes)
	}
	c := a.as(t, f.alfa, f.ana)
	a.get(t, c, "/planning-poker") // calienta conexiones y plantillas
	start := time.Now()
	resp, body := a.get(t, c, "/planning-poker")
	elapsed := time.Since(start)
	pokerAssertStatus(t, resp, http.StatusOK)
	if n := strings.Count(body, `class="poker-round"`); n != 20 {
		t.Errorf("se esperaban 20 rondas abiertas, se vieron %d", n)
	}
	if n := strings.Count(body, `data-ronda-cerrada=`); n != 50 {
		t.Errorf("se esperaban 50 rondas cerradas, se vieron %d", n)
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("GET /planning-poker tardó %v (límite 500 ms)", elapsed)
	}
	t.Logf("GET /planning-poker con 20 rondas abiertas, 50 cerradas y 10 integrantes: %v", elapsed)
}
