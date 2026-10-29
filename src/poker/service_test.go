package poker

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// HU-05a, T009: Service con un Store de prueba.

type fakeStore struct {
	openCalls, voteCalls int
	gotOpen              [3]int64
	gotVote              [3]int64
	gotValue             int
	opened               OpenedRound
	outcome              VoteOutcome
	err                  error
	closeCalls           int         // HU-05b
	gotClose             [2]int64    // proyecto y ronda recibidos por CloseRound
	closed               ClosedRound // lo que devuelve CloseRound
}

func (f *fakeStore) OpenRound(_ context.Context, projectID, storyID, memberID int64) (OpenedRound, error) {
	f.openCalls++
	f.gotOpen = [3]int64{projectID, storyID, memberID}
	return f.opened, f.err
}

func (f *fakeStore) CastVote(_ context.Context, projectID, roundID, memberID int64, value int) (VoteOutcome, error) {
	f.voteCalls++
	f.gotVote = [3]int64{projectID, roundID, memberID}
	f.gotValue = value
	return f.outcome, f.err
}

func (f *fakeStore) CloseRound(_ context.Context, projectID, roundID int64) (ClosedRound, error) {
	f.closeCalls++
	f.gotClose = [2]int64{projectID, roundID}
	return f.closed, f.err
}

func TestService_OpenRound_OK(t *testing.T) {
	store := &fakeStore{opened: OpenedRound{RoundID: 9, StoryID: 4, StoryTitle: "Login", Number: 1}}
	got, err := NewService(store).OpenRound(context.Background(), 1, 4, 7)
	if err != nil || got != store.opened {
		t.Fatalf("se esperaba la ronda del almacén, se obtuvo %+v, %v", got, err)
	}
	if store.gotOpen != [3]int64{1, 4, 7} {
		t.Errorf("el almacén recibió %v, se esperaba proyecto 1, historia 4, integrante 7", store.gotOpen)
	}
}

func TestService_OpenRound_Validaciones(t *testing.T) {
	cases := []struct {
		name          string
		story, member int64
		want          error
	}{
		{"historia cero", 0, 7, ErrStoryNotFound},
		{"integrante cero", 4, 0, ErrMemberRequired},
	}
	for _, c := range cases {
		store := &fakeStore{}
		if _, err := NewService(store).OpenRound(context.Background(), 1, c.story, c.member); !errors.Is(err, c.want) {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.name, c.want, err)
		}
		if store.openCalls != 0 {
			t.Errorf("%s: no debía llamarse al almacén", c.name)
		}
	}
}

func TestService_Vote_ValorVacioAntesDelAlmacen(t *testing.T) {
	store := &fakeStore{}
	// Valor vacío sobre una ronda inexistente: primero el valor (422).
	if _, err := NewService(store).Vote(context.Background(), 1, 0, 7, ""); !errors.Is(err, ErrValueRequired) {
		t.Errorf("se esperaba ErrValueRequired, se obtuvo %v", err)
	}
	if _, err := NewService(store).Vote(context.Background(), 1, 3, 7, "4"); !errors.Is(err, ErrValueNotInSeries) {
		t.Errorf("se esperaba ErrValueNotInSeries, se obtuvo %v", err)
	}
	if store.voteCalls != 0 {
		t.Error("no debía llamarse al almacén")
	}
}

func TestService_Vote_RondaYIntegrante(t *testing.T) {
	store := &fakeStore{}
	if _, err := NewService(store).Vote(context.Background(), 1, 0, 7, "5"); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("ronda cero: se esperaba ErrRoundNotFound, se obtuvo %v", err)
	}
	if _, err := NewService(store).Vote(context.Background(), 1, 3, 0, "5"); !errors.Is(err, ErrMemberRequired) {
		t.Errorf("integrante cero: se esperaba ErrMemberRequired, se obtuvo %v", err)
	}
	if store.voteCalls != 0 {
		t.Error("no debía llamarse al almacén")
	}
}

func TestService_Vote_DevuelveUpdatedYPropagaErrores(t *testing.T) {
	store := &fakeStore{outcome: VoteOutcome{RoundID: 3, Value: 8, Updated: true}}
	got, err := NewService(store).Vote(context.Background(), 1, 3, 7, "8")
	if err != nil || !got.Updated || got.Value != 8 {
		t.Fatalf("se esperaba el resultado del almacén, se obtuvo %+v, %v", got, err)
	}
	if store.gotVote != [3]int64{1, 3, 7} || store.gotValue != 8 {
		t.Errorf("el almacén recibió %v y %d", store.gotVote, store.gotValue)
	}
	for _, domainErr := range []error{ErrRoundClosed, ErrRoundNotFound, fmt.Errorf("envuelto: %w", ErrRoundClosed)} {
		store := &fakeStore{err: domainErr}
		if _, err := NewService(store).Vote(context.Background(), 1, 3, 7, "8"); !errors.Is(err, ErrRoundClosed) && !errors.Is(err, ErrRoundNotFound) {
			t.Errorf("el error %v debía poder reconocerse con errors.Is, se obtuvo %v", domainErr, err)
		}
	}
	store = &fakeStore{err: ErrStoryAlreadyRounded}
	if _, err := NewService(store).OpenRound(context.Background(), 1, 4, 7); !errors.Is(err, ErrStoryAlreadyRounded) {
		t.Errorf("OpenRound debía propagar el error del almacén, se obtuvo %v", err)
	}
}

// HU-05b, T007: Service.CloseRound con un Store de prueba.

func TestService_CloseRound_OK(t *testing.T) {
	store := &fakeStore{closed: ClosedRound{RoundID: 9, StoryID: 4, StoryTitle: "Login", Number: 1}}
	got, err := NewService(store).CloseRound(context.Background(), 1, 9, 7)
	if err != nil || got != store.closed {
		t.Fatalf("se esperaba la ronda cerrada del almacén, se obtuvo %+v, %v", got, err)
	}
	if store.closeCalls != 1 || store.gotClose != [2]int64{1, 9} {
		t.Errorf("el almacén recibió %v en %d llamadas, se esperaba proyecto 1 y ronda 9 una vez", store.gotClose, store.closeCalls)
	}
}

func TestService_CloseRound_Validaciones(t *testing.T) {
	cases := []struct {
		name          string
		round, member int64
		want          error
	}{
		{"ronda cero", 0, 7, ErrRoundNotFound},
		{"integrante cero", 9, 0, ErrMemberRequired},
	}
	for _, c := range cases {
		store := &fakeStore{}
		if _, err := NewService(store).CloseRound(context.Background(), 1, c.round, c.member); !errors.Is(err, c.want) {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.name, c.want, err)
		}
		if store.closeCalls != 0 {
			t.Errorf("%s: no debía llamarse al almacén", c.name)
		}
	}

	other := errors.New("falla del almacén")
	for _, storeErr := range []error{ErrNoVotes, ErrRoundClosed, ErrRoundNotFound, other, fmt.Errorf("envuelto: %w", ErrNoVotes)} {
		store := &fakeStore{err: storeErr}
		if _, err := NewService(store).CloseRound(context.Background(), 1, 9, 7); !errors.Is(err, storeErr) {
			t.Errorf("el error %v debía propagarse de forma que errors.Is funcione, se obtuvo %v", storeErr, err)
		}
	}
}
