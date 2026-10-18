package sprint

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemorySprintStoreGeneratesSequentialSprintIDs(t *testing.T) {
	store := newMemorySprintStore()
	store.addProject(Project{ID: 1, Status: "Activo"})
	for index, want := range []string{"Sprint 1", "Sprint 2"} {
		sprint, err := store.CreateSprint(context.Background(), validTestSprint(1))
		if err != nil {
			t.Fatalf("CreateSprint() devolvió error: %v", err)
		}
		if sprint.DisplayID != want {
			t.Errorf("Sprint %d: DisplayID = %q, se esperaba %q", index+1, sprint.DisplayID, want)
		}
	}
}

func TestMemorySprintStoreAssignsExclusivelyAndAllowsRemoveThenAssign(t *testing.T) {
	store := newMemorySprintStore()
	store.addProject(Project{ID: 1, Status: "Activo"})
	store.addStory(UserStory{ID: 10, ProjectID: 1, Status: Pendiente})
	first, err := store.CreateSprint(context.Background(), validTestSprint(1))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateSprint(context.Background(), validTestSprint(1))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AssignStories(context.Background(), first.ID, []int64{10}); err != nil {
		t.Fatalf("primera asignación: %v", err)
	}
	if err := store.AssignStories(context.Background(), second.ID, []int64{10}); !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("segunda asignación = %v, se esperaba conflicto", err)
	}
	story, _ := store.story(10)
	if story.Status != EnSprint || story.ActiveSprintID != first.ID {
		t.Fatalf("la asignación existente no se conservó: %+v", story)
	}
	if err := store.RemoveStory(context.Background(), first.ID, 10); err != nil {
		t.Fatalf("RemoveStory(): %v", err)
	}
	if err := store.AssignStories(context.Background(), second.ID, []int64{10}); err != nil {
		t.Fatalf("asignación después de quitar: %v", err)
	}
	story, _ = store.story(10)
	if story.Status != EnSprint || story.ActiveSprintID != second.ID {
		t.Fatalf("la historia no se reasignó: %+v", story)
	}
}

func TestMemorySprintStoreAssignmentIsAtomic(t *testing.T) {
	store := newMemorySprintStore()
	store.addProject(Project{ID: 1, Status: "Activo"})
	store.addStory(UserStory{ID: 10, ProjectID: 1, Status: Pendiente})
	store.addStory(UserStory{ID: 11, ProjectID: 1, Status: InProgress})
	sprint, err := store.CreateSprint(context.Background(), validTestSprint(1))
	if err != nil {
		t.Fatal(err)
	}
	err = store.AssignStories(context.Background(), sprint.ID, []int64{10, 11})
	if !errors.Is(err, ErrStoryNotAvailable) {
		t.Fatalf("AssignStories() = %v, se esperaba historia no disponible", err)
	}
	story, _ := store.story(10)
	if story.Status != Pendiente || story.ActiveSprintID != 0 {
		t.Fatalf("la asignación parcial alteró la primera historia: %+v", story)
	}
}

func TestMemorySprintStoreInvalidCreationDoesNotConsumeSprintNumber(t *testing.T) {
	store := newMemorySprintStore()
	store.addProject(Project{ID: 1, Status: "Activo"})
	if _, err := store.CreateSprint(context.Background(), Sprint{
		ProjectID: 1,
		Goal:      "Objetivo inválido",
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		StoryIDs:  []int64{99},
	}); !errors.Is(err, ErrStoryNotAvailable) {
		t.Fatalf("creación inválida = %v, se esperaba historia no disponible", err)
	}
	sprint, err := store.CreateSprint(context.Background(), validTestSprint(1))
	if err != nil {
		t.Fatalf("creación válida posterior devolvió error: %v", err)
	}
	if sprint.DisplayID != "Sprint 1" {
		t.Errorf("DisplayID = %q, se esperaba Sprint 1", sprint.DisplayID)
	}
}

func validTestSprint(projectID int64) Sprint {
	return Sprint{
		ProjectID: projectID,
		Goal:      "Objetivo de prueba",
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
	}
}
