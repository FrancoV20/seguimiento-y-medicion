package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

type defectsPageData struct {
	Defects         []DefectRow
	Stories         []DefectStoryOption
	Severities      []defect.Severity
	ActiveSprint    int64
	HasActiveSprint bool
}

type defectsHandlers struct {
	server  *Server
	service *defect.Service
	catalog DefectCatalog
}

// RegisterDefects registra la pantalla de defectos.
func RegisterDefects(s *Server, service *defect.Service, catalog DefectCatalog) error {
	h := &defectsHandlers{server: s, service: service, catalog: catalog}
	return s.Register(Section{Name: "Defectos", Path: "/defectos", RequiresProject: true}, Routes{
		"GET /defectos":  http.HandlerFunc(h.list),
		"POST /defectos": http.HandlerFunc(h.register),
	})
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
		Defects:         defects,
		Stories:         stories,
		Severities:      []defect.Severity{defect.SeverityCritical, defect.SeverityHigh, defect.SeverityMedium, defect.SeverityLow},
		ActiveSprint:    sprintNumber,
		HasActiveSprint: hasActiveSprint,
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
	h.server.renderer.Page(w, r, "defects.html", PageData{
		Title: "Defectos",
		Form:  NewFormView(),
		Data:  data,
	})
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
	if len(fieldErrors) != 0 {
		h.server.renderer.Fail(w, r, errors.New("formulario de defectos inválido"))
		return
	}
	registered, err := h.service.Register(r.Context(), input)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	SetFlash(w, r, FlashSuccess, fmt.Sprintf(MsgDefectRegistered, registered.SprintNumber))
	RedirectSeeOther(w, r, "/defectos")
}
