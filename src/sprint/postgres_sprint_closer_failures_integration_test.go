package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

// Ajuste 3 de la revisión de HU-15: ramas de error de PostgreSQLSprintCloser. Cada falla tiene que
// revertir todo el cierre.

func TestPostgreSQLSprintCloserRechazaArgumentosInvalidos(t *testing.T) {
	ctx := context.Background()

	t.Run("sin conexión", func(t *testing.T) {
		var nilCloser *PostgreSQLSprintCloser
		if _, err := nilCloser.CloseSprint(ctx, 1, allowClosure); err == nil {
			t.Error("CloseSprint() con closer nulo debía fallar")
		}
		if _, err := NewPostgreSQLSprintCloser(nil).CloseSprint(ctx, 1, allowClosure); err == nil {
			t.Error("CloseSprint() sin conexión debía fallar")
		}
		if _, err := NewPostgreSQLSprintCloser(nil).FindSprint(ctx, 1, 1); !errors.Is(err, ErrPersistence) {
			t.Errorf("FindSprint() sin conexión = %v, se esperaba ErrPersistence", err)
		}
	})

	db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, InProgress)
	closer := NewPostgreSQLSprintCloser(db)

	t.Run("sin validación", func(t *testing.T) {
		if _, err := closer.CloseSprint(ctx, sprintID, nil); err == nil {
			t.Fatal("CloseSprint() sin validación debía fallar")
		}
		assertCloseRolledBack(t, db, sprintID, stories)
	})

	for _, invalidID := range []int64{0, -1, 9_000_000_000} {
		t.Run(fmt.Sprintf("Sprint %d", invalidID), func(t *testing.T) {
			checked := false
			_, err := closer.CloseSprint(ctx, invalidID, func(context.Context, []ClosingStory) error {
				checked = true
				return nil
			})
			if !errors.Is(err, ErrSprintNotFound) {
				t.Errorf("error = %v, se esperaba ErrSprintNotFound", err)
			}
			if checked {
				t.Error("no debía validar historias de un Sprint inexistente")
			}
		})
	}

	t.Run("contexto cancelado", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := closer.CloseSprint(canceled, sprintID, allowClosure); err == nil {
			t.Fatal("CloseSprint() con contexto cancelado debía fallar")
		}
		assertCloseRolledBack(t, db, sprintID, stories)
	})
}

func TestPostgreSQLSprintCloserSprintFinalizadoNoSeValida(t *testing.T) {
	db := openStatusTestDB(t)
	projectID := newStatusProject(t, db, "Proyecto Sprint finalizado HU-15")
	sprintID := newStatusSprint(t, db, projectID, SprintFinished)

	checked := false
	_, err := NewPostgreSQLSprintCloser(db).CloseSprint(context.Background(), sprintID,
		func(context.Context, []ClosingStory) error {
			checked = true
			return nil
		})

	if !errors.Is(err, ErrSprintAlreadyClosed) {
		t.Errorf("error = %v, se esperaba ErrSprintAlreadyClosed", err)
	}
	if checked {
		t.Error("no debía validar historias de un Sprint finalizado")
	}
}

func TestPostgreSQLSprintCloserEntregaALaValidacionLasHistoriasBloqueadas(t *testing.T) {
	db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, InProgress)
	// La primera historia tiene dos registros de 2 horas; la segunda no tiene horas.
	addStatusEffort(t, db, stories[0].id)
	addStatusEffort(t, db, stories[0].id)

	var received []ClosingStory
	_, err := NewPostgreSQLSprintCloser(db).CloseSprint(context.Background(), sprintID,
		func(_ context.Context, locked []ClosingStory) error {
			received = locked
			return nil
		})
	if err != nil {
		t.Fatalf("CloseSprint() devolvió error: %v", err)
	}

	if len(received) != len(stories) {
		t.Fatalf("la validación recibió %d historias, se esperaban %d", len(received), len(stories))
	}
	for i, story := range stories {
		got := received[i]
		if got.ID != story.id || got.Status != story.status {
			t.Errorf("historia %d recibida = id %d estado %q; se esperaba id %d estado %q",
				i, got.ID, got.Status, story.id, story.status)
		}
		if got.StoryPoints == nil || *got.StoryPoints != story.storyPoints {
			t.Errorf("Story Points de la historia %d = %v; se esperaba %d", story.id, got.StoryPoints, story.storyPoints)
		}
	}
	if received[0].ActualHours == nil || received[0].ActualHours.Cmp(big.NewRat(4, 1)) != 0 {
		t.Errorf("horas de la historia %d = %v; se esperaban 4", stories[0].id, received[0].ActualHours)
	}
	if received[1].ActualHours != nil {
		t.Errorf("la historia %d no tiene horas y se recibieron %v", stories[1].id, received[1].ActualHours)
	}
}

func TestPostgreSQLSprintCloserValidacionQueFallaRevierteElCierre(t *testing.T) {
	db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, InProgress)
	blocked := errors.New("validación de prueba rechazada")

	_, err := NewPostgreSQLSprintCloser(db).CloseSprint(context.Background(), sprintID,
		func(context.Context, []ClosingStory) error { return blocked })

	if !errors.Is(err, blocked) {
		t.Fatalf("error = %v, se esperaba el error de la validación", err)
	}
	assertCloseRolledBack(t, db, sprintID, stories)
}

func TestPostgreSQLSprintCloserFallaEnCadaPasoRevierteTodo(t *testing.T) {
	cases := []struct {
		name    string
		table   string
		event   string
		when    string
		wantErr string
	}{
		{
			name:    "registro del Sprint",
			table:   "sprint_story_results",
			event:   "INSERT",
			when:    "NEW.sprint_id = %d",
			wantErr: "record sprint",
		},
		{
			name:    "quitar asignaciones",
			table:   "sprint_story_assignments",
			event:   "DELETE",
			when:    "OLD.sprint_id = %d",
			wantErr: "remove incomplete story assignments",
		},
		{
			name:    "devolver historias",
			table:   "user_stories",
			event:   "UPDATE",
			when:    "OLD.active_sprint_id = %d AND NEW.active_sprint_id IS NULL",
			wantErr: "return incomplete stories",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, InProgress)
			injectCloseFailure(t, db, tc.table, tc.event, fmt.Sprintf(tc.when, sprintID), sprintID)

			_, err := NewPostgreSQLSprintCloser(db).CloseSprint(context.Background(), sprintID, allowClosure)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, se esperaba una falla en %q", err, tc.wantErr)
			}
			assertCloseRolledBack(t, db, sprintID, stories)
		})
	}
}

func TestPostgreSQLSprintCloserFallaAlBloquear(t *testing.T) {
	cases := []struct {
		name    string
		lock    string
		wantErr string
	}{
		{
			name:    "Sprint bloqueado por otra transacción",
			lock:    `SELECT id FROM sprints WHERE id = $1 FOR UPDATE`,
			wantErr: "lock sprint",
		},
		{
			name:    "historia bloqueada por otra transacción",
			lock:    `SELECT id FROM user_stories WHERE active_sprint_id = $1 FOR UPDATE`,
			wantErr: "lock stories",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, _, sprintID, stories := newSprintCloserIntegrationFixture(t, Done, InProgress)
			ctx := context.Background()

			holder, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("no se pudo abrir la transacción que bloquea: %v", err)
			}
			defer func() { _ = holder.Rollback() }()
			if _, err := holder.ExecContext(ctx, tc.lock, sprintID); err != nil {
				t.Fatalf("no se pudo tomar el bloqueo: %v", err)
			}

			_, err = NewPostgreSQLSprintCloser(openLockTimeoutDB(t)).CloseSprint(ctx, sprintID, allowClosure)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, se esperaba una falla en %q", err, tc.wantErr)
			}
			_ = holder.Rollback()
			assertCloseRolledBack(t, db, sprintID, stories)
		})
	}
}

// openLockTimeoutDB abre otra conexión a la base de prueba en la que esperar un bloqueo falla
// enseguida, para simular un error al bloquear sin esperar indefinidamente.
func openLockTimeoutDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := testDatabaseURL(t)
	switch {
	case !strings.Contains(dsn, "://"):
		dsn += " lock_timeout=200ms"
	case strings.Contains(dsn, "?"):
		dsn += "&lock_timeout=200ms"
	default:
		dsn += "?lock_timeout=200ms"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión con lock_timeout: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// injectCloseFailure crea un trigger que hace fallar un paso del cierre de ese Sprint y lo quita
// al terminar la prueba (antes de la limpieza de las fixtures).
func injectCloseFailure(t *testing.T, db *sql.DB, table, event, when string, sprintID int64) {
	t.Helper()
	suffix := fmt.Sprintf("%s_%s_%d", strings.ToLower(event), table, sprintID)
	functionName := "hu15_fail_fn_" + suffix
	triggerName := "hu15_fail_" + suffix

	if _, err := db.Exec(fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger
		LANGUAGE plpgsql
		AS $body$
		BEGIN
			RAISE EXCEPTION 'injected HU-15 failure on %s %s';
		END;
		$body$`,
		functionName, event, table,
	)); err != nil {
		t.Fatalf("no se pudo crear la función de fallo controlado: %v", err)
	}
	if _, err := db.Exec(fmt.Sprintf(`
		CREATE TRIGGER %s
		BEFORE %s ON %s
		FOR EACH ROW
		WHEN (%s)
		EXECUTE FUNCTION %s()`,
		triggerName, event, table, when, functionName,
	)); err != nil {
		_, _ = db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName))
		t.Fatalf("no se pudo crear el trigger de fallo controlado: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s`, triggerName, table)); err != nil {
			t.Errorf("no se pudo quitar el trigger de fallo controlado: %v", err)
		}
		if _, err := db.Exec(fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName)); err != nil {
			t.Errorf("no se pudo quitar la función de fallo controlado: %v", err)
		}
	})
}

// assertCloseRolledBack comprueba que un cierre fallido no dejó cambios.
func assertCloseRolledBack(t *testing.T, db *sql.DB, sprintID int64, stories []sprintCloserTestStory) {
	t.Helper()
	var sprintStatus string
	if err := db.QueryRow(`SELECT status FROM sprints WHERE id = $1`, sprintID).Scan(&sprintStatus); err != nil {
		t.Fatalf("no se pudo leer el Sprint %d: %v", sprintID, err)
	}
	if sprintStatus != string(SprintActive) {
		t.Errorf("estado del Sprint = %q, se esperaba que siguiera %q", sprintStatus, SprintActive)
	}
	var results int
	if err := db.QueryRow(
		`SELECT count(*) FROM sprint_story_results WHERE sprint_id = $1`, sprintID,
	).Scan(&results); err != nil {
		t.Fatalf("no se pudo leer sprint_story_results: %v", err)
	}
	if results != 0 {
		t.Errorf("quedaron %d filas en sprint_story_results después de la falla", results)
	}
	for _, story := range stories {
		var (
			status       string
			activeSprint sql.NullInt64
			assignments  int
		)
		if err := db.QueryRow(`
			SELECT us.status, us.active_sprint_id,
			       (SELECT count(*) FROM sprint_story_assignments a WHERE a.story_id = us.id)
			FROM user_stories us
			WHERE us.id = $1`,
			story.id,
		).Scan(&status, &activeSprint, &assignments); err != nil {
			t.Fatalf("no se pudo leer la historia %d: %v", story.id, err)
		}
		if status != string(story.status) || !activeSprint.Valid || activeSprint.Int64 != sprintID || assignments != 1 {
			t.Errorf("la historia %d cambió pese a la falla: estado=%q sprint=%v asignaciones=%d",
				story.id, status, activeSprint, assignments)
		}
	}
}
