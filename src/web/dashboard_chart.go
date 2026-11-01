package web

import (
	"fmt"
	"strings"
)

// Medidas de los gráficos de barras del Dashboard (HU-08), en unidades del SVG.
const (
	dashboardChartHeight = 120 // alto del área de barras
	dashboardBarWidth    = 40
	dashboardBarGap      = 16
	dashboardChartTop    = 10 // margen superior
)

// dashboardBarHeights devuelve el alto de cada barra, proporcional al mayor valor: redondeo mitad
// hacia arriba, y un valor mayor que 0 nunca mide menos de 1. Si el máximo es 0, todas miden 0 (no
// se divide).
func dashboardBarHeights(values []int, maxHeight int) []int {
	heights := make([]int, len(values))
	highest := 0
	for _, v := range values {
		if v > highest {
			highest = v
		}
	}
	if highest == 0 {
		return heights
	}
	for i, v := range values {
		h := (v*maxHeight*2 + highest) / (2 * highest)
		if v > 0 && h < 1 {
			h = 1
		}
		heights[i] = h
	}
	return heights
}

// dashboardBar es una barra del gráfico, con su posición dentro del SVG.
type dashboardBar struct {
	Label  string // "Pendiente" o "Sprint 3"
	Value  int
	X, Y   int // esquina superior izquierda de la barra
	Height int
	TextX  int // centro de la barra, para la etiqueta y el valor
	LabelY int // debajo de la línea base
	ValueY int // encima de la barra
}

// dashboardChart es un gráfico de barras SVG armado en el servidor, sin JavaScript.
type dashboardChart struct {
	AriaLabel string // descripción accesible con todos los valores
	Width     int
	Height    int
	BarWidth  int
	Bars      []dashboardBar
}

// newDashboardChart arma un gráfico de barras con una barra por valor. La descripción accesible es
// "Título: Etiqueta valor, Etiqueta valor, …".
func newDashboardChart(title string, labels []string, values []int) dashboardChart {
	heights := dashboardBarHeights(values, dashboardChartHeight)
	chart := dashboardChart{
		Width:    len(values)*(dashboardBarWidth+dashboardBarGap) + dashboardBarGap,
		Height:   dashboardChartHeight + 30,
		BarWidth: dashboardBarWidth,
		Bars:     make([]dashboardBar, 0, len(values)),
	}
	parts := make([]string, 0, len(values))
	baseline := dashboardChartTop + dashboardChartHeight
	for i, v := range values {
		x := dashboardBarGap + i*(dashboardBarWidth+dashboardBarGap)
		y := baseline - heights[i]
		chart.Bars = append(chart.Bars, dashboardBar{
			Label:  labels[i],
			Value:  v,
			X:      x,
			Y:      y,
			Height: heights[i],
			TextX:  x + dashboardBarWidth/2,
			LabelY: baseline + 15,
			ValueY: y - 2,
		})
		parts = append(parts, fmt.Sprintf("%s %d", labels[i], v))
	}
	chart.AriaLabel = title + ": " + strings.Join(parts, ", ")
	return chart
}

// newDashboardVelocityChart arma el gráfico de velocidad, con la descripción accesible
// "Velocidad por Sprint: Sprint 1, 8 Story Points; Sprint 2, 13 Story Points".
func newDashboardVelocityChart(rows []dashboardVelocityRow) dashboardChart {
	labels := make([]string, 0, len(rows))
	values := make([]int, 0, len(rows))
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, row.Label)
		values = append(values, row.Points)
		parts = append(parts, fmt.Sprintf("%s, %d Story Points", row.Label, row.Points))
	}
	chart := newDashboardChart(titleDashboardVelocityChart, labels, values)
	chart.AriaLabel = titleDashboardVelocityChart + ": " + strings.Join(parts, "; ")
	return chart
}
