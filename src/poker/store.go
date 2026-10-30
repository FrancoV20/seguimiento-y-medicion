package poker

import "context"

// Store es el puerto de escritura de Planning Poker. Las implementaciones aplican CheckOpen,
// CheckVote y CheckClose dentro de su transacción y devuelven los errores del dominio tal cual.
type Store interface {
	OpenRound(ctx context.Context, projectID, storyID, memberID int64) (OpenedRound, error)
	CastVote(ctx context.Context, projectID, roundID, memberID int64, value int) (VoteOutcome, error)
	CloseRound(ctx context.Context, projectID, roundID int64) (ClosedRound, error)
}
