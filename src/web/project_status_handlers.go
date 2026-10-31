package web

import (
	"errors"
	"net/http"
	"time"
)

// pathProjectStatus es la dirección de la sección "Estado del proyecto" (HU-12).
const pathProjectStatus = "/estado"

type projectStatusHandlers struct {
	rd      *Renderer
	catalog ProjectStatusCatalog
	board   BoardCatalog
	now     func() time.Time
}

// RegisterProjectStatus registra la sección "Estado del proyecto" (GET /estado), que exige un
// proyecto activo. now es el reloj (nil = time.Now). Con un argumento nil devuelve un error y
// no registra nada.
func RegisterProjectStatus(s *Server, catalog ProjectStatusCatalog, board BoardCatalog, now func() time.Time) error {
	if s == nil || catalog == nil || board == nil {
		return errors.New("Estado del proyecto necesita el servidor, el catálogo y el Tablero")
	}
	if now == nil {
		now = time.Now
	}
	h := &projectStatusHandlers{rd: s.renderer, catalog: catalog, board: board, now: now}
	return s.Register(
		Section{Name: TitleProjectStatus, Path: pathProjectStatus, RequiresProject: true},
		Routes{"GET " + pathProjectStatus: http.HandlerFunc(h.show)},
	)
}

// show atiende GET /estado. Lee las dos fuentes antes de escribir nada: ante cualquier falla
// muestra la página de error, nunca una pantalla con datos parciales.
func (h *projectStatusHandlers) show(w http.ResponseWriter, r *http.Request) {
	projectID := projectIDFrom(r)
	summary, err := h.catalog.StatusSummary(r.Context(), projectID)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	board, found, err := h.board.ActiveBoard(r.Context(), projectID)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	view := buildProjectStatusView(summary, board, found, h.now())
	h.rd.Page(w, r, "project_status.html", PageData{Title: TitleProjectStatus, Data: view})
}
