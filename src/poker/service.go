package poker

import "context"

// Service aplica las validaciones que no necesitan la base y delega en el almacenamiento.
type Service struct {
	store Store
}

// NewService crea el servicio sobre un almacenamiento.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// OpenRound abre la ronda 1 de una historia del proyecto a nombre del integrante.
func (s *Service) OpenRound(ctx context.Context, projectID, storyID, memberID int64) (OpenedRound, error) {
	if storyID == 0 {
		return OpenedRound{}, ErrStoryNotFound
	}
	if memberID == 0 {
		return OpenedRound{}, ErrMemberRequired
	}
	return s.store.OpenRound(ctx, projectID, storyID, memberID)
}

// Vote registra o reemplaza el voto del integrante. El valor se valida primero (422 antes que
// 404/409): un voto inválido nunca llega al almacenamiento.
func (s *Service) Vote(ctx context.Context, projectID, roundID, memberID int64, rawValue string) (VoteOutcome, error) {
	value, err := ParseValue(rawValue)
	if err != nil {
		return VoteOutcome{}, err
	}
	if roundID == 0 {
		return VoteOutcome{}, ErrRoundNotFound
	}
	if memberID == 0 {
		return VoteOutcome{}, ErrMemberRequired
	}
	return s.store.CastVote(ctx, projectID, roundID, memberID, value)
}
