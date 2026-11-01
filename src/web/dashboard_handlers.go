package web

import (
	"context"
	"errors"
	"net/http"
)

// pathDashboard es la dirección de la sección "Dashboard" (HU-08).
const pathDashboard = "/dashboard"

type dashboardHandlers struct {
	rd        *Renderer
	catalog   DashboardCatalog
	board     BoardCatalog
	quality   QualityReader
	deviation MetricsSource
}

// RegisterDashboard registra la sección "Dashboard" (GET /dashboard), de solo lectura y con
// proyecto activo. Reutiliza el Tablero de HU-14, la calidad de HU-10 y la fuente de métricas de
// solo lectura de HU-25. Con un argumento nil devuelve un error y no registra nada.
func RegisterDashboard(s *Server, catalog DashboardCatalog, board BoardCatalog, quality QualityReader, deviation MetricsSource) error {
	if s == nil || catalog == nil || board == nil || quality == nil || deviation == nil {
		return errors.New("el Dashboard necesita el servidor, el catálogo, el Tablero, la calidad y la fuente de métricas")
	}
	h := &dashboardHandlers{rd: s.renderer, catalog: catalog, board: board, quality: quality, deviation: deviation}
	return s.Register(
		Section{Name: TitleDashboard, Path: pathDashboard, RequiresProject: true},
		Routes{"GET " + pathDashboard: http.HandlerFunc(h.show)},
	)
}

// show atiende GET /dashboard: lee todos los datos y recién después escribe la respuesta. Ante
// cualquier falla muestra la página de error de HU-20a, nunca números parciales.
func (h *dashboardHandlers) show(w http.ResponseWriter, r *http.Request) {
	view, err := h.load(r.Context(), projectIDFrom(r))
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	h.rd.Page(w, r, "dashboard.html", PageData{Title: TitleDashboard, Data: view})
}

// load lee los datos del Dashboard en este orden: catálogo, Tablero, velocidad de HU-10 por Sprint
// finalizado y desviación de HU-07 del último. Sin Sprints finalizados no le pide nada a HU-10 ni a
// HU-07.
func (h *dashboardHandlers) load(ctx context.Context, projectID int64) (dashboardView, error) {
	summary, err := h.catalog.DashboardSummary(ctx, projectID)
	if err != nil {
		return dashboardView{}, err
	}
	board, found, err := h.board.ActiveBoard(ctx, projectID)
	if err != nil {
		return dashboardView{}, err
	}

	velocities := make([]int, 0, len(summary.Finished))
	for _, finished := range summary.Finished {
		quality, err := h.quality.SprintQuality(ctx, projectID, finished.ID)
		if err != nil {
			return dashboardView{}, err
		}
		velocities = append(velocities, quality.PointsCompleted)
	}

	var deviation SprintMetrics
	if n := len(summary.Finished); n > 0 {
		deviation, err = h.deviation.Metrics(ctx, projectID, summary.Finished[n-1].ID)
		if err != nil {
			return dashboardView{}, err
		}
	}
	return buildDashboardView(summary, board, found, velocities, deviation), nil
}
