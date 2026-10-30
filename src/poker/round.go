package poker

// RoundStatus es el estado de una ronda.
type RoundStatus string

const (
	StatusOpen   RoundStatus = "Abierta"
	StatusClosed RoundStatus = "Cerrada"
)

// CheckOpen decide si se puede abrir la ronda 1 de una historia según sus rondas existentes:
// con una abierta, ErrRoundAlreadyOpen; con alguna cerrada, ErrStoryAlreadyRounded (HU-18 se
// ocupa de las rondas siguientes).
func CheckOpen(existing []RoundStatus) error {
	if len(existing) == 0 {
		return nil
	}
	for _, s := range existing {
		if s == StatusOpen {
			return ErrRoundAlreadyOpen
		}
	}
	return ErrStoryAlreadyRounded
}

// CheckVote decide si se puede votar en una ronda con ese estado.
func CheckVote(status RoundStatus) error {
	if status != StatusOpen {
		return ErrRoundClosed
	}
	return nil
}

// CheckClose decide si se puede cerrar una ronda con ese estado y esa cantidad de votos (HU-05b):
// una ronda que no está abierta se rechaza antes de mirar sus votos.
func CheckClose(status RoundStatus, votes int) error {
	if status != StatusOpen {
		return ErrRoundClosed
	}
	if votes == 0 {
		return ErrNoVotes
	}
	return nil
}

// OpenedRound es la ronda recién abierta.
type OpenedRound struct {
	RoundID, StoryID int64
	StoryTitle       string
	Number           int
}

// VoteOutcome es el resultado de un voto: Updated indica que reemplazó uno anterior.
type VoteOutcome struct {
	RoundID int64
	Value   int
	Updated bool
}

// ClosedRound es la ronda recién cerrada (HU-05b).
type ClosedRound struct {
	RoundID, StoryID int64
	StoryTitle       string
	Number           int
}
