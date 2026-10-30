package web

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// HU-05b, T017: lo que se ve en "Rondas cerradas" (servidor de newPokerApp con los datos de «Alfa»;
// las rondas se preparan cerradas con closeAt).

const pokerWarningAlert = `<div class="alert alert-warning" role="status"><strong class="alert-label">Advertencia:</strong>`

// newAlfaApp arma el servidor de HU-05a con los datos de «Alfa».
func newAlfaApp(t *testing.T, opts ...pokerOption) (*pokerApp, *alfaWebFixture) {
	t.Helper()
	a := newPokerApp(t, opts...)
	return a, newAlfaWebFixture(t, a.fx.db)
}

// page devuelve la pantalla de Planning Poker del integrante, sin mensajes pendientes.
func (a *pokerApp) page(t *testing.T, f *alfaWebFixture, member int64) string {
	t.Helper()
	c := a.as(t, f.alfa, member)
	a.get(t, c, "/planning-poker") // consume cualquier mensaje de una sola vista
	resp, body := a.get(t, c, "/planning-poker")
	pokerAssertStatus(t, resp, http.StatusOK)
	return body
}

// T017 (E2, E4)
func TestHU05b_AlertaDeDivergencia(t *testing.T) {
	a, f := newAlfaApp(t)
	f.closeAt(t, f.rounds["Login"], alfaBase)
	f.closeAt(t, f.rounds["Reportes"], alfaBase.Add(time.Minute))
	body := a.page(t, f, f.ana)
	for title, text := range map[string]string{
		"Login":    "Divergencia: los votos van de 2 a 13. Debatan antes de acordar la estimación",
		"Reportes": "Divergencia: los votos van de 3 a 13. Debatan antes de acordar la estimación",
	} {
		section := closedSection(body, f.rounds[title])
		if !strings.Contains(section, pokerWarningAlert) || !strings.Contains(section, text) {
			t.Errorf("«%s»: se esperaba la alerta con «Advertencia:» y «%s»:\n%s", title, text, section)
		}
	}
}

// T017 (E3, E5)
func TestHU05b_SinDivergenciaEnElLimite(t *testing.T) {
	a, f := newAlfaApp(t)
	f.closeAt(t, f.rounds["Pagos"], alfaBase)
	f.closeAt(t, f.rounds["Perfil"], alfaBase.Add(time.Minute))
	body := a.page(t, f, f.ana)

	pagos := closedSection(body, f.rounds["Pagos"])
	if !strings.Contains(pagos, "Sin divergencia: los votos van de 3 a 8") {
		t.Errorf("«Pagos»: falta «Sin divergencia: los votos van de 3 a 8»:\n%s", pagos)
	}
	if strings.Contains(pagos, "alert-warning") || strings.Contains(pagos, "Divergencia:") {
		t.Errorf("«Pagos» está en el límite de dos niveles: no debía mostrar la alerta:\n%s", pagos)
	}
	if row := closedRow(t, body, f.rounds["Perfil"], f.caro); !strings.Contains(row, "No votó") {
		t.Errorf("«Perfil»: Caro debía figurar «No votó», se obtuvo %q", row)
	}
	if perfil := closedSection(body, f.rounds["Perfil"]); !strings.Contains(perfil, "Sin divergencia: los votos van de 5 a 5") {
		t.Errorf("«Perfil»: falta «Sin divergencia: los votos van de 5 a 5»:\n%s", perfil)
	}
}

// T017 (E7, FR-010)
func TestHU05b_VotosParaTodos(t *testing.T) {
	a, f := newAlfaApp(t)
	f.closeAt(t, f.rounds["Login"], alfaBase)
	ana := closedRoundsSection(t, a.page(t, f, f.ana))
	beto := closedRoundsSection(t, a.page(t, f, f.beto))
	if ana != beto {
		t.Errorf("Ana y Beto debían ver la misma sección «Rondas cerradas»:\nAna:\n%s\nBeto:\n%s", ana, beto)
	}
	if !strings.Contains(ana, "Divergencia: los votos van de 2 a 13") {
		t.Error("la sección debía mostrar la alerta de «Login»")
	}
}

// T017 (FR-011, SC-004): con «Login» cerrada y «Pagos» abierta, el voto de Ana en «Pagos» no aparece
// en la página de Beto, que no cambia ni un byte cuando Ana cambia ese voto.
func TestHU05b_VotoAbiertoNuncaSeRevela(t *testing.T) {
	a, f := newAlfaApp(t)
	f.closeAt(t, f.rounds["Login"], alfaBase)
	pagos := f.rounds["Pagos"]

	before := a.page(t, f, f.beto)
	if row := roundRow(t, before, pagos, f.ana); !strings.Contains(row, LabelPokerVoted) || strings.Contains(row, "3") {
		t.Errorf("la fila de Ana en «Pagos» debía decir «Votó» sin el valor: %q", row)
	}
	if closedSection(before, f.rounds["Login"]) == "" {
		t.Fatal("«Login» debía estar entre las rondas cerradas")
	}

	resp, _ := a.vote(t, a.as(t, f.alfa, f.ana), pagos, "21")
	pokerAssertRedirect(t, resp)
	if got := f.vote(t, pagos, f.ana); got != 21 {
		t.Fatalf("el voto de Ana debía cambiar a 21, quedó %d", got)
	}
	if after := a.page(t, f, f.beto); after != before {
		t.Error("la página de Beto cambió cuando Ana cambió su voto de una ronda abierta: depende del valor ajeno")
	}
}

// T017 (FR-007)
func TestHU05b_SinRondasCerradas(t *testing.T) {
	a, f := newAlfaApp(t)
	body := a.page(t, f, f.ana)
	section := closedRoundsSection(t, body)
	if !strings.Contains(section, "Rondas cerradas") || !strings.Contains(section, "Aún no hay rondas cerradas") {
		t.Errorf("sin rondas cerradas debía verse «Aún no hay rondas cerradas»:\n%s", section)
	}
	if strings.Contains(body, "data-ronda-cerrada") {
		t.Error("no debía mostrarse ninguna ronda cerrada")
	}
}

// T017 (caso límite): una ronda cerrada sin votos (solo posible con datos externos) no rompe la
// pantalla: todos "No votó", sin rango ni alerta.
func TestHU05b_RondaCerradaSinVotos(t *testing.T) {
	a, f := newAlfaApp(t)
	ayuda := f.rounds["Ayuda"]
	f.closeAt(t, ayuda, alfaBase)
	body := a.page(t, f, f.ana)
	for _, m := range []int64{f.ana, f.beto, f.caro} {
		if row := closedRow(t, body, ayuda, m); !strings.Contains(row, "No votó") {
			t.Errorf("el integrante %d debía figurar «No votó», se obtuvo %q", m, row)
		}
	}
	if section := closedSection(body, ayuda); strings.Contains(section, "los votos van de") || strings.Contains(section, "alert") {
		t.Errorf("una ronda sin votos no debía mostrar rango ni alerta:\n%s", section)
	}
}

// T017 (FR-017): título e integrante con <script> se muestran escapados en "Rondas cerradas" y en
// el mensaje de cierre.
func TestHU05b_TextosEscapados(t *testing.T) {
	a, f := newAlfaApp(t)
	const title, name = `<script>alert("t")</script>`, `<script>alert("n")</script>`
	const titleEsc, nameEsc = `&lt;script&gt;alert(&#34;t&#34;)&lt;/script&gt;`, `&lt;script&gt;alert(&#34;n&#34;)&lt;/script&gt;`
	round := f.openRound(t, f.alfa, title, f.ana)
	member := f.addMember(t, name)
	f.setVote(t, round, member, 5)

	c := a.as(t, f.alfa, f.ana)
	a.get(t, c, "/planning-poker")
	resp, _ := a.closeRound(t, c, round)
	pokerAssertRedirect(t, resp)
	_, body := a.get(t, c, "/planning-poker")

	if strings.Contains(body, title) || strings.Contains(body, name) {
		t.Error("un texto con <script> se insertó sin escapar")
	}
	if !strings.Contains(body, "Ronda cerrada para la historia «"+titleEsc+"»") {
		t.Error("el mensaje de cierre debía mostrar el título escapado")
	}
	section := closedSection(body, round)
	if !strings.Contains(section, titleEsc) || !strings.Contains(section, nameEsc) {
		t.Errorf("«Rondas cerradas» debía mostrar el título y el integrante escapados:\n%s", section)
	}
}

// T017 (SC-006): 20 rondas abiertas, 50 cerradas y 10 integrantes en menos de 500 ms.
func TestHU05b_Rendimiento(t *testing.T) {
	a, f := newAlfaApp(t)
	members := []int64{f.ana, f.beto, f.caro}
	for i := len(members); i < 10; i++ {
		members = append(members, f.addMember(t, fmt.Sprintf("Integrante %02d", i)))
	}
	for i := 0; i < 15; i++ { // «Alfa» ya tiene 5 rondas abiertas
		round := f.openRound(t, f.alfa, fmt.Sprintf("Abierta %02d", i), f.ana)
		for j, m := range members[:i%10+1] {
			f.setVote(t, round, m, poker.Series[j%len(poker.Series)])
		}
	}
	for i := 0; i < 50; i++ {
		round := f.openRound(t, f.alfa, fmt.Sprintf("Cerrada %02d", i), f.ana)
		for j, m := range members {
			f.setVote(t, round, m, poker.Series[(i+j)%len(poker.Series)])
		}
		f.closeAt(t, round, alfaBase.Add(time.Duration(i)*time.Minute))
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
