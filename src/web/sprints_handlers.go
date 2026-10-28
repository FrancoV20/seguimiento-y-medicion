package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

type sprintsPageData struct {
	Active       SprintView
	HasActive    bool
	StoryOptions []sprintStoryOptionView
	Notice       string
	BacklogEmpty bool
}

type sprintStoryOptionView struct {
	StoryOption
	Selected bool
}

type sprintsHandlers struct {
	server  *Server
	catalog SprintCatalog
	actions SprintActions
}

// RegisterSprints registra la sección y las rutas de creación y consulta de Sprints.
func RegisterSprints(server *Server, catalog SprintCatalog, actions SprintActions) error {
	h := &sprintsHandlers{server: server, catalog: catalog, actions: actions}
	return server.Register(Section{
		Name:            "Sprints",
		Path:            "/sprints",
		RequiresProject: true,
	}, Routes{
		"GET /sprints":                   http.HandlerFunc(h.get),
		"POST /sprints":                  http.HandlerFunc(h.create),
		"POST /sprints/historias":        http.HandlerFunc(h.assignStories),
		"POST /sprints/historias/quitar": http.HandlerFunc(h.removeStory),
	})
}

func (h *sprintsHandlers) get(w http.ResponseWriter, r *http.Request) {
	activeContext := ActiveContextFrom(r.Context())
	if activeContext == nil || activeContext.ProjectID <= 0 {
		SetFlash(w, r, FlashWarning, MsgProjectRequired)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a=%2Fsprints")
		return
	}

	data, err := h.loadPageData(r, activeContext.ProjectID, nil)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	form := (*FormView)(nil)
	if !data.HasActive {
		form = NewFormView()
		data.Notice = MsgNoActiveSprint
	} else {
		data.Notice = activeSprintMessage(data.Active.Number)
	}
	h.server.renderer.Page(w, r, "sprints.html", PageData{Title: "Sprints", Form: form, Data: data})
}

func (h *sprintsHandlers) create(w http.ResponseWriter, r *http.Request) {
	activeContext := ActiveContextFrom(r.Context())
	if activeContext == nil || activeContext.ProjectID <= 0 {
		SetFlash(w, r, FlashWarning, MsgProjectRequired)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a=%2Fsprints")
		return
	}
	_ = r.ParseForm()
	form, input := buildSprintFormInput(activeContext.ProjectID, r.PostForm)
	data, err := h.loadPageData(r, activeContext.ProjectID, form)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if !form.HasErrors() {
		storyOptions := make(map[int64]struct{}, len(data.StoryOptions))
		for _, option := range data.StoryOptions {
			storyOptions[option.ID] = struct{}{}
		}
		for _, storyID := range input.StoryIDs {
			if _, belongsToProject := storyOptions[storyID]; !belongsToProject {
				form.AddError(FieldStory, sprint.ErrStoryNotAvailable.Error())
				break
			}
		}
	}
	if form.HasErrors() {
		h.server.renderer.PageStatus(w, r, http.StatusUnprocessableEntity, "sprints.html",
			PageData{Title: "Sprints", Form: form, Data: data})
		return
	}

	project := sprint.Project{
		ID:     activeContext.ProjectID,
		Name:   activeContext.ProjectName,
		Status: "Activo",
	}
	if data.HasActive {
		project.ActiveSprintNumber = data.Active.Number
	}
	stories := make([]sprint.UserStory, 0, len(data.StoryOptions))
	for _, option := range data.StoryOptions {
		story := sprint.UserStory{
			ID:        option.ID,
			ProjectID: activeContext.ProjectID,
			Status:    option.Status,
		}
		if option.AssignedSprintNumber > 0 {
			story.ActiveSprintID = 1
			story.ActiveSprintNumber = option.AssignedSprintNumber
		}
		stories = append(stories, story)
	}
	request := sprint.SprintCreationRequest{
		ProjectID: activeContext.ProjectID,
		Goal:      input.Goal,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
		StoryIDs:  input.StoryIDs,
	}
	result, err := h.actions.CreateSprint(r.Context(), request, project, stories)
	if err != nil {
		h.handleCreateError(w, r, form, data, err)
		return
	}

	message := MsgSprintCreated
	if len(request.StoryIDs) == 0 || result.Warning != "" {
		message = MsgSprintWithoutStories
	}
	SetFlash(w, r, FlashSuccess, message)
	RedirectSeeOther(w, r, "/sprints")
}

func (h *sprintsHandlers) assignStories(w http.ResponseWriter, r *http.Request) {
	activeContext := ActiveContextFrom(r.Context())
	if activeContext == nil || activeContext.ProjectID <= 0 {
		SetFlash(w, r, FlashWarning, MsgProjectRequired)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a=%2Fsprints")
		return
	}
	_ = r.ParseForm()
	form, storyIDs := buildStoryAssignmentInput(r.PostForm)
	data, err := h.loadPageData(r, activeContext.ProjectID, form)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if !data.HasActive {
		form.AddError(FieldStory, MsgNoActiveSprint)
		h.server.renderer.PageStatus(w, r, http.StatusConflict, "sprints.html",
			PageData{Title: "Sprints", Form: form, Data: data})
		return
	}
	if form.HasErrors() {
		h.server.renderer.PageStatus(w, r, http.StatusUnprocessableEntity, "sprints.html",
			PageData{Title: "Sprints", Form: form, Data: data})
		return
	}

	optionsByID := make(map[int64]StoryOption, len(data.StoryOptions))
	for _, option := range data.StoryOptions {
		optionsByID[option.ID] = option.StoryOption
	}
	for _, storyID := range storyIDs {
		option, belongsToProject := optionsByID[storyID]
		if !belongsToProject || !option.Available {
			form.AddError(FieldStory, fmt.Sprintf("%s: historia %d", sprint.ErrStoryNotAvailable.Error(), storyID))
			h.server.renderer.PageStatus(w, r, http.StatusConflict, "sprints.html",
				PageData{Title: "Sprints", Form: form, Data: data})
			return
		}
	}

	if err := h.actions.AssignStories(r.Context(), data.Active.ID, storyIDs); err != nil {
		mapped := mapSprintAssignmentError(err)
		if mapped.Field != "" {
			form.AddError(mapped.Field, mapped.Message)
		}
		pageData := PageData{Title: "Sprints", Form: form, Data: data}
		if mapped.StatusCode == http.StatusInternalServerError {
			pageData.Flash = &Flash{Kind: FlashError, Text: mapped.Message}
		}
		h.server.renderer.PageStatus(w, r, mapped.StatusCode, "sprints.html", pageData)
		return
	}

	SetFlash(w, r, FlashSuccess, MsgStoriesAdded)
	RedirectSeeOther(w, r, "/sprints")
}

// removeStory quita una historia del Sprint activo del proyecto activo. El dominio (HU-04)
// aplica la regla de quitar; un rechazo responde 409 con el motivo y sin cambios (T048).
func (h *sprintsHandlers) removeStory(w http.ResponseWriter, r *http.Request) {
	activeContext := ActiveContextFrom(r.Context())
	if activeContext == nil || activeContext.ProjectID <= 0 {
		SetFlash(w, r, FlashWarning, MsgProjectRequired)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a=%2Fsprints")
		return
	}
	_ = r.ParseForm()
	data, err := h.loadPageData(r, activeContext.ProjectID, nil)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	reject := func(status int, message string) {
		if data.HasActive {
			data.Notice = activeSprintMessage(data.Active.Number)
		}
		h.server.renderer.PageStatus(w, r, status, "sprints.html", PageData{
			Title: "Sprints",
			Data:  data,
			Flash: &Flash{Kind: FlashError, Text: message},
		})
	}
	if !data.HasActive {
		reject(http.StatusConflict, MsgNoActiveSprint)
		return
	}

	rawID := strings.TrimSpace(r.PostForm.Get(FieldStoryID))
	storyID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || storyID <= 0 {
		reject(http.StatusUnprocessableEntity, fmt.Sprintf("El identificador de historia %q no es válido", rawID))
		return
	}
	belongsToActiveSprint := false
	for _, story := range data.Active.Stories {
		if story.ID == storyID {
			belongsToActiveSprint = true
			break
		}
	}
	if !belongsToActiveSprint {
		reject(http.StatusConflict, sprint.ErrStoryNotAvailable.Error())
		return
	}

	if err := h.actions.RemoveStory(r.Context(), data.Active.ID, storyID); err != nil {
		var removalError *sprint.StoryRemovalError
		switch {
		case errors.As(err, &removalError):
			reject(http.StatusConflict, removalError.Message())
		case errors.Is(err, sprint.ErrStoryNotAvailable):
			reject(http.StatusConflict, sprint.ErrStoryNotAvailable.Error())
		default:
			reject(http.StatusInternalServerError, MsgOperationFailed)
		}
		return
	}

	SetFlash(w, r, FlashSuccess, MsgStoryRemoved)
	RedirectSeeOther(w, r, "/sprints")
}

func (h *sprintsHandlers) loadPageData(r *http.Request, projectID int64, form *FormView) (*sprintsPageData, error) {
	active, err := h.catalog.ActiveSprint(r.Context(), projectID)
	if err != nil {
		return nil, err
	}
	options, err := h.catalog.ListStoryOptions(r.Context(), projectID)
	if err != nil {
		return nil, err
	}
	selected := selectedStoryIDs(form)
	data := &sprintsPageData{
		StoryOptions: make([]sprintStoryOptionView, 0, len(options)),
		BacklogEmpty: len(options) == 0,
	}
	if active != nil {
		data.Active = *active
		data.HasActive = true
	}
	for _, option := range options {
		_, isSelected := selected[option.ID]
		data.StoryOptions = append(data.StoryOptions, sprintStoryOptionView{StoryOption: option, Selected: isSelected})
	}
	return data, nil
}

func selectedStoryIDs(form *FormView) map[int64]struct{} {
	selected := make(map[int64]struct{})
	if form == nil {
		return selected
	}
	for _, rawID := range strings.Split(form.Values[FieldStory], ",") {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err == nil && id > 0 {
			selected[id] = struct{}{}
		}
	}
	return selected
}

func (h *sprintsHandlers) handleCreateError(w http.ResponseWriter, r *http.Request, form *FormView, data *sprintsPageData, err error) {
	mapped := mapSprintError(err)
	if errors.Is(err, sprint.ErrActiveSprintExists) {
		data.Notice = mapped.Message
		h.server.renderer.PageStatus(w, r, mapped.StatusCode, "sprints.html", PageData{Title: "Sprints", Data: data})
		return
	}
	if mapped.Field != "" {
		form.AddError(mapped.Field, mapped.Message)
	}
	pageData := PageData{Title: "Sprints", Form: form, Data: data}
	if mapped.StatusCode == http.StatusInternalServerError {
		pageData.Flash = &Flash{Kind: FlashError, Text: mapped.Message}
	}
	h.server.renderer.PageStatus(w, r, mapped.StatusCode, "sprints.html", pageData)
}

func activeSprintMessage(number int64) string {
	return fmt.Sprintf("Ya hay un Sprint activo (Sprint %d)", number)
}
