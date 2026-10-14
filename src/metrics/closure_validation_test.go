package metrics

import (
	"math/big"
	"reflect"
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
				ID:          "negative-hours",
				Status:      "Terminada",
				StoryPoints: closureTestIntPointer(3),
				ActualHours: closureTestRat(t, "-1"),
			},
			wantMessage: "horas reales negativas",
		},
		{
			name: "story points outside Fibonacci sequence",
			story: CompletedStory{
				ID:          "invalid-points",
				Status:      "Terminada",
				StoryPoints: closureTestIntPointer(4),
				ActualHours: closureTestRat(t, "2"),
			},
			wantMessage: "Story Points inválidos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSprintClosure(SprintData{
				ID:      "sprint-invalid",
				Status:  "Activo",
				Stories: []CompletedStory{tt.story},
			})
			if err == nil {
				t.Fatal("ValidateSprintClosure() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.story.ID) {
				t.Errorf("error %q does not identify story %q", err, tt.story.ID)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("error %q does not contain reason %q", err, tt.wantMessage)
			}
		})
	}
}

func TestClosureValidationErrorFormatsMultipleReasons(t *testing.T) {
	err := ValidateSprintClosure(SprintData{
		ID:     "sprint-invalid",
		Status: "Activo",
		Stories: []CompletedStory{
			{
				ID:          "negative-hours",
				Status:      "Terminada",
				StoryPoints: closureTestIntPointer(3),
				ActualHours: closureTestRat(t, "-1"),
			},
			{
				ID:          "invalid-points",
				Status:      "Terminada",
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

	want := "Historia negative-hours: horas reales negativas\n" +
		"Historia invalid-points: Story Points inválidos"
	if got := err.Error(); got != want {
		t.Errorf("ClosureValidationError.Error() = %q, want %q", got, want)
	}
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
