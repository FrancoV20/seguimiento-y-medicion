package web

import "fmt"

// Textos de Planning Poker (HU-05a), definidos en un solo lugar.
const (
	TitlePoker                  = "Planning Poker"
	MsgPokerVoteRegistered      = "Voto registrado"
	MsgPokerVoteUpdated         = "Voto actualizado"
	MsgPokerSelectValue         = "Seleccioná un valor de la serie"
	MsgPokerValueNotInSeries    = "El valor no pertenece a la serie de Fibonacci"
	MsgPokerRoundClosed         = "La ronda ya está cerrada"
	MsgPokerRoundNotFound       = "La ronda no existe"
	MsgPokerStoryNotFound       = "La historia no existe"
	MsgPokerRoundAlreadyOpen    = "Ya hay una ronda abierta para esta historia"
	MsgPokerStoryAlreadyRounded = "La historia ya tuvo una ronda de votación"
	MsgPokerNoStories           = "Aún no hay historias"
	MsgPokerNoOpenRounds        = "No hay rondas abiertas"
	LabelPokerVoted             = "Votó"
	LabelPokerNotVoted          = "Sin votar"
	MsgPokerNoVotes             = "La ronda no tiene votos"    // HU-05b
	MsgPokerNoClosedRounds      = "Aún no hay rondas cerradas" // HU-05b
	LabelPokerDidNotVote        = "No votó"                    // HU-05b
)

// pokerRoundOpenedMessage es el mensaje de éxito al abrir una ronda.
func pokerRoundOpenedMessage(title string) string {
	return "Ronda abierta para la historia «" + title + "»"
}

// pokerRoundClosedMessage es el mensaje de éxito al cerrar una ronda (HU-05b).
func pokerRoundClosedMessage(title string) string {
	return "Ronda cerrada para la historia «" + title + "»"
}

// pokerDivergenceMessage es la alerta de una ronda cerrada con divergencia (HU-05b).
func pokerDivergenceMessage(low, high int) string {
	return fmt.Sprintf("Divergencia: los votos van de %d a %d. Debatan antes de acordar la estimación", low, high)
}

// pokerNoDivergenceMessage es el rango de una ronda cerrada sin divergencia (HU-05b).
func pokerNoDivergenceMessage(low, high int) string {
	return fmt.Sprintf("Sin divergencia: los votos van de %d a %d", low, high)
}
