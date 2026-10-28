package web

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// Servidor con rutas de prueba para los errores de US3.
func servidorConRutasDeError() *Server {
	s := NewServer(quietLogger())
	s.HandleFunc("GET /prueba/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := PathID(r, "id")
		if err != nil {
			s.Renderer().Fail(w, r, err)
			return
		}
		fmt.Fprintf(w, "id=%d", id)
	})
	return s
}

func pedir(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func tieneEstructuraComun(t *testing.T, body string) {
	t.Helper()
	for _, esperado := range []string{"<nav", `href="/static/app.css"`, `<a href="/">` + LinkBackHome + `</a>`, "</html>"} {
		if !strings.Contains(body, esperado) {
			t.Errorf("la página de error debe tener la estructura común y el enlace al inicio (falta %q)", esperado)
		}
	}
}

// T031: 404 con estructura común para direcciones inexistentes y parámetros malformados (FR-012, SC-003).

func TestErrores_404ConEstructuraComun(t *testing.T) {
	h := servidorConRutasDeError().Handler()
	for _, path := range []string{"/no-existe", "/prueba/abc", "/prueba/-5", "/prueba/0", "/prueba/99999999999999999999", "/prueba/1.5"} {
		rec := pedir(h, http.MethodGet, path)

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: se esperaba 404, se obtuvo %d", path, rec.Code)
			continue
		}
		body := rec.Body.String()
		if !strings.Contains(body, TitleNotFound) {
			t.Errorf("%s: la página debe decir %q", path, TitleNotFound)
		}
		tieneEstructuraComun(t, body)
	}
}

func TestErrores_IdentificadorValidoFunciona(t *testing.T) {
	rec := pedir(servidorConRutasDeError().Handler(), http.MethodGet, "/prueba/7")

	if rec.Code != http.StatusOK || rec.Body.String() != "id=7" {
		t.Errorf("un identificador válido debe funcionar, se obtuvo %d %q", rec.Code, rec.Body.String())
	}
}

// T033: 405 con estructura común y servidor activo (FR-013).

func TestErrores_405ConEstructuraComunYServidorActivo(t *testing.T) {
	h := servidorConRutasDeError().Handler()

	rec := pedir(h, http.MethodPost, "/")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("se esperaba 405, se obtuvo %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Allow"), http.MethodGet) {
		t.Errorf("la respuesta 405 debe indicar los métodos permitidos, Allow=%q", rec.Header().Get("Allow"))
	}
	if !strings.Contains(rec.Body.String(), TitleMethodNotAllowed) {
		t.Errorf("la página debe decir %q", TitleMethodNotAllowed)
	}
	tieneEstructuraComun(t, rec.Body.String())

	if pedir(h, http.MethodGet, "/").Code != http.StatusOK {
		t.Error("después de un 405 el servidor debe seguir respondiendo")
	}
}

// 503 con el texto pedido y la estructura común (FR-015).

func TestFail_BaseNoDisponibleMuestra503(t *testing.T) {
	s := NewServer(quietLogger())
	s.HandleFunc("GET /datos", func(w http.ResponseWriter, r *http.Request) {
		s.Renderer().Fail(w, r, fmt.Errorf("consulta: %w", driver.ErrBadConn))
	})

	rec := pedir(s.Handler(), http.MethodGet, "/datos")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("se esperaba 503, se obtuvo %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "El servicio no está disponible temporalmente") {
		t.Error("la página 503 debe decir 'El servicio no está disponible temporalmente'")
	}
	tieneEstructuraComun(t, rec.Body.String())
}

func TestIsDatabaseUnavailable(t *testing.T) {
	casos := []struct {
		nombre string
		err    error
		caida  bool
	}{
		{"conexión inválida", fmt.Errorf("q: %w", driver.ErrBadConn), true},
		{"tiempo agotado", fmt.Errorf("q: %w", context.DeadlineExceeded), true},
		{"conexión rechazada", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, true},
		{"conexión perdida (PostgreSQL 08006)", &pgconn.PgError{Code: "08006"}, true},
		{"servidor apagándose (PostgreSQL 57P01)", &pgconn.PgError{Code: "57P01"}, true},
		{"clave duplicada (PostgreSQL 23505)", &pgconn.PgError{Code: "23505"}, false},
		{"error de negocio", errors.New("el proyecto no está activo"), false},
		{"sin error", nil, false},
	}
	for _, c := range casos {
		if got := IsDatabaseUnavailable(c.err); got != c.caida {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.nombre, c.caida, got)
		}
	}
}

// T035: ninguna página de error expone rutas, SQL, trazas ni credenciales (FR-016, SC-004).

func TestErrores_NoExponenDetallesInternos(t *testing.T) {
	const databaseURL = "postgres://sym_user:Sup3rS3cret@localhost:5433/seguimiento_y_medicion?sslmode=disable"
	detalle := "SELECT * FROM projects WHERE id = 1; C:/repo/src/web/server.go:42 " + databaseURL + " goroutine 7 [running]"
	s := servidorConRutasDeError()
	s.HandleFunc("GET /interno", func(w http.ResponseWriter, r *http.Request) { s.Renderer().Fail(w, r, errors.New(detalle)) })
	s.HandleFunc("GET /base", func(w http.ResponseWriter, r *http.Request) {
		s.Renderer().Fail(w, r, Unavailable(errors.New(detalle)))
	})
	s.HandleFunc("GET /no-esta", func(w http.ResponseWriter, r *http.Request) { s.Renderer().Fail(w, r, NotFound(errors.New(detalle))) })
	s.HandleFunc("GET /panico", func(http.ResponseWriter, *http.Request) { panic(detalle) })
	h := s.Handler()

	casos := []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/interno", http.StatusInternalServerError},
		{http.MethodGet, "/base", http.StatusServiceUnavailable},
		{http.MethodGet, "/no-esta", http.StatusNotFound},
		{http.MethodGet, "/panico", http.StatusInternalServerError},
		{http.MethodGet, "/no-existe", http.StatusNotFound},
		{http.MethodGet, "/prueba/abc", http.StatusNotFound},
		{http.MethodPost, "/", http.StatusMethodNotAllowed},
	}
	for _, c := range casos {
		rec := pedir(h, c.method, c.path)
		if rec.Code != c.status {
			t.Errorf("%s %s: se esperaba %d, se obtuvo %d", c.method, c.path, c.status, rec.Code)
		}
		body := rec.Body.String()
		for _, prohibido := range []string{"SELECT", ".go:", "postgres://", "Sup3rS3cret", "goroutine", databaseURL} {
			if strings.Contains(body, prohibido) {
				t.Errorf("%s %s: la página expone %q", c.method, c.path, prohibido)
			}
		}
	}
}
