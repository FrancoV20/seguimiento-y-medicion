package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

// resolve atiende POST /defectos/resolver: resuelve el defecto del campo "defecto" en el Sprint
// activo del proyecto activo. El proyecto sale siempre del contexto, nunca del formulario.
func (h *defectsHandlers) resolve(w http.ResponseWriter, r *http.Request) {
	h.handleResolutionAction(w, r, func(ctx context.Context, projectID, defectID int64) (string, error) {
		resolved, err := h.resolution.Resolve(ctx, projectID, defectID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(MsgDefectResolved, resolved.SprintNumber), nil
	})
}

// reopen atiende POST /defectos/reabrir: vuelve a abrir el defecto del campo "defecto" del
// proyecto activo. No necesita Sprint activo.
func (h *defectsHandlers) reopen(w http.ResponseWriter, r *http.Request) {
	h.handleResolutionAction(w, r, func(ctx context.Context, projectID, defectID int64) (string, error) {
		if err := h.resolution.Reopen(ctx, projectID, defectID); err != nil {
			return "", err
		}
		return MsgDefectReopened, nil
	})
}

// resolutionAction ejecuta una acción de HU-19 y devuelve el mensaje de éxito.
type resolutionAction func(ctx context.Context, projectID, defectID int64) (string, error)

// handleResolutionAction tiene el recorrido común de resolver y reabrir: lee el formulario, toma
// el proyecto del contexto activo y el defecto del campo "defecto", ejecuta la acción y, si sale
// bien, deja el mensaje de una sola vista y redirige con 303 a /defectos; si falla, vuelve a
// mostrar la pantalla con el error.
func (h *defectsHandlers) handleResolutionAction(w http.ResponseWriter, r *http.Request, action resolutionAction) {
	if err := r.ParseForm(); err != nil {
		h.server.renderer.Fail(w, r, fmt.Errorf("leer formulario de %s: %w", r.URL.Path, err))
		return
	}
	ac := ActiveContextFrom(r.Context())
	if ac == nil || ac.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("falta el contexto del proyecto activo"))
		return
	}
	defectID := parseID(r.PostForm.Get(DefectIDField))
	message, err := action(r.Context(), ac.ProjectID, defectID)
	if err != nil {
		h.renderResolutionError(w, r, ac.ProjectID, err)
		return
	}
	SetFlash(w, r, FlashSuccess, message)
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
