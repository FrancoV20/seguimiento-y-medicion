package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Pruebas del firmador de cookies (HU-26, T004-T008, T011, T018).

var testSigner = NewCookieSigner([]byte("clave-de-prueba-de-32-caracteres!"))

// withCookieSecret fija la clave global durante la prueba y la restaura al terminar.
func withCookieSecret(t *testing.T, secret string) {
	t.Helper()
	old := cookieSigner.Load()
	t.Cleanup(func() { cookieSigner.Store(old) })
	SetCookieSecret([]byte(secret))
}

func TestCookieSigner_RoundTrip(t *testing.T) {
	signed := testSigner.Sign(CookieProject, "7")
	if !strings.HasPrefix(signed, "7.") || len(signed) <= len("7.") {
		t.Fatalf("formato inesperado: %q", signed)
	}
	if v, ok := testSigner.Verify(CookieProject, signed); !ok || v != "7" {
		t.Errorf("se esperaba (7, true), se obtuvo (%q, %v)", v, ok)
	}
}

func TestCookieSigner_ValorModificado(t *testing.T) {
	signed := testSigner.Sign(CookieProject, "7")
	tampered := "8" + signed[1:]
	if _, ok := testSigner.Verify(CookieProject, tampered); ok {
		t.Errorf("un valor modificado no debía aceptarse: %q", tampered)
	}
}

func TestCookieSigner_FirmaModificada(t *testing.T) {
	signed := testSigner.Sign(CookieMember, "12")
	start := len("12.")
	// Se cambia un carácter en distintas posiciones de la firma, incluido el último (que en
	// base64 sin relleno lleva bits de relleno: Strict los rechaza).
	for _, pos := range []int{start, start + 10, len(signed) - 1} {
		for _, repl := range []byte{'A', 'B', '_'} {
			if signed[pos] == repl {
				continue
			}
			tampered := signed[:pos] + string(repl) + signed[pos+1:]
			if _, ok := testSigner.Verify(CookieMember, tampered); ok {
				t.Errorf("una firma modificada en la posición %d no debía aceptarse: %q", pos, tampered)
			}
		}
	}
}

func TestCookieSigner_LigadoAlNombre(t *testing.T) {
	signed := testSigner.Sign(CookieProject, "7")
	if _, ok := testSigner.Verify(CookieMember, signed); ok {
		t.Error("la firma de ctx_proyecto no debía valer en ctx_integrante")
	}
}

func TestCookieSigner_SinFirma(t *testing.T) {
	if _, ok := testSigner.Verify(CookieProject, "7"); ok {
		t.Error("un valor sin firma (formato anterior) no debía aceptarse")
	}
}

func TestCookieSigner_MalFormada(t *testing.T) {
	for _, v := range []string{"", ".", "7.", ".abc", "abc", "7.!!!", "7.abc.def"} {
		if _, ok := testSigner.Verify(CookieProject, v); ok {
			t.Errorf("%q no debía aceptarse", v)
		}
	}
}

func TestCookieSigner_OtraClave(t *testing.T) {
	other := NewCookieSigner([]byte("otra-clave-distinta-de-32-caracteres"))
	if _, ok := testSigner.Verify(CookieProject, other.Sign(CookieProject, "7")); ok {
		t.Error("una firma hecha con otra clave no debía aceptarse")
	}
}

// T011: readIDCookie trata cualquier firma inválida como cookie manipulada: (0, true).
func TestReadIDCookie_FirmaInvalidaEsManipulada(t *testing.T) {
	withCookieSecret(t, "clave-de-prueba-de-32-caracteres!")
	signer := cookieSigner.Load()
	other := NewCookieSigner([]byte("otra-clave-distinta-de-32-caracteres"))
	firmaDe7 := signer.Sign(CookieProject, "7")

	cases := map[string]string{
		"valor cambiado":               "8" + firmaDe7[1:],
		"sin firma (formato anterior)": "7",
		"vacío":                        "",
		"solo punto":                   ".",
		"firma vacía":                  "7.",
		"valor vacío":                  ".abc",
		"texto":                        "abc",
		"firma de otra cookie":         signer.Sign(CookieMember, "7"),
		"firma de otra clave":          other.Sign(CookieProject, "7"),
		"firmado pero no numérico":     signer.Sign(CookieProject, "abc"),
		"firmado pero negativo":        signer.Sign(CookieProject, "-1"),
		"firmado con cero adelante":    signer.Sign(CookieProject, "05"),
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: CookieProject, Value: value})
			if id, present := readIDCookie(req, CookieProject); id != 0 || !present {
				t.Errorf("%q: se esperaba (0, true), se obtuvo (%d, %v)", value, id, present)
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieProject, Value: firmaDe7})
	if id, present := readIDCookie(req, CookieProject); id != 7 || !present {
		t.Errorf("firma correcta: se esperaba (7, true), se obtuvo (%d, %v)", id, present)
	}
}

// T018 (E6): con la misma clave después de reiniciar, la cookie sigue valiendo; con otra, no.
func TestHU26_ReinicioConMismaClave(t *testing.T) {
	const secret = "clave-configurada-de-al-menos-32-caracteres"
	withCookieSecret(t, secret)
	rec := httptest.NewRecorder()
	setContextCookie(rec, httptest.NewRequest(http.MethodGet, "/", nil), CookieProject, 42)
	emitted := rec.Result().Cookies()[0]

	read := func() (int64, bool) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: CookieProject, Value: emitted.Value})
		return readIDCookie(req, CookieProject)
	}

	SetCookieSecret([]byte(secret)) // reinicio con la misma clave
	if id, _ := read(); id != 42 {
		t.Errorf("con la misma clave se esperaba 42, se obtuvo %d", id)
	}
	SetCookieSecret([]byte("otra-clave-configurada-de-32-caracteres")) // reinicio con otra clave
	if id, present := read(); id != 0 || !present {
		t.Errorf("con otra clave se esperaba (0, true), se obtuvo (%d, %v)", id, present)
	}
}
