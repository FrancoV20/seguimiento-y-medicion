package poker

import (
	"errors"
	"testing"
)

// HU-18, T003: reglas de "Nueva ronda" y de la estimación acordada, con los datos del Issue.

func TestCheckNextRound(t *testing.T) {
	closed := func(numbers ...int) []StoryRound {
		rounds := make([]StoryRound, 0, len(numbers))
		for _, n := range numbers {
			rounds = append(rounds, StoryRound{Number: n, Status: StatusClosed})
		}
		return rounds
	}
	reportes := []StoryRound{{Number: 1, Status: StatusClosed}, {Number: 2, Status: StatusOpen}}
	cases := []struct {
		name     string
		from     int
		rounds   []StoryRound
		finished bool
		next     int
		want     error
	}{
		{"«Login» desde la 1", 1, closed(1), false, 2, nil},
		{"«Login» desde la 2 (después del escenario 3)", 2, closed(1, 2), false, 3, nil},
		{"«Login» desde la 1 con la 2 cerrada", 1, closed(1, 2), false, 0, ErrNotLatestRound},
		{"«Reportes» desde la 1 (escenario 8)", 1, reportes, false, 0, ErrRoundAlreadyOpen},
		{"«Reportes» desde la 2", 2, reportes, false, 0, ErrRoundAlreadyOpen},
		{"«Reportes» desde la 1 con Sprint finalizado: la abierta gana", 1, reportes, true, 0, ErrRoundAlreadyOpen},
		{"«Alta» desde la 1 (Sprint finalizado)", 1, closed(1), true, 0, ErrStoryInFinishedSprint},
		{"no es la última y Sprint finalizado: gana no es la última", 1, closed(1, 2), true, 0, ErrNotLatestRound},
		{"rondas desordenadas", 3, []StoryRound{{3, StatusClosed}, {1, StatusClosed}, {2, StatusClosed}}, false, 4, nil},
	}
	for _, c := range cases {
		next, err := CheckNextRound(c.from, c.rounds, c.finished)
		if c.want == nil {
			if err != nil || next != c.next {
				t.Errorf("%s: se esperaba la ronda %d, se obtuvo %d, %v", c.name, c.next, next, err)
			}
			continue
		}
		if !errors.Is(err, c.want) {
			t.Errorf("%s: se esperaba %v, se obtuvo %d, %v", c.name, c.want, next, err)
		}
	}
}

func TestCheckAgree(t *testing.T) {
	cases := []struct {
		name           string
		status         RoundStatus
		number, latest int
		finished       bool
		want           error
	}{
		{"«Pagos» ronda 1 (escenario 4)", StatusClosed, 1, 1, false, nil},
		{"«Reportes» ronda 2 abierta (escenario 7)", StatusOpen, 2, 2, false, ErrRoundStillOpen},
		{"«Reportes» ronda 1 (escenario 9)", StatusClosed, 1, 2, false, ErrNotLatestRound},
		{"«Alta» ronda 1 (escenario 10)", StatusClosed, 1, 1, true, ErrStoryInFinishedSprint},
		{"abierta, no última y Sprint finalizado: gana abierta", StatusOpen, 1, 2, true, ErrRoundStillOpen},
		{"no última y Sprint finalizado: gana no última", StatusClosed, 1, 2, true, ErrNotLatestRound},
	}
	for _, c := range cases {
		if err := CheckAgree(c.status, c.number, c.latest, c.finished); !errors.Is(err, c.want) || (c.want == nil && err != nil) {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.name, c.want, err)
		}
	}
}
