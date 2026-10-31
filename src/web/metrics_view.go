package web

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
)

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

// metricsVelocityText da formato a la velocidad: "8 Story Points".
func metricsVelocityText(velocity int) string {
	return ""
}

// metricsDeviationText da formato al texto de desviación de HU-07.
func metricsDeviationText(deviation string) string {
	return ""
}

// selectMetricsSprint elige el Sprint de la lista según el parámetro sprint de la dirección.
func selectMetricsSprint(sprints []FinishedSprint, raw string) (FinishedSprint, bool) {
	return FinishedSprint{}, false
}

// buildMetricsView arma la vista de la página.
func buildMetricsView(in metricsViewInput) MetricsView {
	return MetricsView{}
}
