package report

import (
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/pdf"
)

// HU-09, T037: el PDF del proyecto, buscando los textos con pdf.ExtractText.

func renderProjectText(t *testing.T, r ProjectReport) string {
	t.Helper()
	b, err := RenderProject(r)
	if err != nil {
		t.Fatalf("RenderProject: %v", err)
	}
	text, err := pdf.ExtractText(b)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	return text
}

func alfaProjectReport() ProjectReport {
	m1, m2 := alfaSprint1Metrics(), alfaSprint2Metrics()
	q1, q2 := alfaSprint1Quality(), alfaSprint2Quality()
	return ProjectReport{
		Data: alfaProject(), Status: "En curso", Progress: "57.14", Partial: true,
		Sprints: []ProjectSprintRow{
			{Number: 1, Quality: &q1, Metrics: &m1},
			{Number: 2, Quality: &q2, Metrics: &m2},
			{Number: 3, Active: true},
		},
		Velocity:    metrics.SummarizeVelocity([]int{3, 10}),
		GeneratedAt: fixedNow,
	}
}

func TestRenderProjectReport(t *testing.T) {
	t.Run("«Alfa»", func(t *testing.T) {
		text := renderProjectText(t, alfaProjectReport())
		wantInOrder(t, text,
			"Reporte del proyecto",
			"Reporte parcial: hay un Sprint en curso",
			"Proyecto: Alfa",
			"Fechas: 18/08/2026 al 30/11/2026",
			"Estado: En curso",
			"Integrantes: Ana, Beto",
			"Generado: 28/10/2026 10:30",
			"Avance",
			"Pendiente 1", "En Sprint 1", "En progreso 1", "Hecho 4",
			"4 de 7 historias completadas (57,14 %)",
			"Sprints",
			"Sprint 1 Finalizado 1 de 1 3 de 3 3 0,00 % 0 detectados y 0 resueltos",
			"Sprint 2 Finalizado 3 de 4 10 de 18 10 +6,25 % (cálculo parcial) 2 detectados y 1 resuelto",
			"Sprint 3 En curso: se calcula al cerrar el Sprint — — — — —",
			"Velocidad acumulada: 13 Story Points",
			"Promedio: 6,50 Story Points por Sprint",
			"Esfuerzo registrado",
			"38 horas estimadas en total (2 historias sin horas estimadas)",
			"30 horas registradas en total",
			"Ana: 23 h", "Beto: 7 h",
			"Defectos",
			"3 defectos registrados: 2 abiertos y 1 resuelto",
			"La foto de perfil no se guarda Media Perfil Sprint 2",
			"El PDF sale en blanco Crítica Exportar Sprint 3",
			"Página 1 de 1",
		)
	})

	t.Run("«Beta» sin Sprints ni historias", func(t *testing.T) {
		text := renderProjectText(t, ProjectReport{Data: betaProject(), Status: "En curso", Progress: "0.00",
			Velocity: metrics.SummarizeVelocity(nil), GeneratedAt: fixedNow})
		wantInOrder(t, text,
			"Integrantes: Ana",
			"Pendiente 0", "En Sprint 0", "En progreso 0", "Hecho 0",
			"0 de 0 historias completadas (0,00 %)",
			"Aún no hay Sprints",
			"0 horas estimadas en total",
			"0 horas registradas en total",
			"No se registraron defectos en este proyecto",
		)
		wantNot(t, text, "Reporte parcial", "Velocidad acumulada", "Promedio", "No hay defectos abiertos")
	})

	t.Run("solo un Sprint en curso", func(t *testing.T) {
		data := betaProject()
		data.Sprints = []ProjectSprint{{ID: 5, Number: 1, Status: "Activo"}}
		text := renderProjectText(t, ProjectReport{Data: data, Status: "En curso", Progress: "0.00", Partial: true,
			Sprints: []ProjectSprintRow{{Number: 1, Active: true}}, Velocity: metrics.SummarizeVelocity(nil), GeneratedAt: fixedNow})
		wantInOrder(t, text, "Velocidad acumulada: 0 Story Points", "Promedio: No disponible")
	})

	t.Run("defectos todos resueltos", func(t *testing.T) {
		r := alfaProjectReport()
		r.Data.DefectsOpen, r.Data.DefectsResolved, r.Data.OpenDefects = 0, 2, nil
		text := renderProjectText(t, r)
		wantInOrder(t, text, "2 defectos registrados: 0 abiertos y 2 resueltos", "No hay defectos abiertos")
		wantNot(t, text, "Detectado en")
	})
}
