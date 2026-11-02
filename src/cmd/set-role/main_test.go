package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/role"
)

// HU-28, T025: comando set-role (escenarios 10 y 11).

const msgUsageText = `Uso: go run ./src/cmd/set-role -proyecto P -integrante N -rol "Scrum Master"`

// fakeRoleStore es «Alfa» (1) con Ana (10, Scrum Master) y Beto (11); Dani (20) es de «Beta» (2).
type fakeRoleStore struct {
	projects map[int64]string
	members  map[[2]int64]string // {proyecto, integrante} -> nombre
	roles    map[[2]int64]role.Role
	setCalls int
	readErr  error
	saveErr  error
}

func newFakeRoleStore() *fakeRoleStore {
	return &fakeRoleStore{
		projects: map[int64]string{1: "Alfa", 2: "Beta"},
		members:  map[[2]int64]string{{1, 10}: "Ana", {1, 11}: "Beto", {2, 20}: "Dani"},
		roles:    map[[2]int64]role.Role{{1, 10}: role.RoleScrumMaster, {1, 11}: role.RoleMember, {2, 20}: role.RoleMember},
	}
}

func (s *fakeRoleStore) ProjectName(_ context.Context, projectID int64) (string, error) {
	if s.readErr != nil {
		return "", s.readErr
	}
	name, ok := s.projects[projectID]
	if !ok {
		return "", role.ErrProjectNotFound
	}
	return name, nil
}

func (s *fakeRoleStore) SetMemberRole(_ context.Context, projectID, memberID int64, r role.Role) (string, error) {
	s.setCalls++
	if s.saveErr != nil {
		return "", s.saveErr
	}
	key := [2]int64{projectID, memberID}
	name, ok := s.members[key]
	if !ok {
		return "", role.ErrMemberNotInProject
	}
	s.roles[key] = r
	return name, nil
}

func (s *fakeRoleStore) snapshot() map[[2]int64]role.Role {
	out := make(map[[2]int64]role.Role, len(s.roles))
	for k, v := range s.roles {
		out[k] = v
	}
	return out
}

func sameRoleMaps(a, b map[[2]int64]role.Role) bool {
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

func TestAssignRole_AsignaScrumMaster(t *testing.T) {
	cases := []struct {
		name             string
		project, member  int64
		r                role.Role
		want             string
		key              [2]int64
		wantRoleAfterSet role.Role
	}{
		{"Beto pasa a Scrum Master", 1, 11, role.RoleScrumMaster, "Beto ahora es Scrum Master de «Alfa»", [2]int64{1, 11}, role.RoleScrumMaster},
		{"Ana pasa a Integrante", 1, 10, role.RoleMember, "Ana ahora es Integrante de «Alfa»", [2]int64{1, 10}, role.RoleMember},
		{"Ana ya era Scrum Master", 1, 10, role.RoleScrumMaster, "Ana ahora es Scrum Master de «Alfa»", [2]int64{1, 10}, role.RoleScrumMaster},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newFakeRoleStore()
			var out bytes.Buffer
			code := assignRole(context.Background(), store, tc.project, tc.member, tc.r, &out)
			if code != 0 || strings.TrimSpace(out.String()) != tc.want {
				t.Fatalf("código %d, salida %q; se esperaba 0 y %q", code, out.String(), tc.want)
			}
			if store.roles[tc.key] != tc.wantRoleAfterSet {
				t.Errorf("rol guardado = %q; se esperaba %q", store.roles[tc.key], tc.wantRoleAfterSet)
			}
		})
	}
}

func TestAssignRole_Rechazos(t *testing.T) {
	cases := []struct {
		name            string
		project, member int64
		store           func() *fakeRoleStore
		want            string
		wantSetCalls    int
	}{
		{"proyecto inexistente", 9, 11, newFakeRoleStore, "El proyecto no existe", 0},
		{"integrante de otro proyecto", 1, 20, newFakeRoleStore, "El integrante no pertenece al proyecto", 1},
		{"integrante inexistente", 1, 99, newFakeRoleStore, "El integrante no pertenece al proyecto", 1},
		{"falla al leer el proyecto", 1, 11, func() *fakeRoleStore {
			s := newFakeRoleStore()
			s.readErr = errors.New("conexión rechazada en 10.0.0.1")
			return s
		}, "No se pudo acceder a la base de datos", 0},
		{"falla al guardar", 1, 11, func() *fakeRoleStore {
			s := newFakeRoleStore()
			s.saveErr = errors.New("conexión rechazada en 10.0.0.1")
			return s
		}, "No se pudo acceder a la base de datos", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := tc.store()
			before := store.snapshot()
			var out bytes.Buffer
			code := assignRole(context.Background(), store, tc.project, tc.member, role.RoleScrumMaster, &out)
			if code != 1 || strings.TrimSpace(out.String()) != tc.want {
				t.Errorf("código %d, salida %q; se esperaba 1 y solo %q", code, out.String(), tc.want)
			}
			if store.setCalls != tc.wantSetCalls {
				t.Errorf("SetMemberRole se llamó %d veces; se esperaban %d", store.setCalls, tc.wantSetCalls)
			}
			if !sameRoleMaps(before, store.roles) {
				t.Error("un rechazo cambió los roles")
			}
			if strings.Contains(out.String(), "10.0.0.1") {
				t.Error("la salida muestra detalles internos de la falla")
			}
		})
	}
}

// envRecorder devuelve los valores dados y anota si se leyó DATABASE_URL.
type envRecorder struct {
	values map[string]string
	read   bool
}

func (e *envRecorder) get(k string) string {
	if k == "DATABASE_URL" {
		e.read = true
	}
	return e.values[k]
}

func TestRun_UsoYEntorno(t *testing.T) {
	ctx := context.Background()
	valid := []string{"-proyecto", "1", "-integrante", "11", "-rol", "Scrum Master"}

	t.Run("opciones faltantes o números inválidos", func(t *testing.T) {
		usage := [][]string{
			nil,
			{"-integrante", "11", "-rol", "Scrum Master"},
			{"-proyecto", "1", "-rol", "Scrum Master"},
			{"-proyecto", "1", "-integrante", "11"},
			{"-proyecto", "abc", "-integrante", "11", "-rol", "Scrum Master"},
			{"-proyecto", "0", "-integrante", "11", "-rol", "Scrum Master"},
			{"-proyecto", "-1", "-integrante", "11", "-rol", "Scrum Master"},
			{"-proyecto", "2147483648", "-integrante", "11", "-rol", "Scrum Master"},
			{"-proyecto", "1", "-integrante", "x", "-rol", "Scrum Master"},
			{"-proyecto", "1", "-integrante", "0", "-rol", "Scrum Master"},
			{"-proyecto", "1", "-integrante", "2147483648", "-rol", "Scrum Master"},
			{"-proyecto", "1", "-integrante", "11", "-rol", "Scrum Master", "sobra"},
			{"-otro", "1"},
		}
		for _, args := range usage {
			env := &envRecorder{values: map[string]string{"DATABASE_URL": "postgres://u:p@127.0.0.1:1/b"}}
			var out bytes.Buffer
			if code := run(ctx, args, &out, env.get); code != 2 || strings.TrimSpace(out.String()) != msgUsageText {
				t.Errorf("%q: código %d, salida %q; se esperaba 2 y el uso", args, code, out.String())
			}
			if env.read {
				t.Errorf("%q: se leyó DATABASE_URL; con un uso incorrecto no debía conectarse", args)
			}
		}
	})

	t.Run("rol no válido, sin conectarse", func(t *testing.T) {
		for _, r := range []string{"Product Owner", "scrum master", "SM", "", "   "} {
			env := &envRecorder{values: map[string]string{"DATABASE_URL": "postgres://u:p@127.0.0.1:1/b"}}
			var out bytes.Buffer
			args := []string{"-proyecto", "1", "-integrante", "11", "-rol", r}
			if code := run(ctx, args, &out, env.get); code != 1 || strings.TrimSpace(out.String()) != "El rol no es válido" {
				t.Errorf("-rol %q: código %d, salida %q; se esperaba 1 y «El rol no es válido»", r, code, out.String())
			}
			if env.read {
				t.Errorf("-rol %q: se leyó DATABASE_URL antes de validar el rol", r)
			}
		}
	})

	t.Run("sin DATABASE_URL", func(t *testing.T) {
		var out bytes.Buffer
		env := &envRecorder{}
		if code := run(ctx, valid, &out, env.get); code != 1 ||
			strings.TrimSpace(out.String()) != "falta la variable de entorno obligatoria DATABASE_URL" {
			t.Errorf("código %d, salida %q; se esperaba 1 y el mensaje de LoadConfig", code, out.String())
		}
	})

	t.Run("base inalcanzable", func(t *testing.T) {
		var out bytes.Buffer
		unreachable := "postgres://usuario:secreta@127.0.0.1:1/base?sslmode=disable&connect_timeout=2"
		env := &envRecorder{values: map[string]string{"DATABASE_URL": unreachable}}
		if code := run(ctx, valid, &out, env.get); code != 1 ||
			strings.TrimSpace(out.String()) != "No se pudo acceder a la base de datos" ||
			strings.Contains(out.String(), "127.0.0.1") || strings.Contains(out.String(), "secreta") {
			t.Errorf("código %d, salida %q; se esperaba 1 y el mensaje sin la dirección", code, out.String())
		}
	})
}

func TestRun_AsignaRol(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	id := func(q string, args ...any) int64 {
		t.Helper()
		var v int64
		if err := db.QueryRow(q, args...).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	var alfa, beta, beto, dani int64
	t.Cleanup(func() {
		for _, p := range []int64{alfa, beta} {
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, p)
		}
		for _, m := range []int64{beto, dani} {
			_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, m)
		}
	})
	alfa = id(`INSERT INTO projects (name, start_date, end_date) VALUES ('Alfa', DATE '2026-01-01', DATE '2026-12-31') RETURNING id`)
	beta = id(`INSERT INTO projects (name, start_date, end_date) VALUES ('Beta', DATE '2026-01-01', DATE '2026-12-31') RETURNING id`)
	beto = id(`INSERT INTO members (display_name) VALUES ('Beto') RETURNING id`)
	dani = id(`INSERT INTO members (display_name) VALUES ('Dani') RETURNING id`)
	for _, link := range [][2]int64{{alfa, beto}, {beta, dani}} {
		if _, err := db.Exec(`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`, link[0], link[1]); err != nil {
			t.Fatal(err)
		}
	}
	roleOf := func(projectID, memberID int64) string {
		t.Helper()
		var r string
		if err := db.QueryRow(`SELECT role FROM project_members WHERE project_id = $1 AND member_id = $2`, projectID, memberID).Scan(&r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	args := func(project, member int64, r string) []string {
		return []string{"-proyecto", strconv.FormatInt(project, 10), "-integrante", strconv.FormatInt(member, 10), "-rol", r}
	}

	// Dado que Beto es "Integrante" de «Alfa», cuando se ejecuta el comando para darle "Scrum Master"
	var out bytes.Buffer
	code := run(context.Background(), args(alfa, beto, "Scrum Master"), &out, os.Getenv)
	if code != 0 || strings.TrimSpace(out.String()) != "Beto ahora es Scrum Master de «Alfa»" {
		t.Fatalf("código %d, salida %q; se esperaba 0 y «Beto ahora es Scrum Master de «Alfa»»", code, out.String())
	}
	if got := roleOf(alfa, beto); got != "Scrum Master" {
		t.Errorf("rol de Beto = %q; se esperaba Scrum Master", got)
	}

	// Un integrante que no es del proyecto o un proyecto inexistente: informa el motivo y no cambia nada.
	out.Reset()
	if code := run(context.Background(), args(alfa, dani, "Scrum Master"), &out, os.Getenv); code != 1 ||
		strings.TrimSpace(out.String()) != "El integrante no pertenece al proyecto" {
		t.Errorf("Dani en «Alfa»: código %d, salida %q", code, out.String())
	}
	if got := roleOf(beta, dani); got != "Integrante" {
		t.Errorf("rol de Dani en «Beta» = %q; no debía cambiar", got)
	}
	var missing int64
	if err := db.QueryRow(`SELECT COALESCE(MAX(id), 0) + 1000 FROM projects`).Scan(&missing); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run(context.Background(), args(missing, beto, "Integrante"), &out, os.Getenv); code != 1 ||
		strings.TrimSpace(out.String()) != "El proyecto no existe" {
		t.Errorf("proyecto inexistente: código %d, salida %q", code, out.String())
	}
	if got := roleOf(alfa, beto); got != "Scrum Master" {
		t.Errorf("rol de Beto = %q; un rechazo no debía cambiarlo", got)
	}
}
