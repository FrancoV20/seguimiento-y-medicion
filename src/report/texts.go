package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// Textos del PDF de un Sprint, en un único lugar (HU-09, FR-021, data-model.md).
const (
	titleReport = "Reporte del Sprint"
	msgPartial  = "Reporte parcial: Sprint en curso"

	fmtProject   = "Proyecto: %s"
	fmtSprint    = "Sprint %d"
	fmtGoal      = "Objetivo: %s"
	fmtDates     = "Fechas: %s al %s"
	fmtStatus    = "Estado: %s"
	fmtGenerated = "Generado: %s"

	statusFinished = "Finalizado"
	statusActive   = "En curso"

	headingStories   = "Historias"
	headingEstimates = "Estimaciones"
	headingEffort    = "Esfuerzo registrado"
	headingMetrics   = "Métricas"
	headingDefects   = "Defectos"

	colStory       = "Historia"
	colStatus      = "Estado"
	colPoints      = "Story Points"
	colEstimated   = "Horas estimadas"
	colActual      = "Horas reales"
	colDescription = "Descripción"
	colSeverity    = "Severidad"
	colResolution  = "Resuelto en"

	msgNotEstimated = "Sin estimar"
	msgReturned     = "%s – volvió al Backlog"
	msgNoStories    = "No hay historias registradas en este Sprint"

	fmtProgress        = "%d de %d historias completadas (%s)"
	fmtPoints          = "%d Story Points planificados y %d completados"
	fmtMemberHours     = "%s: %s h"
	fmtVelocity        = "Velocidad: %d Story Points"
	fmtDeviation       = "Desviación de esfuerzo: %s"
	msgPendingClose    = "Se calculan al cerrar el Sprint"
	fmtPendingProgress = "Historias completadas: %s"
	fmtPendingPoints   = "Story Points planificados y completados: %s"
	fmtPendingMetrics  = "Velocidad y desviación: %s"
	msgNoSprintDefects = "No se registraron defectos en este Sprint"
	msgNoResolution    = "—"
	fmtResolution      = "Sprint %d"
	fmtFooter          = "Página %d de %d"

	notAvailable = "No disponible"
)

// percentText convierte el porcentaje de HU-10 o HU-12 ("75.00") en "75,00 %".
func percentText(p string) string {
	if p == notAvailable {
		return p
	}
	return strings.Replace(p, ".", ",", 1) + " %"
}

// deviationText convierte la desviación de HU-07 ("6.25") en "+6,25 %"; cero va sin signo.
func deviationText(d string) string {
	if d == notAvailable || d == "" {
		return notAvailable
	}
	text := percentText(d)
	if strings.HasPrefix(d, "-") || strings.Trim(d, "0.") == "" {
		return text
	}
	return "+" + text
}

// decimalText convierte un número con punto decimal ("6.50") en "6,50".
func decimalText(v string) string {
	if v == notAvailable {
		return v
	}
	return strings.Replace(v, ".", ",", 1)
}

// hoursText escribe las horas sin ceros sobrantes y con coma decimal ("17", "2,5").
func hoursText(h effort.Hours) string {
	return strings.Replace(h.String(), ".", ",", 1)
}

func dateText(t time.Time) string     { return t.Format("02/01/2006") }
func dateTimeText(t time.Time) string { return t.Format("02/01/2006 15:04") }

// storyStatusText agrega "volvió al Backlog" a las historias no terminadas de un Sprint finalizado.
func storyStatusText(status string, finished bool) string {
	if finished && status != storyDone {
		return fmt.Sprintf(msgReturned, status)
	}
	return status
}

func optionalPoints(p *int) string {
	if p == nil {
		return msgNotEstimated
	}
	return fmt.Sprint(*p)
}

func optionalHours(h *effort.Hours) string {
	if h == nil {
		return msgNotEstimated
	}
	return hoursText(*h)
}

// plural elige la forma singular o plural según n.
func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return fmt.Sprintf(pluralForm, n)
}

// withoutHoursText es el aviso de historias sin horas estimadas; vacío si no hay.
func withoutHoursText(n int) string {
	if n == 0 {
		return ""
	}
	return plural(n, "1 historia sin horas estimadas", "%d historias sin horas estimadas")
}

// estimatedTotalText es "28 horas estimadas en total (1 historia sin horas estimadas)".
func estimatedTotalText(total effort.Hours, without int) string {
	text := hoursText(total) + " horas estimadas en total"
	if total == 100 {
		text = "1 hora estimada en total"
	}
	if w := withoutHoursText(without); w != "" {
		text += " (" + w + ")"
	}
	return text
}

// registeredTotalText es "24 horas registradas en total".
func registeredTotalText(total effort.Hours) string {
	if total == 100 {
		return "1 hora registrada en total"
	}
	return hoursText(total) + " horas registradas en total"
}

func memberHoursText(m MemberHours) string {
	return fmt.Sprintf(fmtMemberHours, m.Name, hoursText(m.Hours))
}

func detectedText(n int) string { return plural(n, "1 defecto detectado", "%d defectos detectados") }
func resolvedText(n int) string { return plural(n, "1 resuelto", "%d resueltos") }

// defectsSummaryText es el resumen de defectos de un Sprint finalizado (HU-10).
func defectsSummaryText(detected, resolved int) string {
	if detected == 0 && resolved == 0 {
		return msgNoSprintDefects
	}
	return detectedText(detected) + " y " + resolvedText(resolved) + " en este Sprint"
}

// defectsSoFarText es el resumen de defectos de un Sprint en curso.
func defectsSoFarText(detected int) string {
	if detected == 0 {
		return msgNoSprintDefects
	}
	return detectedText(detected) + " hasta el momento"
}

func footerText(page, total int) string { return fmt.Sprintf(fmtFooter, page, total) }
