package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// HU-13: edición de una historia del Backlog (GET y POST /backlog/{id}/editar). La sección es la de
// HU-22, que exige proyecto activo; la historia se busca siempre dentro de ese proyecto.

// edit muestra el formulario con los datos actuales. Una historia de un Sprint finalizado no se
// edita: se vuelve a la lista (con el filtro) con la advertencia.
func (h *backlogHandlers) edit(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id == 0 {
		h.server.renderer.Fail(w, r, NotFound(errors.New("identificador de historia inválido")))
		return
	}
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}
	story, err := h.catalog.StoryForEdit(r.Context(), active.ProjectID, id)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	filter, _ := backlogStatusFilter(r.URL.Query().Get(BacklogFieldStatus))
	if story.InFinishedSprint {
		SetFlash(w, r, FlashWarning, MsgStoryInFinishedSprint)
		RedirectSeeOther(w, r, backlogListURL(filter))
		return
	}

	form := NewFormView()
	form.Values[BacklogFieldTitle] = story.Title
	form.Values[BacklogFieldDescription] = story.Description
	form.Values[BacklogFieldPriority] = story.Priority
	if story.StoryPoints != nil {
		form.Values[BacklogFieldStoryPoints] = strconv.Itoa(*story.StoryPoints)
	}
	criteria := story.Criteria
	if len(criteria) == 0 {
		criteria = []string{""}
	}
	h.renderEdit(w, r, http.StatusOK, id, form, criteria, filter, nil)
}

// update guarda la edición. El orden de las respuestas es el de plan.md: 404, Sprint finalizado
// (409), agregar o quitar un criterio (200), validación (422), guardado.
func (h *backlogHandlers) update(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	if id == 0 {
		h.server.renderer.Fail(w, r, NotFound(errors.New("identificador de historia inválido")))
		return
	}
	active := ActiveContextFrom(r.Context())
	if active == nil || active.ProjectID == 0 {
		h.server.renderer.Fail(w, r, errors.New("active project context is required"))
		return
	}
	story, err := h.catalog.StoryForEdit(r.Context(), active.ProjectID, id)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}

	// Se leen los datos sin validarlos, para mostrarlos en cualquier respuesta que siga.
	form := parseStoryFields(r, false)
	filter, _ := backlogStatusFilter(r.PostFormValue(BacklogFieldStatus))
	finishedSprint := &Flash{Kind: FlashError, Text: MsgStoryInFinishedSprint}
	if story.InFinishedSprint {
		h.renderEdit(w, r, http.StatusConflict, id, form.Form, form.Criteria, filter, finishedSprint)
		return
	}
	if !form.applyAction() {
		h.renderEdit(w, r, http.StatusOK, id, form.Form, form.Criteria, filter, nil)
		return
	}
	if form.Form.HasErrors() {
		h.renderEdit(w, r, http.StatusUnprocessableEntity, id, form.Form, form.Criteria, filter, nil)
		return
	}
	if len(form.AcceptanceCriteria) == 0 {
		form.Form.AddError(BacklogFieldCriterion, backlog.ErrAcceptanceCriteriaRequired.Error())
		h.renderEdit(w, r, http.StatusUnprocessableEntity, id, form.Form, form.Criteria, filter, nil)
		return
	}

	result, err := h.creator.UpdateStory(r.Context(), backlog.UpdateStoryRequest{
		ProjectID:          active.ProjectID,
		StoryID:            id,
		Title:              form.Form.Values[BacklogFieldTitle],
		Description:        form.Form.Values[BacklogFieldDescription],
		Priority:           backlog.Priority(form.Form.Values[BacklogFieldPriority]),
		AcceptanceCriteria: form.AcceptanceCriteria,
		StoryPoints:        form.StoryPoints,
	})
	switch {
	case err == nil:
	case mapBacklogDomainError(form, err):
		h.renderEdit(w, r, http.StatusUnprocessableEntity, id, form.Form, form.Criteria, filter, nil)
		return
	case errors.Is(err, backlog.ErrStoryInFinishedSprint):
		h.renderEdit(w, r, http.StatusConflict, id, form.Form, form.Criteria, filter, finishedSprint)
		return
	case errors.Is(err, backlog.ErrStoryNotFound):
		h.server.renderer.Fail(w, r, NotFound(err))
		return
	default:
		h.renderEdit(w, r, http.StatusInternalServerError, id, form.Form, form.Criteria, filter,
			&Flash{Kind: FlashError, Text: MsgOperationFailed})
		return
	}

	kind, message := FlashSuccess, MsgStoryUpdated
	if result != nil && result.Warning != "" {
		kind, message = FlashWarning, MsgStoryUpdatedMissingPoints
	}
	SetFlash(w, r, kind, message)
	RedirectSeeOther(w, r, backlogListURL(filter))
}

// renderEdit muestra el formulario de edición con los valores dados y, si hay, el mensaje.
func (h *backlogHandlers) renderEdit(w http.ResponseWriter, r *http.Request, status int, id int64,
	form *FormView, criteria []string, filter string, flash *Flash) {
	fields := make([]backlogCriterionField, len(criteria))
	for i, value := range criteria {
		fields[i] = backlogCriterionField{Index: i, Value: value}
	}
	if len(fields) == 0 {
		fields = []backlogCriterionField{{Index: 0}}
	}
	h.server.renderer.PageStatus(w, r, status, "backlog_edit.html", PageData{
		Title: TitleStoryEdit,
		Form:  form,
		Flash: flash,
		Data: backlogEditPageData{
			ID:                 id,
			Action:             backlogPath + "/" + strconv.FormatInt(id, 10) + "/editar",
			Filter:             filter,
			BackURL:            backlogListURL(filter),
			Criteria:           fields,
			Priorities:         backlogPriorities(),
			StoryPoints:        backlogStoryPointValues(),
			InvalidPriority:    invalidBacklogPriority(form.Values[BacklogFieldPriority]),
			InvalidStoryPoints: invalidBacklogStoryPoints(form.Values[BacklogFieldStoryPoints]),
		},
	})
}
