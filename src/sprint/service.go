package sprint

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service valida los casos de uso de Sprint y coordina su persistencia.
type Service struct {
	store SprintStore
}

// NewService crea un servicio respaldado por el almacenamiento indicado.
func NewService(store SprintStore) Service {
	return Service{store: store}
}

// CreateSprint valida y persiste el Sprint antes de cambiar las historias recibidas.
func (s Service) CreateSprint(
	ctx context.Context,
	request SprintCreationRequest,
	project Project,
	stories []UserStory,
) (SprintCreationResult, error) {
	if err := validateSprintPeriod(request.StartDate, request.EndDate); err != nil {
		return SprintCreationResult{}, err
	}
	if s.store == nil {
		return SprintCreationResult{}, ErrPersistence
	}

	validatedStories := append([]UserStory(nil), stories...)
	candidate, err := CreateSprint(request, project, validatedStories)
	if err != nil {
		return SprintCreationResult{}, err
	}
	persisted, err := s.store.CreateSprint(ctx, candidate.Sprint)
	if err != nil {
		return SprintCreationResult{}, err
	}
	if persisted.ID <= 0 {
		return SprintCreationResult{}, fmt.Errorf("%w: el store no devolvió un ID de Sprint válido", ErrPersistence)
	}

	storyIndexes := make(map[int64]int, len(stories))
	for index, story := range stories {
		storyIndexes[story.ID] = index
	}
	for _, storyID := range request.StoryIDs {
		index := storyIndexes[storyID]
		stories[index].Status = EnSprint
		stories[index].ActiveSprintID = persisted.ID
	}
	return SprintCreationResult{Sprint: persisted}, nil
}

func validateSprintPeriod(startDate, endDate time.Time) error {
	if startDate.IsZero() || endDate.IsZero() || !endDate.After(startDate) {
		return ErrInvalidSprintPeriod
	}
	return nil
}

// CreateSprint valida los datos y prepara el Sprint junto con sus historias asignadas.
func CreateSprint(request SprintCreationRequest, project Project, stories []UserStory) (SprintCreationResult, error) {
	if request.ProjectID <= 0 || project.ID != request.ProjectID {
		return SprintCreationResult{}, ErrProjectNotFound
	}

	sprint := Sprint{
		ProjectID: request.ProjectID,
		Goal:      strings.TrimSpace(request.Goal),
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		StoryIDs:  append([]int64(nil), request.StoryIDs...),
	}
	if err := validateSprint(sprint); err != nil {
		return SprintCreationResult{}, err
	}

	number := project.NextSprintNumber
	if number <= 0 {
		number = 1
	}
	displayID, err := formatSprintID(number)
	if err != nil {
		return SprintCreationResult{}, err
	}

	indexesByID := make(map[int64]int, len(stories))
	for index, story := range stories {
		if _, exists := indexesByID[story.ID]; exists {
			return SprintCreationResult{}, fmt.Errorf("%w: historia duplicada %d", ErrStoryNotAvailable, story.ID)
		}
		indexesByID[story.ID] = index
	}

	selectedIndexes := make([]int, 0, len(request.StoryIDs))
	selectedIDs := make(map[int64]struct{}, len(request.StoryIDs))
	for _, storyID := range request.StoryIDs {
		if storyID <= 0 {
			return SprintCreationResult{}, ErrStoryNotAvailable
		}
		if _, duplicate := selectedIDs[storyID]; duplicate {
			return SprintCreationResult{}, fmt.Errorf("%w: historia duplicada %d", ErrStoryNotAvailable, storyID)
		}
		selectedIDs[storyID] = struct{}{}

		index, exists := indexesByID[storyID]
		if !exists {
			return SprintCreationResult{}, fmt.Errorf("%w: historia %d inexistente", ErrStoryNotAvailable, storyID)
		}
		if err := validateStoryForProject(stories[index], project.ID); err != nil {
			return SprintCreationResult{}, err
		}
		selectedIndexes = append(selectedIndexes, index)
	}

	// Usa el número secuencial como identificador provisional del agregado en memoria;
	// PostgreSQL reemplaza este ID con su clave técnica generada al persistirlo.
	sprint.ID = number
	sprint.Number = number
	sprint.DisplayID = displayID
	sprint.Status = SprintActive

	assignedStories := make([]UserStory, len(selectedIndexes))
	for i, index := range selectedIndexes {
		assignedStories[i] = stories[index]
		if err := AssignStoryToSprint(&assignedStories[i], sprint.ID); err != nil {
			return SprintCreationResult{}, err
		}
	}
	for i, index := range selectedIndexes {
		stories[index] = assignedStories[i]
	}

	return SprintCreationResult{Sprint: sprint}, nil
}

// MoveStory quita la historia del Sprint actual antes de asignarla al Sprint destino.
func MoveStory(ctx context.Context, store SprintStore, fromSprintID, toSprintID, storyID int64) error {
	if store == nil || fromSprintID <= 0 || toSprintID <= 0 || fromSprintID == toSprintID || storyID <= 0 {
		return ErrStoryNotAvailable
	}
	if err := store.RemoveStory(ctx, fromSprintID, storyID); err != nil {
		return err
	}
	return store.AssignStories(ctx, toSprintID, []int64{storyID})
}
