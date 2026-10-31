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

// reopen atiende POST /defectos/reabrir: vuelve a abrir el defecto del campo "defecto" del
// proyecto activo. No necesita Sprint activo.
func (h *defectsHandlers) reopen(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.server.renderer.Fail(w, r, fmt.Errorf("leer formulario de reapertura: %w", err))
		return
	}
	ac := ActiveContextFrom(r.Context())
	if ac == nil || ac.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("falta el contexto del proyecto activo"))
		return
	}
	defectID := parseID(r.PostForm.Get(DefectIDField))
	if err := h.resolution.Reopen(r.Context(), ac.ProjectID, defectID); err != nil {
		h.renderResolutionError(w, r, ac.ProjectID, err)
		return
	}
	SetFlash(w, r, FlashSuccess, MsgDefectReopened)
	RedirectSeeOther(w, r, "/defectos")
}

// resolutionErrorResponse traduce un error de HU-19 a su respuesta y mensaje: 409 para las reglas y
// los estados ya alcanzados, 404 para un defecto inexistente (o de otro proyecto) y 500 con el
// mensaje genérico para cualquier otro error.
func resolutionErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, defect.ErrNoActiveSprint):
		return http.StatusConflict, MsgDefectResolveNoSprint
	case errors.Is(err, defect.ErrResolutionBeforeDetection):
		return http.StatusConflict, MsgDefectResolutionBeforeDetection
	case errors.Is(err, defect.ErrAlreadyResolved):
		return http.StatusConflict, MsgDefectAlreadyResolved
	case errors.Is(err, defect.ErrAlreadyOpen):
		return http.StatusConflict, MsgDefectAlreadyOpen
	case errors.Is(err, defect.ErrDefectNotFound):
		return http.StatusNotFound, MsgDefectNotFound
	default:
		return http.StatusInternalServerError, MsgOperationFailed
	}
}

// renderResolutionError vuelve a mostrar la pantalla de defectos con el mensaje del error. El
// detalle técnico de una falla de guardado solo va al registro del servidor.
func (h *defectsHandlers) renderResolutionError(w http.ResponseWriter, r *http.Request, projectID int64, err error) {
	status, message := resolutionErrorResponse(err)
	if status == http.StatusInternalServerError {
		h.server.renderer.logger.Error("no se pudo resolver o reabrir el defecto",
			"metodo", r.Method, "ruta", r.URL.Path, "detalle", err.Error())
	}
	data, pageErr := h.pageData(r, projectID)
	if pageErr != nil {
		h.server.renderer.Fail(w, r, pageErr)
		return
	}
	h.renderPage(w, r, status, data, NewFormView(), &Flash{Kind: FlashError, Text: message})
}
