package web

import "context"

// Modelo de vista de Planning Poker (HU-05a). Ningún tipo de las rondas abiertas guarda el valor
// del voto de otro integrante: el único valor es OwnVote, el del integrante identificado. Los
// valores ajenos solo viven en PokerRevealedVoteView, que se llena desde rondas cerradas (HU-05b).

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

// PokerRevealedVoteView es un integrante en una ronda cerrada (HU-05b): su valor ya revelado, o
// nil si no votó.
type PokerRevealedVoteView struct {
	MemberID int64
	Name     string
	Value    *int
}

// PokerClosedRoundView es una ronda cerrada del proyecto con sus votos revelados (HU-05b).
// Latest indica que es la última ronda de su historia e InFinishedSprint que la historia está en
// un Sprint finalizado (HU-18).
type PokerClosedRoundView struct {
	ID, StoryID      int64
	StoryTitle       string
	Number           int
	Votes            []PokerRevealedVoteView
	Latest           bool
	InFinishedSprint bool
}

// PokerBoard es todo lo que muestra la pantalla.
type PokerBoard struct {
	Stories      []PokerStoryOption
	Rounds       []PokerRoundView
	ClosedRounds []PokerClosedRoundView // HU-05b: de la más recientemente cerrada a la más antigua
}

// PokerCatalog es la lectura de la pantalla, filtrada por el proyecto del contexto.
type PokerCatalog interface {
	PokerBoard(ctx context.Context, projectID, memberID int64) (PokerBoard, error)
}
