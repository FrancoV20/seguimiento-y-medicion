package metrics

import (
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

func ValidateSprintClosure(sprint SprintData) error {
	var reasons []string

	for _, story := range sprint.Stories {
		if story.Status != "Terminada" {
			continue
		}

		if story.ActualHours != nil && story.ActualHours.Sign() < 0 {
			reasons = append(reasons, fmt.Sprintf("Historia %s: %s", story.ID, ErrNegativeActualHours))
		}

		if story.StoryPoints == nil || backlog.ValidateStoryPoints(story.StoryPoints) != nil {
			reasons = append(reasons, fmt.Sprintf("Historia %s: %s", story.ID, ErrInvalidStoryPoints))
		}
	}

	if len(reasons) == 0 {
		return nil
	}

	return ClosureValidationError{Reasons: reasons}
}
