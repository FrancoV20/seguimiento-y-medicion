package web

import (
	"net/http"
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
	Stories     []backlogStoryView
	Criteria    []backlogCriterionField
	Priorities  []string
	StoryPoints []int
}

type backlogStoryView struct {
	ID             int64
	Title          string
	Priority       string
	StoryPoints    string
	EstimatedHours string
	Status         string
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
