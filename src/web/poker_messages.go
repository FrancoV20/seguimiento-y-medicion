package web

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
)

// pokerRoundOpenedMessage es el mensaje de éxito al abrir una ronda.
func pokerRoundOpenedMessage(title string) string {
	return "Ronda abierta para la historia «" + title + "»"
}
