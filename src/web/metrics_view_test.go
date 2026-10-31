package web

import (
	"fmt"
	"testing"
	"time"
)

// HU-25 (T010): formato de la vista y elección del Sprint, sin base de datos.

func TestMetricsDeviationText_DosDecimalesComaYSigno(t *testing.T) {
	casos := []struct {
		hu07 string // DeviationPercentage de HU-07
		want string
	}{
		{"20.00", "+20,00 %"},
		{"-8.50", "-8,50 %"},
		{"0.00", "0,00 %"},
		{"150.25", "+150,25 %"},
		{"-100.00", "-100,00 %"},
		{"0.01", "+0,01 %"},
	}
	for _, c := range casos {
		if got := metricsDeviationText(c.hu07); got != c.want {
			t.Errorf("metricsDeviationText(%q) = %q; se esperaba %q", c.hu07, got, c.want)
		}
	}
}

func TestMetricsDeviationText_NoDisponibleSeMuestraComoLoDaHU07(t *testing.T) {
	noData := hu07NoData(t)
	if got := metricsDeviationText(noData); got != noData {
		t.Errorf("metricsDeviationText(%q) = %q; se esperaba el texto de HU-07 sin cambios", noData, got)
	}
}

func TestMetricsVelocityText(t *testing.T) {
	casos := map[int]string{0: "0 Story Points", 8: "8 Story Points", 34: "34 Story Points"}
	for velocity, want := range casos {
		if got := metricsVelocityText(velocity); got != want {
			t.Errorf("metricsVelocityText(%d) = %q; se esperaba %q", velocity, got, want)
		}
	}
}

// metricsTestSprints es una lista del más reciente al más antiguo, como la da el catálogo.
func metricsTestSprints() []FinishedSprint {
	day := func(s string) time.Time {
		d, _ := time.Parse("2006-01-02", s)
		return d
	}
	return []FinishedSprint{
		{ID: 30, Number: 3, Goal: "Reportes", StartDate: day("2026-09-29"), EndDate: day("2026-10-12")},
		{ID: 20, Number: 2, Goal: "Pagos online", StartDate: day("2026-09-15"), EndDate: day("2026-09-28")},
		{ID: 10, Number: 1, Goal: "Alta de usuarios", StartDate: day("2026-09-01"), EndDate: day("2026-09-14")},
	}
}

func TestSelectMetricsSprint(t *testing.T) {
	list := metricsTestSprints()
	casos := []struct {
		name   string
		raw    string
		wantID int64
	}{
		{"sin parámetro: el más reciente", "", 30},
		{"válido de la lista", "20", 20},
		{"el más antiguo de la lista", "10", 10},
		{"no numérico", "abc", 30},
		{"con signo", "+20", 30},
		{"negativo", "-20", 30},
		{"con ceros a la izquierda", "020", 30},
		{"demasiado grande", "99999999999999999999", 30},
		{"inexistente", "999", 30},
		{"activo (no está en la lista)", "40", 30},
		{"de otro proyecto (no está en la lista)", "77", 30},
	}
	for _, c := range casos {
		t.Run(c.name, func(t *testing.T) {
			got, ok := selectMetricsSprint(list, c.raw)
			if !ok || got.ID != c.wantID {
				t.Errorf("selectMetricsSprint(%q) = %d, %v; se esperaba %d, true", c.raw, got.ID, ok, c.wantID)
			}
		})
	}
	t.Run("sin lista no hay Sprint elegido", func(t *testing.T) {
		if got, ok := selectMetricsSprint(nil, "20"); ok {
			t.Errorf("selectMetricsSprint(nil) = %+v, true; se esperaba sin elección", got)
		}
	})
}

func TestBuildMetricsView_ConSprintElegido(t *testing.T) {
	list := metricsTestSprints()
	view := buildMetricsView(metricsViewInput{
		Sprints:     list,
		Selected:    list[1],
		HasSelected: true,
		Metrics:     SprintMetrics{Velocity: 8, Deviation: "20.00", PartialUsed: 2, PartialTotal: 2},
	})

	if len(view.Sprints) != 3 {
		t.Fatalf("la vista tiene %d Sprints; se esperaban 3", len(view.Sprints))
	}
	wantRows := []MetricsSprintRow{
		{ID: 30, Label: "Sprint 3", Goal: "Reportes", Dates: "Del 29/09/2026 al 12/10/2026"},
		{ID: 20, Label: "Sprint 2", Goal: "Pagos online", Dates: "Del 15/09/2026 al 28/09/2026", Selected: true},
		{ID: 10, Label: "Sprint 1", Goal: "Alta de usuarios", Dates: "Del 01/09/2026 al 14/09/2026"},
	}
	for i, want := range wantRows {
		if view.Sprints[i] != want {
			t.Errorf("fila %d = %+v; se esperaba %+v", i, view.Sprints[i], want)
		}
	}
	if !view.HasSelected || view.SelectedLabel != "Sprint 2" || view.SelectedGoal != "Pagos online" ||
		view.SelectedDates != "Del 15/09/2026 al 28/09/2026" {
		t.Errorf("Sprint elegido = %v %q %q %q; se esperaba Sprint 2, Pagos online y sus fechas",
			view.HasSelected, view.SelectedLabel, view.SelectedGoal, view.SelectedDates)
	}
	if view.VelocityText != "8 Story Points" {
		t.Errorf("velocidad = %q; se esperaba %q", view.VelocityText, "8 Story Points")
	}
	if view.DeviationText != "+20,00 %" {
		t.Errorf("desviación = %q; se esperaba %q", view.DeviationText, "+20,00 %")
	}
	if view.DeviationFormula != MsgMetricsDeviationFormula {
		t.Errorf("fórmula = %q; se esperaba %q", view.DeviationFormula, MsgMetricsDeviationFormula)
	}
	if view.PartialAlert != "" || view.EmptyMessage != "" || view.ActiveHint != "" {
		t.Errorf("no se esperaban alerta, mensaje vacío ni aviso: %q %q %q",
			view.PartialAlert, view.EmptyMessage, view.ActiveHint)
	}
}

func TestBuildMetricsView_SinHorasEstimadasYVelocidadCero(t *testing.T) {
	list := metricsTestSprints()
	noData := hu07NoData(t)
	view := buildMetricsView(metricsViewInput{
		Sprints: list, Selected: list[0], HasSelected: true,
		Metrics: SprintMetrics{Velocity: 0, Deviation: noData},
	})
	if view.VelocityText != "0 Story Points" {
		t.Errorf("velocidad = %q; se esperaba %q", view.VelocityText, "0 Story Points")
	}
	if view.DeviationText != noData {
		t.Errorf("desviación = %q; se esperaba %q", view.DeviationText, noData)
	}
	if view.PartialAlert != "" {
		t.Errorf("no se esperaba alerta de cálculo parcial: %q", view.PartialAlert)
	}
}

func TestBuildMetricsView_CalculoParcialMuestraElAvisoDeHU07(t *testing.T) {
	list := metricsTestSprints()
	warning := hu07PartialWarning(t, 1, 3)
	view := buildMetricsView(metricsViewInput{
		Sprints: list, Selected: list[0], HasSelected: true,
		Metrics: SprintMetrics{
			Velocity: 10, Deviation: "-25.00",
			PartialUsed: 1, PartialTotal: 3, Partial: true, PartialAlert: warning,
		},
	})
	if view.PartialAlert != warning {
		t.Errorf("alerta = %q; se esperaba %q", view.PartialAlert, warning)
	}
	if view.DeviationText != "-25,00 %" {
		t.Errorf("desviación = %q; se esperaba %q", view.DeviationText, "-25,00 %")
	}
}

func TestBuildMetricsView_SinSprintsFinalizados(t *testing.T) {
	t.Run("sin Sprint activo", func(t *testing.T) {
		view := buildMetricsView(metricsViewInput{})
		if view.EmptyMessage != MsgMetricsNoSprints {
			t.Errorf("mensaje = %q; se esperaba %q", view.EmptyMessage, MsgMetricsNoSprints)
		}
		if view.HasSelected || len(view.Sprints) != 0 || view.VelocityText != "" || view.DeviationText != "" {
			t.Errorf("sin Sprints no debe haber métricas: %+v", view)
		}
		if view.ActiveHint != "" {
			t.Errorf("sin Sprint activo no debe haber aviso: %q", view.ActiveHint)
		}
	})
	t.Run("con Sprint activo", func(t *testing.T) {
		view := buildMetricsView(metricsViewInput{ActiveNumber: 1, HasActive: true})
		if view.EmptyMessage != MsgMetricsNoSprints {
			t.Errorf("mensaje = %q; se esperaba %q", view.EmptyMessage, MsgMetricsNoSprints)
		}
		if want := fmt.Sprintf(MsgMetricsActiveHint, 1); view.ActiveHint != want {
			t.Errorf("aviso = %q; se esperaba %q", view.ActiveHint, want)
		}
	})
}

func TestBuildMetricsView_AvisoDelSprintEnCurso(t *testing.T) {
	list := metricsTestSprints()
	view := buildMetricsView(metricsViewInput{
		Sprints: list, Selected: list[0], HasSelected: true,
		Metrics:      SprintMetrics{Velocity: 3, Deviation: "0.00"},
		ActiveNumber: 4, HasActive: true,
	})
	if want := fmt.Sprintf(MsgMetricsActiveHint, 4); view.ActiveHint != want {
		t.Errorf("aviso = %q; se esperaba %q", view.ActiveHint, want)
	}
	if view.EmptyMessage != "" {
		t.Errorf("con Sprints finalizados no debe mostrarse %q", view.EmptyMessage)
	}
	if view.DeviationText != "0,00 %" {
		t.Errorf("desviación = %q; se esperaba %q", view.DeviationText, "0,00 %")
	}
	for _, row := range view.Sprints {
		if row.Label == "Sprint 4" {
			t.Errorf("el Sprint activo no debe figurar en la lista")
		}
	}
}
