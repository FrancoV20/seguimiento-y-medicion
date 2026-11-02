package role

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"
)

// HU-28, T007: PostgreSQLStore contra PostgreSQL real, con «Alfa» (Ana, Beto, Caro) y «Beta» (Dani)
// (escenarios 10 y 11).

func openRoleTestDB(t *testing.T) *sql.DB {
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

type roleFixture struct {
	alfa, beta, missingProject      int64
	ana, beto, caro, dani, noMember int64
}

// newRoleFixture crea «Alfa» con Ana, Beto y Caro y «Beta» con Dani, todos como 'Integrante'
// salvo Ana, que es 'Scrum Master'. También reserva un proyecto y un integrante que ya no existen.
func newRoleFixture(t *testing.T, db *sql.DB) roleFixture {
	t.Helper()
	var f roleFixture
	project := func(name string) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(`INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, '2026-03-01', '2026-12-31') RETURNING id`, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, id) })
		return id
	}
	member := func(name string) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM members WHERE id = $1`, id) })
		return id
	}
	join := func(projectID, memberID int64) {
		t.Helper()
		if _, err := db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, projectID, memberID); err != nil {
			t.Fatal(err)
		}
	}

	f.alfa, f.beta = project("Alfa"), project("Beta")
	f.ana, f.beto, f.caro, f.dani = member("Ana"), member("Beto"), member("Caro"), member("Dani")
	join(f.alfa, f.ana)
	join(f.alfa, f.beto)
	join(f.alfa, f.caro)
	join(f.beta, f.dani)
	if _, err := db.Exec(`UPDATE project_members SET role = 'Scrum Master' WHERE project_id = $1 AND member_id = $2`,
		f.alfa, f.ana); err != nil {
		t.Fatal(err)
	}

	f.missingProject, f.noMember = project("Borrado"), member("Borrado")
	if _, err := db.Exec(`DELETE FROM projects WHERE id = $1`, f.missingProject); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM members WHERE id = $1`, f.noMember); err != nil {
		t.Fatal(err)
	}
	return f
}

// rolesOf devuelve el rol de cada integrante del proyecto.
func rolesOf(t *testing.T, db *sql.DB, projectID int64) map[int64]string {
	t.Helper()
	rows, err := db.Query(`SELECT member_id, role FROM project_members WHERE project_id = $1`, projectID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var r string
		if err := rows.Scan(&id, &r); err != nil {
			t.Fatal(err)
		}
		out[id] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func sameRoles(a, b map[int64]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestPostgreSQLStore(t *testing.T) {
	db := openRoleTestDB(t)
	ctx := context.Background()
	f := newRoleFixture(t, db)
	var store Store = NewPostgreSQLStore(db)

	t.Run("ProjectName", func(t *testing.T) {
		if name, err := store.ProjectName(ctx, f.alfa); err != nil || name != "Alfa" {
			t.Errorf("ProjectName(Alfa) = %q, %v; se esperaba Alfa", name, err)
		}
		for _, id := range []int64{f.missingProject, 0, -1, 2147483648} {
			if _, err := store.ProjectName(ctx, id); !errors.Is(err, ErrProjectNotFound) {
				t.Errorf("ProjectName(%d) error = %v; se esperaba ErrProjectNotFound", id, err)
			}
		}
	})

	t.Run("SetMemberRole asigna Scrum Master a Beto", func(t *testing.T) {
		before := rolesOf(t, db, f.alfa)
		name, err := store.SetMemberRole(ctx, f.alfa, f.beto, RoleScrumMaster)
		if err != nil || name != "Beto" {
			t.Fatalf("SetMemberRole(Alfa, Beto, Scrum Master) = %q, %v; se esperaba Beto", name, err)
		}
		after := rolesOf(t, db, f.alfa)
		if after[f.beto] != "Scrum Master" {
			t.Errorf("rol de Beto = %q; se esperaba Scrum Master", after[f.beto])
		}
		if after[f.ana] != before[f.ana] || after[f.caro] != before[f.caro] {
			t.Errorf("cambiaron otros integrantes: antes %v, después %v", before, after)
		}
	})

	t.Run("SetMemberRole devuelve a Ana a Integrante", func(t *testing.T) {
		name, err := store.SetMemberRole(ctx, f.alfa, f.ana, RoleMember)
		if err != nil || name != "Ana" {
			t.Fatalf("SetMemberRole(Alfa, Ana, Integrante) = %q, %v; se esperaba Ana", name, err)
		}
		if got := rolesOf(t, db, f.alfa)[f.ana]; got != "Integrante" {
			t.Errorf("rol de Ana = %q; se esperaba Integrante", got)
		}
	})

	t.Run("integrante que no es del proyecto", func(t *testing.T) {
		alfaBefore, betaBefore := rolesOf(t, db, f.alfa), rolesOf(t, db, f.beta)
		for _, id := range []int64{f.dani, f.noMember, 0, -1, 2147483648} {
			name, err := store.SetMemberRole(ctx, f.alfa, id, RoleScrumMaster)
			if !errors.Is(err, ErrMemberNotInProject) || name != "" {
				t.Errorf("SetMemberRole(Alfa, %d) = %q, %v; se esperaba ErrMemberNotInProject", id, name, err)
			}
		}
		if !sameRoles(rolesOf(t, db, f.alfa), alfaBefore) || !sameRoles(rolesOf(t, db, f.beta), betaBefore) {
			t.Error("un rechazo cambió los roles de «Alfa» o «Beta»")
		}
	})

	t.Run("base cerrada", func(t *testing.T) {
		closed, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		_ = closed.Close()
		down := NewPostgreSQLStore(closed)
		if _, err := down.ProjectName(ctx, f.alfa); err == nil || errors.Is(err, ErrProjectNotFound) || errors.Is(err, ErrMemberNotInProject) {
			t.Errorf("ProjectName con la base cerrada: error = %v; se esperaba una falla distinta de los dos rechazos", err)
		}
		if _, err := down.SetMemberRole(ctx, f.alfa, f.caro, RoleScrumMaster); err == nil ||
			errors.Is(err, ErrProjectNotFound) || errors.Is(err, ErrMemberNotInProject) {
			t.Errorf("SetMemberRole con la base cerrada: error = %v; se esperaba una falla distinta de los dos rechazos", err)
		}
		if got := rolesOf(t, db, f.alfa)[f.caro]; got != "Integrante" {
			t.Errorf("rol de Caro = %q; con la base cerrada no debía cambiar", got)
		}
	})
}
