package web

import (
	"context"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

const pathPoker = "/planning-poker"

// PokerActions es lo que la pantalla necesita del dominio; lo cumple *poker.Service.
type PokerActions interface {
	OpenRound(ctx context.Context, projectID, storyID, memberID int64) (poker.OpenedRound, error)
	Vote(ctx context.Context, projectID, roundID, memberID int64, rawValue string) (poker.VoteOutcome, error)
}

var _ PokerActions = (*poker.Service)(nil)

type pokerHandlers struct {
	server  *Server
	actions PokerActions
	catalog PokerCatalog
}

// pokerRoundPage es una ronda lista para la plantilla: el voto propio sin puntero.
type pokerRoundPage struct {
	PokerRoundView
	HasOwnVote bool
	OwnValue   int
}

// pokerPageData son los datos de poker.html.
type pokerPageData struct {
	Board        PokerBoard
	Rounds       []pokerRoundPage
	Series       []int
	NoStories    string
	NoOpenRounds string
	Voted        string
	NotVoted     string
}

// RegisterPlanningPoker registra la sección "Planning Poker" (HU-05a). Exige proyecto activo e
// integrante identificado: sin ellos, HU-20b redirige antes de llegar a los handlers.
func RegisterPlanningPoker(s *Server, actions PokerActions, catalog PokerCatalog) error {
	h := &pokerHandlers{server: s, actions: actions, catalog: catalog}
	return s.Register(
		Section{Name: TitlePoker, Path: pathPoker, RequiresProject: true, RequiresMember: true},
		Routes{
			"GET " + pathPoker:             http.HandlerFunc(h.show),
			"POST " + pathPoker + "/abrir": http.HandlerFunc(h.open),
			"POST " + pathPoker + "/votar": http.HandlerFunc(h.vote),
		},
	)
}

// show muestra las historias y las rondas abiertas del proyecto activo.
func (h *pokerHandlers) show(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, nil)
}

// open abre la ronda 1 de una historia a nombre del integrante identificado; un campo
// "integrante" enviado se ignora.
func (h *pokerHandlers) open(w http.ResponseWriter, r *http.Request) {
	ac := ActiveContextFrom(r.Context())
	story, _, _ := pokerFormIDs(r)
	round, err := h.actions.OpenRound(r.Context(), ac.ProjectID, story, ac.MemberID)
	if err != nil {
		h.reject(w, r, err)
		return
	}
	h.done(w, r, pokerRoundOpenedMessage(round.StoryTitle))
}

// vote registra o reemplaza el voto del integrante identificado; un campo "integrante"
// enviado se ignora.
func (h *pokerHandlers) vote(w http.ResponseWriter, r *http.Request) {
	ac := ActiveContextFrom(r.Context())
	_, round, value := pokerFormIDs(r)
	outcome, err := h.actions.Vote(r.Context(), ac.ProjectID, round, ac.MemberID, value)
	if err != nil {
		h.reject(w, r, err)
		return
	}
	message := MsgPokerVoteRegistered
	if outcome.Updated {
		message = MsgPokerVoteUpdated
	}
	h.done(w, r, message)
}

// done deja el mensaje de éxito y vuelve a la pantalla con 303, así recargar no reenvía el
// formulario (E2).
func (h *pokerHandlers) done(w http.ResponseWriter, r *http.Request, message string) {
	SetFlash(w, r, FlashSuccess, message)
	RedirectSeeOther(w, r, pathPoker)
}

// reject muestra la pantalla con el código y el mensaje del error; una falla se registra con su
// detalle y se muestra con el mensaje genérico.
func (h *pokerHandlers) reject(w http.ResponseWriter, r *http.Request, err error) {
	status, text := pokerOutcomeFor(err)
	if status == http.StatusInternalServerError {
		h.server.logger.Error("Planning Poker: no se pudo guardar", "ruta", r.URL.Path, "detalle", err.Error())
	}
	h.render(w, r, status, &Flash{Kind: FlashError, Text: text})
}

// render lee la pantalla del proyecto activo; si la lectura falla, muestra la página de error.
func (h *pokerHandlers) render(w http.ResponseWriter, r *http.Request, status int, flash *Flash) {
	ac := ActiveContextFrom(r.Context())
	board, err := h.catalog.PokerBoard(r.Context(), ac.ProjectID, ac.MemberID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	rounds := make([]pokerRoundPage, 0, len(board.Rounds))
	for _, round := range board.Rounds {
		page := pokerRoundPage{PokerRoundView: round}
		if round.OwnVote != nil {
			page.HasOwnVote, page.OwnValue = true, *round.OwnVote
		}
		rounds = append(rounds, page)
	}
	h.server.renderer.PageStatus(w, r, status, "poker.html", PageData{
		Title: TitlePoker,
		Flash: flash,
		Data: pokerPageData{
			Board: board, Rounds: rounds, Series: poker.Series,
			NoStories: MsgPokerNoStories, NoOpenRounds: MsgPokerNoOpenRounds,
			Voted: LabelPokerVoted, NotVoted: LabelPokerNotVoted,
		},
	})
}
