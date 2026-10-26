package web

import (
	"context"

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

// SprintActions reúne las operaciones de HU-04 usadas por la interfaz.
type SprintActions interface {
	CreateSprint(ctx context.Context, request sprint.SprintCreationRequest, project sprint.Project, stories []sprint.UserStory) (sprint.SprintCreationResult, error)
	AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error
	RemoveStory(ctx context.Context, sprintID int64, storyID int64) error
}
