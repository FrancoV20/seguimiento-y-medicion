package web

import (
	"net/http"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

const projectsPageTitle = "Gestión de Proyectos"

type projectsPageData struct {
	Projects        []ProjectRow
	Members         []MemberOption
	NoMembers       string
	SelectedMembers map[string]bool
}

type projectsHandlers struct {
	server  *Server
	service *project.ProjectService
	catalog ProjectCatalog
}

// RegisterProjects registra la pantalla de gestión de proyectos.
func RegisterProjects(s *Server, service *project.ProjectService, catalog ProjectCatalog) error {
	handlers := projectsHandlers{server: s, service: service, catalog: catalog}
	return s.Register(Section{Name: "Proyectos", Path: "/proyectos"}, Routes{
		"GET /proyectos":  http.HandlerFunc(handlers.get),
		"POST /proyectos": http.HandlerFunc(handlers.post),
	})
}

func (h projectsHandlers) get(w http.ResponseWriter, r *http.Request) {
	data, err := h.pageData(r, nil)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, data, NewFormView(), nil)
}

func (h projectsHandlers) post(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.server.renderer.Fail(w, r, Internal(err))
		return
	}

	form, input, fieldErrors := parseProjectForm(r.PostForm)
	data, err := h.pageData(r, form.MemberIDs)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	formView := projectFormView(form, fieldErrors)

	if len(data.Members) == 0 {
		data.NoMembers = MsgNoMembersAvailable
		h.render(w, r, http.StatusUnprocessableEntity, data, formView, nil)
		return
	}
	if len(fieldErrors) > 0 {
		h.render(w, r, http.StatusUnprocessableEntity, data, formView, nil)
		return
	}

	result, err := h.service.CreateProject(project.ProjectCreationRequest{
		Name:      input.Name,
		Members:   input.MemberIDs,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
	if err != nil {
		if field, message, ok := projectFieldError(err); ok {
			formView.AddError(field, message)
			h.render(w, r, http.StatusUnprocessableEntity, data, formView, nil)
			return
		}
		h.render(w, r, http.StatusInternalServerError, data, formView,
			&Flash{Kind: FlashError, Text: MsgOperationFailed})
		return
	}
	if result == nil || result.Project == nil {
		h.render(w, r, http.StatusInternalServerError, data, formView,
			&Flash{Kind: FlashError, Text: MsgOperationFailed})
		return
	}

	SetFlash(w, r, FlashSuccess, result.Message)
	RedirectSeeOther(w, r, "/proyectos")
}

func (h projectsHandlers) pageData(r *http.Request, selected []string) (projectsPageData, error) {
	projects, err := h.catalog.ListProjects(r.Context())
	if err != nil {
		return projectsPageData{}, err
	}
	members, err := h.catalog.ListMembers(r.Context())
	if err != nil {
		return projectsPageData{}, err
	}

	selectedMembers := make(map[string]bool, len(selected))
	for _, id := range selected {
		if numericID, err := strconv.ParseInt(id, 10, 64); err == nil && numericID > 0 {
			selectedMembers[memberOptionKey(numericID)] = true
		}
	}
	return projectsPageData{
		Projects:        projects,
		Members:         members,
		SelectedMembers: selectedMembers,
	}, nil
}

func projectFormView(form ProjectForm, fieldErrors map[string]string) *FormView {
	view := NewFormView()
	view.Values[ProjectNameField] = form.Name
	view.Values[ProjectStartDateField] = form.StartDate
	view.Values[ProjectEndDateField] = form.EndDate
	for _, id := range form.MemberIDs {
		view.Values[ProjectMembersField] += id + ","
	}
	for field, message := range fieldErrors {
		view.AddError(field, message)
	}
	return view
}

func (h projectsHandlers) render(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	data projectsPageData,
	form *FormView,
	flash *Flash,
) {
	if len(data.Members) == 0 {
		data.NoMembers = MsgNoMembersAvailable
	}
	h.server.renderer.PageStatus(w, r, status, "projects.html", PageData{
		Title: projectsPageTitle,
		Data:  data,
		Form:  form,
		Flash: flash,
	})
}

func memberOptionKey(id int64) string {
	return strconv.FormatInt(id, 10)
}
