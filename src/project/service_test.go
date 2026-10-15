package project

import (
	"errors"
	"reflect"
	"strings"
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
	result, err := service.CreateProject(req)

	// Then / Assert: creation succeeds with the Active initial status.
	if err != nil {
		t.Fatalf("did not expect an error, got: %v", err)
	}
	if result == nil || result.Project == nil {
		t.Fatal("expected a project creation result, got nil")
	}
	if result.Project.Status != "Active" {
		t.Errorf("expected initial status %q, got %q", "Active", result.Project.Status)
	}
	if !strings.Contains(result.Message, "Proyecto creado exitosamente") {
		t.Errorf("expected success message, got %q", result.Message)
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
	result, err := service.CreateProject(req)

	// Then / Assert: creation succeeds.
	if err != nil {
		t.Fatalf("did not expect an error, got: %v", err)
	}
	if result == nil || result.Project == nil {
		t.Fatal("expected a project creation result, got nil")
	}
	if !strings.Contains(result.Message, "Proyecto creado exitosamente") {
		t.Errorf("expected success message, got %q", result.Message)
	}
}

func TestProjectService_CreateProject_ValidationErrors(t *testing.T) {
	startDate := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	tests := []struct {
		name          string
		projectName   string
		members       []string
		available     map[int]bool
		expectedError error
	}{
		{
			name:          "rejects empty name",
			projectName:   "",
			members:       []string{"1"},
			available:     map[int]bool{1: true},
			expectedError: ErrEmptyName,
		},
		{
			name:          "rejects empty member list",
			projectName:   "Sentinel Project",
			members:       []string{},
			available:     map[int]bool{1: true},
			expectedError: ErrMissingMembers,
		},
		{
			name:          "rejects duplicate members",
			projectName:   "Sentinel Project",
			members:       []string{"1", "1"},
			available:     map[int]bool{1: true},
			expectedError: ErrDuplicateMember,
		},
		{
			name:          "rejects nonexistent member",
			projectName:   "Sentinel Project",
			members:       []string{"2"},
			available:     map[int]bool{1: true},
			expectedError: ErrMemberNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given / Arrange: the store exposes only the configured member IDs.
			store := &TestProjectStore{Members: test.available}
			service := NewProjectService(store)
			req := ProjectCreationRequest{
				Name:      test.projectName,
				Members:   test.members,
				StartDate: startDate,
				EndDate:   endDate,
			}

			// When / Act: project creation is attempted.
			_, err := service.CreateProject(req)

			// Then / Assert: the expected sentinel is returned and nothing is persisted.
			if !errors.Is(err, test.expectedError) {
				t.Errorf("expected error %v, got %v", test.expectedError, err)
			}
			if len(store.Projects) != 0 {
				t.Errorf("expected no projects in the store, got %d", len(store.Projects))
			}
		})
	}

	t.Run("rejects end date before start date", func(t *testing.T) {
		// Given / Arrange: a valid member and a project period ending one day early.
		store := &TestProjectStore{Members: map[int]bool{1: true}}
		service := NewProjectService(store)
		req := ProjectCreationRequest{
			Name:      "Sentinel Project",
			Members:   []string{"1"},
			StartDate: time.Now(),
			EndDate:   time.Now().Add(-24 * time.Hour),
		}
		originalReq := req
		originalReq.Members = append([]string(nil), req.Members...)

		// When / Act: project creation is attempted.
		result, err := service.CreateProject(req)

		// Then / Assert: the period error is returned without creating a project.
		if !errors.Is(err, ErrInvalidPeriod) {
			t.Errorf("expected error %v, got %v", ErrInvalidPeriod, err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %#v", result)
		}
		if len(store.Projects) != 0 {
			t.Errorf("expected no projects in the store, got %d", len(store.Projects))
		}
		if !reflect.DeepEqual(req, originalReq) {
			t.Errorf("expected request data to remain unchanged, got %#v", req)
		}
	})
}
