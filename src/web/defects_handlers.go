package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

type defectsPageData struct {
	Defects          []DefectRow
	Stories          []DefectStoryOption
	Severities       []defect.Severity
	ActiveSprint     int64
	HasActiveSprint  bool
	NoDefectsMessage string
	// CanResolve habilita las acciones de HU-19 en la lista.
	CanResolve bool
}

type defectsHandlers struct {
	server     *Server
	service    *defect.Service
	resolution *defect.ResolutionService
	catalog    DefectCatalog
}

// RegisterDefects registra la pantalla de defectos sin las acciones de resolución (HU-06).
func RegisterDefects(s *Server, service *defect.Service, catalog DefectCatalog) error {
	return RegisterDefectsWithResolution(s, service, nil, catalog)
}

// RegisterDefectsWithResolution registra la pantalla de defectos una sola vez, con todas sus rutas:
// las de HU-06 y, si hay servicio de resolución, las de HU-19.
func RegisterDefectsWithResolution(s *Server, service *defect.Service, resolution *defect.ResolutionService, catalog DefectCatalog) error {
	h := &defectsHandlers{server: s, service: service, resolution: resolution, catalog: catalog}
	routes := Routes{
		"GET /defectos":  http.HandlerFunc(h.list),
		"POST /defectos": http.HandlerFunc(h.register),
	}
	if resolution != nil {
		routes["POST /defectos/resolver"] = http.HandlerFunc(h.resolve)
	}
	return s.Register(Section{Name: "Defectos", Path: "/defectos", RequiresProject: true}, routes)
}

func (h *defectsHandlers) pageData(r *http.Request, projectID int64) (defectsPageData, error) {
	defects, err := h.catalog.Defects(r.Context(), projectID)
	if err != nil {
		return defectsPageData{}, fmt.Errorf("leer defectos: %w", err)
	}
	stories, err := h.catalog.StoryOptions(r.Context(), projectID)
	if err != nil {
		return defectsPageData{}, fmt.Errorf("leer historias para defectos: %w", err)
	}
	sprintNumber, hasActiveSprint, err := h.catalog.ActiveSprintNumber(r.Context(), projectID)
	if err != nil {
		return defectsPageData{}, fmt.Errorf("leer Sprint activo para defectos: %w", err)
	}
	return defectsPageData{
		Defects:          defects,
		Stories:          stories,
		Severities:       []defect.Severity{defect.SeverityCritical, defect.SeverityHigh, defect.SeverityMedium, defect.SeverityLow},
		ActiveSprint:     sprintNumber,
		HasActiveSprint:  hasActiveSprint,
		NoDefectsMessage: MsgDefectNone,
		CanResolve:       h.resolution != nil,
	}, nil
}

func (h *defectsHandlers) list(w http.ResponseWriter, r *http.Request) {
	ac := ActiveContextFrom(r.Context())
	if ac == nil || ac.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("falta el contexto del proyecto activo"))
		return
	}
	data, err := h.pageData(r, ac.ProjectID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if !data.HasActiveSprint {
		h.renderPage(w, r, http.StatusConflict, data, NewFormView(), &Flash{
			Kind: FlashError,
			Text: MsgDefectNoSprint,
		})
		return
	}
	h.renderPage(w, r, http.StatusOK, data, NewFormView(), nil)
}

func (h *defectsHandlers) register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.server.renderer.Fail(w, r, fmt.Errorf("leer formulario de defectos: %w", err))
		return
	}
	ac := ActiveContextFrom(r.Context())
	if ac == nil || ac.ProjectID <= 0 {
		h.server.renderer.Fail(w, r, errors.New("falta el contexto del proyecto activo"))
		return
	}
	_, input, fieldErrors := parseDefectForm(r.PostForm, ac.ProjectID)
	form := FormFromRequest(r, DefectDescriptionField, DefectSeverityField, DefectStoryField)
	validatedInput, validationErrors := defect.Validate(input)
	for _, violation := range validationErrors {
		field, message, ok := defectFieldError(violation.Err)
		if ok && fieldErrors[field] == "" {
			fieldErrors[field] = message
		}
	}
	if len(fieldErrors) != 0 {
		for field, message := range fieldErrors {
			form.AddError(field, message)
		}
		h.renderFormStatus(w, r, ac.ProjectID, http.StatusUnprocessableEntity, form)
		return
	}
	registered, err := h.service.Register(r.Context(), validatedInput)
	if err != nil {
		switch {
		case errors.Is(err, defect.ErrNoActiveSprint):
			h.renderFormStatus(w, r, ac.ProjectID, http.StatusConflict, nil)
		case errors.Is(err, defect.ErrStoryNotFound):
			form.AddError(DefectStoryField, MsgDefectStoryNotInProject)
			h.renderFormStatus(w, r, ac.ProjectID, http.StatusUnprocessableEntity, form)
		default:
			h.renderFormStatus(w, r, ac.ProjectID, http.StatusInternalServerError, form)
		}
		return
	}
	SetFlash(w, r, FlashSuccess, fmt.Sprintf(MsgDefectRegistered, registered.SprintNumber))
	RedirectSeeOther(w, r, "/defectos")
}

func (h *defectsHandlers) renderFormStatus(w http.ResponseWriter, r *http.Request, projectID int64, status int, form *FormView) {
	data, err := h.pageData(r, projectID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if status == http.StatusConflict {
		h.renderPage(w, r, status, data, nil, &Flash{Kind: FlashError, Text: MsgDefectNoSprint})
		return
	}
	if status == http.StatusInternalServerError {
		h.renderPage(w, r, status, data, form, &Flash{Kind: FlashError, Text: MsgOperationFailed})
		return
	}
	h.renderPage(w, r, status, data, form, nil)
}

func (h *defectsHandlers) renderPage(w http.ResponseWriter, r *http.Request, status int, data defectsPageData, form *FormView, flash *Flash) {
	h.server.renderer.PageStatus(w, r, status, "defects.html", PageData{
		Title: "Defectos",
		Form:  form,
		Flash: flash,
		Data:  data,
	})
}
