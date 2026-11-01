package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

const backlogPath = "/backlog"

// RegisterBacklog registers the project-scoped Backlog page and its GET/POST handlers.
func RegisterBacklog(s *Server, creator *backlog.BacklogService, catalog StoryCatalog) error {
	if s == nil {
		return errors.New("web server is required")
	}
	if creator == nil {
		return errors.New("backlog service is required")
	}
	if catalog == nil {
		return errors.New("story catalog is required")
	}

	handler := &backlogHandlers{server: s, creator: creator, catalog: catalog}
	return s.Register(Section{
		Name:            "Backlog",
		Path:            backlogPath,
		RequiresProject: true,
	}, Routes{
		"GET /backlog":              http.HandlerFunc(handler.get),
		"POST /backlog":             http.HandlerFunc(handler.post),
		"GET /backlog/{id}/editar":  http.HandlerFunc(handler.edit),   // HU-13
		"POST /backlog/{id}/editar": http.HandlerFunc(handler.update), // HU-13
	})
}

type backlogHandlers struct {
	server  *Server
	creator *backlog.BacklogService
	catalog StoryCatalog
}

func (h *backlogHandlers) get(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}

	form := NewFormView()
	// HU-13: filtro por estado; si "estado" viene repetido, cuenta el primero.
	filter, filterErr := backlogStatusFilter(r.URL.Query().Get(BacklogFieldStatus))
	data, err := h.pageData(r, active.ProjectID, []string{""}, form, filter, filterErr)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	h.server.renderer.PageStatus(w, r, http.StatusOK, "backlog.html", PageData{Title: "Backlog", Form: form, Data: data})
}

func (h *backlogHandlers) post(w http.ResponseWriter, r *http.Request) {
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}

	form := parseBacklogForm(r)
	if !form.applyAction() {
		h.render(w, r, http.StatusOK, active.ProjectID, form.Form, form.Criteria)
		return
	}
	if form.Form.HasErrors() {
		h.render(w, r, http.StatusUnprocessableEntity, active.ProjectID, form.Form, form.Criteria)
		return
	}
	if len(form.AcceptanceCriteria) == 0 {
		form.Form.AddError(BacklogFieldCriterion, backlog.ErrAcceptanceCriteriaRequired.Error())
		h.render(w, r, http.StatusUnprocessableEntity, active.ProjectID, form.Form, form.Criteria)
		return
	}

	request := backlog.CreateStoryRequest{
		ProjectID:          active.ProjectID,
		Title:              form.Form.Values[BacklogFieldTitle],
		Description:        form.Form.Values[BacklogFieldDescription],
		Priority:           backlog.Priority(form.Form.Values[BacklogFieldPriority]),
		AcceptanceCriteria: form.AcceptanceCriteria,
		StoryPoints:        form.StoryPoints,
		EstimatedHours:     form.EstimatedHours,
	}
	result, err := h.creator.CreateStory(request)
	if err != nil {
		if mapBacklogDomainError(form, err) {
			h.render(w, r, http.StatusUnprocessableEntity, active.ProjectID, form.Form, form.Criteria)
			return
		}
		data, readErr := h.pageData(r, active.ProjectID, form.Criteria, form.Form, "", "")
		if readErr != nil {
			h.server.renderer.Fail(w, r, readErr)
			return
		}
		h.server.renderer.PageStatus(w, r, http.StatusInternalServerError, "backlog.html",
			PageData{
				Title: "Backlog",
				Form:  form.Form,
				Data:  data,
				Flash: &Flash{Kind: FlashError, Text: MsgOperationFailed},
			})
		return
	}

	message := MsgStoryCreated
	kind := FlashSuccess
	if result != nil && result.Warning != "" {
		message = MsgStoryCreatedMissingPoints
		kind = FlashWarning
	}
	SetFlash(w, r, kind, message)
	RedirectSeeOther(w, r, backlogPath)
}

func (h *backlogHandlers) render(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	projectID int64,
	form *FormView,
	criteria []string,
) {
	data, err := h.pageData(r, projectID, criteria, form, "", "")
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	h.server.renderer.PageStatus(w, r, status, "backlog.html", PageData{
		Title: "Backlog",
		Form:  form,
		Data:  data,
	})
}

func (h *backlogHandlers) pageData(
	r *http.Request,
	projectID int64,
	criteria []string,
	form *FormView,
	filter string,
	filterErr string,
) (backlogPageData, error) {
	stories, err := h.catalog.ListStories(r.Context(), projectID)
	if err != nil {
		return backlogPageData{}, err
	}

	rows := make([]backlogStoryView, 0, len(stories))
	for _, story := range stories {
		if filter != "" && story.Status != filter {
			continue
		}
		row := backlogStoryView{
			ID:               story.ID,
			Title:            story.Title,
			Priority:         story.Priority,
			Status:           story.Status,
			EditURL:          backlogEditURL(story.ID, filter),
			InFinishedSprint: story.InFinishedSprint,
		}
		row.StoryPoints = "Sin estimar"
		if story.StoryPoints != nil {
			row.StoryPoints = strconv.Itoa(*story.StoryPoints)
		}
		row.EstimatedHours = "Sin estimar"
		if story.EstimatedHours != nil {
			row.EstimatedHours = strings.ReplaceAll(
				strconv.FormatFloat(*story.EstimatedHours, 'f', -1, 64),
				".",
				",",
			)
		}
		rows = append(rows, row)
	}

	fields := make([]backlogCriterionField, len(criteria))
	for i, value := range criteria {
		fields[i] = backlogCriterionField{Index: i, Value: value}
	}
	if len(fields) == 0 {
		fields = []backlogCriterionField{{Index: 0}}
	}
	noResults := ""
	if len(stories) > 0 && len(rows) == 0 {
		noResults = MsgNoStoriesWithStatus
	}
	return backlogPageData{
		Stories:            rows,
		HasStories:         len(stories) > 0,
		Filter:             filter,
		Statuses:           backlogStatusOptions(),
		FilterError:        filterErr,
		NoResults:          noResults,
		Criteria:           fields,
		Priorities:         backlogPriorities(),
		StoryPoints:        backlogStoryPointValues(),
		InvalidPriority:    invalidBacklogPriority(form.Values[BacklogFieldPriority]),
		InvalidStoryPoints: invalidBacklogStoryPoints(form.Values[BacklogFieldStoryPoints]),
	}, nil
}
