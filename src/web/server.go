package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ShutdownTimeout es el máximo que se espera a las solicitudes en curso al apagar (HU-20a, T021).
const ShutdownTimeout = 10 * time.Second

// Server es el servidor web de la aplicación.
type Server struct {
	logger          *slog.Logger
	mux             *http.ServeMux
	middlewares     []func(http.Handler) http.Handler
	shutdownTimeout time.Duration
}

// NewServer crea el servidor con la página de inicio registrada.
func NewServer(logger *slog.Logger) *Server {
	s := &Server{
		logger:          logger,
		mux:             http.NewServeMux(),
		shutdownTimeout: ShutdownTimeout,
	}
	s.HandleFunc("GET /{$}", s.home)
	return s
}

// Handle registra un handler para un patrón de http.ServeMux (por ejemplo "GET /proyectos").
func (s *Server) Handle(pattern string, h http.Handler) { s.mux.Handle(pattern, h) }

// HandleFunc registra una función handler.
func (s *Server) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, h)
}

// Use agrega un middleware a la cadena, en el orden en que se registra.
func (s *Server) Use(mw func(http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, mw)
}

// Handler devuelve el handler completo con los middlewares aplicados.
func (s *Server) Handler() http.Handler {
	var h http.Handler = s.mux
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		h = s.middlewares[i](h)
	}
	return h
}

// PortError explica por qué no se puede escuchar en la dirección configurada.
type PortError struct {
	Addr   string
	Reason string
}

func (e *PortError) Error() string { return e.Reason }

// Listen valida la dirección y abre el puerto. Los errores nombran el puerto
// afectado sin detalles internos del sistema operativo.
func (s *Server) Listen(addr string) (net.Listener, error) {
	_, port, err := net.SplitHostPort(addr)
	n, convErr := strconv.Atoi(port)
	if err != nil || convErr != nil || n < 1 || n > 65535 {
		return nil, &PortError{Addr: addr, Reason: fmt.Sprintf(
			"el puerto de la dirección %q no es válido: usá un número entre 1 y 65535, por ejemplo ADDR=:8080", addr)}
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if isAddrInUse(err) {
			return nil, &PortError{Addr: addr, Reason: fmt.Sprintf(
				"el puerto %s ya está en uso: cerrá el programa que lo usa o elegí otro puerto con ADDR", port)}
		}
		return nil, &PortError{Addr: addr, Reason: fmt.Sprintf(
			"no se pudo abrir el puerto %s: revisá que la dirección %q sea correcta y esté disponible", port, addr)}
	}
	return ln, nil
}

// windowsAddrInUse es WSAEADDRINUSE en Windows.
const windowsAddrInUse = syscall.Errno(10048)

func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) || errors.Is(err, windowsAddrInUse) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") || strings.Contains(msg, "only one usage of each socket address")
}

// Serve atiende solicitudes hasta que ctx se cancela. Entonces deja de aceptar
// conexiones nuevas, espera a las solicitudes en curso hasta el límite de apagado
// y llama a onShutdown (por ejemplo, para cerrar la base de datos).
func (s *Server) Serve(ctx context.Context, ln net.Listener, onShutdown func()) error {
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ErrorLog:          slog.NewLogLogger(s.logger.Handler(), slog.LevelWarn),
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	s.logger.Info("servidor en ejecución", "direccion", ln.Addr().String())

	var result error
	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			result = err
		}
	case <-ctx.Done():
		s.logger.Info("deteniendo el servidor: se completan las solicitudes en curso", "limite", s.shutdownTimeout)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Warn("se venció el límite de apagado; se cierran las conexiones restantes")
			_ = srv.Close()
		}
	}
	if onShutdown != nil {
		onShutdown()
	}
	s.logger.Info("servidor detenido")
	return result
}

// home muestra la página de inicio. HU-20a US2 la reemplaza por la plantilla con el layout común.
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(homeHTML))
}

const homeHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Inicio · Software Metrics &amp; Estimation</title>
</head>
<body>
<header class="app-header">
<a class="app-title" href="/">Software Metrics &amp; Estimation</a>
<nav class="app-nav" aria-label="Menú principal"><ul><li><a href="/" class="active" aria-current="page">Inicio</a></li></ul></nav>
<p class="app-context">Sin proyecto seleccionado</p>
</header>
<main class="app-main">
<h1>Inicio</h1>
<p>Administrá proyectos de software: backlog, sprints, estimación, esfuerzo y métricas.</p>
</main>
</body>
</html>
`
