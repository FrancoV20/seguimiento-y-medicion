package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// EstimateActions es la operación de HU-17 que usa la pantalla; la satisface *backlog.EstimateService.
type EstimateActions interface {
	UpdateEstimatedHours(ctx context.Context, projectID, storyID int64, input string) (backlog.EstimateResult, error)
}

var _ EstimateActions = (*backlog.EstimateService)(nil)

// RegisterEstimates registra la sección "Estimaciones" (HU-17) con GET y POST /estimaciones. La
// sección exige proyecto activo: HU-20b redirige a la selección si falta.
func RegisterEstimates(s *Server, actions EstimateActions, catalog EstimateCatalog) error {
	if s == nil {
		return errors.New("web server is required")
	}
	if actions == nil {
		return errors.New("estimate service is required")
	}
	if catalog == nil {
		return errors.New("estimate catalog is required")
	}
	h := &estimatesHandlers{server: s, actions: actions, catalog: catalog}
	return s.Register(Section{Name: TitleEstimates, Path: pathEstimates, RequiresProject: true}, Routes{
		"GET " + pathEstimates:  http.HandlerFunc(h.get),
		"POST " + pathEstimates: http.HandlerFunc(h.post),
	})
}

type estimatesHandlers struct {
	server  *Server
	actions EstimateActions
	catalog EstimateCatalog
}

type estimatesPageData struct {
	Rows []estimateRowView
}

// estimateRowView es una fila ya formateada para la plantilla.
type estimateRowView struct {
	ID             int64
	Title          string
	Status         backlog.StoryStatus
	StoryPoints    string
	EstimatedHours string
	ActualHours    string
	InputValue     string // valor del campo de horas: la estimación actual o lo escrito
	Locked         bool
	LockedReason   string
}

func (h *estimatesHandlers) get(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}
	h.render(w, r, http.StatusOK, active.ProjectID, nil)
}

func (h *estimatesHandlers) post(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}
	submission := parseEstimateSubmission(r)
	result, err := h.actions.UpdateEstimatedHours(r.Context(), active.ProjectID, submission.StoryID, submission.Hours)
	if err != nil {
		// El mapeo de cada error a su mensaje y su fila es de US2 (T026).
		h.render(w, r, http.StatusInternalServerError, active.ProjectID, &Flash{Kind: FlashError, Text: MsgOperationFailed})
		return
	}
	SetFlash(w, r, FlashSuccess, estimateSuccessMessage(result))
	RedirectSeeOther(w, r, pathEstimates)
}

// render muestra la lista del proyecto con el estado HTTP y el mensaje indicados.
func (h *estimatesHandlers) render(w http.ResponseWriter, r *http.Request, status int, projectID int64, flash *Flash) {
	rows, err := h.catalog.ListEstimates(r.Context(), projectID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	data := estimatesPageData{Rows: make([]estimateRowView, 0, len(rows))}
	for _, row := range rows {
		data.Rows = append(data.Rows, newEstimateRowView(row))
	}
	page := PageData{Title: TitleEstimates, Data: data, Flash: flash}
	if status == http.StatusOK {
		h.server.renderer.Page(w, r, "estimates.html", page)
		return
	}
	h.server.renderer.PageStatus(w, r, status, "estimates.html", page)
}

func newEstimateRowView(row EstimateRow) estimateRowView {
	view := estimateRowView{
		ID:           row.ID,
		Title:        row.Title,
		Status:       row.Status,
		ActualHours:  formatEstimateHours(row.ActualHours),
		Locked:       row.Locked,
		LockedReason: row.LockedReason,
	}
	if row.StoryPoints != nil {
		view.StoryPoints = strconv.Itoa(*row.StoryPoints)
	}
	if row.EstimatedHours != nil {
		view.EstimatedHours = formatEstimateHours(*row.EstimatedHours)
		view.InputValue = view.EstimatedHours
	}
	return view
}
