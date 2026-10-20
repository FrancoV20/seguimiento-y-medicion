package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// T034: un pánico produce la página 500 genérica, el servidor sigue y el detalle queda en el log (FR-014).

func TestRecover_PanicoMuestra500YElServidorSigue(t *testing.T) {
	var logs bytes.Buffer
	s := NewServer(slog.New(slog.NewTextHandler(&logs, nil)))
	s.HandleFunc("GET /panico", func(http.ResponseWriter, *http.Request) {
		panic("explotó el cálculo: detalle interno 42")
	})
	h := s.Handler()

	rec := pedir(h, http.MethodGet, "/panico")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("se esperaba 500, se obtuvo %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, TitleInternal) || strings.Contains(body, "detalle interno 42") {
		t.Error("la página 500 debe ser genérica y no mostrar el detalle del pánico")
	}
	tieneEstructuraComun(t, body)
	if !strings.Contains(logs.String(), "detalle interno 42") || !strings.Contains(logs.String(), "goroutine") {
		t.Error("el detalle y la traza del pánico deben quedar en el registro del servidor")
	}
	if pedir(h, http.MethodGet, "/").Code != http.StatusOK {
		t.Error("después de un pánico el servidor debe seguir respondiendo")
	}
}

func TestRecover_CubreTambienLosMiddlewares(t *testing.T) {
	s := NewServer(quietLogger())
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/falla-middleware" {
				panic("pánico en un middleware")
			}
			next.ServeHTTP(w, r)
		})
	})

	rec := pedir(s.Handler(), http.MethodGet, "/falla-middleware")

	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), TitleInternal) {
		t.Errorf("un pánico en un middleware también debe dar la página 500, se obtuvo %d", rec.Code)
	}
}
