package web

import (
	"errors"
	"net/http"
)

// RegisterQuality registra la sección "Calidad y avance" (HU-10, T028) con GET /calidad. Es de solo
// lectura: cualquier otro método recibe el 405 de HU-20a. La sección exige proyecto activo: HU-20b
// redirige a la selección si falta.
func RegisterQuality(s *Server, reader QualityReader, catalog QualityCatalog) error {
	if s == nil {
		return errors.New("web server is required")
	}
	if reader == nil {
		return errors.New("quality reader is required")
	}
	if catalog == nil {
		return errors.New("quality catalog is required")
	}
	h := &qualityHandlers{server: s, reader: reader, catalog: catalog}
	return s.Register(Section{Name: TitleQuality, Path: pathQuality, RequiresProject: true}, Routes{
		"GET " + pathQuality: http.HandlerFunc(h.get),
	})
}

type qualityHandlers struct {
	server  *Server
	reader  QualityReader
	catalog QualityCatalog
}

// get muestra la lista de Sprints finalizados y los números del elegido. Si falla la lista o el
// cálculo, se ve la página de error de HU-20a y ningún número parcial.
func (h *qualityHandlers) get(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}
	list, err := h.catalog.QualitySprints(r.Context(), active.ProjectID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}

	chosen, ok := selectQualitySprint(r.URL.Query().Get("sprint"), list)
	if !ok {
		h.page(w, r, newQualityEmptyPageData(list))
		return
	}
	quality, err := h.reader.SprintQuality(r.Context(), active.ProjectID, chosen.ID)
	if err != nil {
		// Incluye un Sprint que dejó de estar finalizado o de existir entre la lista y el cálculo.
		h.server.renderer.Fail(w, r, err)
		return
	}
	h.page(w, r, newQualityPageData(list, chosen, quality))
}

func (h *qualityHandlers) page(w http.ResponseWriter, r *http.Request, data qualityPageData) {
	h.server.renderer.Page(w, r, "quality.html", PageData{Title: TitleQuality, Data: data})
}
