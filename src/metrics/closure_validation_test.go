package metrics

import (
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestValidateSprintClosureRejectsInvalidCompletedStories(t *testing.T) {
	tests := []struct {
		name        string
		story       CompletedStory
		wantMessage string
	}{
		{
			name: "negative actual hours",
			story: CompletedStory{
				ID:          1,
				Status:      "Hecho",
				StoryPoints: closureTestIntPointer(3),
				ActualHours: closureTestRat(t, "-1"),
			},
			wantMessage: "horas reales negativas",
		},
		{
			name: "story points outside Fibonacci sequence",
			story: CompletedStory{
				ID:          2,
				Status:      "Hecho",
				StoryPoints: closureTestIntPointer(4),
				ActualHours: closureTestRat(t, "2"),
			},
			wantMessage: "Story Points inválidos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSprintClosure(SprintData{
				ID:      1,
				Status:  "Activo",
				Stories: []CompletedStory{tt.story},
			})
			if err == nil {
				t.Fatal("ValidateSprintClosure() error = nil, want validation error")
			}
			storyID := strconv.FormatInt(tt.story.ID, 10)
			if !strings.Contains(err.Error(), storyID) {
				t.Errorf("error %q does not identify story %d", err, tt.story.ID)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("error %q does not contain reason %q", err, tt.wantMessage)
			}
		})
	}
}

func TestClosureValidationErrorFormatsMultipleReasons(t *testing.T) {
	err := ValidateSprintClosure(SprintData{
		ID:     2,
		Status: "Activo",
		Stories: []CompletedStory{
			{
				ID:          3,
				Status:      "Hecho",
				StoryPoints: closureTestIntPointer(3),
				ActualHours: closureTestRat(t, "-1"),
			},
			{
				ID:          4,
				Status:      "Hecho",
				StoryPoints: closureTestIntPointer(4),
				ActualHours: closureTestRat(t, "2"),
			},
		},
	})
	if err == nil {
		t.Fatal("ValidateSprintClosure() error = nil, want validation error")
	}

	if got := reflect.TypeOf(err).Name(); got != "ClosureValidationError" {
		t.Errorf("error type = %q, want %q", got, "ClosureValidationError")
	}

	want := "Historia 3: horas reales negativas\n" +
		"Historia 4: Story Points inválidos"
	if got := err.Error(); got != want {
		t.Errorf("ClosureValidationError.Error() = %q, want %q", got, want)
	}
}

func TestCalculateMetricsWithInvalidSprintDoesNotPersistOrChangeStatus(t *testing.T) {
	tests := []struct {
		name        string
		story       CompletedStory
		wantMessage string
	}{
		{
			name: "negative actual hours",
			story: CompletedStory{
				ID:             5,
				Status:         "Hecho",
				StoryPoints:    closureTestIntPointer(3),
				EstimatedHours: closureTestRat(t, "8"),
				ActualHours:    closureTestRat(t, "-1"),
			},
			wantMessage: ErrNegativeActualHours.Error(),
		},
		{
			name: "invalid story points",
			story: CompletedStory{
				ID:             6,
				Status:         "Hecho",
				StoryPoints:    closureTestIntPointer(4),
				EstimatedHours: closureTestRat(t, "8"),
				ActualHours:    closureTestRat(t, "2"),
			},
			wantMessage: ErrInvalidStoryPoints.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &closureValidationStoreMock{
				sprint: SprintData{
					ID:      3,
					Status:  "Finalizado",
					Stories: []CompletedStory{tt.story},
				},
			}
			originalStatus := store.sprint.Status

			_, err := NewService(store).CalculateMetrics(store.sprint.ID)
			if err == nil {
				t.Fatal("CalculateMetrics() error = nil, want closure validation error")
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("CalculateMetrics() error = %q, want it to contain %q", err, tt.wantMessage)
			}
			if store.saveMetricsCalls != 0 {
				t.Errorf("SaveMetrics() called %d times, want 0", store.saveMetricsCalls)
			}
			if store.sprint.Status != originalStatus {
				t.Errorf("Sprint status = %q, want unchanged status %q", store.sprint.Status, originalStatus)
			}
		})
	}
}

type closureValidationStoreMock struct {
	sprint           SprintData
	saveMetricsCalls int
}

func (s *closureValidationStoreMock) LoadSprintData(int64) (SprintData, error) {
	return s.sprint, nil
}

func (s *closureValidationStoreMock) SaveMetrics(SprintMetrics) error {
	s.saveMetricsCalls++
	return nil
}

func closureTestIntPointer(value int) *int {
	return &value
}

func closureTestRat(t *testing.T, value string) *big.Rat {
	t.Helper()

	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		t.Fatalf("invalid rational number %q", value)
	}
	return rat
}
