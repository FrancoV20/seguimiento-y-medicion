package sprint

import (
	"context"
	"sort"
	"strings"
)

// SprintStore persiste Sprints y administra las asociaciones con historias.
type SprintStore interface {
	// CreateSprint crea un Sprint y sus asignaciones iniciales como una sola operación.
	CreateSprint(ctx context.Context, sprint Sprint) (Sprint, error)
	// AssignStories asigna historias disponibles a un Sprint existente.
	AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error
	// RemoveStory quita una historia del Sprint y la devuelve a Pendiente.
	RemoveStory(ctx context.Context, sprintID int64, storyID int64) error
}

func validateSprint(sprint Sprint) error {
	if strings.TrimSpace(sprint.Goal) == "" {
		return ErrGoalRequired
	}
	if sprint.StartDate.IsZero() || sprint.EndDate.IsZero() || !sprint.EndDate.After(sprint.StartDate) {
		return ErrInvalidSprintPeriod
	}
	if sprint.ProjectID <= 0 {
		return ErrProjectNotFound
	}
	return nil
}

func normalizeStoryIDs(storyIDs []int64) ([]int64, error) {
	ids := append([]int64(nil), storyIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for i, id := range ids {
		if id <= 0 || i > 0 && ids[i-1] == id {
			return nil, ErrStoryNotAvailable
		}
	}
	return ids, nil
}
