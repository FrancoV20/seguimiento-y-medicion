package web

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

const (
	DefectDescriptionField = "descripcion"
	DefectSeverityField    = "severidad"
	DefectStoryField       = "historia"

	MsgDefectRegistered          = "Defecto registrado correctamente. Sprint de detección: Sprint %d"
	MsgDefectDescriptionRequired = "La descripción es obligatoria"
	MsgDefectDescriptionTooLong  = "La descripción no puede superar 1000 caracteres"
	MsgDefectSeverityRequired    = "Seleccione la severidad"
	MsgDefectSeverityInvalid     = "La severidad no es válida"
	MsgDefectStoryRequired       = "Seleccione la historia relacionada"
	MsgDefectStoryNotInProject   = "La historia no pertenece al proyecto"
	MsgDefectNoSprint            = "No hay un Sprint activo; no se puede registrar un defecto"
	MsgDefectNone                = "Aún no hay defectos"
)

// DefectForm conserva los valores del formulario tal como llegaron.
type DefectForm struct {
	Description string
	Severity    string
	StoryID     string
}

func parseDefectForm(values url.Values, projectID int64) (DefectForm, defect.RegisterInput, map[string]string) {
	form := DefectForm{
		Description: values.Get(DefectDescriptionField),
		Severity:    values.Get(DefectSeverityField),
		StoryID:     values.Get(DefectStoryField),
	}
	input := defect.RegisterInput{
		ProjectID:   projectID,
		Description: form.Description,
		Severity:    defect.Severity(form.Severity),
	}
	fieldErrors := make(map[string]string)

	switch {
	case form.StoryID == "":
		fieldErrors[DefectStoryField] = MsgDefectStoryRequired
	default:
		storyID, err := strconv.ParseInt(form.StoryID, 10, 64)
		if err != nil || storyID <= 0 {
			fieldErrors[DefectStoryField] = MsgDefectStoryNotInProject
		} else {
			input.StoryID = storyID
		}
	}
	return form, input, fieldErrors
}

func defectFieldError(err error) (string, string, bool) {
	switch {
	case errors.Is(err, defect.ErrDescriptionRequired):
		return DefectDescriptionField, MsgDefectDescriptionRequired, true
	case errors.Is(err, defect.ErrDescriptionTooLong):
		return DefectDescriptionField, MsgDefectDescriptionTooLong, true
	case errors.Is(err, defect.ErrSeverityRequired):
		return DefectSeverityField, MsgDefectSeverityRequired, true
	case errors.Is(err, defect.ErrSeverityInvalid):
		return DefectSeverityField, MsgDefectSeverityInvalid, true
	case errors.Is(err, defect.ErrStoryRequired):
		return DefectStoryField, MsgDefectStoryRequired, true
	case errors.Is(err, defect.ErrStoryNotFound):
		return DefectStoryField, MsgDefectStoryNotInProject, true
	default:
		return "", "", false
	}
}
