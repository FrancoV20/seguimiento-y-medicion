package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"testing"
	"time"
)

// HU-27, T011: la migración 015 agrega members.password_hash (TEXT NULL) y su down la quita, sobre
// una base descartable, con el mismo patrón que TestMigraciones_SeAplicanEnOrdenSobreUnaBaseDescartable.

const memberPasswordVersion = 15

func TestMigracion_PasswordHash_UpYDown(t *testing.T) {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", raw)
	if err != nil {
		t.Fatalf("no se pudo abrir la base de desarrollo para crear la descartable: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close() })

	name := fmt.Sprintf("hu27_migracion_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("no se pudo crear la base descartable (el usuario necesita permiso CREATEDB): %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	})
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("DATABASE_URL no es válida")
	}
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatalf("no se pudo abrir la base descartable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	exec := func(file string) {
		t.Helper()
		b, err := fs.ReadFile(Migrations, file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(b)); err != nil {
			t.Fatalf("%s falló: %v", path.Base(file), err)
		}
	}
	column := func() (dataType, nullable string, found bool) {
		t.Helper()
		err := db.QueryRowContext(ctx, `SELECT data_type, is_nullable FROM information_schema.columns
			WHERE table_name = 'members' AND column_name = 'password_hash'`).Scan(&dataType, &nullable)
		if err == sql.ErrNoRows {
			return "", "", false
		}
		if err != nil {
			t.Fatal(err)
		}
		return dataType, nullable, true
	}

	var target *migrationPair
	for _, m := range migrationsInOrder(t) {
		if m.version == memberPasswordVersion {
			m := m
			target = &m
			break
		}
		exec(m.up)
	}
	if target == nil {
		t.Fatalf("no existe la migración %03d", memberPasswordVersion)
	}
	// Un integrante previo a la migración queda sin contraseña.
	if _, err := db.ExecContext(ctx, `INSERT INTO members (display_name) VALUES ('Ana')`); err != nil {
		t.Fatal(err)
	}

	exec(target.up)
	if dataType, nullable, found := column(); !found || dataType != "text" || nullable != "YES" {
		t.Errorf("password_hash = %q, %q, %v; se esperaba text y nullable", dataType, nullable, found)
	}
	var withPassword int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM members WHERE password_hash IS NOT NULL`).Scan(&withPassword); err != nil || withPassword != 0 {
		t.Errorf("integrantes previos con contraseña = %d (error: %v); se esperaba 0", withPassword, err)
	}

	exec(target.down)
	if _, _, found := column(); found {
		t.Error("después de la down la columna password_hash sigue existiendo")
	}
	exec(target.up) // se puede volver a aplicar
	if _, _, found := column(); !found {
		t.Error("al volver a aplicar la up no se creó la columna")
	}
}
