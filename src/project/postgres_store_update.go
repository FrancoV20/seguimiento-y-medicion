package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgconn"
)

// Update guarda la modificación de un proyecto en una sola transacción (HU-11): o cambia todo
// (nombre, fechas e integrantes) o no cambia nada. Bloquea el proyecto, quita las bajas antes de
// contar sus horas (así espera a una carga de horas en curso) y revisa los Sprints solo si
// cambian las fechas. Guardar sin cambios es un éxito.
func (s *PostgreSQLProjectStore) Update(ctx context.Context, u ProjectUpdate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project update: %w", err)
	}
	defer tx.Rollback() // sin efecto después del Commit

	// 1. Bloquear el proyecto y leer sus fechas actuales.
	var oldStart, oldEnd sql.NullTime
	err = tx.QueryRowContext(ctx,
		`SELECT start_date, end_date FROM projects WHERE id = $1 FOR NO KEY UPDATE`, u.ID,
	).Scan(&oldStart, &oldEnd)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrProjectNotFound
	}
	if err != nil {
		return fmt.Errorf("lock project: %w", err)
	}

	for _, id := range u.MemberIDs {
		if id <= 0 || id > math.MaxInt32 { // members.id es INTEGER: fuera de rango no puede existir
			return ErrMemberNotFound
		}
	}
	requested := toInt32(u.MemberIDs)

	// 2. Todos los integrantes pedidos existen.
	var existing int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE id = ANY($1::int[])`, requested,
	).Scan(&existing); err != nil {
		return fmt.Errorf("check project members: %w", err)
	}
	if existing != len(requested) {
		return ErrMemberNotFound
	}

	// 3. Pertenencia actual.
	current, err := currentMembers(ctx, tx, u.ID)
	if err != nil {
		return err
	}
	added, removed := MembershipChanges(current, u.MemberIDs)

	if len(removed) > 0 {
		// 4. Bajas primero: el DELETE espera a una carga de horas que tenga la fila bloqueada.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM project_members WHERE project_id = $1 AND member_id = ANY($2::int[])`,
			u.ID, toInt32(removed),
		); err != nil {
			return fmt.Errorf("remove project members: %w", err)
		}
		// 5. Una baja con horas en historias de este proyecto bloquea todo el cambio.
		var name string
		err := tx.QueryRowContext(ctx, `
			SELECT m.display_name
			FROM effort_logs el
			JOIN user_stories us ON us.id = el.story_id
			JOIN members m ON m.id = el.member_id
			WHERE us.project_id = $1 AND el.member_id = ANY($2::int[])
			ORDER BY m.display_name, m.id
			LIMIT 1`, u.ID, toInt32(removed)).Scan(&name)
		if err == nil {
			return &MemberHasEffortError{MemberName: name}
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check member effort: %w", err)
		}
	}

	// 6. Sprints: solo si cambian las fechas.
	if DatesChanged(oldStart.Time, oldEnd.Time, u.StartDate, u.EndDate) {
		sprints, err := projectSprints(ctx, tx, u.ID)
		if err != nil {
			return err
		}
		if out := SprintsLeftOutside(oldStart.Time, oldEnd.Time, u.StartDate, u.EndDate, sprints); out != nil {
			return out
		}
	}

	// 7. Datos del proyecto.
	if _, err := tx.ExecContext(ctx,
		`UPDATE projects SET name = $2, start_date = $3, end_date = $4 WHERE id = $1`,
		u.ID, u.Name, u.StartDate, u.EndDate,
	); err != nil {
		return fmt.Errorf("update project: %w", err)
	}

	// 8. Altas.
	if len(added) > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO project_members (project_id, member_id)
			SELECT $1, unnest($2::int[])
			ON CONFLICT DO NOTHING`, u.ID, toInt32(added),
		); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" { // integrante borrado en el medio
				return ErrMemberNotFound
			}
			return fmt.Errorf("add project members: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project update: %w", err)
	}
	return nil
}

func currentMembers(ctx context.Context, tx *sql.Tx, projectID int) ([]int, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT member_id FROM project_members WHERE project_id = $1 ORDER BY member_id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("read project members: %w", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read project members: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func projectSprints(ctx context.Context, tx *sql.Tx, projectID int) ([]SprintPeriod, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT number, start_date, end_date FROM sprints WHERE project_id = $1 ORDER BY number`, projectID)
	if err != nil {
		return nil, fmt.Errorf("read project sprints: %w", err)
	}
	defer rows.Close()
	var sprints []SprintPeriod
	for rows.Next() {
		var s SprintPeriod
		if err := rows.Scan(&s.Number, &s.StartDate, &s.EndDate); err != nil {
			return nil, fmt.Errorf("read project sprints: %w", err)
		}
		sprints = append(sprints, s)
	}
	return sprints, rows.Err()
}

// toInt32 convierte los identificadores para pasarlos como int[] (las columnas son INTEGER).
func toInt32(ids []int) []int32 {
	out := make([]int32, len(ids))
	for i, id := range ids {
		out[i] = int32(id)
	}
	return out
}
