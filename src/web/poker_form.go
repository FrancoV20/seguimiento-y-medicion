package web

import (
	"errors"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
)

// Campos de los formularios de Planning Poker.
const (
	PokerFieldStory = "historia"
	PokerFieldRound = "ronda"
	PokerFieldValue = "valor"
)

// pokerFormIDs lee los identificadores y el valor enviados. Un identificador vacío, no numérico,
// no canónico o fuera de rango se lee como 0 (el servicio lo trata como inexistente).
func pokerFormIDs(r *http.Request) (story, round int64, value string) {
	_ = r.ParseForm()
	return parseID(r.PostFormValue(PokerFieldStory)), parseID(r.PostFormValue(PokerFieldRound)), r.PostFormValue(PokerFieldValue)
}

// pokerOutcomeFor traduce un error del dominio a su código y su texto (plan.md, decisión 7).
// Cualquier otro error es una falla: 500 con el mensaje genérico.
func pokerOutcomeFor(err error) (status int, text string) {
	switch {
	case errors.Is(err, poker.ErrValueRequired):
		return http.StatusUnprocessableEntity, MsgPokerSelectValue
	case errors.Is(err, poker.ErrValueNotInSeries):
		return http.StatusUnprocessableEntity, MsgPokerValueNotInSeries
	case errors.Is(err, poker.ErrRoundClosed):
		return http.StatusConflict, MsgPokerRoundClosed
	case errors.Is(err, poker.ErrRoundNotFound):
		return http.StatusNotFound, MsgPokerRoundNotFound
	case errors.Is(err, poker.ErrStoryNotFound):
		return http.StatusNotFound, MsgPokerStoryNotFound
	case errors.Is(err, poker.ErrRoundAlreadyOpen):
		return http.StatusConflict, MsgPokerRoundAlreadyOpen
	case errors.Is(err, poker.ErrStoryAlreadyRounded):
		return http.StatusConflict, MsgPokerStoryAlreadyRounded
	default:
		return http.StatusInternalServerError, MsgOperationFailed
	}
}
