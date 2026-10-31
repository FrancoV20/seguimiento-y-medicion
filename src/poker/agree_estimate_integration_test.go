package poker

import (
	"context"
	"errors"
	"testing"
	"time"
)

// HU-18, T012-T013: estimación acordada contra PostgreSQL con los datos del Issue.

func TestPostgresAgreeEstimate_OK(t *testing.T) {
	f := newConsensusFixture(t)
	store := NewPostgreSQLStore(f.db)
	ctx := context.Background()
	before := f.storyState(t, f.pagos)

	agreed, err := store.AgreeEstimate(ctx, f.alfa, f.rounds["Pagos"], 5)
	if err != nil {
		t.Fatalf("AgreeEstimate devolvió error: %v", err)
	}
	if agreed != (AgreedEstimate{StoryID: f.pagos, StoryTitle: "Pagos", StoryPoints: 5}) {
		t.Errorf("estimación inesperada: %+v", agreed)
	}
	if sp := f.storyPoints(t, f.pagos); sp != 5 {
		t.Errorf("«Pagos» debía quedar con 5 Story Points, tiene %d", sp)
	}
	for _, value := range []int{8, 8} { // reemplazar desde la misma ronda; el mismo valor otra vez también es éxito
		if _, err := store.AgreeEstimate(ctx, f.alfa, f.rounds["Pagos"], value); err != nil {
			t.Fatalf("registrar %d otra vez devolvió error: %v", value, err)
		}
	}
	if sp := f.storyPoints(t, f.pagos); sp != 8 {
		t.Errorf("«Pagos» debía quedar con 8 Story Points, tiene %d", sp)
	}
	if after := f.storyState(t, f.pagos); after != before {
		t.Errorf("solo debían cambiar los Story Points: antes %+v, después %+v", before, after)
	}
}

func TestPostgresAgreeEstimate_ConDivergencia(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, f.rounds["Login"], 5); err != nil {
		t.Fatalf("después de debatir la divergencia se esperaba registrar 5, se obtuvo %v", err)
	}
	if sp := f.storyPoints(t, f.login); sp != 5 {
		t.Errorf("«Login» debía quedar con 5 Story Points, tiene %d", sp)
	}
}

func TestPostgresAgreeEstimate_RondaAbierta(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, f.rounds["Reportes 2"], 5); !errors.Is(err, ErrRoundStillOpen) {
		t.Errorf("se esperaba ErrRoundStillOpen, se obtuvo %v", err)
	}
	if sp := f.storyPoints(t, f.reportes); sp != 3 {
		t.Errorf("«Reportes» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

func TestPostgresAgreeEstimate_NoEsLaUltima(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, f.rounds["Reportes"], 5); !errors.Is(err, ErrNotLatestRound) {
		t.Errorf("se esperaba ErrNotLatestRound, se obtuvo %v", err)
	}
	if sp := f.storyPoints(t, f.reportes); sp != 3 {
		t.Errorf("«Reportes» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

func TestPostgresAgreeEstimate_SprintFinalizado(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, f.rounds["Alta"], 5); !errors.Is(err, ErrStoryInFinishedSprint) {
		t.Errorf("se esperaba ErrStoryInFinishedSprint, se obtuvo %v", err)
	}
	if sp := f.storyPoints(t, f.alta); sp != 3 {
		t.Errorf("«Alta» debía seguir con 3 Story Points, tiene %d", sp)
	}
}

func TestPostgresAgreeEstimate_SprintActivo(t *testing.T) {
	f := newConsensusFixture(t)
	f.assignToActiveSprint(t, f.pagos, "En progreso")
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, f.rounds["Pagos"], 5); err != nil {
		t.Fatalf("una historia de un Sprint activo debía aceptar la estimación, se obtuvo %v", err)
	}
	if sp := f.storyPoints(t, f.pagos); sp != 5 {
		t.Errorf("«Pagos» debía quedar con 5 Story Points, tiene %d", sp)
	}
}

func TestPostgresAgreeEstimate_Inexistente(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, 2147483647, 5); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("se esperaba ErrRoundNotFound, se obtuvo %v", err)
	}
}

func TestPostgresAgreeEstimate_OtroProyecto(t *testing.T) {
	f := newConsensusFixture(t)
	if _, err := NewPostgreSQLStore(f.db).AgreeEstimate(context.Background(), f.alfa, f.rounds["Tablero"], 5); !errors.Is(err, ErrRoundNotFound) {
		t.Errorf("la ronda de «Tablero» (otro proyecto) debía tratarse como inexistente, se obtuvo %v", err)
	}
	if sp := f.storyPoints(t, f.tablero); sp != 0 {
		t.Errorf("«Tablero» no debía cambiar, tiene %d Story Points", sp)
	}
}

// Una transacción de prueba retiene «Pagos»; la estimación con un plazo de 200 ms falla con un
// error que no es del dominio y no deja cambios.
func TestPostgresAgreeEstimate_FalloNoDejaCambios(t *testing.T) {
	f := newConsensusFixture(t)
	hold, err := f.db.Begin()
	if err != nil {
		t.Fatalf("no se pudo iniciar la transacción de prueba: %v", err)
	}
	defer hold.Rollback()
	if _, err := hold.Exec(`SELECT id FROM user_stories WHERE id = $1 FOR SHARE`, f.pagos); err != nil {
		t.Fatalf("no se pudo retener «Pagos»: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = NewPostgreSQLStore(f.db).AgreeEstimate(ctx, f.alfa, f.rounds["Pagos"], 5)
	if err == nil {
		t.Fatal("se esperaba un error al guardar")
	}
	for _, domain := range []error{ErrRoundNotFound, ErrRoundStillOpen, ErrNotLatestRound, ErrStoryInFinishedSprint} {
		if errors.Is(err, domain) {
			t.Errorf("la falla no debía confundirse con %v: %v", domain, err)
		}
	}
	_ = hold.Rollback()
	if sp := f.storyPoints(t, f.pagos); sp != 0 {
		t.Errorf("«Pagos» debía seguir sin Story Points, tiene %d", sp)
	}
}

// T013: el cierre de Sprint (HU-15) llega primero; la estimación lo espera en el Sprint.
func TestPostgresAgreeEstimate_CierreDeSprintPrimero(t *testing.T) {
	cases := []struct {
		name   string
		status string // estado de «Pagos» en el Sprint 2 al cerrarlo
		want   error  // resultado de la estimación
		points int    // Story Points de «Pagos» al final
	}{
		{"«Pagos» Hecho queda en el Sprint finalizado", "Hecho", ErrStoryInFinishedSprint, 0},
		{"«Pagos» En progreso vuelve al Backlog", "En progreso", nil, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newConsensusFixture(t)
			f.assignToActiveSprint(t, f.pagos, c.status)
			ctx := context.Background()

			closing, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("no se pudo iniciar el cierre: %v", err)
			}
			defer closing.Rollback()
			closeSprintLikeHU15(t, ctx, closing, f.sprint2)

			done := make(chan error, 1)
			go func() {
				_, err := NewPostgreSQLStore(f.db).AgreeEstimate(ctx, f.alfa, f.rounds["Pagos"], 5)
				done <- err
			}()
			f.waitForWaiters(t, 1) // la estimación espera el Sprint
			if err := closing.Commit(); err != nil {
				t.Fatalf("no se pudo confirmar el cierre: %v", err)
			}

			select {
			case err := <-done:
				if !errors.Is(err, c.want) || (c.want == nil && err != nil) {
					t.Errorf("se esperaba %v, se obtuvo %v", c.want, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("la estimación no terminó después del cierre")
			}
			if sp := f.storyPoints(t, f.pagos); sp != c.points {
				t.Errorf("«Pagos» debía quedar con %d Story Points, tiene %d", c.points, sp)
			}
		})
	}
}

// T013: la estimación llega primero; el cierre la espera en el Sprint y ninguno se bloquea con el
// otro (sin 40P01).
func TestPostgresAgreeEstimate_EstimacionPrimero(t *testing.T) {
	f := newConsensusFixture(t)
	f.assignToActiveSprint(t, f.pagos, "Hecho")
	ctx := context.Background()

	hold, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("no se pudo iniciar la transacción de prueba: %v", err)
	}
	defer hold.Rollback()
	if _, err := hold.Exec(`SELECT id FROM user_stories WHERE id = $1 FOR SHARE`, f.pagos); err != nil {
		t.Fatalf("no se pudo retener «Pagos»: %v", err)
	}

	estimate := make(chan error, 1)
	go func() {
		_, err := NewPostgreSQLStore(f.db).AgreeEstimate(ctx, f.alfa, f.rounds["Pagos"], 5)
		estimate <- err
	}()
	f.waitForWaiters(t, 1) // la estimación tomó la ronda y el Sprint, y espera la historia

	closed := make(chan error, 1)
	go func() {
		tx, err := f.db.BeginTx(ctx, nil)
		if err != nil {
			closed <- err
			return
		}
		defer tx.Rollback()
		closeSprintLikeHU15(t, ctx, tx, f.sprint2)
		closed <- tx.Commit()
	}()
	f.waitForWaiters(t, 2) // el cierre espera el Sprint
	if err := hold.Rollback(); err != nil {
		t.Fatalf("no se pudo liberar «Pagos»: %v", err)
	}

	for name, ch := range map[string]chan error{"estimación": estimate, "cierre": closed} {
		select {
		case err := <-ch:
			if err != nil {
				t.Errorf("%s: se esperaba éxito, se obtuvo %v", name, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: no terminó", name)
		}
	}
	if sp := f.storyPoints(t, f.pagos); sp != 5 {
		t.Errorf("«Pagos» debía quedar con 5 Story Points, tiene %d", sp)
	}
	var status string
	if err := f.db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, f.sprint2).Scan(&status); err != nil || status != "Finalizado" {
		t.Errorf("el Sprint 2 debía quedar Finalizado: %q, %v", status, err)
	}
}

// T013: "Nueva ronda" y la estimación desde la misma ronda se serializan: con el INSERT de la
// ronda 2 frenado, la estimación espera la ronda y, al abrirse la 2, recibe ErrNotLatestRound.
func TestPostgresNextRoundYEstimacion_Serializadas(t *testing.T) {
	f := newConsensusFixture(t)
	ctx := context.Background()
	store := NewPostgreSQLStore(f.db)

	hold, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("no se pudo iniciar la transacción de prueba: %v", err)
	}
	defer hold.Rollback()
	if _, err := hold.Exec(`LOCK TABLE poker_rounds IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		t.Fatalf("no se pudo frenar los INSERT de rondas: %v", err)
	}

	next := make(chan error, 1)
	go func() {
		_, err := store.OpenNextRound(ctx, f.alfa, f.rounds["Pagos"], f.beto)
		next <- err
	}()
	f.waitForWaiters(t, 1) // "Nueva ronda" tiene la ronda 1 FOR SHARE y espera en el INSERT

	agree := make(chan error, 1)
	go func() {
		_, err := store.AgreeEstimate(ctx, f.alfa, f.rounds["Pagos"], 5)
		agree <- err
	}()
	f.waitForWaiters(t, 2) // la estimación espera la ronda 1
	if err := hold.Rollback(); err != nil {
		t.Fatalf("no se pudo liberar la tabla: %v", err)
	}

	select {
	case err := <-next:
		if err != nil {
			t.Errorf("la ronda 2 debía abrirse, se obtuvo %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("\"Nueva ronda\" no terminó")
	}
	select {
	case err := <-agree:
		if !errors.Is(err, ErrNotLatestRound) {
			t.Errorf("la estimación desde la ronda 1 debía recibir ErrNotLatestRound, se obtuvo %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("la estimación no terminó")
	}
	if sp := f.storyPoints(t, f.pagos); sp != 0 {
		t.Errorf("«Pagos» debía seguir sin Story Points, tiene %d", sp)
	}
	if n := len(f.storyRounds(t, f.pagos)); n != 2 {
		t.Errorf("«Pagos» debía quedar con dos rondas, tiene %d", n)
	}
}
