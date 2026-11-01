package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// HU-13: edición de una historia del Backlog (GET y POST /backlog/{id}/editar). La sección es la de
// HU-22, que exige proyecto activo; la historia se busca siempre dentro de ese proyecto.

// edit muestra el formulario con los datos actuales.
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

// update guarda la edición: 404, agregar o quitar un criterio (200) y guardado.
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
	if _, err := h.catalog.StoryForEdit(r.Context(), active.ProjectID, id); err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}

	// Se leen los datos sin validarlos, para mostrarlos en cualquier respuesta que siga.
	form := parseStoryFields(r, false)
	filter, _ := backlogStatusFilter(r.PostFormValue(BacklogFieldStatus))
	// T079 completa aquí el control de la historia de un Sprint finalizado.
	if !form.applyAction() {
		h.renderEdit(w, r, http.StatusOK, id, form.Form, form.Criteria, filter, nil)
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
	if err != nil {
		// T068 y T079 completan el mapeo de los errores.
		h.server.renderer.Fail(w, r, err)
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
