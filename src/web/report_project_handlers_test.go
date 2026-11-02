package web

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/report"
)

// HU-09, T040: reporte del proyecto (GET /reportes/proyecto/pdf) con el servidor de prueba y los datos
// de report_fixtures_test.go.

const projectReportPath = "/reportes/proyecto/pdf"

// Escenario 14.
func TestHU09_ProyectoAdvertencia(t *testing.T) {
	app := newReportApp(t)
	c := app.open(t, "Alfa")
	for _, query := range []string{"", "?confirmar=SI"} {
		resp, body := app.get(t, c, projectReportPath+query)
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			t.Fatalf("%q: se esperaba la página HTML de advertencia, se obtuvo %d %q", query, resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		reportWantNoPDF(t, resp, body)
		reportWantText(t, body,
			"El proyecto tiene un Sprint en curso (Sprint 3): el reporte puede estar incompleto",
			`href="/reportes/proyecto/pdf?confirmar=si"`, "Generar igual",
			`href="/reportes"`, "Cancelar",
		)
	}
	if n := app.fx.metricsRows(t, "Alfa"); n != 0 {
		t.Errorf("la advertencia no debía escribir en sprint_metrics; hay %d filas", n)
	}
}

// Escenarios 15, 16 y 17.
func TestHU09_ProyectoContenido(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Alfa", projectReportPath+"?confirmar=si")
	text := reportPDFText(t, resp, body, fmt.Sprintf("reporte-proyecto-%d.pdf", app.fx.projects["Alfa"]))
	reportWantInOrder(t, text,
		"Reporte del proyecto",
		"Reporte parcial: hay un Sprint en curso",
		"Proyecto: Alfa",
		"Fechas: 18/08/2026 al 30/11/2026",
		"Estado: En curso",
		"Integrantes: Ana, Beto",
		"Generado: 28/10/2026 10:30",
		"Pendiente 1", "En Sprint 1", "En progreso 1", "Hecho 4",
		"4 de 7 historias completadas (57,14 %)",
		"Sprint 1 Finalizado 1 de 1 3 de 3 3 0,00 % 0 detectados y 0 resueltos",
		"Sprint 2 Finalizado 3 de 4 10 de 18 10 +6,25 % (cálculo parcial) 2 detectados y 1 resuelto",
		"Sprint 3 En curso: se calcula al cerrar el Sprint",
		"Velocidad acumulada: 13 Story Points",
		"Promedio: 6,50 Story Points por Sprint",
		"38 horas estimadas en total (2 historias sin horas estimadas)",
		"30 horas registradas en total",
		"Ana: 23 h", "Beto: 7 h",
		"3 defectos registrados: 2 abiertos y 1 resuelto",
		"La foto de perfil no se guarda Media Perfil Sprint 2",
		"El PDF sale en blanco Crítica Exportar Sprint 3",
	)
	reportWantNoText(t, text, "Delta", "Caro", "D1")
}

// Escenario 18.
func TestHU09_ProyectoSinSprints(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Beta", projectReportPath)
	text := reportPDFText(t, resp, body, fmt.Sprintf("reporte-proyecto-%d.pdf", app.fx.projects["Beta"]))
	reportWantInOrder(t, text,
		"Proyecto: Beta",
		"Integrantes: Ana",
		"Pendiente 0", "En Sprint 0", "En progreso 0", "Hecho 0",
		"0 de 0 historias completadas (0,00 %)",
		"Aún no hay Sprints",
		"0 horas estimadas en total",
		"0 horas registradas en total",
		"No se registraron defectos en este proyecto",
	)
	reportWantNoText(t, text, "Reporte parcial", "Velocidad acumulada")
}

// Caso límite: proyecto sin Sprint en curso.
func TestHU09_ProyectoSinSprintActivo(t *testing.T) {
	app := newReportApp(t)
	resp, body := app.page(t, "Gamma", projectReportPath)
	text := reportPDFText(t, resp, body, fmt.Sprintf("reporte-proyecto-%d.pdf", app.fx.projects["Gamma"]))
	reportWantInOrder(t, text,
		// Sin horas estimadas, HU-07 da la desviación "No disponible" y la marca como cálculo parcial.
		"Sprint 1 Finalizado 1 de 1 1 de 1 1 No disponible (cálculo parcial) 1 detectado y 0 resueltos",
		"Sprint 2 Finalizado 1 de 1 1 de 1 1 No disponible (cálculo parcial) 0 detectados y 1 resuelto",
		"Sprint 3 Finalizado 0 de 0 0 de 0 0 No disponible 0 detectados y 0 resueltos",
		"Velocidad acumulada: 2 Story Points",
		"Promedio: 0,67 Story Points por Sprint",
		"1 defecto registrado: 0 abiertos y 1 resuelto",
		"No hay defectos abiertos",
	)
	reportWantNoText(t, text, "Reporte parcial")
}

// Escenario 13 y FR-020 para el reporte del proyecto.
func TestHU09_ProyectoErrores(t *testing.T) {
	cause := errors.New("detalle interno de prueba")

	t.Run("catálogo que falla", func(t *testing.T) {
		app := newReportApp(t, withReportCatalog(func(c ReportCatalog) ReportCatalog {
			return failingReportCatalog{ReportCatalog: c, err: cause}
		}))
		resp, body := app.page(t, "Beta", projectReportPath)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("se esperaba 500, se obtuvo %d", resp.StatusCode)
		}
		reportWantNoPDF(t, resp, body)
		reportWantNoText(t, body, cause.Error())
	})

	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"generador que falla", cause, http.StatusInternalServerError},
		{"base de datos caída", fmt.Errorf("load project report: %w", driver.ErrBadConn), http.StatusServiceUnavailable},
		{"proyecto borrado mientras se arma", report.ErrProjectNotFound, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newReportApp(t, withReportGenerator(func(g ReportGenerator) ReportGenerator {
				return failingReportGenerator{ReportGenerator: g, err: tc.err}
			}))
			resp, body := app.page(t, "Beta", projectReportPath)
			if resp.StatusCode != tc.want {
				t.Errorf("se esperaba %d, se obtuvo %d", tc.want, resp.StatusCode)
			}
			reportWantNoPDF(t, resp, body)
			reportWantNoText(t, body, cause.Error())
		})
	}

	t.Run("sin proyecto y método no soportado", func(t *testing.T) {
		app := newReportApp(t)
		resp, _ := app.get(t, app.client(false), projectReportPath)
		if want := "/contexto/proyecto?volver_a=" + url.QueryEscape(projectReportPath); resp.StatusCode != http.StatusSeeOther ||
			resp.Header.Get("Location") != want {
			t.Errorf("sin proyecto: %d %q, se esperaba 303 %q", resp.StatusCode, resp.Header.Get("Location"), want)
		}
		resp, _ = app.doWithHeaders(t, http.MethodPost, projectReportPath, url.Values{}, map[string]string{"Origin": app.ts.URL})
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("POST: se esperaba 405, se obtuvo %d", resp.StatusCode)
		}
	})
}
