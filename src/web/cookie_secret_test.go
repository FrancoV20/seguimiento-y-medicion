package web

import (
	"errors"
	"strings"
	"testing"
)

// Pruebas de la carga de COOKIE_SECRET (HU-26, T009-T010).

func envWith(values map[string]string) func(string) string {
	return func(k string) string { return values[k] }
}

func TestLoadCookieSecret_Valida(t *testing.T) {
	const secret = "una-clave-configurada-de-32-chars"
	got, temporary, err := LoadCookieSecret(envWith(map[string]string{EnvCookieSecret: "  " + secret + "  "}))
	if err != nil || temporary || string(got) != secret {
		t.Errorf("se esperaba la clave configurada, se obtuvo (%q, %v, %v)", got, temporary, err)
	}
}

func TestLoadCookieSecret_Ausente(t *testing.T) {
	for _, v := range []string{"", "   "} {
		a, tempA, errA := LoadCookieSecret(envWith(map[string]string{EnvCookieSecret: v}))
		b, _, _ := LoadCookieSecret(envWith(nil))
		if errA != nil || !tempA || len(a) != MinCookieSecretLength {
			t.Errorf("%q: se esperaba una clave temporal de %d bytes, se obtuvo (%d bytes, %v, %v)", v, MinCookieSecretLength, len(a), tempA, errA)
		}
		if string(a) == string(b) {
			t.Error("dos claves temporales no debían ser iguales")
		}
	}
}

func TestLoadCookieSecret_Corta(t *testing.T) {
	short := strings.Repeat("x", MinCookieSecretLength-1)
	_, _, err := LoadCookieSecret(envWith(map[string]string{EnvCookieSecret: short}))
	var shortErr *ShortCookieSecretError
	if !errors.As(err, &shortErr) {
		t.Fatalf("se esperaba ShortCookieSecretError, se obtuvo %v", err)
	}
	if !strings.Contains(err.Error(), "32") || !strings.Contains(err.Error(), EnvCookieSecret) {
		t.Errorf("el mensaje debe nombrar la variable y el largo mínimo: %q", err.Error())
	}
	if _, _, err := LoadCookieSecret(envWith(map[string]string{EnvCookieSecret: short + "x"})); err != nil {
		t.Errorf("con exactamente %d caracteres no debía fallar: %v", MinCookieSecretLength, err)
	}
}

func TestLoadCookieSecret_NoExponeLaClave(t *testing.T) {
	const secret = "MiClaveSecreta-123"
	_, _, err := LoadCookieSecret(envWith(map[string]string{EnvCookieSecret: secret}))
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Errorf("el error no debe contener la clave: %v", err)
	}
}
