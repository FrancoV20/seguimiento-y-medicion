package web

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

const (
	FieldGoal      = "objetivo"
	FieldStartDate = "fecha_inicio"
	FieldEndDate   = "fecha_fin"
	FieldStory     = "historia"
	FieldStoryID   = "historia_id"
)

const (
	MsgSprintCreated         = "Sprint creado correctamente"
	MsgSprintWithoutStories  = "Sprint creado correctamente. Todavía no tiene historias asignadas"
	MsgStoriesAdded          = "Historias agregadas correctamente"
	MsgStoryRemoved          = "Historia quitada del Sprint"
	MsgSelectAtLeastOneStory = "Elija al menos una historia"
	MsgNoActiveSprint        = "No hay un Sprint activo"
	MsgMissingStartDate      = "Falta la fecha de inicio"
	MsgMissingEndDate        = "Falta la fecha de fin"
	MsgInvalidDateFormat     = "La fecha no tiene un formato válido"
)

// SprintFormInput contiene los datos de creación parseados desde el formulario.
type SprintFormInput struct {
	ProjectID int64
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	StoryIDs  []int64
}

func buildSprintFormInput(projectID int64, values url.Values) (*FormView, SprintFormInput) {
	form := NewFormView()
	input := SprintFormInput{ProjectID: projectID, Goal: values.Get(FieldGoal)}
	form.Values[FieldGoal] = values.Get(FieldGoal)
	form.Values[FieldStartDate] = values.Get(FieldStartDate)
	form.Values[FieldEndDate] = values.Get(FieldEndDate)
	form.Values[FieldStory] = strings.Join(values[FieldStory], ",")

	parseDate := func(field, value string, missingMessage string) time.Time {
		if value == "" {
			form.AddError(field, missingMessage)
			return time.Time{}
		}
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			form.AddError(field, MsgInvalidDateFormat)
			return time.Time{}
		}
		return parsed
	}
	input.StartDate = parseDate(FieldStartDate, values.Get(FieldStartDate), MsgMissingStartDate)
	input.EndDate = parseDate(FieldEndDate, values.Get(FieldEndDate), MsgMissingEndDate)

	seen := make(map[int64]struct{}, len(values[FieldStory]))
	for _, rawID := range values[FieldStory] {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || id <= 0 {
			form.AddError(FieldStory, fmt.Sprintf("El identificador de historia %q no es válido", rawID))
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		input.StoryIDs = append(input.StoryIDs, id)
	}
	return form, input
}

// SprintActions reúne las operaciones de HU-04 usadas por la interfaz.
type SprintActions interface {
	CreateSprint(ctx context.Context, request sprint.SprintCreationRequest, project sprint.Project, stories []sprint.UserStory) (sprint.SprintCreationResult, error)
	AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error
	RemoveStory(ctx context.Context, sprintID int64, storyID int64) error
}
