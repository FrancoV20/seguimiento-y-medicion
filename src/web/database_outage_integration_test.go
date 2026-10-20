package web

import (
	"context"
	"database/sql"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// T036: caída y recuperación de la base con el servidor en ejecución (FR-015, SC-005).
// Un proxy TCP local delante de PostgreSQL simula la caída sin detener el contenedor.

type pausableProxy struct {
	ln     net.Listener
	target string

	mu     sync.Mutex
	paused bool
	conns  map[net.Conn]struct{}
}

func startPausableProxy(t *testing.T, target string) *pausableProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo abrir el proxy: %v", err)
	}
	p := &pausableProxy{ln: ln, target: target, conns: map[net.Conn]struct{}{}}
	go p.acceptLoop()
	t.Cleanup(func() {
		_ = ln.Close()
		p.Pause()
	})
	return p
}

func (p *pausableProxy) Addr() string { return p.ln.Addr().String() }

func (p *pausableProxy) acceptLoop() {
	for {
		client, err := p.ln.Accept()
		if err != nil {
			return
		}
		p.mu.Lock()
		paused := p.paused
		if !paused {
			p.conns[client] = struct{}{}
		}
		p.mu.Unlock()
		if paused {
			_ = client.Close() // base "caída": acepta y corta
			continue
		}
		go p.pipe(client)
	}
}

func (p *pausableProxy) pipe(client net.Conn) {
	upstream, err := net.Dial("tcp", p.target)
	if err != nil {
		_ = client.Close()
		return
	}
	p.mu.Lock()
	p.conns[upstream] = struct{}{}
	p.mu.Unlock()
	go func() {
		_, _ = io.Copy(upstream, client)
		_ = upstream.Close()
	}()
	_, _ = io.Copy(client, upstream)
	_ = client.Close()
}

// Pause corta todas las conexiones y rechaza las nuevas.
func (p *pausableProxy) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paused = true
	for c := range p.conns {
		_ = c.Close()
	}
	p.conns = map[net.Conn]struct{}{}
}

// Resume vuelve a dejar pasar las conexiones.
func (p *pausableProxy) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paused = false
}

func TestCaidaYRecuperacionDeLaBaseSinReiniciar(t *testing.T) {
	raw := os.Getenv(EnvDatabaseURL)
	if raw == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("DATABASE_URL no es válida")
	}
	proxy := startPausableProxy(t, u.Host)
	u.Host = proxy.Addr()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatalf("no se pudo abrir la base a través del proxy: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s := NewServer(quietLogger())
	s.HandleFunc("GET /datos", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		var uno int
		if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&uno); err != nil {
			s.Renderer().Fail(w, r, err)
			return
		}
		s.Renderer().Page(w, r, "home.html", PageData{Title: "Datos"})
	})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	if code, _ := obtener(t, srv.URL+"/datos"); code != http.StatusOK {
		t.Fatalf("con la base disponible se esperaba 200, se obtuvo %d", code)
	}

	proxy.Pause()
	code, body := obtener(t, srv.URL+"/datos")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("con la base caída se esperaba 503, se obtuvo %d", code)
	}
	if !strings.Contains(body, MsgUnavailable) || !strings.Contains(body, "<nav") {
		t.Error("la página 503 debe mostrar el mensaje y la estructura común sin consultar la base")
	}

	proxy.Resume()
	if code, _ := obtener(t, srv.URL+"/datos"); code != http.StatusOK {
		t.Errorf("al volver la base, la primera solicitud debía responder 200 sin reiniciar, se obtuvo %d", code)
	}
}

func obtener(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}
