package defect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgreSQLResolutionStore resuelve y reabre defectos en PostgreSQL.
type PostgreSQLResolutionStore struct {
	db *sql.DB
}

var _ ResolutionStore = (*PostgreSQLResolutionStore)(nil)

// NewPostgreSQLResolutionStore crea el almacenamiento sobre una conexión abierta.
func NewPostgreSQLResolutionStore(db *sql.DB) *PostgreSQLResolutionStore {
	return &PostgreSQLResolutionStore{db: db}
}

// Resolve registra el Sprint activo del proyecto como Sprint de resolución del defecto. Bloquea el
// defecto (FOR UPDATE) y después el Sprint activo (FOR SHARE), para serializarse con otra
// resolución del mismo defecto y con el cierre del Sprint (HU-15). Cualquier error revierte todo.
func (s *PostgreSQLResolutionStore) Resolve(ctx context.Context, projectID, defectID int64) (Resolved, error) {
	if s == nil || s.db == nil {
		return Resolved{}, errors.New("PostgreSQL database connection is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Resolved{}, fmt.Errorf("begin defect resolution: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var detectionSprintNumber int64
	err = tx.QueryRowContext(ctx,
		`SELECT d.status, ds.number
		 FROM defects d
		 JOIN sprints ds ON ds.id = d.detection_sprint_id
		 WHERE d.id = $1 AND d.project_id = $2
		 FOR UPDATE OF d`,
		defectID, projectID,
	).Scan(&status, &detectionSprintNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return Resolved{}, ErrDefectNotFound
	}
	if err != nil {
		return Resolved{}, fmt.Errorf("lock defect %d: %w", defectID, err)
	}

	var activeSprintID, activeSprintNumber int64
	hasActiveSprint := true
	err = tx.QueryRowContext(ctx,
		`SELECT id, number
		 FROM sprints
		 WHERE project_id = $1 AND status = 'Activo'
		 ORDER BY number DESC
		 LIMIT 1
		 FOR SHARE`,
		projectID,
	).Scan(&activeSprintID, &activeSprintNumber)
	if errors.Is(err, sql.ErrNoRows) {
		hasActiveSprint = false
	} else if err != nil {
		return Resolved{}, fmt.Errorf("lock active sprint of project %d: %w", projectID, err)
	}

	if err := CheckResolve(Status(status), detectionSprintNumber, activeSprintNumber, hasActiveSprint); err != nil {
		return Resolved{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE defects SET status = $1, resolution_sprint_id = $2 WHERE id = $3`,
		string(StatusResolved), activeSprintID, defectID,
	); err != nil {
		return Resolved{}, fmt.Errorf("resolve defect %d: %w", defectID, err)
	}
	if err := tx.Commit(); err != nil {
		return Resolved{}, fmt.Errorf("commit defect %d resolution: %w", defectID, err)
	}
	return Resolved{DefectID: defectID, SprintNumber: activeSprintNumber}, nil
}

// Reopen vuelve el defecto a "Abierto" y borra su Sprint de resolución. No necesita Sprint activo.
// Cualquier error revierte todo.
func (s *PostgreSQLResolutionStore) Reopen(ctx context.Context, projectID, defectID int64) error {
	if s == nil || s.db == nil {
		return errors.New("PostgreSQL database connection is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin defect reopening: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM defects WHERE id = $1 AND project_id = $2 FOR UPDATE`,
		defectID, projectID,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrDefectNotFound
	}
	if err != nil {
		return fmt.Errorf("lock defect %d: %w", defectID, err)
	}

	if err := CheckReopen(Status(status)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE defects SET status = $1, resolution_sprint_id = NULL WHERE id = $2`,
		string(StatusOpen), defectID,
	); err != nil {
		return fmt.Errorf("reopen defect %d: %w", defectID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit defect %d reopening: %w", defectID, err)
	}
	return nil
}
