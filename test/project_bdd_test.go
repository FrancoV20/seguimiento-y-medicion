package test

import (
	"errors"
	"reflect"
	"strings"
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
	result, err := service.CreateProject(request)

	// Then: creation succeeds and returns an active project with an ID.
	if err != nil {
		t.Fatalf("expected project creation to succeed, got error: %v", err)
	}
	if result == nil || result.Project == nil {
		t.Fatal("expected a project creation result, got nil")
	}
	if result.Project.ID <= 0 {
		t.Errorf("expected project ID to be greater than 0, got %d", result.Project.ID)
	}
	if result.Project.Status != "Active" {
		t.Errorf("expected project status %q, got %q", "Active", result.Project.Status)
	}
	if !strings.Contains(result.Message, "Proyecto creado exitosamente") {
		t.Errorf("expected success message, got %q", result.Message)
	}
}

func TestProjectCreation_InvalidDates_BDD(t *testing.T) {
	// Given: a valid member and a request whose end date is one day before its start date.
	store := &project.TestProjectStore{
		Members: map[int]bool{1: true},
	}
	startDate := time.Now()
	request := project.ProjectCreationRequest{
		Name:      "Invalid Dates Project",
		Members:   []string{"1"},
		StartDate: startDate,
		EndDate:   startDate.Add(-24 * time.Hour),
	}
	originalRequest := request
	originalRequest.Members = append([]string(nil), request.Members...)

	// When: the service attempts to create the project.
	service := project.NewProjectService(store)
	result, err := service.CreateProject(request)

	// Then: the period error is returned and nothing is persisted.
	if !errors.Is(err, project.ErrInvalidPeriod) {
		t.Errorf("expected error %v, got %v", project.ErrInvalidPeriod, err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %#v", result)
	}
	if len(store.Projects) != 0 {
		t.Errorf("expected no projects in the store, got %d", len(store.Projects))
	}
	if !reflect.DeepEqual(request, originalRequest) {
		t.Errorf("expected request data to remain unchanged, got %#v", request)
	}
}

func TestProjectCreation_EqualDates_BDD(t *testing.T) {
	// Given: a valid member and a request with equal start and end dates.
	store := &project.TestProjectStore{
		Members: map[int]bool{1: true},
	}
	startDate := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	request := project.ProjectCreationRequest{
		Name:      "Equal Dates Project",
		Members:   []string{"1"},
		StartDate: startDate,
		EndDate:   startDate,
	}

	// When: the service attempts to create the project.
	service := project.NewProjectService(store)
	result, err := service.CreateProject(request)

	// Then: the period is rejected with its field message and nothing is persisted.
	const expectedMessage = "La fecha de finalización debe ser posterior a la fecha de inicio"
	if !errors.Is(err, project.ErrInvalidPeriod) {
		t.Errorf("expected error %v, got %v", project.ErrInvalidPeriod, err)
	}
	if err != nil && err.Error() != expectedMessage {
		t.Errorf("expected error message %q, got %q", expectedMessage, err.Error())
	}
	if result != nil {
		t.Errorf("expected nil result, got %#v", result)
	}
	if len(store.Projects) != 0 {
		t.Errorf("expected no projects in the store, got %d", len(store.Projects))
	}
}

func TestProjectCreation_EndDateOneDayAfter_BDD(t *testing.T) {
	// Given: a valid member and a project whose end date is one day after its start date.
	store := &project.TestProjectStore{
		Members: map[int]bool{1: true},
	}
	startDate := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	request := project.ProjectCreationRequest{
		Name:      "One Day Project",
		Members:   []string{"1"},
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, 0, 1),
	}

	// When: the service creates the project.
	service := project.NewProjectService(store)
	result, err := service.CreateProject(request)

	// Then: creation succeeds.
	if err != nil {
		t.Fatalf("expected project creation to succeed, got error: %v", err)
	}
	if result == nil || result.Project == nil {
		t.Fatal("expected a project creation result, got nil")
	}
	if len(store.Projects) != 1 {
		t.Errorf("expected one project in the store, got %d", len(store.Projects))
	}
}
