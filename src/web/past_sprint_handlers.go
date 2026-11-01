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
			"GET " + pathPastSprints:           http.HandlerFunc(h.list),
			"GET " + pathPastSprints + "/{id}": http.HandlerFunc(h.detail),
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

// detail atiende GET /sprints-anteriores/{id}: el Sprint finalizado del proyecto activo con su
// registro de cierre y la velocidad de HU-07. Un id inválido es 404 sin consultar la base; un Sprint
// inexistente, de otro proyecto o en curso es 404 antes de calcular. No escribe nada hasta tener
// todos los datos, así una falla nunca deja datos parciales.
func (h *pastSprintHandlers) detail(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id == 0 {
		h.rd.Fail(w, r, NotFound(errors.New("identificador de Sprint inválido")))
		return
	}
	detail, err := h.catalog.SprintDetail(r.Context(), projectIDFrom(r), id)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	m, err := h.calc.CalculateMetrics(id)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	view := buildPastSprintDetailView(detail, m.Velocity)
	h.rd.Page(w, r, "past_sprint.html", PageData{Title: TitlePastSprints, Data: view})
}
