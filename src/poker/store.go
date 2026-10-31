package poker

import "context"

// Store es el puerto de escritura de Planning Poker. Las implementaciones aplican CheckOpen,
// CheckVote, CheckClose y CheckNextRound dentro de su transacción y devuelven los errores del
// dominio tal cual.
type Store interface {
	OpenRound(ctx context.Context, projectID, storyID, memberID int64) (OpenedRound, error)
	CastVote(ctx context.Context, projectID, roundID, memberID int64, value int) (VoteOutcome, error)
	CloseRound(ctx context.Context, projectID, roundID int64) (ClosedRound, error)
	// OpenNextRound abre la ronda siguiente de la historia desde una de sus rondas (HU-18).
	OpenNextRound(ctx context.Context, projectID, roundID, memberID int64) (OpenedRound, error)
}
