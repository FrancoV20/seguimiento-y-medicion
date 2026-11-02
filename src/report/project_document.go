package report

import (
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/pdf"
)

// Columnas del reporte del proyecto (research.md §18); cada tabla suma 495 pt o menos.
var (
	progressColumns = []pdf.Column{{Title: colStatus, Width: 150}, {Title: colStories, Width: 80}}
	sprintColumns   = []pdf.Column{
		{Title: colSprint, Width: 45}, {Title: colStatus, Width: 70}, {Title: colStories, Width: 50},
		{Title: colPoints, Width: 55}, {Title: colVelocity, Width: 50}, {Title: colDeviation, Width: 110},
		{Title: colDefects, Width: 115},
	}
	openDefectColumns = []pdf.Column{
		{Title: colDescription, Width: 200}, {Title: colSeverity, Width: 60}, {Title: colStory, Width: 140},
		{Title: colDetectedIn, Width: 95},
	}
)

// RenderProject convierte el reporte del proyecto en PDF, en el orden de la spec: encabezado, avance,
// Sprints, esfuerzo registrado y defectos, con "Página X de Y" en cada página.
func RenderProject(r ProjectReport) ([]byte, error) {
	d := pdf.New()
	data := r.Data

	// 1. Encabezado.
	d.Text(pdf.Bold, sizeTitle, titleProjectReport)
	if r.Partial {
		d.Text(pdf.Bold, sizeHeading, msgProjectPartial)
	}
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtProject, data.Name))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtDates, dateText(data.StartDate), dateText(data.EndDate)))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtStatus, r.Status))
	d.Text(pdf.Regular, sizeText, membersText(data.Members))
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtGenerated, dateTimeText(r.GeneratedAt)))

	// 2. Avance.
	heading(d, headingProgress)
	c := data.Stories
	if err := d.Table(progressColumns, [][]string{
		{statusPending, fmt.Sprint(c.Pending)},
		{statusInSprint, fmt.Sprint(c.InSprint)},
		{statusProgress, fmt.Sprint(c.InProgress)},
		{storyDone, fmt.Sprint(c.Done)},
	}); err != nil {
		return nil, err
	}
	d.Space(4)
	d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtProgress, c.Done, c.Total(), percentText(r.Progress)))

	// 3. Sprints.
	heading(d, headingSprints)
	if len(r.Sprints) == 0 {
		d.Text(pdf.Regular, sizeText, msgNoSprints)
	} else {
		rows := make([][]string, 0, len(r.Sprints))
		for _, s := range r.Sprints {
			rows = append(rows, sprintRow(s))
		}
		if err := d.Table(sprintColumns, rows); err != nil {
			return nil, err
		}
		d.Space(4)
		d.Text(pdf.Regular, sizeText, fmt.Sprintf(fmtTotalVelocity, r.Velocity.Total))
		d.Text(pdf.Regular, sizeText, averageVelocityText(r.Velocity.Average))
	}

	// 4. Esfuerzo registrado.
	heading(d, headingEffort)
	d.Text(pdf.Regular, sizeText, estimatedTotalText(data.EstimatedTotal, data.WithoutEstimate))
	d.Text(pdf.Regular, sizeText, registeredTotalText(data.RegisteredTotal))
	for _, m := range data.MemberHours {
		d.Text(pdf.Regular, sizeText, memberHoursText(m))
	}

	// 5. Defectos.
	heading(d, headingDefects)
	d.Text(pdf.Regular, sizeText, projectDefectsText(data.DefectsOpen, data.DefectsResolved))
	if len(data.OpenDefects) > 0 {
		d.Space(4)
		rows := make([][]string, 0, len(data.OpenDefects))
		for _, df := range data.OpenDefects {
			rows = append(rows, []string{df.Description, df.Severity, df.StoryTitle, fmt.Sprintf(fmtDetectedIn, df.DetectionSprint)})
		}
		if err := d.Table(openDefectColumns, rows); err != nil {
			return nil, err
		}
	} else if data.DefectsOpen+data.DefectsResolved > 0 {
		d.Text(pdf.Regular, sizeText, msgNoOpenDefects)
	}

	// 6. Pie.
	return d.Bytes(footerText)
}

// sprintRow arma la fila de un Sprint: con HU-10 y HU-07 si está finalizado, o "En curso" si no.
func sprintRow(s ProjectSprintRow) []string {
	label := fmt.Sprintf(fmtSprint, s.Number)
	if s.Active {
		return []string{label, msgActiveRow, msgNoResolution, msgNoResolution, msgNoResolution, msgNoResolution, msgNoResolution}
	}
	q, m := s.Quality, s.Metrics
	deviation := deviationText(m.DeviationPercentage)
	if m.IsPartial {
		deviation += msgPartialSuffix
	}
	return []string{
		label, statusFinished,
		fmt.Sprintf(fmtRatio, q.StoriesCompleted, q.StoriesPlanned),
		fmt.Sprintf(fmtRatio, q.PointsCompleted, q.PointsPlanned),
		fmt.Sprint(m.Velocity),
		deviation,
		sprintDefectsCell(q.DefectsDetected, q.DefectsResolved),
	}
}
