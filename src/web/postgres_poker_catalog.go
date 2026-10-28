package web

import (
	"context"
	"database/sql"
)

// PostgreSQLPokerCatalog implementa PokerCatalog sobre PostgreSQL (HU-05a). La consulta de
// participantes nunca lee poker_votes.value; el valor solo se lee filtrando por el integrante
// identificado.
type PostgreSQLPokerCatalog struct {
	db *sql.DB
}

var _ PokerCatalog = (*PostgreSQLPokerCatalog)(nil)

// NewPostgreSQLPokerCatalog crea el catálogo a partir de una conexión ya abierta.
func NewPostgreSQLPokerCatalog(db *sql.DB) *PostgreSQLPokerCatalog {
	return &PostgreSQLPokerCatalog{db: db}
}

const pokerStoriesQuery = `
SELECT id, title, status FROM user_stories
 WHERE project_id = $1
 ORDER BY id`

const pokerOpenRoundsQuery = `
SELECT r.id, r.story_id, us.title, r.round_number
  FROM poker_rounds r
  JOIN user_stories us ON us.id = r.story_id
 WHERE r.project_id = $1 AND r.status = 'Abierta'
 ORDER BY r.created_at, r.id`

const pokerParticipantsQuery = `
SELECT r.id, m.id, m.display_name,
       EXISTS (SELECT 1 FROM poker_votes v
                WHERE v.round_id = r.id AND v.member_id = m.id) AS has_voted
  FROM poker_rounds r
  JOIN project_members pm ON pm.project_id = r.project_id
  JOIN members m ON m.id = pm.member_id
 WHERE r.project_id = $1 AND r.status = 'Abierta'
 ORDER BY r.id, m.display_name, m.id`

const pokerOwnVotesQuery = `
SELECT v.round_id, v.value
  FROM poker_votes v
  JOIN poker_rounds r ON r.id = v.round_id
 WHERE r.project_id = $1 AND r.status = 'Abierta' AND v.member_id = $2`

// PokerBoard lee las cuatro partes de la pantalla en una transacción de solo lectura, para que
// sea coherente aunque alguien vote mientras se arma.
func (c *PostgreSQLPokerCatalog) PokerBoard(ctx context.Context, projectID, memberID int64) (PokerBoard, error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return PokerBoard{}, Unavailable(err)
	}
	defer tx.Rollback()

	board := PokerBoard{Stories: []PokerStoryOption{}, Rounds: []PokerRoundView{}}
	if err := pokerEach(ctx, tx, pokerStoriesQuery, []any{projectID}, func(rows *sql.Rows) error {
		var s PokerStoryOption
		if err := rows.Scan(&s.ID, &s.Title, &s.Status); err != nil {
			return err
		}
		board.Stories = append(board.Stories, s)
		return nil
	}); err != nil {
		return PokerBoard{}, err
	}

	index := map[int64]int{}
	if err := pokerEach(ctx, tx, pokerOpenRoundsQuery, []any{projectID}, func(rows *sql.Rows) error {
		r := PokerRoundView{Participants: []PokerParticipantView{}}
		if err := rows.Scan(&r.ID, &r.StoryID, &r.StoryTitle, &r.Number); err != nil {
			return err
		}
		index[r.ID] = len(board.Rounds)
		board.Rounds = append(board.Rounds, r)
		return nil
	}); err != nil {
		return PokerBoard{}, err
	}

	if err := pokerEach(ctx, tx, pokerParticipantsQuery, []any{projectID}, func(rows *sql.Rows) error {
		var roundID int64
		var p PokerParticipantView
		if err := rows.Scan(&roundID, &p.MemberID, &p.Name, &p.HasVoted); err != nil {
			return err
		}
		if i, ok := index[roundID]; ok {
			p.IsMe = p.MemberID == memberID
			board.Rounds[i].Participants = append(board.Rounds[i].Participants, p)
		}
		return nil
	}); err != nil {
		return PokerBoard{}, err
	}

	if err := pokerEach(ctx, tx, pokerOwnVotesQuery, []any{projectID, memberID}, func(rows *sql.Rows) error {
		var roundID int64
		var value int
		if err := rows.Scan(&roundID, &value); err != nil {
			return err
		}
		if i, ok := index[roundID]; ok {
			v := value
			board.Rounds[i].OwnVote = &v
		}
		return nil
	}); err != nil {
		return PokerBoard{}, err
	}
	return board, nil
}

// pokerEach ejecuta una consulta y llama a scan por cada fila.
func pokerEach(ctx context.Context, tx *sql.Tx, query string, args []any, scan func(*sql.Rows) error) error {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return Unavailable(err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return Unavailable(err)
		}
	}
	if err := rows.Err(); err != nil {
		return Unavailable(err)
	}
	return nil
}
