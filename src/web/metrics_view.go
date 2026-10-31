package web

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// HU-25: textos y armado de la vista "Métricas del Sprint". Los textos propios de la historia
// viven acá; "No disponible" y "Cálculo parcial: ..." llegan de HU-07 y no se reescriben.
const (
	TitleMetrics = "Métricas del Sprint"

	// MsgMetricsNoSprints se muestra cuando el proyecto no tiene Sprints finalizados.
	MsgMetricsNoSprints = "Aún no hay Sprints finalizados"
	// MsgMetricsActiveHint avisa que el Sprint activo (número) se calcula al cerrarlo.
	MsgMetricsActiveHint = "El Sprint %d sigue en curso; sus métricas se calculan al cerrarlo"
	// MsgMetricsDeviationFormula acompaña a la desviación.
	MsgMetricsDeviationFormula = "(horas reales − horas estimadas) ÷ horas estimadas"

	msgMetricsSprintLabel = "Sprint %d"
	msgMetricsDates       = "Del %s al %s"
	msgMetricsVelocity    = "%d Story Points"
)

// metricsDateLayout es dd/mm/aaaa, como HU-12 y HU-15.
const metricsDateLayout = "02/01/2006"

// MetricsSprintRow es un Sprint finalizado de la lista.
type MetricsSprintRow struct {
	ID       int64
	Label    string // "Sprint 3"
	Goal     string
	Dates    string // "Del 01/10/2026 al 14/10/2026"
	Selected bool
}

// MetricsView es lo que muestra metrics.html.
type MetricsView struct {
	Sprints      []MetricsSprintRow
	EmptyMessage string // MsgMetricsNoSprints cuando no hay Sprints finalizados

	HasSelected      bool
	SelectedLabel    string
	SelectedGoal     string
	SelectedDates    string
	VelocityText     string // "8 Story Points"
	DeviationText    string // "+20,00 %" o "No disponible"
	DeviationFormula string
	PartialAlert     string // Warning de HU-07 o vacío

	ActiveHint string // MsgMetricsActiveHint con el número del Sprint activo, o vacío
}

// metricsViewInput reúne lo leído para armar la vista.
type metricsViewInput struct {
	Sprints      []FinishedSprint
	Selected     FinishedSprint
	HasSelected  bool
	Metrics      SprintMetrics
	ActiveNumber int64
	HasActive    bool
}

// metricsVelocityText da formato a la velocidad: "8 Story Points" (cero incluido).
func metricsVelocityText(velocity int) string {
	return fmt.Sprintf(msgMetricsVelocity, velocity)
}

// metricsDeviationText presenta el texto de desviación de HU-07 sin recalcularlo ni redondearlo:
// "20.00" pasa a "+20,00 %", "-8.50" a "-8,50 %" y "0.00" a "0,00 %". Un texto que no es un
// número (el "No disponible" de HU-07) se muestra tal cual.
func metricsDeviationText(deviation string) string {
	value, ok := new(big.Rat).SetString(deviation)
	if !ok {
		return deviation
	}
	text := strings.Replace(deviation, ".", ",", 1) + " %"
	if value.Sign() > 0 {
		text = "+" + text
	}
	return text
}

// selectMetricsSprint elige el Sprint de la lista según el parámetro sprint de la dirección: el de
// ese identificador si es un número válido y está en la lista; si no (vacío, no numérico,
// inexistente, activo o de otro proyecto), el primero, que es el más reciente. Sin lista no hay
// elección.
func selectMetricsSprint(sprints []FinishedSprint, raw string) (FinishedSprint, bool) {
	if len(sprints) == 0 {
		return FinishedSprint{}, false
	}
	if id := parseID(raw); id != 0 {
		for _, s := range sprints {
			if s.ID == id {
				return s, true
			}
		}
	}
	return sprints[0], true
}

// buildMetricsView arma la vista de la página con las métricas de HU-07 ya obtenidas.
func buildMetricsView(in metricsViewInput) MetricsView {
	view := MetricsView{}
	if in.HasActive {
		view.ActiveHint = fmt.Sprintf(MsgMetricsActiveHint, in.ActiveNumber)
	}
	if len(in.Sprints) == 0 {
		view.EmptyMessage = MsgMetricsNoSprints
		return view
	}

	view.Sprints = make([]MetricsSprintRow, 0, len(in.Sprints))
	for _, s := range in.Sprints {
		view.Sprints = append(view.Sprints, MetricsSprintRow{
			ID:       s.ID,
			Label:    fmt.Sprintf(msgMetricsSprintLabel, s.Number),
			Goal:     s.Goal,
			Dates:    metricsDatesText(s.StartDate, s.EndDate),
			Selected: in.HasSelected && s.ID == in.Selected.ID,
		})
	}
	if !in.HasSelected {
		return view
	}

	view.HasSelected = true
	view.SelectedLabel = fmt.Sprintf(msgMetricsSprintLabel, in.Selected.Number)
	view.SelectedGoal = in.Selected.Goal
	view.SelectedDates = metricsDatesText(in.Selected.StartDate, in.Selected.EndDate)
	view.VelocityText = metricsVelocityText(in.Metrics.Velocity)
	view.DeviationText = metricsDeviationText(in.Metrics.Deviation)
	view.DeviationFormula = MsgMetricsDeviationFormula
	if in.Metrics.Partial {
		view.PartialAlert = in.Metrics.PartialAlert
	}
	return view
}

// metricsDatesText arma "Del 01/10/2026 al 14/10/2026".
func metricsDatesText(start, end time.Time) string {
	return fmt.Sprintf(msgMetricsDates, start.Format(metricsDateLayout), end.Format(metricsDateLayout))
}
