package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"
)

// T043: todas las migraciones se aplican en orden sobre una base vacía y descartable.
// La prueba crea su propia base y la borra al terminar: nunca usa la base de desarrollo.

type migrationPair struct {
	version  uint64
	up, down string
}

// migrationsInOrder agrupa los .sql embebidos por versión y los ordena.
func migrationsInOrder(t *testing.T) []migrationPair {
	t.Helper()
	files, err := fs.Glob(Migrations, "migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no se encontraron migraciones embebidas (error: %v)", err)
	}
	byVersion := map[uint64]*migrationPair{}
	for _, f := range files {
		name := path.Base(f)
		prefix, rest, ok := strings.Cut(name, "_")
		v, convErr := strconv.ParseUint(prefix, 10, 64)
		if !ok || convErr != nil {
			t.Fatalf("%s: el nombre debe empezar con el número de versión (NNN_nombre.up.sql)", name)
		}
		m := byVersion[v]
		if m == nil {
			m = &migrationPair{version: v}
			byVersion[v] = m
		}
		switch {
		case strings.HasSuffix(rest, ".up.sql"):
			if m.up != "" {
				t.Fatalf("la versión %d está repetida: %s y %s", v, path.Base(m.up), name)
			}
			m.up = f
		case strings.HasSuffix(rest, ".down.sql"):
			if m.down != "" {
				t.Fatalf("la versión %d está repetida: %s y %s", v, path.Base(m.down), name)
			}
			m.down = f
		default:
			t.Fatalf("%s: debe terminar en .up.sql o .down.sql", name)
		}
	}
	list := make([]migrationPair, 0, len(byVersion))
	for _, m := range byVersion {
		if m.up == "" || m.down == "" {
			t.Errorf("la versión %d no tiene su par .up.sql y .down.sql", m.version)
		}
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].version < list[j].version })
	return list
}

func TestMigraciones_VersionUnicaYParUpDown(t *testing.T) {
	list := migrationsInOrder(t)

	if latest, err := LatestVersion(); err != nil || uint64(latest) != list[len(list)-1].version {
		t.Errorf("LatestVersion=%d no coincide con la última migración %d (error: %v)", latest, list[len(list)-1].version, err)
	}
}

func TestMigraciones_SeAplicanEnOrdenSobreUnaBaseDescartable(t *testing.T) {
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

	name := fmt.Sprintf("hu20a_migraciones_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("no se pudo crear la base descartable (el usuario necesita permiso CREATEDB): %v", err)
	}
	t.Cleanup(func() { _, _ = admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)") })

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("DATABASE_URL no es válida")
	}
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatalf("no se pudo abrir la base descartable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() }) // se ejecuta antes que el DROP

	exec := func(file string) error {
		b, err := fs.ReadFile(Migrations, file)
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, string(b))
		return err
	}
	tablas := func() int {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_tables WHERE schemaname = 'public'`).Scan(&n); err != nil {
			t.Fatalf("no se pudieron contar las tablas: %v", err)
		}
		return n
	}

	list := migrationsInOrder(t)
	for _, m := range list {
		if err := exec(m.up); err != nil {
			t.Fatalf("%s falló sobre una base vacía: revisá que sus dependencias tengan un número menor (%v)", path.Base(m.up), err)
		}
	}
	for _, tabla := range []string{"projects", "user_stories"} {
		var existe bool
		if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, tabla).Scan(&existe); err != nil || !existe {
			t.Errorf("después de migrar debe existir la tabla %s (error: %v)", tabla, err)
		}
	}

	// La semilla de integrantes se puede correr dos veces sin duplicar.
	semilla, err := os.ReadFile("seeds/members.sql")
	if err != nil {
		t.Fatalf("no se pudo leer seeds/members.sql: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := db.ExecContext(ctx, string(semilla)); err != nil {
			t.Fatalf("la semilla de integrantes falló: %v", err)
		}
	}
	var integrantes int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM members`).Scan(&integrantes); err != nil || integrantes != 5 {
		t.Errorf("la semilla debe dejar 5 integrantes sin duplicar, hay %d (error: %v)", integrantes, err)
	}

	// Las migraciones down, en orden inverso, dejan la base vacía.
	for i := len(list) - 1; i >= 0; i-- {
		if err := exec(list[i].down); err != nil {
			t.Fatalf("%s falló: %v", path.Base(list[i].down), err)
		}
	}
	if n := tablas(); n != 0 {
		t.Errorf("después de todas las migraciones down no deben quedar tablas, quedan %d", n)
	}
}
