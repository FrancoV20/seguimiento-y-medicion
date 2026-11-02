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

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
)

// HU-27, T019: comando set-password.

type fakePasswordStore struct {
	names    map[int64]string
	saved    map[int64]string
	setCalls int
	readErr  error
	saveErr  error
}

func newFakeStore() *fakePasswordStore {
	return &fakePasswordStore{names: map[int64]string{3: "Caro"}, saved: map[int64]string{}}
}

func (s *fakePasswordStore) MemberName(_ context.Context, id int64) (string, error) {
	if s.readErr != nil {
		return "", s.readErr
	}
	name, ok := s.names[id]
	if !ok {
		return "", auth.ErrMemberNotFound
	}
	return name, nil
}

func (s *fakePasswordStore) SetPasswordHash(_ context.Context, id int64, hash string) error {
	s.setCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved[id] = hash
	return nil
}

func TestSetPassword_Configura(t *testing.T) {
	store := newFakeStore()
	var out bytes.Buffer
	code := setPassword(context.Background(), store, 3, strings.NewReader("caro-clave-2026\r\ncaro-clave-2026\n"), &out)
	if code != 0 || !strings.Contains(out.String(), "Contraseña configurada para Caro") {
		t.Fatalf("código %d, salida %q; se esperaba 0 y «Contraseña configurada para Caro»", code, out.String())
	}
	if ok, err := auth.VerifyPassword("caro-clave-2026", store.saved[3]); !ok || err != nil {
		t.Errorf("el valor guardado no verifica: %v, %v", ok, err)
	}
	if strings.Contains(out.String(), "caro-clave-2026") {
		t.Error("la salida muestra la contraseña")
	}
}

func TestSetPassword_ContrasenaInvalida(t *testing.T) {
	long := strings.Repeat("a", 129)
	cases := map[string]struct{ input, want string }{
		"corta":               {"corta12\ncorta12\n", "La contraseña debe tener al menos 8 caracteres"},
		"129 caracteres":      {long + "\n" + long + "\n", "La contraseña no puede superar 128 caracteres"},
		"distintas":           {"caro-clave-2026\ncaro-clave-2027\n", "Las contraseñas no coinciden"},
		"entrada vacía":       {"", "La contraseña debe tener al menos 8 caracteres"},
		"sin la confirmación": {"caro-clave-2026\n", "Las contraseñas no coinciden"},
	}
	for name, tc := range cases {
		store := newFakeStore()
		var out bytes.Buffer
		code := setPassword(context.Background(), store, 3, strings.NewReader(tc.input), &out)
		if code != 1 || !strings.Contains(out.String(), tc.want) {
			t.Errorf("%s: código %d, salida %q; se esperaba 1 y %q", name, code, out.String(), tc.want)
		}
		if store.setCalls != 0 {
			t.Errorf("%s: se llamó a SetPasswordHash", name)
		}
	}
}

func TestSetPassword_IntegranteInexistente(t *testing.T) {
	store := newFakeStore()
	var out bytes.Buffer
	input := strings.NewReader("caro-clave-2026\ncaro-clave-2026\n")
	code := setPassword(context.Background(), store, 99, input, &out)
	if code != 1 || strings.TrimSpace(out.String()) != "El integrante no existe" {
		t.Errorf("código %d, salida %q; se esperaba 1 y solo «El integrante no existe»", code, out.String())
	}
	if input.Len() == 0 {
		t.Error("se leyó la contraseña de un integrante que no existe")
	}
}

func TestSetPassword_ErrorAlGuardar(t *testing.T) {
	for name, store := range map[string]*fakePasswordStore{
		"al leer":    {readErr: errors.New("conexión rechazada en 10.0.0.1")},
		"al guardar": {names: map[int64]string{3: "Caro"}, saveErr: errors.New("conexión rechazada en 10.0.0.1")},
	} {
		var out bytes.Buffer
		code := setPassword(context.Background(), store, 3, strings.NewReader("caro-clave-2026\ncaro-clave-2026\n"), &out)
		if code != 1 || !strings.Contains(out.String(), "No se pudo acceder a la base de datos") || strings.Contains(out.String(), "10.0.0.1") {
			t.Errorf("%s: código %d, salida %q; se esperaba 1 y el mensaje sin detalles", name, code, out.String())
		}
	}
}

func TestRun_UsoYEntorno(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(k string) string { return values[k] }
	}
	for _, args := range [][]string{nil, {"-integrante", "abc"}, {"-integrante", "0"}, {"-integrante", "-1"}, {"-integrante", "2147483648"}, {"-otro", "1"}} {
		var out bytes.Buffer
		if code := run(context.Background(), args, strings.NewReader(""), &out, env(nil)); code != 2 ||
			!strings.Contains(out.String(), "Uso: go run ./src/cmd/set-password -integrante N") {
			t.Errorf("%v: código %d, salida %q; se esperaba 2 y el uso", args, code, out.String())
		}
	}

	var out bytes.Buffer
	if code := run(context.Background(), []string{"-integrante", "3"}, strings.NewReader(""), &out, env(nil)); code != 1 ||
		!strings.Contains(out.String(), "falta la variable de entorno obligatoria DATABASE_URL") {
		t.Errorf("sin DATABASE_URL: código %d, salida %q", code, out.String())
	}

	out.Reset()
	unreachable := "postgres://usuario:secreta@127.0.0.1:1/base?sslmode=disable&connect_timeout=2"
	if code := run(context.Background(), []string{"-integrante", "3"}, strings.NewReader(""), &out,
		env(map[string]string{"DATABASE_URL": unreachable})); code != 1 ||
		!strings.Contains(out.String(), "No se pudo acceder a la base de datos") ||
		strings.Contains(out.String(), "127.0.0.1") || strings.Contains(out.String(), "secreta") {
		t.Errorf("base inalcanzable: código %d, salida %q; se esperaba 1 sin la dirección", code, out.String())
	}
}

func TestRun_ConfiguraContrasena(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var caro int64
	if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ('Caro') RETURNING id`).Scan(&caro); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM members WHERE id = $1`, caro) })

	var out bytes.Buffer
	code := run(context.Background(), []string{"-integrante", strconv.FormatInt(caro, 10)},
		strings.NewReader("caro-clave-2026\ncaro-clave-2026\n"), &out, os.Getenv)
	if code != 0 || !strings.Contains(out.String(), "Contraseña configurada para Caro") {
		t.Fatalf("código %d, salida %q", code, out.String())
	}
	var stored string
	if err := db.QueryRow(`SELECT password_hash FROM members WHERE id = $1`, caro).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "pbkdf2-sha256$600000$") {
		t.Errorf("password_hash = %q; se esperaba pbkdf2-sha256$600000$…", stored)
	}
	if ok, err := auth.VerifyPassword("caro-clave-2026", stored); !ok || err != nil {
		t.Errorf("el valor guardado no verifica: %v, %v", ok, err)
	}
}
