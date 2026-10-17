package effort

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para database/sql
)

// errInvalidID se devuelve al guardar con un identificador que no es numérico.
var errInvalidID = errors.New("identificador no numérico")

// PostgreSQLEffortStore implementa EffortStore sobre PostgreSQL con database/sql y pgx.
type PostgreSQLEffortStore struct {
	db *sql.DB
}

// NewPostgreSQLEffortStore crea el store a partir de una conexión ya abierta.
func NewPostgreSQLEffortStore(db *sql.DB) *PostgreSQLEffortStore {
	return &PostgreSQLEffortStore{db: db}
}

// Los identificadores del dominio son texto; en la base son numéricos. Un identificador que
// no es un número válido no puede existir, así que las consultas lo tratan como "no encontrado".
func parseStoryID(id string) (int64, bool) {
	n, err := strconv.ParseInt(id, 10, 64)
	return n, err == nil
}

func parseMemberID(id string) (int64, bool) {
	n, err := strconv.ParseInt(id, 10, 32) // members.id es INTEGER
	return n, err == nil
}

// StoryAcceptsEffort indica si la historia está asignada a un Sprint con estado Activo.
func (s *PostgreSQLEffortStore) StoryAcceptsEffort(ctx context.Context, storyID string) (bool, error) {
	id, ok := parseStoryID(storyID)
	if !ok {
		return false, nil
	}
	var accepts bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_stories us
			JOIN sprints sp ON sp.id = us.active_sprint_id
			WHERE us.id = $1 AND sp.status = 'Activo'
		)`, id).Scan(&accepts)
	return accepts, err
}

// MemberBelongsToStoryProject indica si el integrante pertenece al proyecto de la historia.
func (s *PostgreSQLEffortStore) MemberBelongsToStoryProject(ctx context.Context, memberID, storyID string) (bool, error) {
	mid, ok := parseMemberID(memberID)
	if !ok {
		return false, nil
	}
	sid, ok := parseStoryID(storyID)
	if !ok {
		return false, nil
	}
	var belongs bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_stories us
			JOIN project_members pm ON pm.project_id = us.project_id
			WHERE us.id = $1 AND pm.member_id = $2
		)`, sid, mid).Scan(&belongs)
	return belongs, err
}

// SaveEntry guarda el registro y devuelve el acumulado de la historia en una sola transacción:
// si algo falla, no queda nada guardado.
func (s *PostgreSQLEffortStore) SaveEntry(ctx context.Context, entry EffortEntry) (EffortEntry, Hours, error) {
	sid, ok := parseStoryID(entry.StoryID)
	if !ok {
		return EffortEntry{}, 0, errInvalidID
	}
	mid, ok := parseMemberID(entry.MemberID)
	if !ok {
		return EffortEntry{}, 0, errInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return EffortEntry{}, 0, err
	}
	defer tx.Rollback() // sin efecto si ya se hizo Commit

	year, month, day := entry.WorkDate.Date()
	var id int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, make_date($3::int, $4::int, $5::int), $6, ($7::bigint)::numeric / 100)
		RETURNING id, created_at`,
		sid, mid, year, int(month), day, entry.Activity, int64(entry.Hours),
	).Scan(&id, &entry.CreatedAt)
	if err != nil {
		return EffortEntry{}, 0, err
	}
	entry.ID = strconv.FormatInt(id, 10)

	total, err := accumulatedHours(ctx, tx, sid)
	if err != nil {
		return EffortEntry{}, 0, err
	}
	if err := tx.Commit(); err != nil {
		return EffortEntry{}, 0, err
	}
	return entry, total, nil
}

// AccumulatedHours devuelve la suma de las horas de la historia; 0 si no tiene registros.
func (s *PostgreSQLEffortStore) AccumulatedHours(ctx context.Context, storyID string) (Hours, error) {
	id, ok := parseStoryID(storyID)
	if !ok {
		return 0, nil
	}
	return accumulatedHours(ctx, s.db, id)
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// accumulatedHours suma en centésimas de hora, sin pasar por números de punto flotante.
func accumulatedHours(ctx context.Context, q queryRower, storyID int64) (Hours, error) {
	var total int64
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(hours) * 100, 0)::bigint FROM effort_logs WHERE story_id = $1`, storyID,
	).Scan(&total)
	return Hours(total), err
}

// Garantiza en compilación que PostgreSQLEffortStore cumple el contrato.
var _ EffortStore = (*PostgreSQLEffortStore)(nil)
