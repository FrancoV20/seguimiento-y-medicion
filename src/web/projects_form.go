package web

import (
	"net/url"
	"time"
)

type ProjectForm struct {
	Name      string
	StartDate string
	EndDate   string
	MemberIDs []string
}

type ProjectFormInput struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	MemberIDs []string
}

const (
	ProjectNameField      = "nombre"
	ProjectStartDateField = "fecha_inicio"
	ProjectEndDateField   = "fecha_fin"
	ProjectMembersField   = "integrantes"

	MsgNoMembersAvailable = "No hay integrantes disponibles"
	MsgMissingStartDate   = "Falta la fecha de inicio"
	MsgMissingEndDate     = "Falta la fecha de fin"
	MsgInvalidDateFormat  = "La fecha no tiene un formato válido"
	MsgInvalidMember      = "El integrante seleccionado no es válido"
	MsgProjectCreated     = "Proyecto creado exitosamente"
)

// parseProjectForm is a provisional stub until the T015 implementation.
func parseProjectForm(values url.Values) (ProjectForm, ProjectFormInput, map[string]string) {
	return ProjectForm{
		Name:      values.Get(ProjectNameField),
		StartDate: values.Get(ProjectStartDateField),
		EndDate:   values.Get(ProjectEndDateField),
		MemberIDs: values[ProjectMembersField],
	}, ProjectFormInput{}, map[string]string{}
}
