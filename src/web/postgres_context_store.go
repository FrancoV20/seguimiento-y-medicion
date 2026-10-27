package web

import (
	"context"
	"database/sql"
	"errors"
)

// PostgreSQLContextStore implementa ContextStore contra PostgreSQL (HU-20b, T010).
// Solo lee; no modifica proyectos ni integrantes.
type PostgreSQLContextStore struct {
	db *sql.DB
}

var _ ContextStore = (*PostgreSQLContextStore)(nil)

// NewPostgreSQLContextStore crea el adaptador sobre una conexión abierta.
func NewPostgreSQLContextStore(db *sql.DB) *PostgreSQLContextStore {
	return &PostgreSQLContextStore{db: db}
}

// ListProjects devuelve los proyectos, los más recientes primero.
func (s *PostgreSQLContextStore) ListProjects(ctx context.Context) ([]ProjectSummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM projects ORDER BY id DESC`)
	if err != nil {
		return nil, Unavailable(err)
	}
	defer rows.Close()

	var out []ProjectSummary
	for rows.Next() {
		var p ProjectSummary
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, Unavailable(err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, Unavailable(err)
	}
	return out, nil
}

// ListMembers devuelve los integrantes del proyecto ordenados por nombre.
func (s *PostgreSQLContextStore) ListMembers(ctx context.Context, projectID int64) ([]MemberSummary, error) {
	if !validID(projectID) {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.display_name
		FROM members m
		JOIN project_members pm ON pm.member_id = m.id
		WHERE pm.project_id = $1
		ORDER BY m.display_name, m.id`, projectID)
	if err != nil {
		return nil, Unavailable(err)
	}
	defer rows.Close()

	var out []MemberSummary
	for rows.Next() {
		var m MemberSummary
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, Unavailable(err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, Unavailable(err)
	}
	return out, nil
}

// resolveContextQuery valida proyecto e integrante en una sola consulta.
// Sin filas: el proyecto no existe. display_name nulo: el proyecto existe pero el
// integrante no pertenece a él (con $2 = 0 nunca coincide, porque SERIAL empieza en 1).
const resolveContextQuery = `
	SELECT p.name, m.display_name
	FROM projects p
	LEFT JOIN project_members pm ON pm.project_id = p.id AND pm.member_id = $2
	LEFT JOIN members m ON m.id = pm.member_id
	WHERE p.id = $1`

// ResolveContext distingue "no existe" (resultado vacío, sin error) de "base no disponible"
// (error KindUnavailable), para que el middleware no limpie la selección ante una caída.
func (s *PostgreSQLContextStore) ResolveContext(ctx context.Context, projectID, memberID int64) (ResolvedContext, error) {
	if !validID(projectID) {
		return ResolvedContext{}, nil
	}
	if !validID(memberID) {
		memberID = 0
	}

	var projectName string
	var memberName sql.NullString
	err := s.db.QueryRowContext(ctx, resolveContextQuery, projectID, memberID).Scan(&projectName, &memberName)
	if errors.Is(err, sql.ErrNoRows) {
		return ResolvedContext{}, nil
	}
	if err != nil {
		return ResolvedContext{}, Unavailable(err)
	}

	rc := ResolvedContext{ProjectExists: true, ProjectName: projectName}
	if memberName.Valid {
		rc.MemberBelongs = true
		rc.MemberName = memberName.String
	}
	return rc, nil
}