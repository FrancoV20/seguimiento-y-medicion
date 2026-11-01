package web

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// Backlog form field names and user-facing messages shared by handlers and templates.
const (
	BacklogFieldTitle          = "titulo"
	BacklogFieldDescription    = "descripcion"
	BacklogFieldPriority       = "prioridad"
	BacklogFieldStoryPoints    = "story_points"
	BacklogFieldEstimatedHours = "horas_estimadas"
	BacklogFieldCriterion      = "criterio"
	BacklogFieldAction         = "accion"

	MsgStoryCreated              = "Historia creada correctamente"
	MsgStoryCreatedMissingPoints = "Historia creada correctamente. Falta completar la estimación de Story Points"
	MsgInvalidStoryPoints        = "Los Story Points no son un valor válido"
	MsgHoursTooLarge             = "Las horas estimadas no pueden superar 9999,99"
	MsgInvalidEstimatedHours     = "El valor de horas no es válido"
)

// Textos y campos de la consulta y edición del Backlog (HU-13).
const (
	BacklogFieldStatus = "estado"

	TitleStoryEdit               = "Editar historia"
	MsgStoryUpdated              = "Historia modificada correctamente"
	MsgStoryUpdatedMissingPoints = "Historia modificada correctamente. Falta completar la estimación de Story Points"
	MsgStoryInFinishedSprint     = "La historia pertenece a un Sprint finalizado; no se puede editar"
	MsgInvalidStatusFilter       = "El estado no es válido"
	MsgNoStoriesWithStatus       = "No hay historias con ese estado"
	LabelEditStory               = "Editar"
	LabelFinishedSprint          = "Sprint finalizado"
	LabelAllStatuses             = "Todos"
)

const (
	backlogActionSave   = "guardar"
	backlogActionAdd    = "agregar_criterio"
	backlogActionRemove = "quitar_criterio:"
)

type backlogForm struct {
	Form               *FormView
	Action             string
	Criteria           []string
	AcceptanceCriteria []string
	StoryPoints        *int
	EstimatedHours     *float64
}

type backlogCriterionField struct {
	Index int
	Value string
}

type backlogPageData struct {
	Stories            []backlogStoryView
	HasStories         bool     // HU-13: el proyecto tiene historias, aunque el filtro no deje ninguna
	Filter             string   // HU-13: estado elegido; "" = Todos
	Statuses           []string // HU-13: Pendiente, En Sprint, En progreso, Hecho
	FilterError        string   // HU-13: MsgInvalidStatusFilter o ""
	NoResults          string   // HU-13: MsgNoStoriesWithStatus o ""
	Criteria           []backlogCriterionField
	Priorities         []string
	StoryPoints        []int
	InvalidPriority    string
	InvalidStoryPoints string
}

type backlogStoryView struct {
	ID             int64
	Title          string
	Priority       string
	StoryPoints    string
	EstimatedHours string
	Status         string
	// HU-13
	EditURL          string // /backlog/{id}/editar[?estado=…]
	InFinishedSprint bool
}

func parseBacklogForm(r *http.Request) *backlogForm {
	form := FormFromRequest(r,
		BacklogFieldTitle,
		BacklogFieldDescription,
		BacklogFieldPriority,
		BacklogFieldStoryPoints,
		BacklogFieldEstimatedHours,
	)
	criteria := append([]string(nil), r.PostForm[BacklogFieldCriterion]...)
	if len(criteria) == 0 {
		criteria = []string{""}
	}

	action := strings.TrimSpace(r.PostFormValue(BacklogFieldAction))
	if action == "" {
		action = backlogActionSave
	}
	parsed := &backlogForm{
		Form:     form,
		Action:   action,
		Criteria: criteria,
	}

	for _, criterion := range criteria {
		if criterion = strings.TrimSpace(criterion); criterion != "" {
			parsed.AcceptanceCriteria = append(parsed.AcceptanceCriteria, criterion)
		}
	}

	rawPoints := strings.TrimSpace(form.Values[BacklogFieldStoryPoints])
	if rawPoints != "" {
		points, err := strconv.Atoi(rawPoints)
		if err != nil {
			form.AddError(BacklogFieldStoryPoints, MsgInvalidStoryPoints)
		} else {
			parsed.StoryPoints = &points
		}
	}

	rawHours := strings.TrimSpace(form.Values[BacklogFieldEstimatedHours])
	if rawHours != "" {
		normalized := strings.ReplaceAll(rawHours, ",", ".")
		hours, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			form.AddError(BacklogFieldEstimatedHours, MsgInvalidEstimatedHours)
		} else {
			parsed.EstimatedHours = &hours
		}
	}

	return parsed
}

func mapBacklogDomainError(form *backlogForm, err error) bool {
	if form == nil || form.Form == nil {
		return false
	}
	switch {
	case errors.Is(err, backlog.ErrTitleRequired):
		form.Form.AddError(BacklogFieldTitle, backlog.ErrTitleRequired.Error())
	case errors.Is(err, backlog.ErrDescriptionRequired):
		form.Form.AddError(BacklogFieldDescription, backlog.ErrDescriptionRequired.Error())
	case errors.Is(err, backlog.ErrInvalidPriority):
		form.Form.AddError(BacklogFieldPriority, backlog.ErrInvalidPriority.Error())
	case errors.Is(err, backlog.ErrAcceptanceCriteriaRequired):
		form.Form.AddError(BacklogFieldCriterion, backlog.ErrAcceptanceCriteriaRequired.Error())
	case errors.Is(err, backlog.ErrInvalidStoryPoints):
		form.Form.AddError(BacklogFieldStoryPoints, MsgInvalidStoryPoints)
	case errors.Is(err, backlog.ErrInvalidEstimatedHours):
		message := MsgInvalidEstimatedHours
		if form.EstimatedHours != nil && *form.EstimatedHours > 9999.99 {
			message = MsgHoursTooLarge
		}
		form.Form.AddError(BacklogFieldEstimatedHours, message)
	default:
		return false
	}
	return true
}

// applyAction mutates only the displayed criterion fields. It returns true only for save.
func (f *backlogForm) applyAction() bool {
	if f == nil {
		return true
	}
	switch {
	case f.Action == backlogActionAdd:
		f.Criteria = append(f.Criteria, "")
		return false
	case strings.HasPrefix(f.Action, backlogActionRemove):
		index, err := strconv.Atoi(strings.TrimPrefix(f.Action, backlogActionRemove))
		if err == nil && index >= 0 && index < len(f.Criteria) && len(f.Criteria) > 1 {
			f.Criteria = append(f.Criteria[:index], f.Criteria[index+1:]...)
		}
		return false
	default:
		return true
	}
}

func backlogPriorities() []string {
	return []string{
		string(backlog.PriorityHigh),
		string(backlog.PriorityMedium),
		string(backlog.PriorityLow),
	}
}

func backlogStoryPointValues() []int {
	return []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}
}

func invalidBacklogPriority(value string) string {
	for _, priority := range backlogPriorities() {
		if value == priority {
			return ""
		}
	}
	return value
}

func invalidBacklogStoryPoints(value string) string {
	points, err := strconv.Atoi(value)
	if err != nil {
		return value
	}
	for _, allowed := range backlogStoryPointValues() {
		if points == allowed && strconv.Itoa(points) == value {
			return ""
		}
	}
	return value
}

// backlogStatusFilter valida el estado del filtro contra backlog.StoryStatuses. "Todos" y el valor
// vacío son la lista completa (""). Un estado desconocido (incluidas otras mayúsculas) también da la
// lista completa, con MsgInvalidStatusFilter.
func backlogStatusFilter(raw string) (filter, errMsg string) {
	if raw == "" || raw == LabelAllStatuses {
		return "", ""
	}
	for _, status := range backlog.StoryStatuses() {
		if raw == string(status) {
			return raw, ""
		}
	}
	return "", MsgInvalidStatusFilter
}

// backlogListURL es la dirección de la lista con el filtro, solo si es un estado válido; si no,
// /backlog. Nunca devuelve una dirección fuera del Backlog.
func backlogListURL(estado string) string {
	if filter, _ := backlogStatusFilter(estado); filter != "" {
		return backlogPath + "?" + BacklogFieldStatus + "=" + url.QueryEscape(filter)
	}
	return backlogPath
}

// backlogEditURL es la dirección de edición de la historia, con el filtro de la lista si lo hay.
func backlogEditURL(id int64, filter string) string {
	path := backlogPath + "/" + strconv.FormatInt(id, 10) + "/editar"
	if filter != "" {
		path += "?" + BacklogFieldStatus + "=" + url.QueryEscape(filter)
	}
	return path
}

// backlogStatusOptions son los estados del selector del filtro.
func backlogStatusOptions() []string {
	statuses := backlog.StoryStatuses()
	options := make([]string, 0, len(statuses))
	for _, status := range statuses {
		options = append(options, string(status))
	}
	return options
}
