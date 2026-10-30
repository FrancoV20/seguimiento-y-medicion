package sprint

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// Ajuste 4 de la revisión de HU-15: sprint_story_results debe usar los mismos tipos que las
// columnas que referencia (BIGINT) y solo aceptar los cuatro estados de una historia.

func TestSprintStoryResultsSchema_ClavesSonBigint(t *testing.T) {
	db := openStatusTestDB(t)

	for _, column := range []string{"sprint_id", "story_id"} {
		var dataType string
		if err := db.QueryRow(`
			SELECT data_type
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'sprint_story_results'
			  AND column_name = $1`,
			column,
		).Scan(&dataType); err != nil {
			t.Fatalf("no se pudo leer el tipo de sprint_story_results.%s: %v", column, err)
		}
		if dataType != "bigint" {
			t.Errorf("sprint_story_results.%s es %q, se esperaba \"bigint\" como la columna que referencia", column, dataType)
		}
	}
}

func TestSprintStoryResultsSchema_FinalStatusSoloAceptaLosCuatroEstados(t *testing.T) {
	db := openStatusTestDB(t)
	projectID := newStatusProject(t, db, "Proyecto esquema sprint_story_results HU-15")
	sprintID := newStatusSprint(t, db, projectID, SprintActive)
	storyID := newStatusStory(t, db, projectID, EnSprint, sprintID)
	ctx := context.Background()

	// Cada inserción se hace en una transacción que se revierte: la prueba no deja filas.
	insert := func(status string) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("no se pudo abrir la transacción: %v", err)
		}
		defer func() { _ = tx.Rollback() }()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
			VALUES ($1, $2, $3, 5)`,
			sprintID, storyID, status,
		)
		return err
	}

	for _, status := range []StoryStatus{Pendiente, EnSprint, InProgress, Done} {
		if err := insert(string(status)); err != nil {
			t.Errorf("final_status %q debe aceptarse: %v", status, err)
		}
	}

	for _, invalid := range []string{"Cualquiera", "hecho", "", "Finalizado"} {
		err := insert(invalid)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Errorf("final_status %q debe rechazarse por un CHECK (código 23514), se obtuvo: %v", invalid, err)
		}
	}
}
