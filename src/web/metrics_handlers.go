package web

import (
	"errors"
	"net/http"
)

// errMetricsNotImplemented marca el esqueleto del handler de HU-25 antes del corte GREEN.
var errMetricsNotImplemented = errors.New("métricas del Sprint: no implementado")

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

// show atiende GET /metricas.
func (h *metricsHandlers) show(w http.ResponseWriter, r *http.Request) {
	h.rd.Fail(w, r, errMetricsNotImplemented)
}
