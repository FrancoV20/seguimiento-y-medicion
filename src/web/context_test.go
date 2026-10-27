package web

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadIDCookie_SinCookieEsAusencia(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	id, present := readIDCookie(req, CookieProject)

	if id != 0 || present {
		t.Errorf("sin cookie se esperaba (0, false), se obtuvo (%d, %v)", id, present)
	}
}

func TestReadIDCookie_Valores(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  int64
	}{
		{"valido", "7", 7},
		{"maximo", "2147483647", 2147483647},
		{"vacio", "", 0},
		{"no numerico", "abc", 0},
		{"cero", "0", 0},
		{"negativo", "-1", 0},
		{"muy grande", "99999999999", 0},
		{"justo fuera de rango", "2147483648", 0},
		{"con signo", "+5", 0},
		{"con ceros a la izquierda", "05", 0},
		{"decimal", "1.5", 0},
		{"etiqueta HTML", "<script>", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			// HU-26: el valor viaja firmado; parseID sigue rechazando los valores inválidos.
			req.AddCookie(&http.Cookie{Name: CookieProject, Value: cookieSigner.Load().Sign(CookieProject, tc.value)})

			id, present := readIDCookie(req, CookieProject)

			if id != tc.want {
				t.Errorf("valor %q: se esperaba id %d, se obtuvo %d", tc.value, tc.want, id)
			}
			if !present {
				t.Errorf("valor %q: la cookie estaba presente y se informó ausente", tc.value)
			}
		})
	}
}

func TestSetContextCookie_AtributosDeSesion(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	setContextCookie(rec, req, CookieMember, 42)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("se esperaba 1 cookie, hay %d", len(cookies))
	}
	c := cookies[0]
	if v, ok := verifyContextValue(CookieMember, c.Value); c.Name != CookieMember || !ok || v != "42" {
		t.Errorf("cookie inesperada (se esperaba 42 firmado): %s=%s", c.Name, c.Value)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Errorf("faltan HttpOnly, SameSite=Lax o Path=/: %+v", c)
	}
	if !c.Expires.IsZero() || c.MaxAge != 0 {
		t.Errorf("debe ser de sesión (sin Expires ni Max-Age): %+v", c)
	}
	if c.Secure {
		t.Errorf("por HTTP no debe llevar Secure")
	}
}

func TestSetContextCookie_SecureConHTTPS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	setContextCookie(rec, req, CookieProject, 1)

	if c := rec.Result().Cookies()[0]; !c.Secure {
		t.Errorf("por HTTPS la cookie debe llevar Secure")
	}
}

func TestClearContextCookie_Expira(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	clearContextCookie(rec, req, CookieProject)

	c := rec.Result().Cookies()[0]
	if c.Name != CookieProject || c.Value != "" || c.MaxAge >= 0 {
		t.Errorf("la cookie debía quedar vacía y vencida: %+v", c)
	}
	if !c.HttpOnly || c.Path != "/" {
		t.Errorf("al borrar se conservan HttpOnly y Path=/ para que el navegador la reemplace: %+v", c)
	}
}

// HU-26, T014: readIDCookie trata cualquier firma inválida como cookie manipulada: (0, true).
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
