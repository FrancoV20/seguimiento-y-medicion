package sprint

import (
	"context"
	"database/sql"
	"fmt"
)

func formatSprintID(number int64) (string, error) {
	if number <= 0 {
		return "", ErrInvalidSprintNumber
	}
	return fmt.Sprintf("Sprint %d", number), nil
}

func nextSprintNumber(ctx context.Context, tx *sql.Tx, projectID int64) (int64, error) {
	var number int64
	err := tx.QueryRowContext(ctx,
		`SELECT next_sprint_number FROM projects WHERE id = $1 FOR UPDATE`,
		projectID,
	).Scan(&number)
	if err == sql.ErrNoRows {
		return 0, ErrProjectNotFound
	}
	if err != nil {
		return 0, persistenceError("leer la secuencia del proyecto")
	}

	var highestExisting int64
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(number), 0) FROM sprints WHERE project_id = $1`,
		projectID,
	).Scan(&highestExisting)
	if err != nil {
		return 0, persistenceError("consultar los Sprints existentes")
	}
	if number <= highestExisting {
		number = highestExisting + 1
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE projects SET next_sprint_number = $1 WHERE id = $2`,
		number+1,
		projectID,
	); err != nil {
		return 0, persistenceError("actualizar la secuencia del proyecto")
	}
	return number, nil
}
