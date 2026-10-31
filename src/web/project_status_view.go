package web

import (
	"fmt"
	"strings"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-12: textos y armado de la vista "Estado del proyecto". Todos los textos viven acá.
const (
	TitleProjectStatus = "Estado del proyecto"

	msgProjectStatusDates      = "Del %s al %s"
	msgProjectStatusState      = "Estado: %s"
	msgProjectStatusProgress   = "%d de %d historias completadas (%s)"
	msgProjectStatusSprint     = "%s en curso"
	msgProjectStatusSprintGoal = "Objetivo: %s"

	// MsgProjectStatusNoSprint se muestra cuando el proyecto no tiene un Sprint activo.
	MsgProjectStatusNoSprint = "No hay un Sprint en curso"
)

// projectStatusDateLayout es dd/mm/aaaa.
const projectStatusDateLayout = "02/01/2006"

// projectStatusDateText da formato dd/mm/aaaa a una fecha.
func projectStatusDateText(t time.Time) string {
	return t.Format(projectStatusDateLayout)
}

// projectStatusPercentText pasa "50.00" (punto, de project.ProgressPercentage) a "50,00 %".
func projectStatusPercentText(percentage string) string {
	return strings.Replace(percentage, ".", ",", 1) + " %"
}

// projectStatusRow es una fila de la tabla Estado / Historias.
type projectStatusRow struct {
	Label string
	Count int
}

// projectStatusView es lo que muestra project_status.html.
type projectStatusView struct {
	Name     string
	Dates    string             // "Del 01/10/2026 al 30/11/2026"
	State    string             // "Estado: En curso"
	Progress string             // "4 de 8 historias completadas (50,00 %)"
	Rows     []projectStatusRow // cuatro filas, en orden, con ceros

	HasSprint   bool
	SprintLabel string // "Sprint 2 en curso"
	SprintGoal  string // "Objetivo: Pagos online"
	SprintDates string // "Del 19/10/2026 al 01/11/2026"
	NoSprint    string // MsgProjectStatusNoSprint
}

// buildProjectStatusView arma la vista con las reglas del dominio. today es la fecha de referencia;
// found indica si el proyecto tiene un Sprint activo (board).
func buildProjectStatusView(summary ProjectStatusSummary, board BoardView, found bool, today time.Time) projectStatusView {
	counts := summary.Counts
	total := counts.Total()
	view := projectStatusView{
		Name:  summary.Name,
		Dates: projectStatusDatesText(summary.StartDate, summary.EndDate),
		State: fmt.Sprintf(msgProjectStatusState, project.StatusOn(summary.StartDate, summary.EndDate, today)),
		Progress: fmt.Sprintf(msgProjectStatusProgress, counts.Done, total,
			projectStatusPercentText(project.ProgressPercentage(counts.Done, total))),
		Rows: []projectStatusRow{
			{Label: string(sprint.Pendiente), Count: counts.Pending},
			{Label: string(sprint.EnSprint), Count: counts.InSprint},
			{Label: string(sprint.InProgress), Count: counts.InProgress},
			{Label: string(sprint.Done), Count: counts.Done},
		},
	}
	if !found {
		view.NoSprint = MsgProjectStatusNoSprint
		return view
	}
	view.HasSprint = true
	view.SprintLabel = fmt.Sprintf(msgProjectStatusSprint, board.Label)
	view.SprintGoal = fmt.Sprintf(msgProjectStatusSprintGoal, board.Goal)
	view.SprintDates = projectStatusDatesText(board.StartDate, board.EndDate)
	return view
}

// projectStatusDatesText arma "Del 01/10/2026 al 30/11/2026".
func projectStatusDatesText(start, end time.Time) string {
	return fmt.Sprintf(msgProjectStatusDates, projectStatusDateText(start), projectStatusDateText(end))
}
