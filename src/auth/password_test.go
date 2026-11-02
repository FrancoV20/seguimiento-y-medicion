package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// HU-27, T005: contraseñas (valor guardado, verificación y reglas de una contraseña nueva).

const anaPassword = "ana-clave-2026"

func testHash(t *testing.T, password string) string {
	t.Helper()
	stored, err := HashPasswordIterations(password, 1000)
	if err != nil {
		t.Fatalf("HashPasswordIterations: %v", err)
	}
	return stored
}

func TestHashPassword_FormatoYSalDistinta(t *testing.T) {
	first, second := testHash(t, anaPassword), testHash(t, anaPassword)
	if first == second {
		t.Fatal("dos valores de la misma contraseña son iguales; la sal no es aleatoria")
	}
	for _, stored := range []string{first, second} {
		parts := strings.Split(stored, "$")
		if len(parts) != 4 || parts[0] != "pbkdf2-sha256" || parts[1] != "1000" {
			t.Fatalf("valor guardado %q; se esperaba pbkdf2-sha256$1000$sal$hash", stored)
		}
		salt, err := base64.RawStdEncoding.DecodeString(parts[2])
		if err != nil || len(salt) != 16 {
			t.Errorf("sal %q: %d bytes, %v; se esperaban 16", parts[2], len(salt), err)
		}
		hash, err := base64.RawStdEncoding.DecodeString(parts[3])
		if err != nil || len(hash) != 32 {
			t.Errorf("hash %q: %d bytes, %v; se esperaban 32", parts[3], len(hash), err)
		}
		if strings.Contains(stored, anaPassword) {
			t.Error("el valor guardado contiene la contraseña")
		}
		if ok, err := VerifyPassword(anaPassword, stored); !ok || err != nil {
			t.Errorf("VerifyPassword = %v, %v; se esperaba true", ok, err)
		}
	}
}

func TestVerifyPassword_CorrectaEIncorrecta(t *testing.T) {
	stored := testHash(t, anaPassword)
	for _, tc := range []struct {
		password string
		want     bool
	}{{anaPassword, true}, {"otra-clave", false}, {"", false}} {
		if ok, err := VerifyPassword(tc.password, stored); ok != tc.want || err != nil {
			t.Errorf("VerifyPassword(%q) = %v, %v; se esperaba %v", tc.password, ok, err, tc.want)
		}
	}
}

func TestVerifyPassword_LeeLasIteracionesGuardadas(t *testing.T) {
	stored := testHash(t, anaPassword)
	if !strings.HasPrefix(stored, "pbkdf2-sha256$1000$") {
		t.Fatalf("valor guardado %q; se esperaban 1000 iteraciones", stored)
	}
	if ok, err := VerifyPassword(anaPassword, stored); !ok || err != nil {
		t.Errorf("un valor de 1.000 iteraciones no verifica con el código de producción: %v, %v", ok, err)
	}
}

func TestVerifyPassword_ValorGuardadoMalFormado(t *testing.T) {
	parts := strings.Split(testHash(t, anaPassword), "$")
	salt, hash := parts[2], parts[3]
	short := base64.RawStdEncoding.EncodeToString(make([]byte, 8))
	half := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	cases := map[string]string{
		"vacío":                 "",
		"texto":                 "abc",
		"3 partes":              "pbkdf2-sha256$1000$" + salt,
		"5 partes":              "pbkdf2-sha256$1000$" + salt + "$" + hash + "$extra",
		"esquema bcrypt":        "bcrypt$1000$" + salt + "$" + hash,
		"iteraciones 0":         "pbkdf2-sha256$0$" + salt + "$" + hash,
		"iteraciones -1":        "pbkdf2-sha256$-1$" + salt + "$" + hash,
		"iteraciones abc":       "pbkdf2-sha256$abc$" + salt + "$" + hash,
		"iteraciones 10000001":  "pbkdf2-sha256$10000001$" + salt + "$" + hash,
		"sal que no es base64":  "pbkdf2-sha256$1000$!!!$" + hash,
		"hash que no es base64": "pbkdf2-sha256$1000$" + salt + "$!!!",
		"sal de 8 bytes":        "pbkdf2-sha256$1000$" + short + "$" + hash,
		"hash de 16 bytes":      "pbkdf2-sha256$1000$" + salt + "$" + half,
	}
	for name, stored := range cases {
		if ok, err := VerifyPassword(anaPassword, stored); ok || !errors.Is(err, ErrMalformedHash) {
			t.Errorf("%s: VerifyPassword = %v, %v; se esperaba ErrMalformedHash", name, ok, err)
		}
	}
}

func TestHashPassword_IteracionesReales(t *testing.T) {
	stored, err := HashPassword(anaPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(stored, "pbkdf2-sha256$600000$") {
		t.Errorf("valor guardado %q; se esperaban 600000 iteraciones", stored)
	}
	if ok, err := VerifyPassword(anaPassword, stored); !ok || err != nil {
		t.Errorf("VerifyPassword = %v, %v; se esperaba true", ok, err)
	}
	if _, err := HashPasswordIterations(anaPassword, 0); err == nil {
		t.Error("HashPasswordIterations con 0 iteraciones debía fallar")
	}
}

func TestValidateNewPassword(t *testing.T) {
	long128, long129 := strings.Repeat("a", 128), strings.Repeat("a", 129)
	cases := []struct {
		name              string
		password, confirm string
		want              error
	}{
		{"7 caracteres", "corta12", "corta12", ErrPasswordTooShort},
		{"válida", "caro-clave-2026", "caro-clave-2026", nil},
		{"8 caracteres exactos", "12345678", "12345678", nil},
		{"8 runas y 10 bytes", "ñandú-12", "ñandú-12", nil},
		{"128 caracteres", long128, long128, nil},
		{"129 caracteres", long129, long129, ErrPasswordTooLong},
		{"no coinciden", "caro-clave-2026", "caro-clave-2027", ErrPasswordMismatch},
		{"el largo primero", "corta12", "otra", ErrPasswordTooShort},
		{"los espacios cuentan", " caro-clave-2026 ", " caro-clave-2026 ", nil},
	}
	for _, tc := range cases {
		if err := ValidateNewPassword(tc.password, tc.confirm); !errors.Is(err, tc.want) {
			t.Errorf("%s: error = %v; se esperaba %v", tc.name, err, tc.want)
		}
	}
	for err, text := range map[error]string{
		ErrPasswordTooShort: "La contraseña debe tener al menos 8 caracteres",
		ErrPasswordTooLong:  "La contraseña no puede superar 128 caracteres",
		ErrPasswordMismatch: "Las contraseñas no coinciden",
	} {
		if err.Error() != text {
			t.Errorf("texto = %q; se esperaba %q", err.Error(), text)
		}
	}
}
