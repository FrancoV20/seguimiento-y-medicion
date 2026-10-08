package project

import (
	"testing"
	"time"
)

func TestNewProject_RejectsMissingRequiredFields(t *testing.T) {
	startDate := time.Date(2026, time.October, 6, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 14)

	tests := []struct {
		name        string
		projectName string
		members     []string
	}{
		{
			name:        "rejects empty project name",
			projectName: "",
			members:     []string{"Gabriel"},
		},
		{
			name:        "rejects missing members",
			projectName: "Academic project",
			members:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given / Arrange: a project has a missing required field.
			// When / Act: attempt to create the project.
			_, err := NewProject(test.projectName, test.members, startDate, endDate)

			// Then / Assert: creation must fail with an explicit error.
			if err == nil {
				t.Fatal("expected an error for a missing required field")
			}
		})
	}
}
func TestNewProject_RejectsInvalidDates(t *testing.T) {
	name := "Sentinel Project"
	members := []string{"Juan Ignacio", "Franco"}

	t.Run("rejects end date before start date", func(t *testing.T) {
		// Arrange
		startDate := time.Now()
		endDate := startDate.AddDate(0, -1, 0) // 1 month BEFORE start

		// Act
		_, err := NewProject(name, members, startDate, endDate)

		// Assert
		if err == nil {
			t.Fatal("expected an error when end date is before start date")
		}
	})
}

func TestNewProject_SuccessInitialState(t *testing.T) {
	// Arrange: Datos 100% válidos
	name := "Sentinel Project"
	members := []string{"Juan Ignacio", "Franco"}
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0) // 1 mes DESPUÉS (válido)

	// Act
	p, err := NewProject(name, members, startDate, endDate)

	// Assert: No debe haber error
	if err != nil {
		t.Fatalf("did not expect an error for a valid project, got: %v", err)
	}

	// T006: El proyecto debe nacer con estado "Active"
	expectedStatus := "Active"
	if p.Status != expectedStatus {
		t.Errorf("expected initial status %q, got %q", expectedStatus, p.Status)
	}
}
