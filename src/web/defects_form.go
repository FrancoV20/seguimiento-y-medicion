package web

import (
	"net/url"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

const (
	DefectDescriptionField = "descripcion"
	DefectSeverityField    = "severidad"
	DefectStoryField       = "historia"
)

// DefectForm conserva los valores del formulario tal como llegaron.
type DefectForm struct {
	Description string
	Severity    string
	StoryID     string
}

func parseDefectForm(_ url.Values, projectID int64) (DefectForm, defect.RegisterInput, map[string]string) {
	return DefectForm{}, defect.RegisterInput{ProjectID: projectID}, nil
}

func defectFieldError(_ error) (string, string, bool) {
	return "", "", false
}
