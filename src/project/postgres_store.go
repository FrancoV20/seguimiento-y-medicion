package project

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgreSQLProjectStore struct {
	db *sql.DB
}

func NewPostgreSQLProjectStore(db *sql.DB) *PostgreSQLProjectStore {
	return &PostgreSQLProjectStore{db: db}
}

func (s *PostgreSQLProjectStore) Create(p *Project) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin project transaction: %w", err)
	}
	defer tx.Rollback()

	var projectID int
	err = tx.QueryRow(
		`INSERT INTO projects (name, status, start_date, end_date)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		p.Name,
		p.Status,
		p.StartDate,
		p.EndDate,
	).Scan(&projectID)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	for _, memberIDString := range p.Members {
		memberID, err := strconv.Atoi(memberIDString)
		if err != nil {
			return fmt.Errorf("invalid member ID %q: %w", memberIDString, err)
		}

		_, err = tx.Exec(
			`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`,
			projectID,
			memberID,
		)
		if err != nil {
			return fmt.Errorf("insert project member %d: %w", memberID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project transaction: %w", err)
	}

	p.ID = projectID
	return nil
}

func (s *PostgreSQLProjectStore) MemberExists(id int) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM members WHERE id = $1)`,
		id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check member existence: %w", err)
	}
	return exists, nil
}

var _ ProjectStore = (*PostgreSQLProjectStore)(nil)
