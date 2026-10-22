package sprint

import (
	"fmt"
)

// AssignStoryToSprint asigna a un Sprint una historia que está Pendiente.
func AssignStoryToSprint(story *UserStory, sprintID int64) error {
	if story == nil || sprintID <= 0 {
		return ErrStoryNotAvailable
	}
	if story.Status == EnSprint || story.ActiveSprintID != 0 {
		return &StoryAlreadyAssignedError{StoryID: story.ID, SprintNumber: story.ActiveSprintNumber}
	}
	if story.Status != Pendiente {
		return ErrStoryNotAvailable
	}
	story.Status = EnSprint
	story.ActiveSprintID = sprintID
	story.ActiveSprintNumber = 0 // lo completa quien conoce el número del Sprint
	return nil
}

// RemoveStoryFromSprint elimina la asignación y devuelve la historia a Pendiente.
func RemoveStoryFromSprint(story *UserStory, sprintID int64) error {
	if story == nil || sprintID <= 0 || story.Status != EnSprint || story.ActiveSprintID != sprintID {
		return ErrStoryNotAvailable
	}
	story.Status = Pendiente
	story.ActiveSprintID = 0
	story.ActiveSprintNumber = 0
	return nil
}

func validateStoryForProject(story UserStory, projectID int64) error {
	if story.ProjectID != projectID {
		return fmt.Errorf("%w: la historia %d pertenece a otro proyecto", ErrStoryNotAvailable, story.ID)
	}
	if story.Status == EnSprint || story.ActiveSprintID != 0 {
		return &StoryAlreadyAssignedError{StoryID: story.ID, SprintNumber: story.ActiveSprintNumber}
	}
	if story.Status != Pendiente {
		return fmt.Errorf("%w: historia %d", ErrStoryNotAvailable, story.ID)
	}
	return nil
}
