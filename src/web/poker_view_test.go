package web

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
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
	anaRow, betoRow := participantRow(t, before, a.fx.ana), participantRow(t, before, a.fx.beto)
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
