package role

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// PostgreSQLStore implementa Store contra projects, members y project_members (HU-28).
type PostgreSQLStore struct {
	db *sql.DB
}

var _ Store = (*PostgreSQLStore)(nil)

// NewPostgreSQLStore crea el adaptador sobre una conexión abierta.
func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// validID indica si el identificador puede existir: las columnas id son SERIAL (INTEGER de 32 bits).
func validID(id int64) bool { return id >= 1 && id <= math.MaxInt32 }

// ProjectName devuelve el nombre del proyecto, o ErrProjectNotFound.
func (s *PostgreSQLStore) ProjectName(ctx context.Context, projectID int64) (string, error) {
	if !validID(projectID) {
		return "", ErrProjectNotFound
	}
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM projects WHERE id = $1`, projectID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrProjectNotFound
	}
	if err != nil {
		return "", fmt.Errorf("role: leer el proyecto %d: %w", projectID, err)
	}
	return name, nil
}

// setMemberRoleQuery cambia el rol solo si el integrante pertenece al proyecto.
const setMemberRoleQuery = `
	UPDATE project_members pm
	SET role = $3
	FROM members m
	WHERE pm.project_id = $1 AND pm.member_id = $2 AND m.id = pm.member_id
	RETURNING m.display_name`

// SetMemberRole guarda el rol del integrante en el proyecto y devuelve su nombre, o
// ErrMemberNotInProject.
func (s *PostgreSQLStore) SetMemberRole(ctx context.Context, projectID, memberID int64, r Role) (string, error) {
	if !validID(projectID) || !validID(memberID) {
		return "", ErrMemberNotInProject
	}
	var name string
	err := s.db.QueryRowContext(ctx, setMemberRoleQuery, projectID, memberID, string(r)).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrMemberNotInProject
	}
	if err != nil {
		return "", fmt.Errorf("role: guardar el rol del integrante %d en el proyecto %d: %w", memberID, projectID, err)
	}
	return name, nil
}
