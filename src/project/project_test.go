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
