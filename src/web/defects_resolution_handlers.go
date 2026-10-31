package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

// resolve atiende POST /defectos/resolver: resuelve el defecto del campo "defecto" en el Sprint
// activo del proyecto activo. El proyecto sale siempre del contexto, nunca del formulario.
func (h *defectsHandlers) resolve(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.server.renderer.Fail(w, r, fmt.Errorf("leer formulario de resolución: %w", err))
		return
	}
	ac := ActiveContextFrom(r.Context())
	if ac == nil || ac.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("falta el contexto del proyecto activo"))
		return
	}
	defectID := parseID(r.PostForm.Get(DefectIDField))
	resolved, err := h.resolution.Resolve(r.Context(), ac.ProjectID, defectID)
	if err != nil {
		h.renderResolutionError(w, r, ac.ProjectID, err)
		return
	}
	SetFlash(w, r, FlashSuccess, fmt.Sprintf(MsgDefectResolved, resolved.SprintNumber))
	RedirectSeeOther(w, r, "/defectos")
}

// resolutionErrorResponse traduce un error de dominio de HU-19 a su respuesta y mensaje.
func resolutionErrorResponse(err error) (int, string, bool) {
	switch {
	case errors.Is(err, defect.ErrNoActiveSprint):
		return http.StatusConflict, MsgDefectResolveNoSprint, true
	case errors.Is(err, defect.ErrResolutionBeforeDetection):
		return http.StatusConflict, MsgDefectResolutionBeforeDetection, true
	case errors.Is(err, defect.ErrAlreadyResolved):
		return http.StatusConflict, MsgDefectAlreadyResolved, true
	default:
		return 0, "", false
	}
}

// renderResolutionError vuelve a mostrar la pantalla de defectos con el mensaje del error.
func (h *defectsHandlers) renderResolutionError(w http.ResponseWriter, r *http.Request, projectID int64, err error) {
	status, message, ok := resolutionErrorResponse(err)
	if !ok {
		h.server.renderer.Fail(w, r, err)
		return
	}
	data, pageErr := h.pageData(r, projectID)
	if pageErr != nil {
		h.server.renderer.Fail(w, r, pageErr)
		return
	}
	h.renderPage(w, r, status, data, NewFormView(), &Flash{Kind: FlashError, Text: message})
}
