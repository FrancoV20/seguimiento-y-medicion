package project

import (
	"testing"
	"time"
)

func TestProjectService_CreateProject_Success(t *testing.T) {
	// Given / Arrange: valid members are available in the test store.
	store := &TestProjectStore{
		Members: map[int]bool{1: true, 2: true},
	}
	service := NewProjectService(store)
	startDate := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	req := ProjectCreationRequest{
		Name:      "Sentinel Project",
		Members:   []string{"1", "2"},
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, 1, 0),
	}

	// When / Act: a valid project is created.
	project, err := service.CreateProject(req)

	// Then / Assert: creation succeeds with the Active initial status.
	if err != nil {
		t.Fatalf("did not expect an error, got: %v", err)
	}
	if project == nil {
		t.Fatal("expected a project, got nil")
	}
	if project.Status != "Active" {
		t.Errorf("expected initial status %q, got %q", "Active", project.Status)
	}
}

func TestProjectService_CreateProject_SuccessWithEqualDates(t *testing.T) {
	// Given / Arrange: valid members are available and both dates are equal.
	store := &TestProjectStore{
		Members: map[int]bool{1: true, 2: true},
	}
	service := NewProjectService(store)
	date := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	req := ProjectCreationRequest{
		Name:      "Single-Day Project",
		Members:   []string{"1", "2"},
		StartDate: date,
		EndDate:   date,
	}

	// When / Act: a project with equal start and end dates is created.
	project, err := service.CreateProject(req)

	// Then / Assert: creation succeeds.
	if err != nil {
		t.Fatalf("did not expect an error, got: %v", err)
	}
	if project == nil {
		t.Fatal("expected a project, got nil")
	}
}
