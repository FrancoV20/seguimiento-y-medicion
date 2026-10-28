// Package poker es el dominio de Planning Poker (HU-05a): rondas de votación de una historia y
// votos ocultos de los integrantes del proyecto.
package poker

import "errors"

// Errores del dominio. Los textos que ve la persona se definen en la pantalla (poker_messages.go).
var (
	ErrValueRequired       = errors.New("poker: falta el valor del voto")
	ErrValueNotInSeries    = errors.New("poker: el valor no pertenece a la serie")
	ErrRoundClosed         = errors.New("poker: la ronda está cerrada")
	ErrRoundNotFound       = errors.New("poker: la ronda no existe")
	ErrStoryNotFound       = errors.New("poker: la historia no existe")
	ErrRoundAlreadyOpen    = errors.New("poker: la historia ya tiene una ronda abierta")
	ErrStoryAlreadyRounded = errors.New("poker: la historia ya tuvo una ronda")
	ErrMemberRequired      = errors.New("poker: falta el integrante")
)
