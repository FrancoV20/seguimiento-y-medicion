package project

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgreSQLProjectStore_Create_Integration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL no está configurada")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database connection: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect to database: %v", err)
	}

	if _, err := db.ExecContext(ctx, `TRUNCATE TABLE project_members, projects, members RESTART IDENTITY`); err != nil {
		t.Fatalf("clean project tables: %v", err)
	}

	var memberID int
	if err := db.QueryRowContext(
		ctx,
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		"Integration User",
	).Scan(&memberID); err != nil {
		t.Fatalf("insert integration member: %v", err)
	}

	var projectID int
	defer func() {
		if projectID != 0 {
			if _, err := db.ExecContext(ctx, `DELETE FROM project_members WHERE project_id = $1`, projectID); err != nil {
				t.Errorf("clean project-member relation: %v", err)
			}
			if _, err := db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, projectID); err != nil {
				t.Errorf("clean integration project: %v", err)
			}
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM members WHERE id = $1`, memberID); err != nil {
			t.Errorf("clean integration member: %v", err)
		}
	}()

	startDate := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	project, err := NewProject(
		"Integration Project",
		[]string{strconv.Itoa(memberID)},
		startDate,
		startDate.AddDate(0, 0, 14),
	)
	if err != nil {
		t.Fatalf("create valid project model: %v", err)
	}

	store := NewPostgreSQLProjectStore(db)
	if err := store.Create(&project); err != nil {
		t.Fatalf("store project: %v", err)
	}
	projectID = project.ID
	if project.ID <= 0 {
		t.Fatalf("expected a generated project ID, got %d", project.ID)
	}

	var storedName string
	var storedStatus string
	var storedMemberID int
	err = db.QueryRowContext(
		ctx,
		`SELECT p.name, p.status, pm.member_id
		 FROM projects p
		 JOIN project_members pm ON pm.project_id = p.id
		 WHERE p.id = $1`,
		project.ID,
	).Scan(&storedName, &storedStatus, &storedMemberID)
	if err != nil {
		t.Fatalf("query stored project and member relation: %v", err)
	}
	if storedName != project.Name {
		t.Errorf("expected stored name %q, got %q", project.Name, storedName)
	}
	if storedStatus != project.Status {
		t.Errorf("expected stored status %q, got %q", project.Status, storedStatus)
	}
	if storedMemberID != memberID {
		t.Errorf("expected related member ID %d, got %d", memberID, storedMemberID)
	}
}
