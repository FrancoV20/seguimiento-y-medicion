package web

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
)

// T009: PostgreSQLContextStore contra PostgreSQL real.

func openContextTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv(EnvDatabaseURL)
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

// contextFixture: dos proyectos; Ana y Beto en A, Carla solo en B.
type contextFixture struct {
	projectA, projectB int64
	nameA, nameB       string
	ana, beto, carla   int64
	anaName, betoName  string
	carlaName          string
}

func newContextFixture(t *testing.T, db *sql.DB) *contextFixture {
	t.Helper()
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	f := &contextFixture{
		nameA:     fmt.Sprintf("HU20b Proyecto A %d", stamp),
		nameB:     fmt.Sprintf("HU20b Proyecto B %d", stamp),
		anaName:   fmt.Sprintf("Ana %d", stamp),
		betoName:  fmt.Sprintf("Beto %d", stamp),
		carlaName: fmt.Sprintf("Carla %d", stamp),
	}
	// Se registra antes de insertar: limpia aunque la preparación falle a mitad.
	t.Cleanup(func() {
		for _, id := range []int64{f.projectA, f.projectB} {
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, id)
		}
		for _, id := range []int64{f.ana, f.beto, f.carla} {
			_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, id)
		}
	})

	insert := func(query, arg string) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRowContext(ctx, query, arg).Scan(&id); err != nil {
			t.Fatalf("no se pudo insertar el dato de prueba: %v", err)
		}
		return id
	}
	newProject := func(name string) int64 {
		return insert(`INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`, name)
	}
	newMember := func(name string) int64 {
		return insert(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
	}
	link := func(projectID, memberID int64) {
		t.Helper()
		if _, err := db.ExecContext(ctx,
			`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, projectID, memberID); err != nil {
			t.Fatalf("no se pudo vincular el integrante: %v", err)
		}
	}

	f.projectA = newProject(f.nameA)
	f.projectB = newProject(f.nameB)
	f.ana = newMember(f.anaName)
	f.beto = newMember(f.betoName)
	f.carla = newMember(f.carlaName)
	link(f.projectA, f.ana)
	link(f.projectA, f.beto)
	link(f.projectB, f.carla)
	return f
}

func TestPostgreSQLContextStore_ListProjects_MasRecientesPrimero(t *testing.T) {
	db := openContextTestDB(t)
	fx := newContextFixture(t, db)

	got, err := NewPostgreSQLContextStore(db).ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects devolvió error: %v", err)
	}
	posA, posB := -1, -1
	for i, p := range got {
		switch p.ID {
		case fx.projectA:
			posA = i
			if p.Name != fx.nameA {
				t.Errorf("nombre del proyecto A: se esperaba %q, se obtuvo %q", fx.nameA, p.Name)
			}
		case fx.projectB:
			posB = i
			if p.Name != fx.nameB {
				t.Errorf("nombre del proyecto B: se esperaba %q, se obtuvo %q", fx.nameB, p.Name)
			}
		}
	}
	if posA < 0 || posB < 0 {
		t.Fatalf("faltan proyectos en la lista (posA=%d, posB=%d)", posA, posB)
	}
	if posB > posA {
		t.Errorf("el proyecto más reciente (B) debía aparecer antes que A")
	}
}

func TestPostgreSQLContextStore_ListMembers_SoloLosDelProyecto(t *testing.T) {
	db := openContextTestDB(t)
	fx := newContextFixture(t, db)
	store := NewPostgreSQLContextStore(db)
	ctx := context.Background()

	a, err := store.ListMembers(ctx, fx.projectA)
	if err != nil {
		t.Fatalf("ListMembers(A) devolvió error: %v", err)
	}
	if len(a) != 2 || a[0].ID != fx.ana || a[0].Name != fx.anaName || a[1].ID != fx.beto {
		t.Errorf("proyecto A: se esperaban Ana y Beto (por nombre), se obtuvo %+v", a)
	}

	b, err := store.ListMembers(ctx, fx.projectB)
	if err != nil || len(b) != 1 || b[0].ID != fx.carla {
		t.Errorf("proyecto B: se esperaba solo Carla, se obtuvo %+v (error: %v)", b, err)
	}

	none, err := store.ListMembers(ctx, MaxID)
	if err != nil || len(none) != 0 {
		t.Errorf("proyecto inexistente: se esperaba lista vacía sin error, se obtuvo %+v (error: %v)", none, err)
	}
}

func TestPostgreSQLContextStore_ResolveContext(t *testing.T) {
	db := openContextTestDB(t)
	fx := newContextFixture(t, db)
	store := NewPostgreSQLContextStore(db)

	cases := []struct {
		name                string
		projectID, memberID int64
		want                ResolvedContext
	}{
		{"proyecto e integrante válidos", fx.projectA, fx.ana,
			ResolvedContext{ProjectExists: true, ProjectName: fx.nameA, MemberBelongs: true, MemberName: fx.anaName}},
		{"sin integrante", fx.projectA, 0,
			ResolvedContext{ProjectExists: true, ProjectName: fx.nameA}},
		{"integrante de otro proyecto", fx.projectA, fx.carla,
			ResolvedContext{ProjectExists: true, ProjectName: fx.nameA}},
		{"integrante inexistente", fx.projectA, MaxID,
			ResolvedContext{ProjectExists: true, ProjectName: fx.nameA}},
		{"integrante fuera de rango", fx.projectA, 99999999999,
			ResolvedContext{ProjectExists: true, ProjectName: fx.nameA}},
		{"proyecto inexistente", MaxID, fx.ana, ResolvedContext{}},
		{"proyecto negativo", -1, fx.ana, ResolvedContext{}},
		{"proyecto fuera de rango", 99999999999, fx.ana, ResolvedContext{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := store.ResolveContext(context.Background(), tc.projectID, tc.memberID)
			if err != nil {
				t.Fatalf("no debía haber error, se obtuvo: %v", err)
			}
			if got != tc.want {
				t.Errorf("se esperaba %+v, se obtuvo %+v", tc.want, got)
			}
		})
	}
}

func TestPostgreSQLContextStore_BaseCaidaEsNoDisponible(t *testing.T) {
	db := openContextTestDB(t)
	_ = db.Close() // simula la caída: toda consulta falla

	_, err := NewPostgreSQLContextStore(db).ResolveContext(context.Background(), 1, 1)

	if err == nil || KindOf(err) != KindUnavailable {
		t.Errorf("con la base caída se esperaba un error KindUnavailable, se obtuvo %v", err)
	}
}

func TestPostgreSQLContextStore_ResolveContext_MenosDe50ms(t *testing.T) {
	db := openContextTestDB(t)
	fx := newContextFixture(t, db)
	store := NewPostgreSQLContextStore(db)
	ctx := context.Background()
	if _, err := store.ResolveContext(ctx, fx.projectA, fx.ana); err != nil { // calienta la conexión
		t.Fatalf("ResolveContext devolvió error: %v", err)
	}

	const runs = 20
	start := time.Now()
	for i := 0; i < runs; i++ {
		if _, err := store.ResolveContext(ctx, fx.projectA, fx.ana); err != nil {
			t.Fatalf("ResolveContext devolvió error: %v", err)
		}
	}
	if avg := time.Since(start) / runs; avg >= 50*time.Millisecond {
		t.Errorf("SC-012: el promedio fue %v, debía ser menor a 50 ms", avg)
	}
}

// HU-27, T012: MemberCredential con «Alfa» (Ana con contraseña, Caro sin) y «Beta» (Dani).
type credentialFixture struct {
	alfa, beta      int64
	ana, caro, dani int64
	anaHash         string
}

func newCredentialFixture(t *testing.T, db *sql.DB) *credentialFixture {
	t.Helper()
	ctx := context.Background()
	f := &credentialFixture{}
	t.Cleanup(func() {
		for _, p := range []int64{f.alfa, f.beta} {
			_, _ = db.Exec(`DELETE FROM project_members WHERE project_id = $1`, p)
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, p)
		}
		for _, m := range []int64{f.ana, f.caro, f.dani} {
			_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, m)
		}
	})
	insert := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			t.Fatalf("no se pudo insertar el dato de prueba: %v", err)
		}
		return id
	}
	stamp := time.Now().UnixNano()
	project := func(name string) int64 {
		return insert(`INSERT INTO projects (name, start_date, end_date) VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`,
			fmt.Sprintf("%s %d", name, stamp))
	}
	member := func(name string, hash any) int64 {
		return insert(`INSERT INTO members (display_name, password_hash) VALUES ($1, $2) RETURNING id`, name, hash)
	}
	var err error
	if f.anaHash, err = auth.HashPasswordIterations("ana-clave-2026", 1000); err != nil {
		t.Fatal(err)
	}
	f.alfa, f.beta = project("HU27 Alfa"), project("HU27 Beta")
	f.ana, f.caro, f.dani = member("Ana", f.anaHash), member("Caro", nil), member("Dani", nil)
	for _, link := range [][2]int64{{f.alfa, f.ana}, {f.alfa, f.caro}, {f.beta, f.dani}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, link[0], link[1]); err != nil {
			t.Fatalf("no se pudo vincular el integrante: %v", err)
		}
	}
	return f
}

func TestPostgreSQLContextStore_MemberCredential(t *testing.T) {
	db := openContextTestDB(t)
	f := newCredentialFixture(t, db)
	store := NewPostgreSQLContextStore(db)
	cases := []struct {
		name              string
		projectID, member int64
		want              MemberCredential
	}{
		{"«Alfa», Ana", f.alfa, f.ana, MemberCredential{Belongs: true, Name: "Ana", PasswordHash: f.anaHash}},
		{"«Alfa», Caro", f.alfa, f.caro, MemberCredential{Belongs: true, Name: "Caro"}},
		{"«Alfa», Dani (de «Beta»)", f.alfa, f.dani, MemberCredential{}},
		{"«Alfa», 0", f.alfa, 0, MemberCredential{}},
		{"«Alfa», 2147483648", f.alfa, 2147483648, MemberCredential{}},
		{"proyecto inexistente, Ana", 2147483647, f.ana, MemberCredential{}},
	}
	for _, tc := range cases {
		got, err := store.MemberCredential(context.Background(), tc.projectID, tc.member)
		if err != nil || got != tc.want {
			t.Errorf("%s: MemberCredential = %+v, %v; se esperaba %+v", tc.name, got, err, tc.want)
		}
	}
}

func TestPostgreSQLContextStore_MemberCredentialBaseCaida(t *testing.T) {
	db := openContextTestDB(t)
	f := newCredentialFixture(t, db)
	closed, err := sql.Open("pgx", os.Getenv(EnvDatabaseURL))
	if err != nil {
		t.Fatal(err)
	}
	_ = closed.Close()
	_, err = NewPostgreSQLContextStore(closed).MemberCredential(context.Background(), f.alfa, f.ana)
	if KindOf(err) != KindUnavailable {
		t.Errorf("con la base cerrada: error %v (tipo %v); se esperaba KindUnavailable", err, KindOf(err))
	}
}
