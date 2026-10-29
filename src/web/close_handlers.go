package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

const closePath = "/cierre"

type closeHandlers struct {
	server  *Server
	service *sprint.CloseSprintService
	catalog BoardCatalog
}

// RegisterClose registra la pantalla y las acciones para cerrar un Sprint.
func RegisterClose(server *Server, service *sprint.CloseSprintService, catalog BoardCatalog) error {
	if server == nil {
		return errors.New("web server is required")
	}
	if service == nil {
		return errors.New("close sprint service is required")
	}
	if catalog == nil {
		return errors.New("board catalog is required")
	}

	handler := &closeHandlers{server: server, service: service, catalog: catalog}
	return server.Register(
		Section{Name: "Cierre de Sprint", Path: closePath, RequiresProject: true},
		Routes{
			"GET " + closePath:  http.HandlerFunc(handler.get),
			"POST " + closePath: http.HandlerFunc(handler.post),
		},
	)
}

func (h *closeHandlers) get(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}

	data, err := buildCloseSprintViewData(r.Context(), h.catalog, active.ProjectID)
	if err != nil {
		h.server.renderer.Fail(w, r, Unavailable(err))
		return
	}
	h.server.renderer.Page(w, r, "close.html", PageData{
		Title: "Cierre de Sprint",
		Form:  NewFormView(),
		Data:  data,
	})
}

func (h *closeHandlers) post(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}

	form, err := parseCloseSprintForm(r)
	if err != nil {
		h.renderError(w, r, http.StatusUnprocessableEntity, MsgSprintNotFound)
		return
	}

	result, err := h.service.CloseSprint(r.Context(), active.ProjectID, form.SprintID)
	if err != nil {
		switch {
		case errors.Is(err, sprint.ErrSprintNotFound):
			h.renderError(w, r, http.StatusUnprocessableEntity, MsgSprintNotFound)
		case errors.Is(err, sprint.ErrSprintAlreadyClosed):
			h.renderError(w, r, http.StatusConflict, MsgSprintAlreadyClosed)
		case errors.Is(err, sprint.ErrCloseBlocked):
			detail := strings.TrimSpace(strings.TrimPrefix(err.Error(), sprint.ErrCloseBlocked.Error()+":"))
			if detail == "" {
				detail = MsgCloseBlocked
			}
			h.renderError(w, r, http.StatusConflict, detail)
		default:
			h.server.renderer.Fail(w, r, err)
		}
		return
	}

	SetFlash(w, r, FlashSuccess, closeSprintSuccessMessage(result.SprintNumber, result.ReturnedCount))
	RedirectSeeOther(w, r, closePath)
}

func (h *closeHandlers) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}
	data, err := buildCloseSprintViewData(r.Context(), h.catalog, active.ProjectID)
	if err != nil {
		h.server.renderer.Fail(w, r, Unavailable(err))
		return
	}
	h.server.renderer.PageStatus(w, r, status, "close.html", PageData{
		Title: "Cierre de Sprint",
		Form:  NewFormView(),
		Data:  data,
		Flash: &Flash{Kind: FlashError, Text: message},
	})
}
