package sprint

import (
    "os"
    "testing"
    "time"
)

func TestPostgreSQLSprintStoreIntegration(t *testing.T) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        t.Skip("DATABASE_URL not set; skipping PostgreSQL integration test")
    }

    start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

    store, err := NewPostgreSQLSprintStore(dsn)
    if err != nil {
        t.Fatalf("NewPostgreSQLSprintStore() returned error: %v", err)
    }

    projectID, err := store.CreateProjectFixture()
    if err != nil {
        t.Fatalf("CreateProjectFixture() returned error: %v", err)
    }

    storyIDs, err := store.CreatePendingStoriesFixture(projectID, 3)
    if err != nil {
        t.Fatalf("CreatePendingStoriesFixture() returned error: %v", err)
    }

    sprint, err := store.CreateSprint(SprintCreationRequest{
        ProjectID: projectID,
        Goal:      "Puntuar integración",
        StartDate: start,
        EndDate:   end,
        StoryIDs:  storyIDs,
    })
    if err != nil {
        t.Fatalf("CreateSprint() returned error: %v", err)
    }

    if sprint.DisplayID != "Sprint 1" {
        t.Fatalf("DisplayID = %q, want %q", sprint.DisplayID, "Sprint 1")
    }

    if len(sprint.StoryIDs) != 3 {
        t.Fatalf("len(StoryIDs) = %d, want 3", len(sprint.StoryIDs))
    }

    if err := store.CleanupProject(projectID); err != nil {
        t.Fatalf("CleanupProject() returned error: %v", err)
    }
}
