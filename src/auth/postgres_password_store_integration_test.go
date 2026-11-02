package auth

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"
)

// HU-27, T013: PostgreSQLPasswordStore contra PostgreSQL real.

func openPasswordTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestPostgreSQLPasswordStore(t *testing.T) {
	db := openPasswordTestDB(t)
	ctx := context.Background()
	var caro int64
	if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ('Caro') RETURNING id`).Scan(&caro); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM members WHERE id = $1`, caro) })
	store := NewPostgreSQLPasswordStore(db)

	if name, err := store.MemberName(ctx, caro); err != nil || name != "Caro" {
		t.Errorf("MemberName(Caro) = %q, %v; se esperaba Caro", name, err)
	}
	for _, id := range []int64{2147483647, 0, -1, 2147483648} {
		if _, err := store.MemberName(ctx, id); !errors.Is(err, ErrMemberNotFound) {
			t.Errorf("MemberName(%d) error = %v; se esperaba ErrMemberNotFound", id, err)
		}
		if err := store.SetPasswordHash(ctx, id, "x"); !errors.Is(err, ErrMemberNotFound) {
			t.Errorf("SetPasswordHash(%d) error = %v; se esperaba ErrMemberNotFound", id, err)
		}
	}

	stored, err := HashPasswordIterations("caro-clave-2026", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetPasswordHash(ctx, caro, stored); err != nil {
		t.Fatalf("SetPasswordHash: %v", err)
	}
	var got sql.NullString
	if err := db.QueryRow(`SELECT password_hash FROM members WHERE id = $1`, caro).Scan(&got); err != nil || got.String != stored {
		t.Errorf("password_hash = %q, %v; se esperaba el valor guardado", got.String, err)
	}

	closed, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	_ = closed.Close()
	down := NewPostgreSQLPasswordStore(closed)
	if _, err := down.MemberName(ctx, caro); err == nil || errors.Is(err, ErrMemberNotFound) {
		t.Errorf("MemberName con la base cerrada: error = %v; se esperaba una falla distinta de ErrMemberNotFound", err)
	}
	if err := down.SetPasswordHash(ctx, caro, stored); err == nil || errors.Is(err, ErrMemberNotFound) {
		t.Errorf("SetPasswordHash con la base cerrada: error = %v; se esperaba una falla distinta de ErrMemberNotFound", err)
	}
}
