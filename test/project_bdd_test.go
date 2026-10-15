package test

import (
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

func TestProjectCreation_BDD(t *testing.T) {
	// Given: a project store with one valid member and a valid creation request.
	store := &project.TestProjectStore{
		Members: map[int]bool{1: true},
	}
	startDate := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	request := project.ProjectCreationRequest{
		Name:      "BDD Project",
		Members:   []string{"1"},
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, 0, 14),
	}

	// When: the service creates the project from the request.
	service := project.NewProjectService(store)
	createdProject, err := service.CreateProject(request)

	// Then: creation succeeds and returns an active project with an ID.
	if err != nil {
		t.Fatalf("expected project creation to succeed, got error: %v", err)
	}
	if createdProject == nil {
		t.Fatal("expected a created project, got nil")
	}
	if createdProject.ID <= 0 {
		t.Errorf("expected project ID to be greater than 0, got %d", createdProject.ID)
	}
	if createdProject.Status != "Active" {
		t.Errorf("expected project status %q, got %q", "Active", createdProject.Status)
	}
}
