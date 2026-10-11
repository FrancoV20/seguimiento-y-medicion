package sprint

import (
    "testing"
    "time"
)

func TestCreateSprint_AssignsThreePendingStoriesAndGeneratesDisplayID(t *testing.T) {
    start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    req := SprintCreationRequest{
        ProjectID: 1,
        Goal:      "Entregar la versión beta",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  []int64{101, 102, 103},
    }

    project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
    stories := []UserStory{
        {ID: 101, ProjectID: 1, Status: Pendiente},
        {ID: 102, ProjectID: 1, Status: Pendiente},
        {ID: 103, ProjectID: 1, Status: Pendiente},
    }

    result, err := CreateSprint(req, project, stories)
    if err != nil {
        t.Fatalf("CreateSprint() returned unexpected error: %v", err)
    }

    if result.Sprint.DisplayID != "Sprint 1" {
        t.Fatalf("DisplayID = %q, want %q", result.Sprint.DisplayID, "Sprint 1")
    }

    if result.Sprint.Status != "Activo" {
        t.Fatalf("Status = %q, want %q", result.Sprint.Status, "Activo")
    }

    if len(result.Sprint.StoryIDs) != 3 {
        t.Fatalf("len(StoryIDs) = %d, want 3", len(result.Sprint.StoryIDs))
    }

    for _, storyID := range result.Sprint.StoryIDs {
        if storyID == 0 {
            t.Fatal("StoryIDs contained a zero value")
        }
    }
}

func TestCreateSprint_RejectsEmptyGoal(t *testing.T) {
    start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    req := SprintCreationRequest{
        ProjectID: 1,
        Goal:      "   ",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  []int64{101},
    }

    project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
    stories := []UserStory{{ID: 101, ProjectID: 1, Status: Pendiente}}

    _, err := CreateSprint(req, project, stories)
    if err == nil {
        t.Fatal("CreateSprint() expected error for empty goal, got nil")
    }
}

func TestCreateSprint_RejectsInactiveStoryAlreadyAssignedToAnotherActiveSprint(t *testing.T) {
    start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    req := SprintCreationRequest{
        ProjectID: 1,
        Goal:      "Ajustar backlog",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  []int64{201},
    }

    project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
    stories := []UserStory{{
        ID:          201,
        ProjectID:   1,
        Status:      EnSprint,
        ActiveSprintID: 99,
    }}

    _, err := CreateSprint(req, project, stories)
    if err == nil {
        t.Fatal("CreateSprint() expected error for conflicting active sprint assignment")
    }
}

func TestCreateSprint_RequiresEndDateAfterStartDate(t *testing.T) {
    start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    req := SprintCreationRequest{
        ProjectID: 1,
        Goal:      "Revisar alcance",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  []int64{301},
    }

    project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
    stories := []UserStory{{ID: 301, ProjectID: 1, Status: Pendiente}}

    _, err := CreateSprint(req, project, stories)
    if err == nil {
        t.Fatal("CreateSprint() expected error when endDate <= startDate")
    }
}

func TestCreateSprint_AllowsEmptyStorySelectionWithWarning(t *testing.T) {
    start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    req := SprintCreationRequest{
        ProjectID: 1,
        Goal:      "Planificar sprint sin historias",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  []int64{},
    }

    project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}

    result, err := CreateSprint(req, project, nil)
    if err != nil {
        t.Fatalf("CreateSprint() unexpected error for empty story selection: %v", err)
    }

    if result.Warning == "" {
        t.Fatal("result.Warning was empty for valid sprint with no stories")
    }

    if len(result.Sprint.StoryIDs) != 0 {
        t.Fatalf("len(StoryIDs) = %d, want 0", len(result.Sprint.StoryIDs))
    }
}

func TestCreateSprint_RemoveThenAssignAllowsStoryMove(t *testing.T) {
    start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
    story := UserStory{ID: 401, ProjectID: 1, Status: EnSprint, ActiveSprintID: 11}

    if err := RemoveStoryFromSprint(project.ID, story.ID, 11); err != nil {
        t.Fatalf("RemoveStoryFromSprint() unexpected error: %v", err)
    }

    if story.Status != Pendiente {
        t.Fatalf("story.Status = %q, want %q", story.Status, Pendiente)
    }

    req := SprintCreationRequest{
        ProjectID: 1,
        Goal:      "Asignar a otro sprint",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  []int64{401},
    }

    result, err := CreateSprint(req, project, []UserStory{story})
    if err != nil {
        t.Fatalf("CreateSprint() after remove then assign returned error: %v", err)
    }

    if result.Sprint.DisplayID != "Sprint 1" {
        t.Fatalf("DisplayID = %q, want %q", result.Sprint.DisplayID, "Sprint 1")
    }
}
