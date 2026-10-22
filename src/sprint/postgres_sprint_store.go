package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLSprintStore implementa SprintStore mediante PostgreSQL.
type PostgreSQLSprintStore struct {
	db *sql.DB
}

var _ SprintStore = (*PostgreSQLSprintStore)(nil)

// NewPostgreSQLSprintStore abre una conexión y comprueba su disponibilidad.
func NewPostgreSQLSprintStore(dsn string) (*PostgreSQLSprintStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, persistenceError("abrir la conexión")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, persistenceError("conectar con la base de datos")
	}
	return &PostgreSQLSprintStore{db: db}, nil
}

// Close cierra la conexión a PostgreSQL.
func (s *PostgreSQLSprintStore) Close() error {
	if err := s.db.Close(); err != nil {
		return persistenceError("cerrar la conexión")
	}
	return nil
}

// CreateSprint persiste el Sprint y sus asociaciones iniciales dentro de una transacción.
func (s *PostgreSQLSprintStore) CreateSprint(ctx context.Context, sprint Sprint) (Sprint, error) {
	if err := validateSprint(sprint); err != nil {
		return Sprint{}, err
	}
	storyIDs, err := normalizeStoryIDs(sprint.StoryIDs)
	if err != nil {
		return Sprint{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Sprint{}, persistenceError("iniciar la creación del Sprint")
	}
	defer tx.Rollback()

	// Un solo Sprint activo: se verifica antes de tomar el número, así un rechazo no lo consume (R3).
	if err := lockProjectWithoutActiveSprint(ctx, tx, sprint.ProjectID); err != nil {
		return Sprint{}, err
	}
	number, err := nextSprintNumber(ctx, tx, sprint.ProjectID)
	if err != nil {
		return Sprint{}, err
	}
	displayID, err := formatSprintID(number)
	if err != nil {
		return Sprint{}, err
	}
	stories, err := lockAssignableStories(ctx, tx, sprint.ProjectID, storyIDs, 0)
	if err != nil {
		return Sprint{}, err
	}

	sprint.Goal = strings.TrimSpace(sprint.Goal)
	sprint.Number = number
	sprint.DisplayID = displayID
	sprint.Status = SprintActive
	sprint.StoryIDs = append([]int64(nil), storyIDs...)
	err = tx.QueryRowContext(ctx, `
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		sprint.ProjectID,
		sprint.Number,
		sprint.Goal,
		sprint.StartDate,
		sprint.EndDate,
		sprint.Status,
	).Scan(&sprint.ID)
	if err != nil {
		return Sprint{}, persistenceError("crear el Sprint")
	}
	if err := persistStoryAssignments(ctx, tx, sprint.ID, stories); err != nil {
		return Sprint{}, err
	}
	if err := tx.Commit(); err != nil {
		return Sprint{}, persistenceError("confirmar la creación del Sprint")
	}
	return sprint, nil
}

// AssignStories asigna las historias disponibles al Sprint en una única transacción.
func (s *PostgreSQLSprintStore) AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error {
	ids, err := normalizeStoryIDs(storyIDs)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return persistenceError("iniciar la asignación de historias")
	}
	defer tx.Rollback()

	var projectID int64
	var status SprintStatus
	err = tx.QueryRowContext(ctx,
		`SELECT project_id, status FROM sprints WHERE id = $1 FOR UPDATE`,
		sprintID,
	).Scan(&projectID, &status)
	if err == sql.ErrNoRows {
		return ErrStoryNotAvailable
	}
	if err != nil {
		return persistenceError("consultar el Sprint para asignación")
	}
	if status != SprintActive {
		return ErrStoryNotAvailable
	}

	stories, err := lockAssignableStories(ctx, tx, projectID, ids, sprintID)
	if err != nil {
		return err
	}
	if err := persistStoryAssignments(ctx, tx, sprintID, stories); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return persistenceError("confirmar la asignación de historias")
	}
	return nil
}

// RemoveStory quita la asociación y devuelve la historia a Pendiente atómicamente.
func (s *PostgreSQLSprintStore) RemoveStory(ctx context.Context, sprintID int64, storyID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return persistenceError("iniciar la eliminación de una historia")
	}
	defer tx.Rollback()

	var status SprintStatus
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM sprints WHERE id = $1 FOR UPDATE`,
		sprintID,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return ErrStoryNotAvailable
	}
	if err != nil {
		return persistenceError("consultar el Sprint para quitar una historia")
	}
	if status != SprintActive {
		return ErrStoryNotAvailable
	}

	var story UserStory
	var activeSprintID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT id, project_id, status, active_sprint_id
		FROM user_stories
		WHERE id = $1
		FOR UPDATE`,
		storyID,
	).Scan(&story.ID, &story.ProjectID, &story.Status, &activeSprintID)
	if err == sql.ErrNoRows {
		return ErrStoryNotAvailable
	}
	if err != nil {
		return persistenceError("consultar la historia para quitar")
	}
	if activeSprintID.Valid {
		story.ActiveSprintID = activeSprintID.Int64
	}
	if err := RemoveStoryFromSprint(&story, sprintID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx,
		`DELETE FROM sprint_story_assignments WHERE sprint_id = $1 AND story_id = $2`,
		sprintID,
		storyID,
	)
	if err != nil {
		return persistenceError("eliminar la asociación de historia")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return persistenceError("verificar la eliminación de la asociación")
	}
	if affected != 1 {
		return ErrStoryNotAvailable
	}
	result, err = tx.ExecContext(ctx,
		`UPDATE user_stories SET status = $1, active_sprint_id = NULL WHERE id = $2`,
		story.Status,
		storyID,
	)
	if err != nil {
		return persistenceError("devolver la historia a Pendiente")
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return persistenceError("verificar la devolución de la historia a Pendiente")
	}
	if affected != 1 {
		return ErrStoryNotAvailable
	}
	if err := tx.Commit(); err != nil {
		return persistenceError("confirmar la eliminación de la asociación")
	}
	return nil
}

func lockAssignableStories(
	ctx context.Context,
	tx *sql.Tx,
	projectID int64,
	storyIDs []int64,
	targetSprintID int64,
) ([]UserStory, error) {
	ids := append([]int64(nil), storyIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	stories := make([]UserStory, 0, len(ids))
	for _, storyID := range ids {
		var story UserStory
		var activeSprintID, activeSprintNumber sql.NullInt64
		err := tx.QueryRowContext(ctx, `
			SELECT us.id, us.project_id, us.status, us.active_sprint_id, s.number
			FROM user_stories us
			LEFT JOIN sprints s ON s.id = us.active_sprint_id
			WHERE us.id = $1
			FOR UPDATE OF us`,
			storyID,
		).Scan(&story.ID, &story.ProjectID, &story.Status, &activeSprintID, &activeSprintNumber)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: historia %d inexistente", ErrStoryNotAvailable, storyID)
		}
		if err != nil {
			return nil, persistenceError("consultar la historia para asignación")
		}
		if activeSprintID.Valid {
			story.ActiveSprintID = activeSprintID.Int64
		}
		if activeSprintNumber.Valid {
			story.ActiveSprintNumber = activeSprintNumber.Int64
		}
		if targetSprintID != 0 && story.ActiveSprintID == targetSprintID && story.Status == EnSprint {
			continue
		}
		if err := validateStoryForProject(story, projectID); err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	return stories, nil
}

func persistStoryAssignments(ctx context.Context, tx *sql.Tx, sprintID int64, stories []UserStory) error {
	for _, story := range stories {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`,
			sprintID,
			story.ID,
		); err != nil {
			if isUniqueViolation(err) {
				return &StoryAlreadyAssignedError{StoryID: story.ID}
			}
			return persistenceError("crear la asociación de historia")
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE user_stories
			SET status = $1, active_sprint_id = $2
			WHERE id = $3 AND status = $4 AND active_sprint_id IS NULL`,
			EnSprint,
			sprintID,
			story.ID,
			Pendiente,
		)
		if err != nil {
			return persistenceError("actualizar el estado de la historia")
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return persistenceError("verificar la actualización del estado de la historia")
		}
		if affected != 1 {
			return &StoryAlreadyAssignedError{StoryID: story.ID}
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var stateError interface{ SQLState() string }
	return errors.As(err, &stateError) && stateError.SQLState() == "23505"
}
