package web

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-08 (T016) [E1, E4, E5, E6, E7, E8, E9]: textos y armado de la vista del Dashboard con datos en
// memoria, sin base de datos.

func TestDashboardTexts(t *testing.T) {
	if got := dashboardDecimalText("10.50"); got != "10,50" {
		t.Errorf("dashboardDecimalText(10.50) = %q", got)
	}
	for n, want := range map[int]string{0: "No hay defectos abiertos", 1: "1 defecto abierto", 3: "3 defectos abiertos"} {
		if got := dashboardOpenDefectsText(n); got != want {
			t.Errorf("dashboardOpenDefectsText(%d) = %q; se esperaba %q", n, got, want)
		}
	}
}

// dashboardTestBoard arma un Tablero con la cantidad de historias de cada grupo.
func dashboardTestBoard(number int64, inSprint, inProgress, done, planned, completed int) BoardView {
	group := func(status sprint.StoryStatus, n int) BoardGroup {
		return BoardGroup{Status: status, Stories: make([]BoardStory, n)}
	}
	return BoardView{
		Number: number,
		Label:  fmt.Sprintf("Sprint %d", number),
		Groups: []BoardGroup{
			group(sprint.EnSprint, inSprint),
			group(sprint.InProgress, inProgress),
			group(sprint.Done, done),
		},
		PlannedPoints:   planned,
		CompletedPoints: completed,
	}
}

func dashboardTestFinished(numbers ...int) []DashboardSprint {
	out := make([]DashboardSprint, 0, len(numbers))
	for _, n := range numbers {
		out = append(out, DashboardSprint{ID: int64(100 + n), Number: n})
	}
	return out
}

func TestBuildDashboardView(t *testing.T) {
	alfa := DashboardSummary{
		Stories:     DashboardStoryCounts{Pending: 2, InSprint: 1, InProgress: 2, Done: 3},
		OpenDefects: 3,
		Finished:    dashboardTestFinished(1, 2),
	}
	board := dashboardTestBoard(3, 1, 2, 1, 11, 2)

	t.Run("Alfa con todos los bloques", func(t *testing.T) {
		view := buildDashboardView(alfa, board, true, []int{8, 13}, SprintMetrics{Deviation: "20.00"})

		if view.NoData {
			t.Fatal("Alfa tiene datos")
		}
		if got := dashboardBarValues(view.StoryRows); !reflect.DeepEqual(got, []int{2, 1, 2, 3}) {
			t.Errorf("historias por estado = %v", got)
		}
		if !view.HasActiveSprint || view.ActiveSprint != "Sprint 3 en curso" || view.ActiveCounts != [3]int{1, 2, 1} ||
			view.ActivePoints != "11 Story Points planificados y 2 completados" {
			t.Errorf("Sprint en curso = %v %q %v %q", view.HasActiveSprint, view.ActiveSprint, view.ActiveCounts, view.ActivePoints)
		}
		if !view.HasFinished || len(view.VelocityRows) != 2 || view.VelocityRows[0] != (dashboardVelocityRow{Label: "Sprint 1", Points: 8}) ||
			view.VelocityRows[1] != (dashboardVelocityRow{Label: "Sprint 2", Points: 13}) {
			t.Errorf("velocidad = %+v", view.VelocityRows)
		}
		if view.VelocityChart.AriaLabel != "Velocidad por Sprint: Sprint 1, 8 Story Points; Sprint 2, 13 Story Points" {
			t.Errorf("AriaLabel de velocidad = %q", view.VelocityChart.AriaLabel)
		}
		if view.TotalVelocity != "Velocidad acumulada: 21 Story Points" || view.AverageVelocity != "Promedio: 10,50 Story Points por Sprint" {
			t.Errorf("acumulada y promedio = %q, %q", view.TotalVelocity, view.AverageVelocity)
		}
		if view.LastTenNotice != "" {
			t.Errorf("con 2 Sprints no debe avisar de los últimos 10: %q", view.LastTenNotice)
		}
		if view.Deviation != "Desviación de esfuerzo del Sprint 2: +20,00 %" || view.PartialWarning != "" {
			t.Errorf("desviación = %q, aviso %q", view.Deviation, view.PartialWarning)
		}
		if view.OpenDefects != "3 defectos abiertos" {
			t.Errorf("defectos = %q", view.OpenDefects)
		}
	})

	t.Run("E1: sin historias, sin finalizados y sin Sprint activo", func(t *testing.T) {
		view := buildDashboardView(DashboardSummary{}, BoardView{}, false, nil, SprintMetrics{})
		if !view.NoData {
			t.Error("un proyecto sin nada debe mostrar solo el mensaje de falta de datos")
		}
	})

	t.Run("un Sprint sin historias no es falta de datos", func(t *testing.T) {
		view := buildDashboardView(DashboardSummary{Finished: dashboardTestFinished(1)}, BoardView{}, false, []int{0},
			SprintMetrics{Deviation: "No disponible"})
		if view.NoData {
			t.Error("con un Sprint finalizado hay datos")
		}
		if view.AverageVelocity != "Promedio: 0,00 Story Points por Sprint" || view.Deviation != "Desviación de esfuerzo del Sprint 1: No disponible" {
			t.Errorf("promedio y desviación = %q, %q", view.AverageVelocity, view.Deviation)
		}
	})

	t.Run("E4: sin Sprint activo", func(t *testing.T) {
		view := buildDashboardView(DashboardSummary{Stories: DashboardStoryCounts{Pending: 2}}, BoardView{}, false, nil, SprintMetrics{})
		if view.NoData || view.HasActiveSprint {
			t.Errorf("NoData=%v HasActiveSprint=%v; se esperaba false y false", view.NoData, view.HasActiveSprint)
		}
	})

	t.Run("E6: sin Sprints finalizados", func(t *testing.T) {
		view := buildDashboardView(DashboardSummary{Stories: DashboardStoryCounts{Pending: 1, InSprint: 1}},
			dashboardTestBoard(1, 1, 0, 0, 3, 0), true, nil, SprintMetrics{})
		if view.NoData || view.HasFinished || !view.HasActiveSprint {
			t.Errorf("NoData=%v HasFinished=%v HasActiveSprint=%v", view.NoData, view.HasFinished, view.HasActiveSprint)
		}
	})

	t.Run("doce finalizados: 10 barras y acumulada de los 12", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
		points := []int{21, 21, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
		view := buildDashboardView(DashboardSummary{Stories: DashboardStoryCounts{Done: 12}, Finished: dashboardTestFinished(numbers...)},
			BoardView{}, false, points, SprintMetrics{Deviation: "No disponible"})
		labels := make([]string, 0, len(view.VelocityRows))
		for _, row := range view.VelocityRows {
			labels = append(labels, row.Label)
		}
		want := []string{"Sprint 3", "Sprint 4", "Sprint 5", "Sprint 6", "Sprint 7", "Sprint 8", "Sprint 9", "Sprint 10", "Sprint 11", "Sprint 12"}
		if !reflect.DeepEqual(labels, want) || len(view.VelocityChart.Bars) != 10 {
			t.Errorf("barras de velocidad = %v (%d en el gráfico)", labels, len(view.VelocityChart.Bars))
		}
		if view.LastTenNotice != "Se muestran los últimos 10 Sprints finalizados" {
			t.Errorf("aviso de los últimos 10 = %q", view.LastTenNotice)
		}
		if view.TotalVelocity != "Velocidad acumulada: 92 Story Points" || view.AverageVelocity != "Promedio: 7,67 Story Points por Sprint" {
			t.Errorf("acumulada y promedio = %q, %q", view.TotalVelocity, view.AverageVelocity)
		}
		if view.Deviation != "Desviación de esfuerzo del Sprint 12: No disponible" {
			t.Errorf("la desviación debe ser la del último finalizado: %q", view.Deviation)
		}
	})

	t.Run("E8 y aviso parcial", func(t *testing.T) {
		alert := "Cálculo parcial: basado en 0 de 1 historias"
		view := buildDashboardView(DashboardSummary{Stories: DashboardStoryCounts{Done: 1}, Finished: dashboardTestFinished(1)},
			BoardView{}, false, []int{5}, SprintMetrics{Deviation: "No disponible", Partial: true, PartialAlert: alert})
		if view.Deviation != "Desviación de esfuerzo del Sprint 1: No disponible" || view.PartialWarning != alert {
			t.Errorf("desviación = %q, aviso %q", view.Deviation, view.PartialWarning)
		}
		if view.TotalVelocity != "Velocidad acumulada: 5 Story Points" {
			t.Errorf("la velocidad debe mostrarse igual: %q", view.TotalVelocity)
		}
	})

	t.Run("el aviso parcial solo aparece si el cálculo es parcial", func(t *testing.T) {
		view := buildDashboardView(alfa, board, true, []int{8, 13}, SprintMetrics{Deviation: "20.00", PartialAlert: "no se muestra"})
		if view.PartialWarning != "" {
			t.Errorf("aviso parcial = %q; se esperaba vacío", view.PartialWarning)
		}
	})
}

func dashboardBarValues(bars []dashboardBar) []int {
	out := make([]int, 0, len(bars))
	for _, bar := range bars {
		out = append(out, bar.Value)
	}
	return out
}
