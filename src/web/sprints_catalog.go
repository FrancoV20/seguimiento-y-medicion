package web

import (
	"context"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// SprintView contiene el Sprint activo y sus historias para la pantalla web.
type SprintView struct {
	ID        int64
	Number    int64
	Label     string
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	Stories   []SprintStoryRow
}

// SprintStoryRow presenta una historia asignada y sus condiciones de remoción.
type SprintStoryRow struct {
	ID                  int64
	Title               string
	StoryPoints         *int
	Status              sprint.StoryStatus
	HasEffort           bool
	CanRemove           bool
	RemoveBlockedReason string
}

// StoryOption representa una historia del Backlog seleccionable o deshabilitada.
type StoryOption struct {
	ID                   int64
	Title                string
	Priority             backlog.Priority
	StoryPoints          *int
	Status               sprint.StoryStatus
	AssignedSprintNumber int64
	Available            bool
	UnavailableReason    string
}

// SprintCatalog expone las lecturas necesarias para la pantalla Sprints.
type SprintCatalog interface {
	ActiveSprint(ctx context.Context, projectID int64) (*SprintView, error)
	ListStoryOptions(ctx context.Context, projectID int64) ([]StoryOption, error)
}
