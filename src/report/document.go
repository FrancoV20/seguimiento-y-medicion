package report

import (
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/pdf"
)

// Medidas del documento (research.md §6).
const (
	sizeTitle   = 16.0
	sizeHeading = 12.0
	sizeText    = 10.0
	spaceBlock  = 10.0
)

var storyColumns = []pdf.Column{
	{Title: colStory, Width: 165}, {Title: colStatus, Width: 135}, {Title: colPoints, Width: 55},
	{Title: colEstimated, Width: 70}, {Title: colActual, Width: 70},
}

var defectColumns = []pdf.Column{
	{Title: colDescription, Width: 170}, {Title: colSeverity, Width: 60}, {Title: colStatus, Width: 60},
	{Title: colStory, Width: 120}, {Title: colResolution, Width: 85},
}

// Render convierte el reporte de un Sprint en PDF, en el orden de la spec: encabezado, historias,
// estimaciones, esfuerzo registrado, métricas y defectos, con "Página X de Y" en cada página.
func Render(r SprintReport) ([]byte, error) {
	d := pdf.New()
	data := r.Data
	finished := !r.Partial

	// 1. Encabezado.
	d.Text(pdf.Bold, sizeTitle, titleReport)
	if r.Partial {
		d.Text(pdf.Bold, sizeHeading, msgPartial)
	}
	status := statusFinished
	if r.Partial {
		status = statusActive
	}
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtProject, data.Sprint.ProjectName))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtSprint, data.Sprint.Number))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtGoal, data.Sprint.Goal))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtDates, dateText(data.Sprint.StartDate), dateText(data.Sprint.EndDate)))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtStatus, status))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtGenerated, dateTimeText(r.GeneratedAt)))

	// 2. Historias.
	heading(d, headingStories)
	if len(data.Stories) == 0 {
		d.Text(pdf.Regular, sizeText, msgNoStories)
	} else {
		rows := make([][]string, 0, len(data.Stories))
		for _, s := range data.Stories {
			rows = append(rows, []string{s.Title, storyStatusText(s.Status, finished), optionalPoints(s.StoryPoints),
				optionalHours(s.EstimatedHours), hoursText(s.ActualHours)})
		}
		if err := d.Table(storyColumns, rows); err != nil {
			return nil, err
		}
		d.Space(4)
	}
	if finished {
		q := r.Quality
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtProgress, q.StoriesCompleted, q.StoriesPlanned, percentText(q.CompletionPercentage)))
	} else {
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtPendingProgress, msgPendingClose))
	}

	// 3. Estimaciones.
	heading(d, headingEstimates)
	if finished {
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtPoints, r.Quality.PointsPlanned, r.Quality.PointsCompleted))
	} else {
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtPendingPoints, msgPendingClose))
	}
	var estimated effort.Hours
	without := 0
	for _, s := range data.Stories {
		if s.EstimatedHours == nil {
			without++
			continue
		}
		estimated = estimated.Add(*s.EstimatedHours)
	}
	d.Text(pdf.Regular, sizeText, estimatedTotalText(estimated, without))

	// 4. Esfuerzo registrado.
	writeEffort(d, data.Members)

	// 5. Métricas.
	heading(d, headingMetrics)
	if finished {
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtVelocity, r.Metrics.Velocity))
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtDeviation, deviationText(r.Metrics.DeviationPercentage)))
		if r.Metrics.IsPartial && r.Metrics.Warning != "" {
			d.Text(pdf.Regular, sizeText, r.Metrics.Warning)
		}
	} else {
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtPendingMetrics, msgPendingClose))
	}

	// 6. Defectos.
	heading(d, headingDefects)
	if len(data.Defects) > 0 {
		rows := make([][]string, 0, len(data.Defects))
		for _, df := range data.Defects {
			resolution := msgNoResolution
			if df.ResolutionSprint > 0 {
				resolution = fmt.Sprintf(fmtResolution, df.ResolutionSprint)
			}
			rows = append(rows, []string{df.Description, df.Severity, df.Status, df.StoryTitle, resolution})
		}
		if err := d.Table(defectColumns, rows); err != nil {
			return nil, err
		}
		d.Space(4)
	}
	if finished {
		d.Text(pdf.Regular, sizeText, defectsSummaryText(r.Quality.DefectsDetected, r.Quality.DefectsResolved))
	} else {
		d.Text(pdf.Regular, sizeText, defectsSoFarText(len(data.Defects)))
	}

	// 7. Pie.
	return d.Bytes(footerText)
}

// heading deja un espacio y escribe el subtítulo de una parte.
func heading(d *pdf.Document, title string) {
	d.Space(spaceBlock)
	d.Text(pdf.Bold, sizeHeading, title)
	d.Space(2)
}

// writeEffort escribe el total registrado y las horas de cada integrante.
func writeEffort(d *pdf.Document, members []MemberHours) {
	heading(d, headingEffort)
	var total effort.Hours
	for _, m := range members {
		total = total.Add(m.Hours)
	}
	d.Text(pdf.Regular, sizeText, registeredTotalText(total))
	for _, m := range members {
		d.Text(pdf.Regular, sizeText, memberHoursText(m))
	}
}
