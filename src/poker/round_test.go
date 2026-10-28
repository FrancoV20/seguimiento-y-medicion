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
