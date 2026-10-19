package test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-20a. Cada test sigue Dado / Cuando / Entonces.

func silencioso() *slog.Logger { return slog.New(slog.DiscardHandler) }

type pingerQueFalla struct {
	fallas, intentos int
	err              error
}

func (p *pingerQueFalla) PingContext(context.Context) error {
	p.intentos++
	if p.intentos <= p.fallas {
		return p.err
	}
	return nil
}

func politicaSinEspera() web.RetryPolicy {
	return web.RetryPolicy{
		Attempts: web.ConnectAttempts,
		Delay:    web.ConnectRetryDelay,
		Sleep:    func(context.Context, time.Duration) error { return nil },
	}
}

type lectorDeEsquema struct{ estado web.SchemaStatus }

func (l lectorDeEsquema) ReadSchemaStatus(context.Context) (web.SchemaStatus, error) {
	return l.estado, nil
}

// US1 - Escenario 1 (T016, FR-001, SC-007).
func TestEscenarioWeb_PaginaDeInicioConMenu(t *testing.T) {
	// Dado que el servidor está en ejecución
	srv := httptest.NewServer(web.NewServer(silencioso()).Handler())
	defer srv.Close()

	// Cuando abro su dirección en el navegador
	inicio := time.Now()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("no se pudo abrir la página de inicio: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	demora := time.Since(inicio)

	// Entonces veo la página de inicio con el menú de navegación
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") || !strings.Contains(ct, "utf-8") {
		t.Errorf("Content-Type inesperado: %q", ct)
	}
	if !strings.Contains(string(body), "<nav") || !strings.Contains(string(body), "Inicio") {
		t.Errorf("la página de inicio debe tener el menú de navegación")
	}
	if demora > 500*time.Millisecond {
		t.Errorf("la página de inicio tardó %v, más de 500 ms", demora)
	}
}

// US1 - Escenario 2 (T016, FR-002).
func TestEscenarioWeb_FaltaDatabaseURL(t *testing.T) {
	// Dado que falta la variable DATABASE_URL
	entorno := func(nombre string) string { return map[string]string{"ADDR": ":8080"}[nombre] }

	// Cuando se inicia el servidor
	_, err := web.LoadConfig(entorno)

	// Entonces informa qué variable falta sin mostrar valores
	var faltante *web.MissingEnvError
	if !errors.As(err, &faltante) || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("se esperaba un error que nombre DATABASE_URL, se obtuvo: %v", err)
	}
}

// US1 - Escenario 3 (T016, FR-003, SC-002).
func TestEscenarioWeb_BaseLentaReintentaYConecta(t *testing.T) {
	// Dado que la base de datos todavía no está lista
	base := &pingerQueFalla{fallas: 5, err: errors.New("connection refused")}

	// Cuando se inicia el servidor
	err := web.WaitForDatabase(context.Background(), base, politicaSinEspera(), silencioso(), "localhost:5433/db", web.Config{}.Redact)

	// Entonces reintenta y continúa con el arranque al conectar
	if err != nil || base.intentos != 6 {
		t.Errorf("debía conectar al sexto intento; intentos=%d, error=%v", base.intentos, err)
	}
}

// US1 - Escenario 4 (T016, FR-004).
func TestEscenarioWeb_BaseCaidaNoExponeLaClave(t *testing.T) {
	// Dado que la base de datos no responde
	url := "postgres://sym_user:Sup3rS3cret@localhost:5433/db"
	base := &pingerQueFalla{fallas: 99, err: errors.New("password Sup3rS3cret rejected")}
	cfg := web.Config{DatabaseURL: url}

	// Cuando se agotan los reintentos
	err := web.WaitForDatabase(context.Background(), base, politicaSinEspera(), silencioso(), web.DescribeDatabase(url), cfg.Redact)

	// Entonces informa que no pudo conectarse sin mostrar la clave ni la URL completa
	if err == nil || strings.Contains(err.Error(), "Sup3rS3cret") || strings.Contains(err.Error(), url) {
		t.Errorf("se esperaba un error sin datos sensibles, se obtuvo: %v", err)
	}
}

// US1 - Escenarios 5 y 6 (T016, FR-005).
func TestEscenarioWeb_EsquemaDesactualizadoODirtyNoArranca(t *testing.T) {
	casos := map[string]struct {
		estado web.SchemaStatus
		pista  string
	}{
		"migraciones pendientes": {web.SchemaStatus{Version: 3}, "migrate"},
		"estado dirty":           {web.SchemaStatus{Version: 7, Dirty: true}, "equipo"},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			// Dado que el esquema no está al día
			lector := lectorDeEsquema{estado: c.estado}

			// Cuando se inicia el servidor
			_, err := web.CheckSchema(context.Background(), lector, 7)

			// Entonces no arranca e indica qué hacer
			if err == nil || !strings.Contains(err.Error(), c.pista) {
				t.Errorf("se esperaba un error con la pista %q, se obtuvo: %v", c.pista, err)
			}
		})
	}
}

// US1 - Escenario 7 (T016, FR-006).
func TestEscenarioWeb_PuertoEnUso(t *testing.T) {
	// Dado que el puerto configurado ya está en uso
	ocupado, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo ocupar un puerto: %v", err)
	}
	defer ocupado.Close()

	// Cuando se inicia el servidor en ese puerto
	_, err = web.NewServer(silencioso()).Listen(ocupado.Addr().String())

	// Entonces informa el problema y el puerto afectado
	_, puerto, _ := net.SplitHostPort(ocupado.Addr().String())
	if err == nil || !strings.Contains(err.Error(), puerto) {
		t.Errorf("se esperaba un error que nombre el puerto %s, se obtuvo: %v", puerto, err)
	}
}

// US2 - Escenario 1 (T027, FR-008).
func TestEscenarioWeb_EstructuraComunConMenuYContexto(t *testing.T) {
	// Dado cualquier página de la aplicación
	srv := httptest.NewServer(web.NewServer(silencioso()).Handler())
	defer srv.Close()

	// Cuando se muestra
	body := leer(t, srv.URL+"/")

	// Entonces incluye el menú, la sección actual resaltada y el espacio de contexto
	for _, esperado := range []string{"<nav", `aria-current="page"`, "Sin proyecto seleccionado"} {
		if !strings.Contains(body, esperado) {
			t.Errorf("la estructura común debe contener %q", esperado)
		}
	}
}

// US2 - Escenario 2 (T027, FR-009).
func TestEscenarioWeb_SoloSeccionesRegistradasEnElMenu(t *testing.T) {
	// Dado que hay una sección registrada y otras todavía no implementadas
	s := web.NewServer(silencioso())
	nada := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if err := s.Register(web.Section{Name: "Proyectos", Path: "/proyectos"}, web.Routes{"GET /proyectos": nada}); err != nil {
		t.Fatalf("no se pudo registrar la sección: %v", err)
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// Cuando se muestra el menú
	body := leer(t, srv.URL+"/")

	// Entonces aparecen solo las registradas
	if !strings.Contains(body, `href="/proyectos"`) {
		t.Error("la sección registrada debe aparecer en el menú")
	}
	if strings.Contains(body, `href="/sprints"`) {
		t.Error("una sección no registrada no debe aparecer en el menú")
	}
}

// US2 - Escenario 3 (T027, FR-010, SC-006).
func TestEscenarioWeb_SinRecursosExternos(t *testing.T) {
	// Dado que no hay conexión a internet
	srv := httptest.NewServer(web.NewServer(silencioso()).Handler())
	defer srv.Close()

	// Cuando abro la aplicación
	body := leer(t, srv.URL+"/")
	css, err := http.Get(srv.URL + "/static/app.css")
	if err != nil {
		t.Fatalf("no se pudo pedir el CSS: %v", err)
	}
	css.Body.Close()

	// Entonces estilos y recursos cargan desde el propio servidor
	if strings.Contains(body, "https://") || strings.Contains(body, "http://") || css.StatusCode != http.StatusOK {
		t.Errorf("la página no debe depender de otros orígenes (CSS: %d)", css.StatusCode)
	}
}

// US2 - Escenario 4 (T027, FR-011).
func TestEscenarioWeb_CaracteresEspeciales(t *testing.T) {
	// Dado un texto con tildes, ñ y símbolos
	s := web.NewServer(silencioso())
	texto := "Planificación del año — ¿listo? ✓"
	rec := httptest.NewRecorder()

	// Cuando se muestra en una pantalla
	s.Renderer().Page(rec, httptest.NewRequest(http.MethodGet, "/", nil), "home.html", web.PageData{Title: texto})

	// Entonces se ve correctamente
	if !strings.Contains(rec.Body.String(), texto) {
		t.Errorf("el texto %q no se ve tal cual", texto)
	}
}

func leer(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}
