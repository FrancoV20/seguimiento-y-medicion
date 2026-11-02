package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// HU-28, T006: la migración 016 agrega project_members.role (VARCHAR(20) NOT NULL DEFAULT
// 'Integrante', solo 'Scrum Master' o 'Integrante') y su down la quita, sobre una base
// descartable, con el mismo patrón que TestMigraciones_SeAplicanEnOrdenSobreUnaBaseDescartable.

const memberRoleVersion = 16

func TestMigracion_ProjectMemberRole_UpYDown(t *testing.T) {
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

	name := fmt.Sprintf("hu28_migracion_%d", time.Now().UnixNano())
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
	type columnInfo struct {
		dataType, nullable, def string
		maxLength               int
	}
	column := func() (columnInfo, bool) {
		t.Helper()
		var c columnInfo
		var def sql.NullString
		var maxLength sql.NullInt64
		err := db.QueryRowContext(ctx, `SELECT data_type, character_maximum_length, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = 'project_members' AND column_name = 'role'`).
			Scan(&c.dataType, &maxLength, &c.nullable, &def)
		if errors.Is(err, sql.ErrNoRows) {
			return columnInfo{}, false
		}
		if err != nil {
			t.Fatal(err)
		}
		c.def, c.maxLength = def.String, int(maxLength.Int64)
		return c, true
	}
	roleOf := func(projectID, memberID int64) string {
		t.Helper()
		var r string
		if err := db.QueryRowContext(ctx, `SELECT role FROM project_members WHERE project_id = $1 AND member_id = $2`,
			projectID, memberID).Scan(&r); err != nil {
			t.Fatalf("no se pudo leer el rol: %v", err)
		}
		return r
	}

	var target *migrationPair
	for _, m := range migrationsInOrder(t) {
		if m.version == memberRoleVersion {
			m := m
			target = &m
			break
		}
		exec(m.up)
	}
	if target == nil {
		t.Fatalf("no existe la migración %03d", memberRoleVersion)
	}

	// Un integrante de un proyecto previo a la migración queda como 'Integrante'.
	var alfa, ana, beto int64
	if err := db.QueryRowContext(ctx, `INSERT INTO projects (name, start_date, end_date)
		VALUES ('Alfa', '2026-03-01', '2026-12-31') RETURNING id`).Scan(&alfa); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO members (display_name) VALUES ('Ana') RETURNING id`).Scan(&ana); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO members (display_name) VALUES ('Beto') RETURNING id`).Scan(&beto); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, alfa, ana); err != nil {
		t.Fatal(err)
	}

	exec(target.up)
	c, found := column()
	if !found {
		t.Fatal("después de la up no existe project_members.role")
	}
	if c.dataType != "character varying" || c.maxLength != 20 || c.nullable != "NO" || c.def != "'Integrante'::character varying" {
		t.Errorf("role = %+v; se esperaba character varying(20), NOT NULL y DEFAULT 'Integrante'", c)
	}
	if got := roleOf(alfa, ana); got != "Integrante" {
		t.Errorf("rol de la fila previa = %q; se esperaba Integrante", got)
	}

	// INSERT sin role toma el valor por defecto.
	if _, err := db.ExecContext(ctx, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, alfa, beto); err != nil {
		t.Fatalf("INSERT sin role: %v", err)
	}
	if got := roleOf(alfa, beto); got != "Integrante" {
		t.Errorf("rol de un INSERT sin role = %q; se esperaba Integrante", got)
	}

	// Solo 'Scrum Master' o 'Integrante'.
	if _, err := db.ExecContext(ctx, `UPDATE project_members SET role = 'Scrum Master' WHERE project_id = $1 AND member_id = $2`,
		alfa, ana); err != nil {
		t.Errorf("UPDATE a 'Scrum Master': %v", err)
	}
	_, err = db.ExecContext(ctx, `UPDATE project_members SET role = 'Product Owner' WHERE project_id = $1 AND member_id = $2`,
		alfa, beto)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Errorf("UPDATE a 'Product Owner': error = %v; se esperaba la violación del CHECK (23514)", err)
	}
	if got := roleOf(alfa, beto); got != "Integrante" {
		t.Errorf("después del UPDATE rechazado el rol es %q; se esperaba Integrante", got)
	}

	exec(target.down)
	if _, found := column(); found {
		t.Error("después de la down la columna role sigue existiendo")
	}
	exec(target.up) // se puede volver a aplicar
	if _, found := column(); !found {
		t.Error("al volver a aplicar la up no se creó la columna")
	}
	if got := roleOf(alfa, ana); got != "Integrante" {
		t.Errorf("después de down y up el rol es %q; se esperaba Integrante", got)
	}
}
