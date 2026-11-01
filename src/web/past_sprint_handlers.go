package web

import (
	"errors"
	"net/http"
)

type pastSprintHandlers struct {
	rd      *Renderer
	catalog PastSprintCatalog
	calc    PastSprintMetrics
}

// RegisterPastSprints registra la sección "Sprints anteriores" (HU-16), que exige un proyecto
// activo y responde solo a GET. Con un argumento nil devuelve un error y no registra nada.
func RegisterPastSprints(s *Server, catalog PastSprintCatalog, calc PastSprintMetrics) error {
	if s == nil || catalog == nil || calc == nil {
		return errors.New("Sprints anteriores necesita el servidor, el catálogo y las métricas de HU-07")
	}
	h := &pastSprintHandlers{rd: s.renderer, catalog: catalog, calc: calc}
	return s.Register(
		Section{Name: TitlePastSprints, Path: pathPastSprints, RequiresProject: true},
		Routes{
			"GET " + pathPastSprints: http.HandlerFunc(h.list),
		},
	)
}

// list atiende GET /sprints-anteriores: los Sprints finalizados del proyecto activo. Lee antes de
// escribir: ante una falla muestra la página de error de HU-20a.
func (h *pastSprintHandlers) list(w http.ResponseWriter, r *http.Request) {
	sprints, err := h.catalog.FinishedSprints(r.Context(), projectIDFrom(r))
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	h.rd.Page(w, r, "past_sprints.html", PageData{Title: TitlePastSprints, Data: buildPastSprintListView(sprints)})
}
