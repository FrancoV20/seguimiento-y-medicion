package test

import (
    "testing"
    "time"
)

func TestSprintBDDScenarios(t *testing.T) {
    t.Run("crear sprint con tres historias pendientes", func(t *testing.T) {
        start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
        end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

        sprint, err := createSprintScenario(start, end, "Sprint Goal válido", []int64{10, 11, 12})
        if err != nil {
            t.Fatalf("createSprintScenario() returned unexpected error: %v", err)
        }

        if sprint.DisplayID != "Sprint 1" {
            t.Fatalf("DisplayID = %q, want %q", sprint.DisplayID, "Sprint 1")
        }
    })

    t.Run("rechaza sprint goal vacío", func(t *testing.T) {
        start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
        end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

        if _, err := createSprintScenario(start, end, "   ", []int64{20}); err == nil {
            t.Fatal("expected error when goal is blank")
        }
    })

    t.Run("rechaza fechas inválidas", func(t *testing.T) {
        start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
        end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

        if _, err := createSprintScenario(start, end, "Fechas inválidas", []int64{30}); err == nil {
            t.Fatal("expected error for endDate <= startDate")
        }
    })

    t.Run("permite sprint sin historias con advertencia", func(t *testing.T) {
        start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
        end := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

        sprint, err := createSprintScenario(start, end, "Sprint sin historias", nil)
        if err != nil {
            t.Fatalf("createSprintScenario() returned unexpected error: %v", err)
        }

        if len(sprint.StoryIDs) != 0 {
            t.Fatalf("len(StoryIDs) = %d, want 0", len(sprint.StoryIDs))
        }
    })
}

func createSprintScenario(start, end time.Time, goal string, storyIDs []int64) (sprintResult, error) {
    return sprintResult{}, nil
}

type sprintResult struct {
    DisplayID string
    StoryIDs  []int64
}
