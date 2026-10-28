package poker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// HU-05a, T011-T018: migración 011 y PostgreSQLStore contra PostgreSQL real.

func TestMigracionPoker_Restricciones(t *testing.T) {
	f := newPokerFixture(t)
	open := f.id(t, `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 1, $3) RETURNING id`, f.projectA, f.login, f.ana)
	cases := map[string]string{
		"estado desconocido":             `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by) VALUES ($1, $2, 2, 'Otra', $3)`,
		"cerrada sin fecha de cierre":    `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by) VALUES ($1, $2, 2, 'Cerrada', $3)`,
		"abierta con fecha de cierre":    `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at) VALUES ($1, $2, 2, 'Abierta', $3, now())`,
		"dos abiertas de la historia":    `INSERT INTO poker_rounds (project_id, story_id, round_number, opened_by) VALUES ($1, $2, 2, $3)`,
		"número repetido en la historia": `INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by, closed_at) VALUES ($1, $2, 1, 'Cerrada', $3, now())`,
	}
	for name, q := range cases {
		if _, err := f.db.Exec(q, f.projectA, f.login, f.ana); err == nil {
			t.Errorf("%s: la base debía rechazarlo", name)
		}
	}
	if _, err := f.db.Exec(`INSERT INTO poker_votes (round_id, member_id, value) VALUES ($1, $2, 4)`, open, f.ana); err == nil {
		t.Error("un voto con valor 4 debía rechazarse")
	}
}

func TestPostgresOpenRound_OK(t *testing.T) {
	f := newPokerFixture(t)
	got, err := NewPostgreSQLStore(f.db).OpenRound(context.Background(), f.projectA, f.login, f.ana)
	if err != nil {
		t.Fatalf("OpenRound devolvió error: %v", err)
	}
	if got.RoundID == 0 || got.StoryID != f.login || got.StoryTitle != f.loginTitle || got.Number != 1 {
		t.Errorf("ronda inesperada: %+v", got)
	}
	var number int
	var status string
	var openedBy, project int64
	var closedNull bool
	if err := f.db.QueryRow(`SELECT round_number, status, opened_by, project_id, closed_at IS NULL FROM poker_rounds WHERE id = $1`, got.RoundID).
		Scan(&number, &status, &openedBy, &project, &closedNull); err != nil {
		t.Fatal(err)
	}
	if number != 1 || status != "Abierta" || openedBy != f.ana || project != f.projectA || !closedNull {
		t.Errorf("fila inesperada: %d %s %d %d %v", number, status, openedBy, project, closedNull)
	}
}

func TestPostgresOpenRound_NoModificaHistoria(t *testing.T) {
	f := newPokerFixture(t)
	read := func() string {
		var s string
		_ = f.db.QueryRow(`SELECT title || '|' || status || '|' || COALESCE(story_points::text, '') FROM user_stories WHERE id = $1`, f.pagos).Scan(&s)
		return s
	}
	before := read()
	if _, err := NewPostgreSQLStore(f.db).OpenRound(context.Background(), f.projectA, f.pagos, f.beto); err != nil {
		t.Fatalf("una historia Hecho con Story Points debía poder abrirse: %v", err)
	}
	if after := read(); after != before {
		t.Errorf("la historia no debía cambiar: %s → %s", before, after)
	}
}

func TestPostgresOpenRound_Inexistente(t *testing.T) {
	f := newPokerFixture(t)
	if _, err := NewPostgreSQLStore(f.db).OpenRound(context.Background(), f.projectA, 9_000_000_000, f.ana); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("se esperaba ErrStoryNotFound, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE project_id = $1`, f.projectA); n != 0 {
		t.Errorf("no debía crearse ninguna ronda, hay %d", n)
	}
}

func TestPostgresOpenRound_OtroProyecto(t *testing.T) {
	f := newPokerFixture(t)
	if _, err := NewPostgreSQLStore(f.db).OpenRound(context.Background(), f.projectA, f.storyB, f.ana); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("se esperaba ErrStoryNotFound, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, f.storyB); n != 0 {
		t.Errorf("no debía crearse ninguna ronda, hay %d", n)
	}
}

func TestPostgresOpenRound_ConRondaAbierta(t *testing.T) {
	f := newPokerFixture(t)
	store := NewPostgreSQLStore(f.db)
	if _, err := store.OpenRound(context.Background(), f.projectA, f.login, f.ana); err != nil {
		t.Fatal(err)
	}
	if _, err := store.OpenRound(context.Background(), f.projectA, f.login, f.beto); !errors.Is(err, ErrRoundAlreadyOpen) {
		t.Errorf("se esperaba ErrRoundAlreadyOpen, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, f.login); n != 1 {
		t.Errorf("debía quedar una sola ronda, hay %d", n)
	}
}

func TestPostgresOpenRound_YaTuvoRonda(t *testing.T) {
	f := newPokerFixture(t)
	f.insertClosedRound(t, f.login, f.projectA)
	if _, err := NewPostgreSQLStore(f.db).OpenRound(context.Background(), f.projectA, f.login, f.ana); !errors.Is(err, ErrStoryAlreadyRounded) {
		t.Errorf("se esperaba ErrStoryAlreadyRounded, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, f.login); n != 1 {
		t.Errorf("la cantidad de rondas no debía cambiar, hay %d", n)
	}
}

func TestPostgresOpenRound_Concurrente(t *testing.T) {
	f := newPokerFixture(t)
	store := NewPostgreSQLStore(f.db)
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	for i, member := range []int64{f.ana, f.beto} {
		wg.Add(1)
		go func(i int, member int64) {
			defer wg.Done()
			<-start
			_, errs[i] = store.OpenRound(context.Background(), f.projectA, f.login, member)
		}(i, member)
	}
	close(start)
	wg.Wait()
	ok, already := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrRoundAlreadyOpen):
			already++
		default:
			t.Errorf("error inesperado: %v", err)
		}
	}
	if ok != 1 || already != 1 {
		t.Errorf("se esperaba uno OK y otro ErrRoundAlreadyOpen, se obtuvo %v", errs)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, f.login); n != 1 {
		t.Errorf("debía quedar una sola ronda, hay %d", n)
	}
}

// waitForLock espera hasta que alguna sesión de la base quede esperando un bloqueo.
func waitForLock(t *testing.T, f *pokerFixture) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if f.count(t, `SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("la operación no quedó esperando el bloqueo")
}

// Un cierre en curso (como el de HU-05b, con FOR UPDATE) hace esperar al voto, que termina
// rechazado sin dejar nada guardado.
func TestPostgresCastVote_EsperaAlCierre(t *testing.T) {
	f := newPokerFixture(t)
	ctx := context.Background()
	store := NewPostgreSQLStore(f.db)
	round, err := store.OpenRound(ctx, f.projectA, f.login, f.ana)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT 1 FROM poker_rounds WHERE id = $1 FOR UPDATE`, round.RoundID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`, round.RoundID); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := store.CastVote(ctx, f.projectA, round.RoundID, f.beto, 5)
		done <- err
	}()
	waitForLock(t, f)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, ErrRoundClosed) {
		t.Errorf("se esperaba ErrRoundClosed, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round.RoundID); n != 0 {
		t.Errorf("no debía quedar ningún voto, hay %d", n)
	}
}

func (f *pokerFixture) openRound(t *testing.T) OpenedRound {
	t.Helper()
	round, err := NewPostgreSQLStore(f.db).OpenRound(context.Background(), f.projectA, f.login, f.ana)
	if err != nil {
		t.Fatalf("no se pudo abrir la ronda: %v", err)
	}
	return round
}

func TestPostgresCastVote_Nuevo(t *testing.T) {
	f := newPokerFixture(t)
	round := f.openRound(t)
	got, err := NewPostgreSQLStore(f.db).CastVote(context.Background(), f.projectA, round.RoundID, f.ana, 5)
	if err != nil || got.Updated || got.Value != 5 || got.RoundID != round.RoundID {
		t.Fatalf("se esperaba un voto nuevo de 5, se obtuvo %+v, %v", got, err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1 AND member_id = $2 AND value = 5`, round.RoundID, f.ana); n != 1 {
		t.Errorf("se esperaba una fila con 5, hay %d", n)
	}
}

func TestPostgresCastVote_Reemplaza(t *testing.T) {
	f := newPokerFixture(t)
	round := f.openRound(t)
	store := NewPostgreSQLStore(f.db)
	if _, err := store.CastVote(context.Background(), f.projectA, round.RoundID, f.ana, 5); err != nil {
		t.Fatal(err)
	}
	var first time.Time
	_ = f.db.QueryRow(`SELECT updated_at FROM poker_votes WHERE round_id = $1 AND member_id = $2`, round.RoundID, f.ana).Scan(&first)
	time.Sleep(5 * time.Millisecond)
	got, err := store.CastVote(context.Background(), f.projectA, round.RoundID, f.ana, 8)
	if err != nil || !got.Updated || got.Value != 8 {
		t.Fatalf("se esperaba un cambio de voto a 8, se obtuvo %+v, %v", got, err)
	}
	var value int
	var second time.Time
	_ = f.db.QueryRow(`SELECT value, updated_at FROM poker_votes WHERE round_id = $1 AND member_id = $2`, round.RoundID, f.ana).Scan(&value, &second)
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1 AND member_id = $2`, round.RoundID, f.ana); n != 1 || value != 8 || !second.After(first) {
		t.Errorf("se esperaba una sola fila con 8 y fecha posterior: %d filas, valor %d, %v → %v", n, value, first, second)
	}
}

func TestPostgresCastVote_DosIntegrantes(t *testing.T) {
	f := newPokerFixture(t)
	round := f.openRound(t)
	store := NewPostgreSQLStore(f.db)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, m := range []int64{f.ana, f.beto} {
		wg.Add(1)
		go func(i int, m int64) {
			defer wg.Done()
			_, errs[i] = store.CastVote(context.Background(), f.projectA, round.RoundID, m, 3)
		}(i, m)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("los dos votos debían guardarse: %v", errs)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round.RoundID); n != 2 {
		t.Errorf("se esperaban dos votos, hay %d", n)
	}
}

func TestPostgresCastVote_RondaCerrada(t *testing.T) {
	f := newPokerFixture(t)
	closed := f.insertClosedRound(t, f.login, f.projectA)
	if _, err := NewPostgreSQLStore(f.db).CastVote(context.Background(), f.projectA, closed, f.ana, 5); !errors.Is(err, ErrRoundClosed) {
		t.Errorf("se esperaba ErrRoundClosed, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, closed); n != 0 {
		t.Errorf("no debía guardarse ningún voto, hay %d", n)
	}
}

func TestPostgresCastVote_Inexistente(t *testing.T) {
	f := newPokerFixture(t)
	if _, err := NewPostgreSQLStore(f.db).CastVote(context.Background(), f.projectA, 9_000_000_000, f.ana, 5); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("se esperaba ErrRoundNotFound, se obtuvo %v", err)
	}
}

func TestPostgresCastVote_OtroProyecto(t *testing.T) {
	f := newPokerFixture(t)
	round := f.openRound(t)
	if _, err := NewPostgreSQLStore(f.db).CastVote(context.Background(), f.projectB, round.RoundID, f.caro, 5); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("se esperaba ErrRoundNotFound, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round.RoundID); n != 0 {
		t.Errorf("no debía guardarse ningún voto, hay %d", n)
	}
}

// Un integrante que no existe hace fallar la clave foránea: error que no es de dominio y nada guardado.
func TestPostgresCastVote_FalloNoDejaCambios(t *testing.T) {
	f := newPokerFixture(t)
	round := f.openRound(t)
	store := NewPostgreSQLStore(f.db)
	domain := []error{ErrValueRequired, ErrValueNotInSeries, ErrRoundClosed, ErrRoundNotFound, ErrStoryNotFound, ErrRoundAlreadyOpen, ErrStoryAlreadyRounded, ErrMemberRequired}
	isDomain := func(err error) bool {
		for _, d := range domain {
			if errors.Is(err, d) {
				return true
			}
		}
		return false
	}
	_, err := store.CastVote(context.Background(), f.projectA, round.RoundID, 2147483000, 5)
	if err == nil || isDomain(err) {
		t.Errorf("CastVote: se esperaba un error de almacenamiento, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, round.RoundID); n != 0 {
		t.Errorf("no debía quedar ningún voto, hay %d", n)
	}
	_, err = store.OpenRound(context.Background(), f.projectA, f.pagos, 2147483000)
	if err == nil || isDomain(err) {
		t.Errorf("OpenRound: se esperaba un error de almacenamiento, se obtuvo %v", err)
	}
	if n := f.count(t, `SELECT COUNT(*) FROM poker_rounds WHERE story_id = $1`, f.pagos); n != 0 {
		t.Errorf("no debía quedar ninguna ronda, hay %d", n)
	}
}
