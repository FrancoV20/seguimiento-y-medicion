package web

import (
	"fmt"
	"strconv"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-16: textos y armado de las vistas de "Sprints anteriores". Todos los textos viven acá.
const (
	TitlePastSprints = "Sprints anteriores"

	MsgPastSprintsEmpty        = "Aún no hay Sprints anteriores"
	MsgPastSprintAllCompleted  = "Todas las historias se completaron"
	MsgPastSprintNoneCompleted = "Ninguna historia se completó"
	MsgPastSprintNoRecord      = "No hay registro de las historias de este Sprint"
	LabelPastSprintUnestimated = "Sin estimar"
	LabelPastSprintReturned    = "volvió al Backlog"
	LinkPastSprintsBack        = "Volver a Sprints anteriores"

	msgPastSprintLabel    = "Sprint %d"
	msgPastSprintGoal     = "Objetivo: %s"
	msgPastSprintDates    = "Del %s al %s"
	msgPastSprintVelocity = "Velocidad: %d Story Points"

	pathPastSprints = "/sprints-anteriores"
)

// pastSprintDateLayout es dd/mm/aaaa.
const pastSprintDateLayout = "02/01/2006"

// pastSprintDateText da formato dd/mm/aaaa a una fecha.
func pastSprintDateText(t time.Time) string {
	return t.Format(pastSprintDateLayout)
}

// pastSprintDatesText arma "Del 01/09/2026 al 14/09/2026".
func pastSprintDatesText(start, end time.Time) string {
	return fmt.Sprintf(msgPastSprintDates, pastSprintDateText(start), pastSprintDateText(end))
}

// pastSprintItem es un Sprint de la lista.
type pastSprintItem struct {
	Label string // "Sprint 2"
	Goal  string // "Gestionar perfiles y reportes"
	Dates string // "Del 01/09/2026 al 14/09/2026"
	Link  string // "/sprints-anteriores/{id}"
}

// pastSprintListView es lo que muestra past_sprints.html.
type pastSprintListView struct {
	Sprints []pastSprintItem // por número descendente
	Empty   string           // MsgPastSprintsEmpty si no hay Sprints
}

// buildPastSprintListView arma la lista en el orden recibido (el catálogo ya ordena).
func buildPastSprintListView(sprints []PastSprintSummary) pastSprintListView {
	view := pastSprintListView{Sprints: make([]pastSprintItem, 0, len(sprints))}
	for _, s := range sprints {
		view.Sprints = append(view.Sprints, pastSprintItem{
			Label: fmt.Sprintf(msgPastSprintLabel, s.Number),
			Goal:  s.Goal,
			Dates: pastSprintDatesText(s.StartDate, s.EndDate),
			Link:  pathPastSprints + "/" + strconv.FormatInt(s.ID, 10),
		})
	}
	if len(view.Sprints) == 0 {
		view.Empty = MsgPastSprintsEmpty
	}
	return view
}

// pastSprintPointsText muestra los Story Points al cerrar; sin estimar, "Sin estimar".
func pastSprintPointsText(points *int) string {
	if points == nil {
		return LabelPastSprintUnestimated
	}
	return strconv.Itoa(*points)
}

// pastSprintRow es una fila de las tablas Completadas o No completadas.
type pastSprintRow struct {
	Title       string
	StoryPoints string // "5" o "Sin estimar"
	FinalStatus string // "Hecho", "En progreso" o "En Sprint"
	Returned    string // LabelPastSprintReturned en las no completadas; vacío en las completadas
}

// pastSprintDetailView es lo que muestra past_sprint.html.
type pastSprintDetailView struct {
	Label    string // "Sprint 2"
	Goal     string // "Objetivo: Gestionar perfiles y reportes"
	Dates    string // "Del 01/09/2026 al 14/09/2026"
	Velocity string // "Velocidad: 10 Story Points"

	NoRecord      string          // MsgPastSprintNoRecord si no hay filas; entonces no hay tablas
	Completed     []pastSprintRow // por StoryID
	NoneCompleted string          // MsgPastSprintNoneCompleted si hay registro y no hay completadas
	Pending       []pastSprintRow // por StoryID
	AllCompleted  string          // MsgPastSprintAllCompleted si hay registro y no hay no completadas
	Back          string          // LinkPastSprintsBack
}

// buildPastSprintDetailView arma el detalle: separa el registro de cierre por estado al cerrar
// ("Hecho" = completada), conserva el orden y muestra la velocidad que calculó HU-07.
func buildPastSprintDetailView(detail PastSprintDetail, velocity int) pastSprintDetailView {
	view := pastSprintDetailView{
		Label:    fmt.Sprintf(msgPastSprintLabel, detail.Number),
		Goal:     fmt.Sprintf(msgPastSprintGoal, detail.Goal),
		Dates:    pastSprintDatesText(detail.StartDate, detail.EndDate),
		Velocity: fmt.Sprintf(msgPastSprintVelocity, velocity),
		Back:     LinkPastSprintsBack,
	}
	if len(detail.Stories) == 0 {
		view.NoRecord = MsgPastSprintNoRecord
		return view
	}
	for _, s := range detail.Stories {
		row := pastSprintRow{Title: s.Title, StoryPoints: pastSprintPointsText(s.StoryPoints), FinalStatus: s.FinalStatus}
		if s.FinalStatus == string(sprint.Done) {
			view.Completed = append(view.Completed, row)
			continue
		}
		row.Returned = LabelPastSprintReturned
		view.Pending = append(view.Pending, row)
	}
	if len(view.Completed) == 0 {
		view.NoneCompleted = MsgPastSprintNoneCompleted
	}
	if len(view.Pending) == 0 {
		view.AllCompleted = MsgPastSprintAllCompleted
	}
	return view
}
