package web

import (
	"fmt"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// Textos del Dashboard (HU-08), en un solo lugar.
const (
	TitleDashboard              = "Dashboard"
	MsgDashboardNoData          = "Aún no hay datos suficientes para calcular métricas"
	MsgDashboardNoActiveSprint  = "No hay un Sprint en curso"
	MsgDashboardLastTenSprints  = "Se muestran los últimos 10 Sprints finalizados"
	MsgDashboardOneOpenDefect   = "1 defecto abierto"
	MsgDashboardNoOpenDefects   = "No hay defectos abiertos"
	msgDashboardActiveSprint    = "Sprint %d en curso"
	msgDashboardSprintPoints    = "%d Story Points planificados y %d completados"
	msgDashboardTotalVelocity   = "Velocidad acumulada: %d Story Points"
	msgDashboardAverageVelocity = "Promedio: %s Story Points por Sprint"
	msgDashboardDeviation       = "Desviación de esfuerzo del Sprint %d: %s"
	msgDashboardOpenDefects     = "%d defectos abiertos"
	titleDashboardStoriesChart  = "Historias por estado"
	titleDashboardVelocityChart = "Velocidad por Sprint"

	// dashboardMaxVelocityBars es la cantidad de Sprints finalizados que muestra el gráfico.
	dashboardMaxVelocityBars = 10
)

// dashboardStoryLabels son los estados de las historias, en el orden del gráfico y la tabla.
var dashboardStoryLabels = []string{"Pendiente", "En Sprint", "En progreso", "Hecho"}

// dashboardVelocityRow es un Sprint finalizado del gráfico de velocidad.
type dashboardVelocityRow struct {
	Label  string // "Sprint 1"
	Points int
}

// dashboardView es lo que muestra la plantilla dashboard.html.
type dashboardView struct {
	NoData                bool // proyecto sin historias, sin Sprints: solo NoDataMessage
	NoDataMessage         string
	NoActiveSprintMessage string

	StoriesChart dashboardChart
	StoryRows    []dashboardBar // las mismas barras, para la tabla

	HasActiveSprint bool
	ActiveSprint    string // "Sprint 3 en curso"
	ActiveCounts    [3]int // En Sprint, En progreso y Hecho
	ActivePoints    string // "11 Story Points planificados y 2 completados"

	HasFinished     bool // sin Sprints finalizados, velocidad y desviación muestran NoDataMessage
	VelocityChart   dashboardChart
	VelocityRows    []dashboardVelocityRow // los últimos 10, del más antiguo al más reciente
	LastTenNotice   string
	TotalVelocity   string // "Velocidad acumulada: 21 Story Points"
	AverageVelocity string // "Promedio: 10,50 Story Points por Sprint"
	Deviation       string // "Desviación de esfuerzo del Sprint 2: +20,00 %"
	PartialWarning  string // aviso de cálculo parcial de HU-07, tal cual

	OpenDefects string // "3 defectos abiertos"
}

// dashboardDecimalText pasa el decimal con punto de HU-07 ("10.50") al formato de pantalla ("10,50").
func dashboardDecimalText(value string) string {
	return strings.Replace(value, ".", ",", 1)
}

// dashboardOpenDefectsText arma el texto de defectos abiertos.
func dashboardOpenDefectsText(n int) string {
	switch n {
	case 0:
		return MsgDashboardNoOpenDefects
	case 1:
		return MsgDashboardOneOpenDefect
	default:
		return fmt.Sprintf(msgDashboardOpenDefects, n)
	}
}

// buildDashboardView arma la vista con los datos ya leídos: el resumen del catálogo, el Tablero del
// Sprint activo (found indica si hay uno), los Story Points completados de cada Sprint finalizado
// (en el orden de summary.Finished) y las métricas de HU-07 del último Sprint finalizado.
func buildDashboardView(summary DashboardSummary, board BoardView, found bool, velocities []int, deviation SprintMetrics) dashboardView {
	view := dashboardView{NoDataMessage: MsgDashboardNoData, NoActiveSprintMessage: MsgDashboardNoActiveSprint}
	stories := summary.Stories
	if stories == (DashboardStoryCounts{}) && len(summary.Finished) == 0 && !found {
		view.NoData = true
		return view
	}

	view.StoriesChart = newDashboardChart(titleDashboardStoriesChart, dashboardStoryLabels,
		[]int{stories.Pending, stories.InSprint, stories.InProgress, stories.Done})
	view.StoryRows = view.StoriesChart.Bars
	if found {
		setDashboardActiveSprint(&view, board)
	}
	if len(summary.Finished) > 0 {
		setDashboardVelocity(&view, summary.Finished, velocities, deviation)
	}
	view.OpenDefects = dashboardOpenDefectsText(summary.OpenDefects)
	return view
}

// setDashboardActiveSprint completa el bloque del Sprint en curso con el Tablero de HU-14.
func setDashboardActiveSprint(view *dashboardView, board BoardView) {
	view.HasActiveSprint = true
	view.ActiveSprint = fmt.Sprintf(msgDashboardActiveSprint, board.Number)
	for i, group := range board.Groups {
		if i < len(view.ActiveCounts) {
			view.ActiveCounts[i] = len(group.Stories)
		}
	}
	view.ActivePoints = fmt.Sprintf(msgDashboardSprintPoints, board.PlannedPoints, board.CompletedPoints)
}

// setDashboardVelocity completa el bloque de velocidad y desviación: el gráfico con los últimos 10
// Sprints finalizados, la acumulada y el promedio de todos, y la desviación del último.
func setDashboardVelocity(view *dashboardView, finished []DashboardSprint, velocities []int, deviation SprintMetrics) {
	n := len(finished)
	view.HasFinished = true
	start := 0
	if n > dashboardMaxVelocityBars {
		start = n - dashboardMaxVelocityBars
		view.LastTenNotice = MsgDashboardLastTenSprints
	}
	for i := start; i < n; i++ {
		points := 0
		if i < len(velocities) {
			points = velocities[i]
		}
		view.VelocityRows = append(view.VelocityRows, dashboardVelocityRow{
			Label:  fmt.Sprintf("Sprint %d", finished[i].Number),
			Points: points,
		})
	}
	view.VelocityChart = newDashboardVelocityChart(view.VelocityRows)

	velocity := metrics.SummarizeVelocity(velocities)
	view.TotalVelocity = fmt.Sprintf(msgDashboardTotalVelocity, velocity.Total)
	view.AverageVelocity = fmt.Sprintf(msgDashboardAverageVelocity, dashboardDecimalText(velocity.Average))

	view.Deviation = fmt.Sprintf(msgDashboardDeviation, finished[n-1].Number, metricsDeviationText(deviation.Deviation))
	if deviation.Partial {
		view.PartialWarning = deviation.PartialAlert
	}
}
