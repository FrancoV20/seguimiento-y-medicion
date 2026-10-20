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
	registry        *Registry
	renderer        *Renderer
	middlewares     []func(http.Handler) http.Handler
	shutdownTimeout time.Duration
}

// NewServer crea el servidor con la página de inicio registrada.
func NewServer(logger *slog.Logger) *Server {
	registry := NewRegistry()
	s := &Server{
		logger:          logger,
		mux:             http.NewServeMux(),
		registry:        registry,
		renderer:        NewRenderer(registry, logger),
		shutdownTimeout: ShutdownTimeout,
	}
	s.HandleFunc("GET /{$}", s.home)
	s.Handle("GET /static/", staticHandler())
	return s
}

// Renderer devuelve el renderizador de páginas con el layout común.
func (s *Server) Renderer() *Renderer { return s.renderer }

// Routes asocia patrones de http.ServeMux (por ejemplo "GET /proyectos/{id}") con sus handlers.
type Routes map[string]http.Handler

// Register agrega una sección al menú y registra sus rutas (HU-20a, T028).
// Una dirección o ruta repetida es un error de programación: el arranque debe fallar.
func (s *Server) Register(section Section, routes Routes) (err error) {
	if err := s.registry.Add(section); err != nil {
		return err
	}
	defer func() {
		if rec := recover(); rec != nil {
			s.registry.remove(strings.TrimSpace(section.Path))
			err = fmt.Errorf("la sección %q no se pudo registrar: %v", section.Name, rec)
		}
	}()
	for pattern, h := range routes {
		s.mux.Handle(pattern, h)
	}
	return nil
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

// Handler devuelve el handler completo: la recuperación de pánicos primero,
// después los middlewares en el orden de Use y al final las rutas.
func (s *Server) Handler() http.Handler {
	var h http.Handler = http.HandlerFunc(s.route)
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		h = s.middlewares[i](h)
	}
	return recoverPanics(s.logger, s.renderer, h)
}

// route atiende con el ServeMux y reemplaza sus respuestas 404 y 405 en texto
// plano por las páginas de error con la estructura común (HU-20a, T041).
func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	h, pattern := s.mux.Handler(r)
	if pattern != "" {
		s.mux.ServeHTTP(w, r) // ServeHTTP completa los valores de la ruta ({id})
		return
	}
	// Sin patrón: el mux respondería 404 o 405. Se averigua cuál sin escribir nada.
	probe := &statusProbe{header: http.Header{}}
	h.ServeHTTP(probe, r)
	if probe.status == http.StatusMethodNotAllowed {
		if allow := probe.header.Get("Allow"); allow != "" {
			w.Header().Set("Allow", allow)
		}
		s.renderer.RenderError(w, r, http.StatusMethodNotAllowed, TitleMethodNotAllowed, MsgMethodNotAllowed)
		return
	}
	s.renderer.RenderError(w, r, http.StatusNotFound, TitleNotFound, MsgNotFound)
}

// statusProbe es un ResponseWriter que solo guarda el código y los encabezados.
type statusProbe struct {
	header http.Header
	status int
}

func (p *statusProbe) Header() http.Header { return p.header }

func (p *statusProbe) Write(b []byte) (int, error) {
	if p.status == 0 {
		p.status = http.StatusOK
	}
	return len(b), nil
}

func (p *statusProbe) WriteHeader(code int) {
	if p.status == 0 {
		p.status = code
	}
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

// home muestra la página de inicio con el layout común.
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	s.renderer.Page(w, r, "home.html", PageData{Title: HomeName})
}
