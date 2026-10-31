package test

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

// Escenarios BDD de HU-25 "Métricas del Sprint" (presenta HU-07). Usan la API pública de web, el
// servidor real, PostgreSQL real, el servicio real de HU-07 y el cierre real de HU-15.

type metricasBDDApp struct {
	db         *sql.DB
	servidor   *httptest.Server
	proyecto   int64
	integrante int64
}

type metricasBDDHistoria struct {
	titulo    string
	puntos    int
	estado    sprint.StoryStatus
	estimadas string // vacío: sin horas estimadas
	reales    string // vacío: sin horas registradas
}

func nuevaMetricasBDDApp(t *testing.T) *metricasBDDApp {
	t.Helper()
	dsn := os.Getenv(web.EnvDatabaseURL)
	if dsn == "" {
		t.Skip("DATABASE_URL no definida: se omite el escenario con PostgreSQL")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	a := &metricasBDDApp{db: db}
	sello := time.Now().UnixNano()
	// Se registra antes de insertar: limpia aunque la preparación falle a mitad. Primero las horas
	// y las historias (sin borrado en cascada) y después el proyecto y el integrante.
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`, a.proyecto)
		_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, a.proyecto)
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, a.proyecto)
		_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, a.integrante)
	})
	if err := db.QueryRow(`INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, DATE '2026-01-01', DATE '2026-12-31') RETURNING id`,
		fmt.Sprintf("BDD HU-25 %d", sello)).Scan(&a.proyecto); err != nil {
		t.Fatalf("no se pudo crear el proyecto del escenario: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("BDD HU-25 integrante %d", sello)).Scan(&a.integrante); err != nil {
		t.Fatalf("no se pudo crear el integrante del escenario: %v", err)
	}

	server := web.NewServer(silencioso())
	web.RegisterContext(server, web.NewPostgreSQLContextStore(db))
	if err := web.RegisterMetrics(server, web.NewPostgreSQLMetricsCatalog(db), web.NewHU07MetricsSource(db)); err != nil {
		t.Fatalf("no se pudo registrar la sección Métricas del Sprint: %v", err)
	}
	a.servidor = httptest.NewServer(server.Handler())
	t.Cleanup(a.servidor.Close)
	return a
}

// sprintCerrado crea un Sprint activo con sus historias y horas y lo cierra con HU-15.
func (a *metricasBDDApp) sprintCerrado(t *testing.T, numero int, objetivo string, historias ...metricasBDDHistoria) int64 {
	t.Helper()
	var sprintID int64
	inicio := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, (numero-1)*14)
	if err := a.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, 'Activo') RETURNING id`,
		a.proyecto, numero, objetivo, inicio, inicio.AddDate(0, 0, 13)).Scan(&sprintID); err != nil {
		t.Fatalf("no se pudo crear el Sprint %d del escenario: %v", numero, err)
	}
	for _, h := range historias {
		var estimadas sql.NullString
		if h.estimadas != "" {
			estimadas = sql.NullString{String: h.estimadas, Valid: true}
		}
		var id int64
		if err := a.db.QueryRow(`
			INSERT INTO user_stories (project_id, title, description, priority, story_points, status,
			                          estimated_hours, active_sprint_id)
			VALUES ($1, $2, 'Historia del escenario BDD de HU-25', 'Media', $3, $4, $5::numeric, $6)
			RETURNING id`,
			a.proyecto, h.titulo, h.puntos, string(h.estado), estimadas, sprintID).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear la historia %q: %v", h.titulo, err)
		}
		if _, err := a.db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`,
			sprintID, id); err != nil {
			t.Fatalf("no se pudo asignar la historia %q: %v", h.titulo, err)
		}
		if h.reales != "" {
			if _, err := a.db.Exec(`
				INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
				VALUES ($1, $2, DATE '2026-09-05', 'Trabajo del escenario HU-25', $3::numeric)`,
				id, a.integrante, h.reales); err != nil {
				t.Fatalf("no se pudieron registrar las horas de %q: %v", h.titulo, err)
			}
		}
	}
	cierre := sprint.NewCloseSprintService(sprint.NewPostgreSQLSprintCloser(a.db), cierreBDDValidator{})
	if _, err := cierre.CloseSprint(context.Background(), a.proyecto, sprintID); err != nil {
		t.Fatalf("HU-15 no pudo cerrar el Sprint %d del escenario: %v", numero, err)
	}
	return sprintID
}

// abrir elige el proyecto con HU-20b y abre la ruta de la pantalla.
func (a *metricasBDDApp) abrir(t *testing.T, ruta string) (*http.Response, string) {
	t.Helper()
	cliente := formNavegador(false)
	respuesta, err := cliente.PostForm(a.servidor.URL+"/contexto/proyecto", url.Values{
		"proyecto_id": {idTexto(a.proyecto)},
		"volver_a":    {"/metricas"},
	})
	if err != nil {
		t.Fatalf("no se pudo elegir el proyecto activo: %v", err)
	}
	_ = formCuerpo(t, respuesta)
	if respuesta.StatusCode != http.StatusSeeOther {
		t.Fatalf("elegir proyecto: se esperaba 303, se obtuvo %d", respuesta.StatusCode)
	}
	respuesta, err = cliente.Get(a.servidor.URL + ruta)
	if err != nil {
		t.Fatalf("GET %s: %v", ruta, err)
	}
	return respuesta, formCuerpo(t, respuesta)
}

// bloqueMetricas devuelve el bloque de métricas del Sprint elegido (<section id="metricas-sprint">).
func bloqueMetricas(t *testing.T, cuerpo string) string {
	t.Helper()
	inicio := strings.Index(cuerpo, `<section id="metricas-sprint"`)
	if inicio < 0 {
		t.Fatalf("la pantalla no muestra el bloque de métricas del Sprint elegido")
	}
	fin := strings.Index(cuerpo[inicio:], "</section>")
	if fin < 0 {
		t.Fatalf("el bloque de métricas del Sprint elegido no está cerrado")
	}
	return cuerpo[inicio : inicio+fin]
}

func debeContener(t *testing.T, donde, texto string, esperados ...string) {
	t.Helper()
	for _, e := range esperados {
		if !strings.Contains(texto, e) {
			t.Errorf("%s debía contener %q", donde, e)
		}
	}
}

// noDisponibleHU07 es el texto de HU-07 para la desviación sin horas estimadas.
func noDisponibleHU07(t *testing.T) string {
	t.Helper()
	texto, err := metrics.CalculateDeviation(nil, nil)
	if err != nil {
		t.Fatalf("HU-07 CalculateDeviation devolvió error: %v", err)
	}
	return texto
}

// avisoParcialHU07 es el aviso que arma HU-07 para un cálculo con usadas de total historias.
func avisoParcialHU07(t *testing.T, usadas, total int) string {
	t.Helper()
	puntos := 3
	historias := make([]metrics.CompletedStory, 0, total)
	for i := 0; i < total; i++ {
		h := metrics.CompletedStory{ID: int64(i + 1), Status: "Hecho", StoryPoints: &puntos}
		if i < usadas {
			h.EstimatedHours, h.ActualHours = big.NewRat(1, 1), big.NewRat(1, 1)
		}
		historias = append(historias, h)
	}
	store := &metrics.TestMetricsStore{SprintData: metrics.SprintData{ID: 1, Status: "Finalizado", Stories: historias}}
	resultado, err := metrics.NewService(store).CalculateMetrics(1)
	if err != nil || resultado.Warning == "" {
		t.Fatalf("HU-07 no armó el aviso de cálculo parcial (%v)", err)
	}
	return resultado.Warning
}

func TestMetricsBDD_MetricasDeUnSprintFinalizado(t *testing.T) {
	t.Run("Escenario 1: métricas de un Sprint finalizado", func(t *testing.T) {
		// Given un Sprint finalizado con 8 Story Points completados, 10 horas estimadas y 12 reales.
		app := nuevaMetricasBDDApp(t)
		app.sprintCerrado(t, 1, "Primer Sprint de métricas",
			metricasBDDHistoria{"Alta de clientes", 5, sprint.Done, "4", "5"},
			metricasBDDHistoria{"Baja de clientes", 3, sprint.Done, "6", "7"},
			metricasBDDHistoria{"Reporte pendiente", 13, sprint.InProgress, "30", "2"},
		)

		// When abro sus métricas.
		respuesta, cuerpo := app.abrir(t, "/metricas")

		// Then veo la velocidad y el porcentaje de desviación con su fórmula.
		if respuesta.StatusCode != http.StatusOK {
			t.Fatalf("GET /metricas = %d; se esperaba 200", respuesta.StatusCode)
		}
		debeContener(t, "el bloque de métricas", bloqueMetricas(t, cuerpo),
			"Sprint 1", "8 Story Points", "+20,00 %", web.MsgMetricsDeviationFormula)
	})
}

func TestMetricsBDD_SprintSinEsfuerzoEstimado(t *testing.T) {
	t.Run("Escenario 2: Sprint sin horas estimadas", func(t *testing.T) {
		// Given un Sprint finalizado cuyas historias completadas no tienen horas estimadas.
		app := nuevaMetricasBDDApp(t)
		app.sprintCerrado(t, 1, "Sprint sin estimaciones",
			metricasBDDHistoria{"Historia sin estimar", 5, sprint.Done, "", "3"},
		)

		// When abro sus métricas.
		respuesta, cuerpo := app.abrir(t, "/metricas")

		// Then veo "No disponible" en lugar de un error o un valor inválido, y la velocidad igual.
		if respuesta.StatusCode != http.StatusOK {
			t.Fatalf("GET /metricas = %d; se esperaba 200", respuesta.StatusCode)
		}
		bloque := bloqueMetricas(t, cuerpo)
		debeContener(t, "el bloque de métricas", bloque, noDisponibleHU07(t), "5 Story Points")
		for _, invalido := range []string{"NaN", "Inf", "%!"} {
			if strings.Contains(bloque, invalido) {
				t.Errorf("el bloque de métricas muestra un valor inválido %q", invalido)
			}
		}
	})
}

func TestMetricsBDD_CalculoParcial(t *testing.T) {
	t.Run("Escenario 3: Sprint con datos incompletos", func(t *testing.T) {
		// Given un Sprint finalizado con tres historias completadas y solo una con horas estimadas y reales.
		app := nuevaMetricasBDDApp(t)
		app.sprintCerrado(t, 1, "Sprint con datos incompletos",
			metricasBDDHistoria{"Con datos", 5, sprint.Done, "8", "6"},
			metricasBDDHistoria{"Sin estimar", 3, sprint.Done, "", "2"},
			metricasBDDHistoria{"Sin horas reales", 2, sprint.Done, "4", ""},
		)

		// When abro sus métricas.
		respuesta, cuerpo := app.abrir(t, "/metricas")

		// Then veo la alerta "Cálculo parcial: basado en X de Y historias" con los valores de HU-07.
		if respuesta.StatusCode != http.StatusOK {
			t.Fatalf("GET /metricas = %d; se esperaba 200", respuesta.StatusCode)
		}
		debeContener(t, "el bloque de métricas", bloqueMetricas(t, cuerpo), avisoParcialHU07(t, 1, 3))
	})
}

func TestMetricsBDD_VelocidadCero(t *testing.T) {
	t.Run("Escenario 4: Sprint finalizado sin historias completadas", func(t *testing.T) {
		// Given un Sprint cerrado por HU-15 sin ninguna historia en "Hecho".
		app := nuevaMetricasBDDApp(t)
		app.sprintCerrado(t, 1, "Sprint sin completar",
			metricasBDDHistoria{"Quedó en progreso", 5, sprint.InProgress, "6", "4"},
			metricasBDDHistoria{"Quedó sin empezar", 3, sprint.EnSprint, "2", ""},
		)

		// When abro sus métricas.
		respuesta, cuerpo := app.abrir(t, "/metricas")

		// Then veo la velocidad en cero y la desviación "No disponible", sin error.
		if respuesta.StatusCode != http.StatusOK {
			t.Fatalf("GET /metricas = %d; se esperaba 200", respuesta.StatusCode)
		}
		debeContener(t, "el bloque de métricas", bloqueMetricas(t, cuerpo), "0 Story Points", noDisponibleHU07(t))
	})
}
