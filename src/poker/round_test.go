package poker

import (
	"errors"
	"testing"
)

// HU-05a, T008: reglas puras de apertura y voto.

func TestCheckOpen_SinRondas(t *testing.T) {
	if err := CheckOpen(nil); err != nil {
		t.Errorf("sin rondas se podía abrir, se obtuvo %v", err)
	}
}

func TestCheckOpen_Abierta(t *testing.T) {
	for _, existing := range [][]RoundStatus{{StatusOpen}, {StatusClosed, StatusOpen}} {
		if err := CheckOpen(existing); !errors.Is(err, ErrRoundAlreadyOpen) {
			t.Errorf("%v: se esperaba ErrRoundAlreadyOpen, se obtuvo %v", existing, err)
		}
	}
}

func TestCheckOpen_Cerrada(t *testing.T) {
	if err := CheckOpen([]RoundStatus{StatusClosed}); !errors.Is(err, ErrStoryAlreadyRounded) {
		t.Errorf("se esperaba ErrStoryAlreadyRounded, se obtuvo %v", err)
	}
}

func TestCheckVote_Abierta(t *testing.T) {
	if err := CheckVote(StatusOpen); err != nil {
		t.Errorf("en una ronda abierta se podía votar, se obtuvo %v", err)
	}
}

func TestCheckVote_Cerrada(t *testing.T) {
	if err := CheckVote(StatusClosed); !errors.Is(err, ErrRoundClosed) {
		t.Errorf("se esperaba ErrRoundClosed, se obtuvo %v", err)
	}
}

// HU-05b, T007: regla pura de cierre. Una ronda cerrada se rechaza antes de mirar sus votos.
func TestCheckClose(t *testing.T) {
	cases := []struct {
		name   string
		status RoundStatus
		votes  int
		want   error
	}{
		{"abierta con votos", StatusOpen, 3, nil},
		{"cerrada con votos", StatusClosed, 3, ErrRoundClosed},
		{"cerrada sin votos: cerrada antes que sin votos", StatusClosed, 0, ErrRoundClosed},
		{"abierta sin votos", StatusOpen, 0, ErrNoVotes},
	}
	for _, c := range cases {
		err := CheckClose(c.status, c.votes)
		if c.want == nil && err != nil || c.want != nil && !errors.Is(err, c.want) {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.name, c.want, err)
		}
	}
}
