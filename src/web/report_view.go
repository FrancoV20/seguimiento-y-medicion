package web

import (
	"fmt"
	"net/http"
	"strconv"
)

// HU-09: textos y armado de las vistas de "Reportes". Todos los textos de la pantalla viven acá; los
// del PDF, en src/report/texts.go.
const (
	TitleReports             = "Reportes"
	MsgReportNoSprints       = "Aún no hay Sprints para exportar"
	LinkReportExport         = "Exportar PDF"
	LinkReportGenerateAnyway = "Generar igual"
	LinkReportCancel         = "Cancelar"
	LinkReportProject        = "Exportar reporte del proyecto"
	ReportStatusActive       = "En curso"
	ReportStatusFinished     = "Finalizado"

	msgReportActiveWarning = "El Sprint %d sigue en curso: el reporte puede estar incompleto"
	msgReportSprint        = "Sprint %d"

	pathReports       = "/reportes"
	reportProjectPath = "/reportes/proyecto/pdf"
	reportConfirmKey  = "confirmar"
	reportConfirmYes  = "si"
)

// reportPDFPath es la descarga del PDF de un Sprint: "/reportes/sprints/7/pdf".
func reportPDFPath(id int64) string {
	return pathReports + "/sprints/" + strconv.FormatInt(id, 10) + "/pdf"
}

// reportConfirmPath es la descarga confirmada de un Sprint en curso.
func reportConfirmPath(id int64) string {
	return reportPDFPath(id) + "?" + reportConfirmKey + "=" + reportConfirmYes
}

// reportFileName es el nombre del PDF de un Sprint: "reporte-sprint-2.pdf".
func reportFileName(number int) string { return fmt.Sprintf("reporte-sprint-%d.pdf", number) }

// reportProjectFileName es el nombre del PDF del proyecto: "reporte-proyecto-5.pdf".
func reportProjectFileName(id int64) string { return fmt.Sprintf("reporte-proyecto-%d.pdf", id) }

// reportConfirmed indica si la descarga viene con confirmar=si (exactamente).
func reportConfirmed(r *http.Request) bool {
	return r.URL.Query().Get(reportConfirmKey) == reportConfirmYes
}

// reportSprintRow es un Sprint de la lista.
type reportSprintRow struct {
	Label     string // "Sprint 3"
	Goal      string
	Status    string // "En curso" o "Finalizado"
	ExportURL string
}

// reportListData es lo que muestra reports.html.
type reportListData struct {
	Sprints    []reportSprintRow
	NoSprints  string // MsgReportNoSprints si la lista está vacía
	ProjectURL string // el botón del reporte del proyecto está siempre
}

// buildReportListData arma la lista en el orden recibido (el catálogo ya ordena).
func buildReportListData(sprints []ReportSprintOption) reportListData {
	data := reportListData{Sprints: make([]reportSprintRow, 0, len(sprints)), ProjectURL: reportProjectPath}
	for _, s := range sprints {
		status := ReportStatusFinished
		if s.Active {
			status = ReportStatusActive
		}
		data.Sprints = append(data.Sprints, reportSprintRow{
			Label: fmt.Sprintf(msgReportSprint, s.Number), Goal: s.Goal, Status: status, ExportURL: reportPDFPath(s.ID),
		})
	}
	if len(sprints) == 0 {
		data.NoSprints = MsgReportNoSprints
	}
	return data
}

// reportWarningData es lo que muestra report_warning.html.
type reportWarningData struct {
	Message     string
	GenerateURL string
	CancelURL   string
}

// sprintWarningData arma la advertencia de un Sprint en curso.
func sprintWarningData(s ReportSprintOption) reportWarningData {
	return reportWarningData{
		Message:     fmt.Sprintf(msgReportActiveWarning, s.Number),
		GenerateURL: reportConfirmPath(s.ID),
		CancelURL:   pathReports,
	}
}
