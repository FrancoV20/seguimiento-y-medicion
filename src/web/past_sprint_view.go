package web

import (
	"fmt"
	"strconv"
	"time"
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
