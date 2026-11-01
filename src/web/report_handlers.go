package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/report"
)

type reportHandlers struct {
	rd        *Renderer
	catalog   ReportCatalog
	generator ReportGenerator
}

// RegisterReports registra la sección "Reportes" (HU-09), que exige un proyecto activo y responde
// solo a GET. Con un argumento nil devuelve un error y no registra nada.
func RegisterReports(s *Server, catalog ReportCatalog, generator ReportGenerator) error {
	if s == nil || catalog == nil || generator == nil {
		return errors.New("Reportes necesita el servidor, el catálogo y el generador de reportes")
	}
	h := &reportHandlers{rd: s.renderer, catalog: catalog, generator: generator}
	return s.Register(
		Section{Name: TitleReports, Path: pathReports, RequiresProject: true},
		Routes{
			"GET " + pathReports:                       http.HandlerFunc(h.list),
			"GET " + pathReports + "/sprints/{id}/pdf": http.HandlerFunc(h.sprintPDF),
			"GET " + reportProjectPath:                 http.HandlerFunc(h.projectPDF),
		},
	)
}

// list atiende GET /reportes: los Sprints del proyecto activo con su enlace de descarga.
func (h *reportHandlers) list(w http.ResponseWriter, r *http.Request) {
	sprints, err := h.catalog.ReportSprints(r.Context(), projectIDFrom(r))
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	h.rd.Page(w, r, "reports.html", PageData{Title: TitleReports, Data: buildReportListData(sprints)})
}

// sprintPDF atiende GET /reportes/sprints/{id}/pdf. Un id inválido es 404 sin consultar la base; un
// Sprint inexistente o de otro proyecto, 404; un Sprint en curso sin confirmar muestra la
// advertencia. No escribe nada en la respuesta hasta tener el PDF completo.
func (h *reportHandlers) sprintPDF(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id == 0 {
		h.rd.Fail(w, r, NotFound(errors.New("identificador de Sprint inválido")))
		return
	}
	projectID := projectIDFrom(r)
	sprint, found, err := h.catalog.ReportSprint(r.Context(), projectID, id)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	if !found {
		h.rd.Fail(w, r, NotFound(errors.New("el Sprint no existe en el proyecto activo")))
		return
	}
	if sprint.Active && !reportConfirmed(r) {
		h.rd.Page(w, r, "report_warning.html", PageData{Title: TitleReports, Data: sprintWarningData(sprint)})
		return
	}
	b, err := h.generator.SprintPDF(r.Context(), projectID, id)
	if errors.Is(err, report.ErrSprintNotFound) {
		h.rd.Fail(w, r, NotFound(err))
		return
	}
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	writePDF(w, reportFileName(sprint.Number), b)
}

// projectPDF atiende GET /reportes/proyecto/pdf con el proyecto del contexto. Si hay un Sprint en
// curso y no viene confirmar=si, muestra la advertencia. No escribe nada hasta tener el PDF completo.
func (h *reportHandlers) projectPDF(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFrom(r)
	sprints, err := h.catalog.ReportSprints(r.Context(), projectID)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	for _, s := range sprints {
		if s.Active && !reportConfirmed(r) {
			h.rd.Page(w, r, "report_warning.html", PageData{Title: TitleReports, Data: projectWarningData(s.Number)})
			return
		}
	}
	b, err := h.generator.ProjectPDF(r.Context(), projectID)
	if errors.Is(err, report.ErrProjectNotFound) {
		h.rd.Fail(w, r, NotFound(err))
		return
	}
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	writePDF(w, reportProjectFileName(projectID), b)
}

// writePDF envía el PDF ya armado como descarga.
func writePDF(w http.ResponseWriter, fileName string, b []byte) {
	header := w.Header()
	header.Set("Content-Type", "application/pdf")
	header.Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	header.Set("Cache-Control", "no-store")
	header.Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
