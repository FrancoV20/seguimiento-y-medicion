package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetFlash_AtributosDeLaCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	SetFlash(rec, req, FlashSuccess, "Guardado")

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != CookieFlash {
		t.Fatalf("se esperaba la cookie %q, se obtuvo %+v", CookieFlash, cookies)
	}
	c := cookies[0]
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
		t.Errorf("faltan HttpOnly, SameSite=Lax o Path=/: %+v", c)
	}
	if c.MaxAge <= 0 || c.MaxAge > 120 {
		t.Errorf("el flash debe durar poco (1 a 120 s), Max-Age=%d", c.MaxAge)
	}
}

func TestFlash_IdaYVuelta(t *testing.T) {
	cases := []struct {
		kind FlashKind
		text string
	}{
		{FlashSuccess, "Proyecto creado"},
		{FlashError, MsgOperationFailed},
		{FlashWarning, "Sin Story Points: ñandú & <b>"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			rec := httptest.NewRecorder()
			SetFlash(rec, httptest.NewRequest(http.MethodGet, "/", nil), tc.kind, tc.text)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(rec.Result().Cookies()[0])

			got := flashFromRequest(req)

			if got == nil || got.Kind != tc.kind || got.Text != tc.text {
				t.Errorf("se esperaba %q/%q, se obtuvo %+v", tc.kind, tc.text, got)
			}
		})
	}
}

func TestFlashFromRequest_SinCookieEsNil(t *testing.T) {
	if got := flashFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); got != nil {
		t.Errorf("sin cookie se esperaba nil, se obtuvo %+v", got)
	}
}

func TestFlashFromRequest_ValoresInvalidosSeIgnoran(t *testing.T) {
	cases := []struct{ name, value string }{
		{"vacia", ""},
		{"sin formato", "hola"},
		{"tipo desconocido", "k=danger&t=hola"},
		{"sin tipo", "t=hola"},
		{"sin texto", "k=success&t="},
		{"muy largo", "k=success&t=" + strings.Repeat("a", flashMaxText+1)},
		{"escape malformado", "k=success&t=%zz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.AddCookie(&http.Cookie{Name: CookieFlash, Value: tc.value})

			if got := flashFromRequest(req); got != nil {
				t.Errorf("valor %q: se esperaba nil, se obtuvo %+v", tc.value, got)
			}
		})
	}
}