package metrics

import (
	"math/big"
	"testing"
)

func TestCalculateMetricsVelocityUsesOnlyCompletedStories(t *testing.T) {
	store := &TestMetricsStore{
		SprintData: SprintData{
			ID:     "sprint-1",
			Status: "Finalizado",
			Stories: []CompletedStory{
				{ID: "completed-1", Status: "Terminada", StoryPoints: intPointer(5)},
				{ID: "completed-2", Status: "Terminada", StoryPoints: intPointer(3)},
				{ID: "in-progress", Status: "En progreso", StoryPoints: intPointer(13)},
				{ID: "pending", Status: "Pendiente", StoryPoints: intPointer(8)},
			},
		},
	}

	got, err := NewService(store).CalculateMetrics("sprint-1")
	if err != nil {
		t.Fatalf("CalculateMetrics() error = %v", err)
	}
	if got.Velocity != 8 {
		t.Errorf("Velocity = %d, want 8 (completed stories only)", got.Velocity)
	}
}

func TestCalculateMetricsTotalsAndPersistsCompletedStoryEffort(t *testing.T) {
	store := &TestMetricsStore{
		SprintData: SprintData{
			ID:     "sprint-2",
			Status: "Finalizado",
			Stories: []CompletedStory{
				{
					ID:             "completed-1",
					Status:         "Terminada",
					StoryPoints:    intPointer(3),
					EstimatedHours: serviceTestRat(t, "8"),
					ActualHours:    serviceTestRat(t, "10"),
				},
				{
					ID:             "completed-2",
					Status:         "Terminada",
					StoryPoints:    intPointer(5),
					EstimatedHours: serviceTestRat(t, "12"),
					ActualHours:    serviceTestRat(t, "15"),
				},
				{
					ID:             "in-progress",
					Status:         "En progreso",
					StoryPoints:    intPointer(21),
					EstimatedHours: serviceTestRat(t, "100"),
					ActualHours:    serviceTestRat(t, "200"),
				},
			},
		},
	}

	got, err := NewService(store).CalculateMetrics("sprint-2")
	if err != nil {
		t.Fatalf("CalculateMetrics() error = %v", err)
	}

	if got.Velocity != 8 {
		t.Errorf("Velocity = %d, want 8", got.Velocity)
	}
	assertRatEqual(t, "EstimatedHoursTotal", got.EstimatedHoursTotal, "20")
	assertRatEqual(t, "ActualHoursTotal", got.ActualHoursTotal, "25")
	if got.DeviationPercentage != "25.00" {
		t.Errorf("DeviationPercentage = %q, want %q", got.DeviationPercentage, "25.00")
	}
	if len(store.SavedMetrics) != 1 {
		t.Fatalf("saved metrics count = %d, want 1", len(store.SavedMetrics))
	}
	if store.SavedMetrics[0].SprintID != "sprint-2" {
		t.Errorf("saved SprintID = %q, want %q", store.SavedMetrics[0].SprintID, "sprint-2")
	}
}

func intPointer(value int) *int {
	return &value
}

func serviceTestRat(t *testing.T, value string) *big.Rat {
	t.Helper()

	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		t.Fatalf("invalid rational number %q", value)
	}
	return rat
}

func assertRatEqual(t *testing.T, field string, got *big.Rat, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %s", field, want)
		return
	}

	wantRat, ok := new(big.Rat).SetString(want)
	if !ok {
		t.Fatalf("invalid expected rational number %q", want)
	}
	if got.Cmp(wantRat) != 0 {
		t.Errorf("%s = %s, want %s", field, got.RatString(), wantRat.RatString())
	}
}
