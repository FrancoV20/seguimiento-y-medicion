package poker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"
)

// PostgreSQLStore implementa Store sobre PostgreSQL (transacciones de data-model.md).
type PostgreSQLStore struct {
	db *sql.DB
}

var _ Store = (*PostgreSQLStore)(nil)

// NewPostgreSQLStore crea el almacenamiento a partir de una conexión ya abierta.
func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// OpenRound bloquea la historia del proyecto, revisa sus rondas y crea la ronda 1.
func (s *PostgreSQLStore) OpenRound(ctx context.Context, projectID, storyID, memberID int64) (OpenedRound, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OpenedRound{}, fmt.Errorf("begin open round: %w", err)
	}
	defer tx.Rollback() // sin efecto después del Commit

	round := OpenedRound{StoryID: storyID, Number: 1}
	err = tx.QueryRowContext(ctx,
		`SELECT title FROM user_stories WHERE id = $1 AND project_id = $2 FOR NO KEY UPDATE`,
		storyID, projectID).Scan(&round.StoryTitle)
	if errors.Is(err, sql.ErrNoRows) {
		return OpenedRound{}, ErrStoryNotFound
	}
	if err != nil {
		return OpenedRound{}, fmt.Errorf("lock story: %w", err)
	}

	statuses, err := roundStatuses(ctx, tx, storyID)
	if err != nil {
		return OpenedRound{}, err
	}
	if err := CheckOpen(statuses); err != nil {
		return OpenedRound{}, err
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO poker_rounds (project_id, story_id, round_number, status, opened_by)
		VALUES ($2, $1, 1, 'Abierta', $3)
		RETURNING id`, storyID, projectID, memberID).Scan(&round.RoundID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "uq_poker_rounds_one_open_per_story":
				return OpenedRound{}, ErrRoundAlreadyOpen
			case "uq_poker_rounds_story_number":
				return OpenedRound{}, ErrStoryAlreadyRounded
			}
		}
		return OpenedRound{}, fmt.Errorf("insert round: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return OpenedRound{}, fmt.Errorf("commit open round: %w", err)
	}
	return round, nil
}

func roundStatuses(ctx context.Context, tx *sql.Tx, storyID int64) ([]RoundStatus, error) {
	rows, err := tx.QueryContext(ctx, `SELECT status FROM poker_rounds WHERE story_id = $1`, storyID)
	if err != nil {
		return nil, fmt.Errorf("read rounds: %w", err)
	}
	defer rows.Close()
	var statuses []RoundStatus
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("read rounds: %w", err)
		}
		statuses = append(statuses, RoundStatus(s))
	}
	return statuses, rows.Err()
}

// CastVote toma la ronda con bloqueo compartido (un cierre con FOR UPDATE la espera o la hace
// esperar), revisa que esté abierta y registra o reemplaza el voto del integrante.
func (s *PostgreSQLStore) CastVote(ctx context.Context, projectID, roundID, memberID int64, value int) (VoteOutcome, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return VoteOutcome{}, fmt.Errorf("begin vote: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM poker_rounds WHERE id = $1 AND project_id = $2 FOR SHARE`,
		roundID, projectID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return VoteOutcome{}, ErrRoundNotFound
	}
	if err != nil {
		return VoteOutcome{}, fmt.Errorf("lock round: %w", err)
	}
	if err := CheckVote(RoundStatus(status)); err != nil {
		return VoteOutcome{}, err
	}

	// El CTE ve el estado anterior a la sentencia: inserted es falso si ya había votado.
	var inserted bool
	err = tx.QueryRowContext(ctx, `
		WITH previous AS (
			SELECT 1 FROM poker_votes WHERE round_id = $1 AND member_id = $2
		)
		INSERT INTO poker_votes (round_id, member_id, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (round_id, member_id)
		DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		RETURNING NOT EXISTS (SELECT 1 FROM previous)`, roundID, memberID, value).Scan(&inserted)
	if err != nil {
		return VoteOutcome{}, fmt.Errorf("save vote: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return VoteOutcome{}, fmt.Errorf("commit vote: %w", err)
	}
	return VoteOutcome{RoundID: roundID, Value: value, Updated: !inserted}, nil
}

// CloseRound cierra la ronda del proyecto en una transacción (HU-05b): la toma con bloqueo
// exclusivo (un voto en curso la tiene FOR SHARE y el cierre lo espera; un voto que llega
// después espera al cierre), cuenta sus votos, aplica CheckClose y la cierra con una sola
// sentencia. Sin fila, la ronda no existe o es de otro proyecto: ErrRoundNotFound antes que
// cualquier otro rechazo.
func (s *PostgreSQLStore) CloseRound(ctx context.Context, projectID, roundID int64) (ClosedRound, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ClosedRound{}, fmt.Errorf("begin close round: %w", err)
	}
	defer tx.Rollback()

	closed := ClosedRound{RoundID: roundID}
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT r.status, r.story_id, r.round_number, us.title
		  FROM poker_rounds r
		  JOIN user_stories us ON us.id = r.story_id
		 WHERE r.id = $1 AND r.project_id = $2
		   FOR UPDATE OF r`, roundID, projectID).Scan(&status, &closed.StoryID, &closed.Number, &closed.StoryTitle)
	if errors.Is(err, sql.ErrNoRows) {
		return ClosedRound{}, ErrRoundNotFound
	}
	if err != nil {
		return ClosedRound{}, fmt.Errorf("lock round: %w", err)
	}

	var votes int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM poker_votes WHERE round_id = $1`, roundID).Scan(&votes); err != nil {
		return ClosedRound{}, fmt.Errorf("count votes: %w", err)
	}
	if err := CheckClose(RoundStatus(status), votes); err != nil {
		return ClosedRound{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE poker_rounds SET status = 'Cerrada', closed_at = now() WHERE id = $1`, roundID); err != nil {
		return ClosedRound{}, fmt.Errorf("close round: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ClosedRound{}, fmt.Errorf("commit close round: %w", err)
	}
	return closed, nil
}
