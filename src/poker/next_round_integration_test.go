package poker

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// HU-18, T008: "Nueva ronda" contra PostgreSQL con los datos del Issue.

func TestPostgresOpenNextRound_OK(t *testing.T) {
	f := newConsensusFixture(t)
	store := NewPostgreSQLStore(f.db)
	ctx := context.Background()

	opened, err := store.OpenNextRound(ctx, f.alfa, f.rounds["Login"], f.beto)
	if err != nil {
		t.Fatalf("OpenNextRound devolvió error: %v", err)
	}
	if opened.Number != 2 || opened.StoryID != f.login || opened.StoryTitle != "Login" || opened.RoundID == 0 {
		t.Errorf("ronda abierta inesperada: %+v", opened)
	}
	rounds := f.storyRounds(t, f.login)
	if len(rounds) != 2 {
		t.Fatalf("«Login» debía quedar con dos rondas, tiene %d", len(rounds))
	}
	if r := rounds[1]; r.ID != opened.RoundID || r.Number != 2 || r.Status != "Abierta" || r.OpenedBy != f.beto || r.Votes != 0 {
		t.Errorf("la ronda 2 debía estar abierta por Beto y sin votos: %+v", r)
	}
	var project int64
	if err := f.db.QueryRow(`SELECT project_id FROM poker_rounds WHERE id = $1`, opened.RoundID).Scan(&project); err != nil || project != f.alfa {
		t.Errorf("la ronda 2 debía ser de «Alfa»: %d, %v", project, err)
	}
	if r := rounds[0]; r.Status != "Cerrada" ||
		f.vote(t, r.ID, f.ana) != 2 || f.vote(t, r.ID, f.beto) != 13 || f.vote(t, r.ID, f.caro) != 5 {
		t.Errorf("la ronda 1 debía seguir cerrada con Ana 2, Beto 13 y Caro 5: %+v", r)
	}

	f.closeRound(t, opened.RoundID)
	third, err := store.OpenNextRound(ctx, f.alfa, opened.RoundID, f.ana)
	if err != nil || third.Number != 3 {
		t.Errorf("desde la ronda 2 cerrada se esperaba la ronda 3, se obtuvo %+v, %v", third, err)
	}
}

func TestPostgresOpenNextRound_NoEsLaUltima(t *testing.T) {
	f := newConsensusFixture(t)
	store := NewPostgreSQLStore(f.db)
	second, err := store.OpenNextRound(context.Background(), f.alfa, f.rounds["Login"], f.beto)
	if err != nil {
		t.Fatalf("no se pudo abrir la ronda 2: %v", err)
	}
	f.closeRound(t, second.RoundID)

	if _, err := store.OpenNextRound(context.Background(), f.alfa, f.rounds["Login"], f.beto); !errors.Is(err, ErrNotLatestRound) {
		t.Errorf("desde la ronda 1 con la 2 cerrada se esperaba ErrNotLatestRound, se obtuvo %v", err)
	}
	if n := len(f.storyRounds(t, f.login)); n != 2 {
		t.Errorf("«Login» debía seguir con dos rondas, tiene %d", n)
	}
}

func TestPostgresOpenNextRound_ConRondaAbierta(t *testing.T) {
	f := newConsensusFixture(t)
	store := NewPostgreSQLStore(f.db)
	for _, from := range []string{"Reportes", "Reportes 2"} {
		if _, err := store.OpenNextRound(context.Background(), f.alfa, f.rounds[from], f.beto); !errors.Is(err, ErrRoundAlreadyOpen) {
			t.Errorf("desde «%s» se esperaba ErrRoundAlreadyOpen, se obtuvo %v", from, err)
		}
	}
	if n := len(f.storyRounds(t, f.reportes)); n != 2 {
		t.Errorf("«Reportes» debía seguir con dos rondas, tiene %d", n)
	}
}

func TestPostgresOpenNextRound_SprintFinalizado(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).OpenNextRound(context.Background(), f.alfa, f.rounds["Alta"], f.beto); !errors.Is(err, ErrStoryInFinishedSprint) {
		t.Errorf("se esperaba ErrStoryInFinishedSprint, se obtuvo %v", err)
	}
	if n := len(f.storyRounds(t, f.alta)); n != 1 {
		t.Errorf("«Alta» debía seguir con una ronda, tiene %d", n)
	}
}

func TestPostgresOpenNextRound_Inexistente(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).OpenNextRound(context.Background(), f.alfa, 2147483647, f.beto); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("se esperaba ErrRoundNotFound, se obtuvo %v", err)
	}
}

func TestPostgresOpenNextRound_OtroProyecto(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).OpenNextRound(context.Background(), f.alfa, f.rounds["Tablero"], f.beto); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("la ronda de «Tablero» (otro proyecto) debía tratarse como inexistente, se obtuvo %v", err)
	}
	if n := len(f.storyRounds(t, f.tablero)); n != 1 {
		t.Errorf("«Tablero» debía seguir con una ronda, tiene %d", n)
	}
}

// Dos "Nueva ronda" a la vez desde la ronda 1 de «Login»: una abre la ronda 2 y la otra recibe
// ErrRoundAlreadyOpen.
func TestPostgresOpenNextRound_Concurrente(t *testing.T) {
	f := newConsensusFixture(t)
	store := NewPostgreSQLStore(f.db)
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, member := range []int64{f.ana, f.beto} {
		wg.Add(1)
		go func(i int, member int64) {
			defer wg.Done()
			<-start
			_, errs[i] = store.OpenNextRound(context.Background(), f.alfa, f.rounds["Login"], member)
		}(i, member)
	}
	close(start)
	wg.Wait()

	ok, open := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrRoundAlreadyOpen):
			open++
		default:
			t.Errorf("error inesperado: %v", err)
		}
	}
	if ok != 1 || open != 1 {
		t.Errorf("se esperaba un éxito y un ErrRoundAlreadyOpen, se obtuvo %v", errs)
	}
	if n := len(f.storyRounds(t, f.login)); n != 2 {
		t.Errorf("«Login» debía quedar con dos rondas, tiene %d", n)
	}
}

// Un integrante que no existe hace fallar el INSERT: el error no es del dominio y no queda nada.
func TestPostgresOpenNextRound_FalloNoDejaCambios(t *testing.T) {
	f := newConsensusFixture(t)
	_, err := NewPostgreSQLStore(f.db).OpenNextRound(context.Background(), f.alfa, f.rounds["Login"], 2147483647)
	if err == nil {
		t.Fatal("se esperaba un error al guardar")
	}
	for _, domain := range []error{ErrRoundNotFound, ErrRoundAlreadyOpen, ErrNotLatestRound, ErrStoryInFinishedSprint} {
		if errors.Is(err, domain) {
			t.Errorf("la falla no debía confundirse con %v: %v", domain, err)
		}
	}
	if n := len(f.storyRounds(t, f.login)); n != 1 {
		t.Errorf("«Login» debía seguir con una ronda, tiene %d", n)
	}
}
