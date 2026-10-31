package web

import (
	"context"
	"testing"
)

// HU-18, T018: la lectura de "Rondas cerradas" indica la última ronda de cada historia y si la
// historia está en un Sprint finalizado.

func consensusClosedRounds(t *testing.T, f *consensusWebFixture) map[int64]PokerClosedRoundView {
	t.Helper()
	board, err := NewPostgreSQLPokerCatalog(f.db).PokerBoard(context.Background(), f.alfa, f.ana)
	if err != nil {
		t.Fatalf("PokerBoard devolvió error: %v", err)
	}
	out := map[int64]PokerClosedRoundView{}
	for _, r := range board.ClosedRounds {
		out[r.ID] = r
	}
	return out
}

func TestPostgresPokerCatalog_UltimaRonda(t *testing.T) {
	f := newConsensusWebFixture(t, openContextTestDB(t))
	closed := consensusClosedRounds(t, f)
	for _, title := range []string{"Login", "Pagos", "Alta"} {
		if r, ok := closed[f.rounds[title]]; !ok || !r.Latest {
			t.Errorf("la ronda 1 de «%s» debía ser la última: %+v", title, r)
		}
	}
	if r := closed[f.rounds["Reportes"]]; r.Latest {
		t.Error("la ronda 1 de «Reportes» no es la última: la 2 está abierta")
	}

	f.openRound(t, f.alfa, "Login", 2, f.beto)
	if r := consensusClosedRounds(t, f)[f.rounds["Login"]]; r.Latest {
		t.Error("abierta la ronda 2 de «Login», su ronda 1 ya no es la última")
	}
}

func TestPostgresPokerCatalog_SprintFinalizado(t *testing.T) {
	f := newConsensusWebFixture(t, openContextTestDB(t))
	f.assignToActiveSprint(t, "Pagos", "En progreso")
	closed := consensusClosedRounds(t, f)
	if !closed[f.rounds["Alta"]].InFinishedSprint {
		t.Error("«Alta» está en el Sprint 1 finalizado")
	}
	for _, title := range []string{"Login", "Pagos"} {
		if closed[f.rounds[title]].InFinishedSprint {
			t.Errorf("«%s» no está en un Sprint finalizado", title)
		}
	}
}
