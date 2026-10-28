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

// Motivos por los que una historia del Sprint no se puede quitar (T047).
const (
	RemoveReasonInProgress  = "Ya está en progreso"
	RemoveReasonDone        = "Ya está hecha"
	RemoveReasonHasEffort   = "Tiene horas registradas"
	RemoveReasonNotInSprint = "No está en el Sprint"
)

// CanRemoveStory indica si una historia asignada se puede quitar del Sprint: solo las que
// están "En Sprint" y no tienen horas registradas. Si no se puede, devuelve el motivo.
func CanRemoveStory(status StoryStatus, hasEffort bool) (bool, string) {
	switch status {
	case InProgress:
		return false, RemoveReasonInProgress
	case Done:
		return false, RemoveReasonDone
	case EnSprint:
		if hasEffort {
			return false, RemoveReasonHasEffort
		}
		return true, ""
	default:
		return false, RemoveReasonNotInSprint
	}
}

// RemoveStoryFromSprint elimina la asignación y devuelve la historia a Pendiente. Rechaza con
// *StoryRemovalError una historia que no pertenece al Sprint o que CanRemoveStory no admite.
func RemoveStoryFromSprint(story *UserStory, sprintID int64) error {
	if story == nil {
		return &StoryRemovalError{}
	}
	if sprintID <= 0 || story.ActiveSprintID != sprintID {
		return &StoryRemovalError{StoryID: story.ID}
	}
	if canRemove, reason := CanRemoveStory(story.Status, story.HasEffort); !canRemove {
		return &StoryRemovalError{StoryID: story.ID, Reason: reason}
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
