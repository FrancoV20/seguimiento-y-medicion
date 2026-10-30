package poker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// HU-05b, T009-T010: PostgreSQLStore.CloseRound contra PostgreSQL real, con los datos de «Alfa».

// domainErrors son todos los errores del dominio de Planning Poker.
var domainErrors = []error{ErrValueRequired, ErrValueNotInSeries, ErrRoundClosed, ErrRoundNotFound, ErrStoryNotFound,
	ErrRoundAlreadyOpen, ErrStoryAlreadyRounded, ErrMemberRequired, ErrNoVotes}

func isDomainError(err error) bool {
	for _, d := range domainErrors {
		if errors.Is(err, d) {
			return true
		}
	}
	return false
}

// storyRow resume la historia para comprobar que el cierre no la modifica.
func (f *alfaFixture) storyRow(t *testing.T, story int64) string {
	t.Helper()
	var s string
	if err := f.db.QueryRow(`SELECT title || '|' || status || '|' || COALESCE(story_points::text, '') FROM user_stories WHERE id = $1`, story).Scan(&s); err != nil {
		t.Fatalf("no se pudo leer la historia: %v", err)
	}
	return s
}

// T009 (E1)
func TestPostgresCloseRound_OK(t *testing.T) {
	f := newAlfaFixture(t)
	round := f.rounds["Login"]
	storyBefore := f.storyRow(t, f.login)

	got, err := NewPostgreSQLStore(f.db).CloseRound(context.Background(), f.alfa, round)
	if err != nil {
		t.Fatalf("CloseRound devolvió error: %v", err)
	}
	want := ClosedRound{RoundID: round, StoryID: f.login, StoryTitle: "Login", Number: 1}
	if got != want {
		t.Errorf("se esperaba %+v, se obtuvo %+v", want, got)
	}
	if s := f.status(t, round); s != "Cerrada" {
		t.Errorf("la ronda debía quedar 'Cerrada', quedó %q", s)
	}
	if !f.closedAt(t, round).Valid {
		t.Error("la ronda cerrada debía tener fecha de cierre")
	}
	for member, value := range map[int64]int{f.ana: 2, f.beto: 13, f.caro: 5} {
		if got := f.vote(t, round, member); got != value {
			t.Errorf("el voto del integrante %d debía seguir en %d, quedó %d", member, value, got)
		}
	}
	if after := f.storyRow(t, f.login); after != storyBefore {
		t.Errorf("la historia no debía cambiar: %s → %s", storyBefore, after)
	}
}

// T009 (E5)
func TestPostgresCloseRound_SinTodosLosVotos(t *testing.T) {
	f := newAlfaFixture(t)
	round := f.rounds["Perfil"]
	got, err := NewPostgreSQLStore(f.db).CloseRound(context.Background(), f.alfa, round)
	if err != nil {
		t.Fatalf("una ronda con dos de tres votos debía cerrarse: %v", err)
	}
	if got.StoryTitle != "Perfil" || f.status(t, round) != "Cerrada" || !f.closedAt(t, round).Valid {
		t.Errorf("«Perfil» debía quedar cerrada con fecha de cierre: %+v", got)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round); n != 2 {
		t.Errorf("los dos votos debían conservarse, hay %d", n)
	}
}

// T009 (E6)
func TestPostgresCloseRound_SinVotos(t *testing.T) {
	f := newAlfaFixture(t)
	round := f.rounds["Ayuda"]
	if _, err := NewPostgreSQLStore(f.db).CloseRound(context.Background(), f.alfa, round); !errors.Is(err, ErrNoVotes) {
		t.Errorf("se esperaba ErrNoVotes, se obtuvo %v", err)
	}
	if s := f.status(t, round); s != "Abierta" {
		t.Errorf("«Ayuda» debía seguir 'Abierta', quedó %q", s)
	}
	if f.closedAt(t, round).Valid {
		t.Error("«Ayuda» no debía tener fecha de cierre")
	}
}

// T009 (E9)
func TestPostgresCloseRound_YaCerrada(t *testing.T) {
	f := newAlfaFixture(t)
	round := f.rounds["Login"]
	store := NewPostgreSQLStore(f.db)
	if _, err := store.CloseRound(context.Background(), f.alfa, round); err != nil {
		t.Fatalf("el primer cierre debía funcionar: %v", err)
	}
	first := f.closedAt(t, round)
	time.Sleep(5 * time.Millisecond)
	if _, err := store.CloseRound(context.Background(), f.alfa, round); !errors.Is(err, ErrRoundClosed) {
		t.Errorf("se esperaba ErrRoundClosed, se obtuvo %v", err)
	}
	if second := f.closedAt(t, round); !second.Valid || !second.Time.Equal(first.Time) {
		t.Errorf("la fecha de cierre debía seguir siendo la del primer cierre: %v → %v", first.Time, second.Time)
	}
	if got := f.vote(t, round, f.caro); got != 5 {
		t.Errorf("el voto de Caro debía seguir en 5, quedó %d", got)
	}
}

// T009 (E10)
func TestPostgresCloseRound_Inexistente(t *testing.T) {
	f := newAlfaFixture(t)
	if _, err := NewPostgreSQLStore(f.db).CloseRound(context.Background(), f.alfa, 2147483647); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("se esperaba ErrRoundNotFound, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE project_id = $1 AND status = 'Cerrada'`, f.alfa); n != 0 {
		t.Errorf("no debía cerrarse ninguna ronda de «Alfa», hay %d cerradas", n)
	}
}

// T009 (E10): una ronda de «Beta» es inexistente desde «Alfa», abierta o cerrada sin votos
// (el 404 se decide antes que el 409).
func TestPostgresCloseRound_OtroProyecto(t *testing.T) {
	f := newAlfaFixture(t)
	store := NewPostgreSQLStore(f.db)
	tablero, agenda := f.rounds["Tablero"], f.rounds["Agenda"]
	for name, round := range map[string]int64{"Tablero (abierta)": tablero, "Agenda (cerrada sin votos)": agenda} {
		if _, err := store.CloseRound(context.Background(), f.alfa, round); !errors.Is(err, ErrRoundNotFound) {
			t.Errorf("%s: se esperaba ErrRoundNotFound, se obtuvo %v", name, err)
		}
	}
	if s := f.status(t, tablero); s != "Abierta" || f.closedAt(t, tablero).Valid {
		t.Errorf("«Tablero» debía seguir abierta sin fecha de cierre, quedó %q", s)
	}
	if got := f.vote(t, tablero, f.dani); got != 8 {
		t.Errorf("el voto de Dani debía seguir en 8, quedó %d", got)
	}
	if at := f.closedAt(t, agenda); !at.Valid || !at.Time.Equal(agendaClosedAt) {
		t.Errorf("la fecha de cierre de «Agenda» no debía cambiar: %v", at.Time)
	}
}

// T010 (E8, SC-005): el cierre toma la ronda primero; el voto de Caro espera y, al confirmarse el
// cierre, se rechaza sin cambiar su voto anterior. El LOCK TABLE de la prueba deja al cierre
// detenido en el COUNT con la ronda ya bloqueada, así el orden es determinista.
func TestPostgresCloseRound_VotoEsperaAlCierre(t *testing.T) {
	f := newAlfaFixture(t)
	ctx := context.Background()
	store := NewPostgreSQLStore(f.db)
	round := f.rounds["Login"]

	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`LOCK TABLE poker_votes IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}

	closeDone := make(chan error, 1)
	go func() {
		_, err := store.CloseRound(ctx, f.alfa, round)
		closeDone <- err
	}()
	waitForWaiters(t, f, 1)

	voteDone := make(chan error, 1)
	go func() {
		_, err := store.CastVote(ctx, f.alfa, round, f.caro, 8)
		voteDone <- err
	}()
	waitForWaiters(t, f, 2)

	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := <-closeDone; err != nil {
		t.Errorf("el cierre debía funcionar, se obtuvo %v", err)
	}
	if err := <-voteDone; !errors.Is(err, ErrRoundClosed) {
		t.Errorf("el voto debía rechazarse con ErrRoundClosed, se obtuvo %v", err)
	}
	if got := f.vote(t, round, f.caro); got != 5 {
		t.Errorf("el voto de Caro debía seguir en 5, quedó %d", got)
	}
	if s := f.status(t, round); s != "Cerrada" {
		t.Errorf("«Login» debía quedar 'Cerrada', quedó %q", s)
	}
}

// T010 (E9): dos cierres simultáneos de «Pagos»: uno la cierra y el otro recibe ErrRoundClosed.
func TestPostgresCloseRound_Concurrente(t *testing.T) {
	f := newAlfaFixture(t)
	store := NewPostgreSQLStore(f.db)
	round := f.rounds["Pagos"]
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = store.CloseRound(context.Background(), f.alfa, round)
		}(i)
	}
	close(start)
	wg.Wait()
	ok, closed := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrRoundClosed):
			closed++
		default:
			t.Errorf("error inesperado: %v", err)
		}
	}
	if ok != 1 || closed != 1 {
		t.Errorf("se esperaba un cierre OK y otro ErrRoundClosed, se obtuvo %v", errs)
	}
	if s := f.status(t, round); s != "Cerrada" || !f.closedAt(t, round).Valid {
		t.Errorf("«Pagos» debía quedar cerrada con fecha de cierre, quedó %q", s)
	}
}

// T010 (caso límite): un voto en curso tiene la ronda tomada FOR SHARE; el cierre espera y, al
// confirmarse el voto, lo cuenta: «Ayuda» se cierra con ese voto.
func TestPostgresCloseRound_EsperaAlVotoEnCurso(t *testing.T) {
	f := newAlfaFixture(t)
	ctx := context.Background()
	round := f.rounds["Ayuda"]

	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT 1 FROM poker_rounds WHERE id = $1 FOR SHARE`, round); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, 3)`, round, f.ana); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := NewPostgreSQLStore(f.db).CloseRound(ctx, f.alfa, round)
		done <- err
	}()
	waitForWaiters(t, f, 1)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Errorf("el cierre debía contar el voto confirmado y funcionar, se obtuvo %v", err)
	}
	if s := f.status(t, round); s != "Cerrada" {
		t.Errorf("«Ayuda» debía quedar 'Cerrada', quedó %q", s)
	}
	if got := f.vote(t, round, f.ana); got != 3 {
		t.Errorf("el voto de Ana debía quedar en 3, quedó %d", got)
	}
}

// T010 (E12): el cierre queda esperando un voto en curso y su contexto vence: devuelve un error
// que no es del dominio y la ronda sigue abierta.
func TestPostgresCloseRound_FalloNoDejaCambios(t *testing.T) {
	f := newAlfaFixture(t)
	round := f.rounds["Login"]

	tx, err := f.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT 1 FROM poker_rounds WHERE id = $1 FOR SHARE`, round); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = NewPostgreSQLStore(f.db).CloseRound(ctx, f.alfa, round)
	if err == nil || isDomainError(err) {
		t.Errorf("se esperaba un error de almacenamiento, se obtuvo %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if s := f.status(t, round); s != "Abierta" {
		t.Errorf("«Login» debía seguir 'Abierta', quedó %q", s)
	}
	if f.closedAt(t, round).Valid {
		t.Error("«Login» no debía tener fecha de cierre")
	}
}
