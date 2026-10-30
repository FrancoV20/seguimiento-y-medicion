package effort

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// HU-11, T022: SaveEntry vuelve a comprobar la pertenencia dentro de su transacción.

func membershipEntry(f fixture, member int64) EffortEntry {
	return EffortEntry{StoryID: idStr(f.activeStory), MemberID: idStr(member),
		WorkDate: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Activity: "HU11", Hours: 100}
}

func countEntries(t *testing.T, db *sql.DB, story int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM effort_logs WHERE story_id = $1`, story).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func waitForLockedSession(t *testing.T, db *sql.DB) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&n)
		if n > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("SaveEntry no quedó esperando el bloqueo")
}

func TestPostgresSaveEntry_RechazaIntegranteQuitado(t *testing.T) {
	db := openTestDB(t)
	f := newFixture(t, db)
	_, _, err := NewPostgreSQLEffortStore(db).SaveEntry(context.Background(), membershipEntry(f, f.foreignMember))
	if !errors.Is(err, errMemberNotInProject) {
		t.Fatalf("se esperaba errMemberNotInProject, se obtuvo %v", err)
	}
	if n := countEntries(t, db, f.activeStory); n != 0 {
		t.Errorf("no debía guardarse nada, hay %d registros", n)
	}
}

func TestPostgresSaveEntry_EsperaBajaEnCurso(t *testing.T) {
	for _, commit := range []bool{true, false} {
		name := "baja confirmada"
		if !commit {
			name = "baja deshecha"
		}
		t.Run(name, func(t *testing.T) {
			db := openTestDB(t)
			f := newFixture(t, db)
			ctx := context.Background()
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err := tx.Exec(`DELETE FROM project_members WHERE project_id = $1 AND member_id = $2`, f.project, f.member); err != nil {
				t.Fatal(err)
			}

			done := make(chan error, 1)
			go func() {
				_, _, err := NewPostgreSQLEffortStore(db).SaveEntry(ctx, membershipEntry(f, f.member))
				done <- err
			}()
			waitForLockedSession(t, db)
			if commit {
				err = tx.Commit()
			} else {
				err = tx.Rollback()
			}
			if err != nil {
				t.Fatal(err)
			}

			err = <-done
			n := countEntries(t, db, f.activeStory)
			if commit && (!errors.Is(err, errMemberNotInProject) || n != 0) {
				t.Errorf("con la baja confirmada se esperaba errMemberNotInProject sin registro, se obtuvo %v y %d registros", err, n)
			}
			if !commit && (err != nil || n != 1) {
				t.Errorf("con la baja deshecha se esperaba guardar, se obtuvo %v y %d registros", err, n)
			}
		})
	}
}
