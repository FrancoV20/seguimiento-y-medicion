package web

import (
	"reflect"
	"testing"
)

// HU-08 (T015) [E2, E5, E11]: alturas de las barras y armado de los gráficos SVG, sin base de datos.

func TestDashboardBarHeights(t *testing.T) {
	tests := []struct {
		name   string
		values []int
		want   []int
	}{
		{name: "historias por estado de Alfa", values: []int{2, 1, 2, 3}, want: []int{80, 40, 80, 120}},
		{name: "velocidad 8 y 13 (73,85 sube a 74)", values: []int{8, 13}, want: []int{74, 120}},
		{name: "todo en cero, sin dividir", values: []int{0, 0, 0, 0}, want: []int{0, 0, 0, 0}},
		{name: "un valor chico mide al menos 1", values: []int{1, 1000}, want: []int{1, 120}},
		{name: "lista vacía", values: []int{}, want: []int{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dashboardBarHeights(tc.values, dashboardChartHeight)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dashboardBarHeights(%v) = %v; se esperaba %v", tc.values, got, tc.want)
			}
		})
	}
}

func TestNewDashboardChart(t *testing.T) {
	chart := newDashboardChart("Historias por estado",
		[]string{"Pendiente", "En Sprint", "En progreso", "Hecho"}, []int{2, 1, 2, 3})

	if chart.AriaLabel != "Historias por estado: Pendiente 2, En Sprint 1, En progreso 2, Hecho 3" {
		t.Errorf("AriaLabel = %q", chart.AriaLabel)
	}
	wantWidth := 4*(dashboardBarWidth+dashboardBarGap) + dashboardBarGap
	if chart.Width != wantWidth || chart.Height != dashboardChartHeight+30 {
		t.Errorf("tamaño = %dx%d; se esperaba %dx%d", chart.Width, chart.Height, wantWidth, dashboardChartHeight+30)
	}
	wantHeights := []int{80, 40, 80, 120}
	if len(chart.Bars) != 4 {
		t.Fatalf("barras = %d; se esperaban 4", len(chart.Bars))
	}
	for i, bar := range chart.Bars {
		wantX := dashboardBarGap + i*(dashboardBarWidth+dashboardBarGap)
		wantY := 10 + dashboardChartHeight - wantHeights[i]
		if bar.X != wantX || bar.Y != wantY || bar.Height != wantHeights[i] {
			t.Errorf("barra %d = x %d, y %d, alto %d; se esperaba x %d, y %d, alto %d",
				i, bar.X, bar.Y, bar.Height, wantX, wantY, wantHeights[i])
		}
	}
	if chart.Bars[0].Label != "Pendiente" || chart.Bars[0].Value != 2 || chart.Bars[3].Label != "Hecho" || chart.Bars[3].Value != 3 {
		t.Errorf("etiquetas o valores incorrectos: %+v", chart.Bars)
	}
}

func TestNewDashboardChart_SinBarras(t *testing.T) {
	chart := newDashboardChart("Historias por estado", nil, nil)
	if len(chart.Bars) != 0 || chart.Width != dashboardBarGap {
		t.Errorf("un gráfico sin valores no debe tener barras: %+v", chart)
	}
}
