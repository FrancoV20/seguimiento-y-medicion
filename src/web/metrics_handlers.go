package web

import (
	"errors"
	"net/http"
)

// pathMetrics es la dirección de la sección "Métricas del Sprint" (HU-25).
const pathMetrics = "/metricas"

type metricsHandlers struct {
	rd      *Renderer
	catalog MetricsCatalog
	source  MetricsSource
}

// RegisterMetrics registra la sección "Métricas del Sprint" (GET /metricas), que exige un proyecto
// activo. Con un argumento nil devuelve un error y no registra nada.
func RegisterMetrics(s *Server, catalog MetricsCatalog, source MetricsSource) error {
	if s == nil || catalog == nil || source == nil {
		return errors.New("Métricas del Sprint necesita el servidor, el catálogo y la fuente de métricas")
	}
	h := &metricsHandlers{rd: s.renderer, catalog: catalog, source: source}
	return s.Register(
		Section{Name: TitleMetrics, Path: pathMetrics, RequiresProject: true},
		Routes{"GET " + pathMetrics: http.HandlerFunc(h.show)},
	)
}

// show atiende GET /metricas: lee los Sprints finalizados del proyecto del contexto, pide a HU-07
// las métricas del más reciente y arma la vista. Lee todo antes de escribir: ante cualquier falla
// muestra la página de error de HU-20a, nunca métricas incompletas.
func (h *metricsHandlers) show(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFrom(r)
	sprints, err := h.catalog.FinishedSprints(r.Context(), projectID)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	in := metricsViewInput{Sprints: sprints}
	if len(sprints) > 0 {
		in.Selected, in.HasSelected = sprints[0], true
		in.Metrics, err = h.source.Metrics(r.Context(), projectID, in.Selected.ID)
		if err != nil {
			h.rd.Fail(w, r, err)
			return
		}
	}
	h.rd.Page(w, r, "metrics.html", PageData{Title: TitleMetrics, Data: buildMetricsView(in)})
}
