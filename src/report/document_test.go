package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/pdf"
)

// HU-09, T013: el PDF de un Sprint, buscando los textos con pdf.ExtractText.

func renderText(t *testing.T, r SprintReport) string {
	t.Helper()
	b, err := Render(r)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-1.4")) {
		t.Fatalf("Render no devolvió un PDF")
	}
	text, err := pdf.ExtractText(b)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	return text
}

// wantInOrder comprueba que cada texto aparece después del anterior.
func wantInOrder(t *testing.T, text string, wants ...string) {
	t.Helper()
	pos := 0
	for _, want := range wants {
		i := strings.Index(text[pos:], want)
		if i < 0 {
			if strings.Contains(text, want) {
				t.Errorf("%q aparece fuera de orden", want)
			} else {
				t.Errorf("falta %q en el PDF", want)
			}
			continue
		}
		pos += i + len(want)
	}
	if t.Failed() {
		t.Logf("texto del PDF:\n%s", text)
	}
}

func wantNot(t *testing.T, text string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(text, u) {
			t.Errorf("el PDF no debe contener %q", u)
		}
	}
}

func finishedReport(data Data, m metrics.SprintMetrics, q metrics.SprintQuality) SprintReport {
	return SprintReport{Data: data, Metrics: &m, Quality: &q, GeneratedAt: fixedNow}
}

func TestRenderSprintReport(t *testing.T) {
	t.Run("Sprint 2 finalizado de «Alfa»", func(t *testing.T) {
		text := renderText(t, finishedReport(alfaSprint2(), alfaSprint2Metrics(), alfaSprint2Quality()))
		wantInOrder(t, text,
			"Reporte del Sprint",
			"Proyecto: Alfa",
			"Sprint 2",
			"Objetivo: Gestionar perfiles y reportes",
			"Fechas: 01/09/2026 al 14/09/2026",
			"Estado: Finalizado",
			"Generado: 28/10/2026 10:30",
			"Historias",
			"Login Hecho 5 10 11",
			"Tablero Hecho 3 6 6",
			"Perfil Hecho 2 Sin estimar 4",
			"Reportes En progreso – volvió al Backlog 8 12 3",
			"3 de 4 historias completadas (75,00 %)",
			"Estimaciones",
			"18 Story Points planificados y 10 completados",
			"28 horas estimadas en total (1 historia sin horas estimadas)",
			"Esfuerzo registrado",
			"24 horas registradas en total",
			"Ana: 17 h",
			"Beto: 7 h",
			"Métricas",
			"Velocidad: 10 Story Points",
			"Desviación de esfuerzo: +6,25 %",
			"Cálculo parcial: basado en 2 de 3 historias",
			"Defectos",
			"El login no valida la contraseña Alta Resuelto Login Sprint 2",
			"La foto de perfil no se guarda Media Abierto Perfil —",
			"2 defectos detectados y 1 resuelto en este Sprint",
			"Página 1 de 1",
		)
		wantNot(t, text, "Reporte parcial", "Se calculan al cerrar el Sprint")
	})

	t.Run("Sprint 1 sin defectos", func(t *testing.T) {
		text := renderText(t, finishedReport(alfaSprint1(), alfaSprint1Metrics(), alfaSprint1Quality()))
		wantInOrder(t, text,
			"Alta Hecho 3 4 4",
			"1 de 1 historias completadas (100,00 %)",
			"4 horas estimadas en total",
			"Desviación de esfuerzo: 0,00 %",
			"No se registraron defectos en este Sprint",
		)
		wantNot(t, text, "Cálculo parcial", "sin horas estimadas", "Descripción")
	})

	t.Run("Sprint 3 en curso", func(t *testing.T) {
		text := renderText(t, SprintReport{Data: alfaSprint3(), Partial: true, GeneratedAt: fixedNow})
		wantInOrder(t, text,
			"Reporte del Sprint",
			"Reporte parcial: Sprint en curso",
			"Sprint 3",
			"Estado: En curso",
			"Exportar En progreso 5 6 2",
			"Ayuda En Sprint Sin estimar Sin estimar 0",
			"Historias completadas: Se calculan al cerrar el Sprint",
			"Story Points planificados y completados: Se calculan al cerrar el Sprint",
			"6 horas estimadas en total (1 historia sin horas estimadas)",
			"2 horas registradas en total",
			"Ana: 2 h",
			"Velocidad y desviación: Se calculan al cerrar el Sprint",
			"El PDF sale en blanco Crítica Abierto Exportar —",
			"1 defecto detectado hasta el momento",
		)
		if n := strings.Count(text, "Se calculan al cerrar el Sprint"); n != 3 {
			t.Errorf("\"Se calculan al cerrar el Sprint\" aparece %d veces, se esperaban 3", n)
		}
		wantNot(t, text, "Velocidad:", "volvió al Backlog", "Estado: Finalizado")
	})

	t.Run("Sprint sin historias", func(t *testing.T) {
		data := Data{Sprint: SprintInfo{ProjectName: "Gamma", Number: 3, Goal: "Cerrar", StartDate: day("2026-09-29"),
			EndDate: day("2026-10-12"), Status: "Finalizado"}}
		q := metrics.SprintQuality{CompletionPercentage: "No disponible"}
		text := renderText(t, finishedReport(data, metrics.SprintMetrics{DeviationPercentage: "No disponible"}, q))
		wantInOrder(t, text,
			"No hay historias registradas en este Sprint",
			"0 de 0 historias completadas (No disponible)",
			"0 Story Points planificados y 0 completados",
			"0 horas estimadas en total",
			"0 horas registradas en total",
			"Velocidad: 0 Story Points",
			"Desviación de esfuerzo: No disponible",
			"No se registraron defectos en este Sprint",
		)
	})
}
