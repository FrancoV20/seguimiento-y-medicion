package web

import (
	"net/url"
	"strconv"
	"strings"
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

func parseProjectForm(values url.Values) (ProjectForm, ProjectFormInput, map[string]string) {
	form := ProjectForm{
		Name:      values.Get(ProjectNameField),
		StartDate: values.Get(ProjectStartDateField),
		EndDate:   values.Get(ProjectEndDateField),
		MemberIDs: append([]string(nil), values[ProjectMembersField]...),
	}
	input := ProjectFormInput{Name: strings.TrimSpace(form.Name)}
	fieldErrors := make(map[string]string)

	parseDate := func(field, value, missingMessage string) time.Time {
		if value == "" {
			fieldErrors[field] = missingMessage
			return time.Time{}
		}
		date, err := time.Parse("2006-01-02", value)
		if err != nil || date.Format("2006-01-02") != value {
			fieldErrors[field] = MsgInvalidDateFormat
			return time.Time{}
		}
		return date
	}

	input.StartDate = parseDate(ProjectStartDateField, form.StartDate, MsgMissingStartDate)
	input.EndDate = parseDate(ProjectEndDateField, form.EndDate, MsgMissingEndDate)

	seen := make(map[int]struct{}, len(form.MemberIDs))
	for _, rawID := range form.MemberIDs {
		id, err := strconv.Atoi(rawID)
		if err != nil || id <= 0 {
			fieldErrors[ProjectMembersField] = MsgInvalidMember
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		input.MemberIDs = append(input.MemberIDs, strconv.Itoa(id))
	}

	return form, input, fieldErrors
}
