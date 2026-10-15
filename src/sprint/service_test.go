package sprint

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCreateSprint_AssignsThreePendingStoriesAndGeneratesDisplayID(t *testing.T) {
	req := validCreationRequest([]int64{101, 102, 103})
	project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
	stories := []UserStory{
		{ID: 101, ProjectID: 1, Status: Pendiente},
		{ID: 102, ProjectID: 1, Status: Pendiente},
		{ID: 103, ProjectID: 1, Status: Pendiente},
	}

	result, err := CreateSprint(req, project, stories)
	if err != nil {
		t.Fatalf("CreateSprint() devolvió un error inesperado: %v", err)
	}
	if result.Sprint.DisplayID != "Sprint 1" {
		t.Errorf("DisplayID = %q, se esperaba %q", result.Sprint.DisplayID, "Sprint 1")
	}
	if result.Sprint.Status != SprintActive {
		t.Errorf("estado = %q, se esperaba %q", result.Sprint.Status, SprintActive)
	}
	if result.Sprint.Goal != req.Goal {
		t.Errorf("Goal = %q, se esperaba %q", result.Sprint.Goal, req.Goal)
	}
	if !reflect.DeepEqual(result.Sprint.StoryIDs, req.StoryIDs) {
		t.Errorf("StoryIDs = %v, se esperaban %v", result.Sprint.StoryIDs, req.StoryIDs)
	}
	for _, story := range stories {
		if story.Status != EnSprint {
			t.Errorf("estado de historia %d = %q, se esperaba %q", story.ID, story.Status, EnSprint)
		}
		if story.ActiveSprintID != result.Sprint.ID {
			t.Errorf("Sprint activo de historia %d = %d, se esperaba %d", story.ID, story.ActiveSprintID, result.Sprint.ID)
		}
	}
}

func TestCreateSprint_RejectsEmptyOrWhitespaceGoal(t *testing.T) {
	tests := []struct {
		name string
		goal string
	}{
		{name: "vacío", goal: ""},
		{name: "solo espacios", goal: "   \t\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreationRequest([]int64{101})
			req.Goal = tt.goal
			project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
			stories := []UserStory{{ID: 101, ProjectID: 1, Status: Pendiente}}

			_, err := CreateSprint(req, project, stories)
			if !errors.Is(err, ErrGoalRequired) {
				t.Fatalf("CreateSprint() error = %v, se esperaba ErrGoalRequired", err)
			}
		})
	}
}

func TestSprintCreationRequestDoesNotExposeGeneratedIdentifier(t *testing.T) {
	requestType := reflect.TypeOf(SprintCreationRequest{})
	for _, forbiddenField := range []string{"ID", "Number", "DisplayID"} {
		if _, exists := requestType.FieldByName(forbiddenField); exists {
			t.Errorf("SprintCreationRequest expone el campo generado %q", forbiddenField)
		}
	}
}

func TestCreateSprint_RejectsStoryAssignedToAnotherActiveSprint(t *testing.T) {
	req := validCreationRequest([]int64{201})
	project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
	stories := []UserStory{{
		ID:             201,
		ProjectID:      1,
		Status:         EnSprint,
		ActiveSprintID: 99,
	}}

	_, err := CreateSprint(req, project, stories)
	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("CreateSprint() error = %v, se esperaba ErrActiveSprintConflict", err)
	}
	if stories[0].Status != EnSprint || stories[0].ActiveSprintID != 99 {
		t.Fatalf("el rechazo alteró la asignación existente: %+v", stories[0])
	}
}

func TestCreateSprint_AllowsStoryMoveOnlyAfterRemovingCurrentAssignment(t *testing.T) {
	project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
	story := UserStory{ID: 401, ProjectID: 1, Status: EnSprint, ActiveSprintID: 11}
	before := story

	_, err := CreateSprint(validCreationRequest([]int64{story.ID}), project, []UserStory{story})
	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("asignación directa = %v, se esperaba ErrActiveSprintConflict", err)
	}
	if story != before {
		t.Fatalf("la asignación directa modificó la historia: antes=%+v después=%+v", before, story)
	}

	if err := RemoveStoryFromSprint(&story, 11); err != nil {
		t.Fatalf("RemoveStoryFromSprint() devolvió error: %v", err)
	}
	if story.Status != Pendiente || story.ActiveSprintID != 0 {
		t.Fatalf("la historia no volvió a Pendiente al quitarla: %+v", story)
	}

	storiesAfterRemoval := []UserStory{story}
	result, err := CreateSprint(validCreationRequest([]int64{story.ID}), project, storiesAfterRemoval)
	if err != nil {
		t.Fatalf("la creación después de quitar la historia falló: %v", err)
	}
	if !reflect.DeepEqual(result.Sprint.StoryIDs, []int64{story.ID}) {
		t.Errorf("StoryIDs = %v, se esperaba [%d]", result.Sprint.StoryIDs, story.ID)
	}
	if storiesAfterRemoval[0].Status != EnSprint {
		t.Errorf("estado después de reasignar = %q, se esperaba %q", storiesAfterRemoval[0].Status, EnSprint)
	}
	if storiesAfterRemoval[0].ActiveSprintID != result.Sprint.ID {
		t.Errorf("Sprint activo después de reasignar = %d, se esperaba %d", storiesAfterRemoval[0].ActiveSprintID, result.Sprint.ID)
	}
}

func validCreationRequest(storyIDs []int64) SprintCreationRequest {
	return SprintCreationRequest{
		ProjectID: 1,
		Goal:      "Entregar la versión beta",
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		StoryIDs:  storyIDs,
	}
}
