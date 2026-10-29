package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

// projectEditPageData son los datos de project_edit.html (HU-11).
type projectEditPageData struct {
	ID              int64
	Members         []MemberOption
	SelectedMembers map[string]bool
}

// projectEditID lee el identificador de la dirección. Un valor no canónico, cero, negativo o
// fuera de rango no puede existir: se responde 404 sin consultar la base.
func projectEditID(r *http.Request) (int64, error) {
	id := parseID(r.PathValue("id"))
	if id == 0 {
		return 0, NotFound(fmt.Errorf("identificador de proyecto %q no válido", r.PathValue("id")))
	}
	return id, nil
}

// edit muestra el formulario con los datos actuales (GET /proyectos/{id}/modificar).
func (h projectsHandlers) edit(w http.ResponseWriter, r *http.Request) {
	id, err := projectEditID(r)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	current, err := h.catalog.ProjectForEdit(r.Context(), id)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	members, err := h.catalog.ListMembers(r.Context())
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}

	form := NewFormView()
	form.Values[ProjectNameField] = current.Name
	form.Values[ProjectStartDateField] = current.StartDate.Format("2006-01-02")
	form.Values[ProjectEndDateField] = current.EndDate.Format("2006-01-02")
	selected := make(map[string]bool, len(current.MemberIDs))
	for _, memberID := range current.MemberIDs {
		selected[memberOptionKey(memberID)] = true
	}
	h.renderEdit(w, r, http.StatusOK, projectEditPageData{ID: id, Members: members, SelectedMembers: selected}, form, nil)
}

// update guarda la modificación (POST /proyectos/{id}/modificar). El 404 se decide antes de
// validar el formulario; los rechazos vuelven a mostrar lo enviado.
func (h projectsHandlers) update(w http.ResponseWriter, r *http.Request) {
	id, err := projectEditID(r)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if _, err := h.catalog.ProjectForEdit(r.Context(), id); err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.server.renderer.Fail(w, r, Internal(err))
		return
	}

	form, input, fieldErrors := parseProjectForm(r.PostForm)
	members, err := h.catalog.ListMembers(r.Context())
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	data := projectEditPageData{ID: id, Members: members, SelectedMembers: selectedMemberKeys(form.MemberIDs)}
	formView := projectFormView(form, fieldErrors)
	if len(fieldErrors) > 0 {
		h.renderEdit(w, r, http.StatusUnprocessableEntity, data, formView, nil)
		return
	}

	err = h.service.UpdateProject(r.Context(), project.ProjectUpdateRequest{
		ID:        int(id),
		Name:      input.Name,
		Members:   input.MemberIDs,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
	if err != nil {
		h.updateFailed(w, r, err, data, formView)
		return
	}
	SetFlash(w, r, FlashSuccess, MsgProjectUpdated)
	RedirectSeeOther(w, r, "/proyectos")
}

// updateFailed traduce un error de la modificación a su respuesta (tabla de data-model.md).
func (h projectsHandlers) updateFailed(w http.ResponseWriter, r *http.Request, err error, data projectEditPageData, form *FormView) {
	var hasEffort *project.MemberHasEffortError
	var outOfRange *project.SprintOutOfRangeError
	switch {
	case errors.Is(err, project.ErrProjectNotFound):
		h.server.renderer.Fail(w, r, NotFound(err))
	case errors.As(err, &hasEffort):
		form.AddError(ProjectMembersField, fmt.Sprintf(MsgMemberHasEffort, hasEffort.MemberName))
		h.renderEdit(w, r, http.StatusConflict, data, form, nil)
	case errors.As(err, &outOfRange):
		field := ProjectEndDateField
		if outOfRange.Field == project.DateFieldStart {
			field = ProjectStartDateField
		}
		form.AddError(field, fmt.Sprintf(MsgSprintOutOfRange, outOfRange.Number))
		h.renderEdit(w, r, http.StatusConflict, data, form, nil)
	default:
		if field, message, ok := projectFieldError(err); ok {
			form.AddError(field, message)
			h.renderEdit(w, r, http.StatusUnprocessableEntity, data, form, nil)
			return
		}
		h.server.logger.Error("no se pudo modificar el proyecto", "proyecto", data.ID, "detalle", err.Error())
		h.renderEdit(w, r, http.StatusInternalServerError, data, form, &Flash{Kind: FlashError, Text: MsgOperationFailed})
	}
}

// selectedMemberKeys marca los integrantes enviados para volver a mostrarlos (FR-017).
func selectedMemberKeys(ids []string) map[string]bool {
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		if n, err := strconv.ParseInt(id, 10, 64); err == nil && n > 0 {
			selected[memberOptionKey(n)] = true
		}
	}
	return selected
}

func (h projectsHandlers) renderEdit(w http.ResponseWriter, r *http.Request, status int, data projectEditPageData, form *FormView, flash *Flash) {
	h.server.renderer.PageStatus(w, r, status, "project_edit.html", PageData{
		Title: TitleProjectEdit,
		Data:  data,
		Form:  form,
		Flash: flash,
	})
}
