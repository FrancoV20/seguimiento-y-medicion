package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// PostgreSQLPasswordStore implementa PasswordStore contra la tabla members (HU-27).
type PostgreSQLPasswordStore struct {
	db *sql.DB
}

var _ PasswordStore = (*PostgreSQLPasswordStore)(nil)

// NewPostgreSQLPasswordStore crea el adaptador sobre una conexión abierta.
func NewPostgreSQLPasswordStore(db *sql.DB) *PostgreSQLPasswordStore {
	return &PostgreSQLPasswordStore{db: db}
}

// validMemberID indica si el identificador puede existir: members.id es SERIAL (INTEGER de 32 bits).
func validMemberID(id int64) bool { return id >= 1 && id <= math.MaxInt32 }

// MemberName devuelve el nombre del integrante, o ErrMemberNotFound.
func (s *PostgreSQLPasswordStore) MemberName(ctx context.Context, memberID int64) (string, error) {
	if !validMemberID(memberID) {
		return "", ErrMemberNotFound
	}
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT display_name FROM members WHERE id = $1`, memberID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrMemberNotFound
	}
	if err != nil {
		return "", fmt.Errorf("auth: leer el integrante %d: %w", memberID, err)
	}
	return name, nil
}

// SetPasswordHash guarda el valor de la contraseña del integrante, o ErrMemberNotFound.
func (s *PostgreSQLPasswordStore) SetPasswordHash(ctx context.Context, memberID int64, hash string) error {
	if !validMemberID(memberID) {
		return ErrMemberNotFound
	}
	res, err := s.db.ExecContext(ctx, `UPDATE members SET password_hash = $2 WHERE id = $1`, memberID, hash)
	if err != nil {
		return fmt.Errorf("auth: guardar la contraseña del integrante %d: %w", memberID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: guardar la contraseña del integrante %d: %w", memberID, err)
	}
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}
