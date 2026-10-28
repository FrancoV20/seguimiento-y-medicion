package web

import "context"

// Modelo de vista de Planning Poker (HU-05a). Ningún tipo guarda el valor del voto de otro
// integrante: el único valor es OwnVote, el del integrante identificado.

// PokerStoryOption es una historia del selector "Abrir ronda".
type PokerStoryOption struct {
	ID     int64
	Title  string
	Status string
}

// PokerParticipantView es un integrante en una ronda abierta: solo si votó, nunca qué votó.
type PokerParticipantView struct {
	MemberID int64
	Name     string
	HasVoted bool
	IsMe     bool
}

// PokerRoundView es una ronda abierta del proyecto.
type PokerRoundView struct {
	ID, StoryID  int64
	StoryTitle   string
	Number       int
	Participants []PokerParticipantView
	OwnVote      *int // voto del integrante identificado; nil si no votó
}

// PokerBoard es todo lo que muestra la pantalla.
type PokerBoard struct {
	Stories []PokerStoryOption
	Rounds  []PokerRoundView
}

// PokerCatalog es la lectura de la pantalla, filtrada por el proyecto del contexto.
type PokerCatalog interface {
	PokerBoard(ctx context.Context, projectID, memberID int64) (PokerBoard, error)
}
