package web

import (
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

func TestCloseFormCalculatesPendingStoriesSummary(t *testing.T) {
	board := BoardView{
		ID:     15,
		Number: 3,
		Label:  "Sprint 3",
		Groups: []BoardGroup{
			{
				Status: sprint.EnSprint,
				Stories: []BoardStory{
					{ID: 3, Title: "Historia En Sprint", StoryPoints: closeFormIntPointer(2), Status: sprint.EnSprint},
				},
			},
			{
				Status: sprint.InProgress,
				Stories: []BoardStory{
					{ID: 4, Title: "Historia En progreso", StoryPoints: closeFormIntPointer(8), Status: sprint.InProgress},
				},
			},
			{
				Status: sprint.Done,
				Stories: []BoardStory{
					{ID: 1, Title: "Historia Hecho 1", StoryPoints: closeFormIntPointer(3), Status: sprint.Done},
					{ID: 2, Title: "Historia Hecho 2", StoryPoints: closeFormIntPointer(5), Status: sprint.Done},
				},
			},
		},
		PlannedPoints:   18,
		CompletedPoints: 8,
	}

	got := buildCloseSummary(board)

	if len(got.Completed) != 2 {
		t.Errorf("historias completadas = %d; se esperaban 2", len(got.Completed))
	}
	if len(got.ToReturn) != 2 {
		t.Errorf("historias que serán devueltas a Pendiente = %d; se esperaban exactamente 2", len(got.ToReturn))
	}
	if got.PlannedPoints != 18 {
		t.Errorf("Story Points planificados = %d; se esperaban 18", got.PlannedPoints)
	}
	if got.CompletedPoints != 8 {
		t.Errorf("Story Points completados = %d; se esperaban 8", got.CompletedPoints)
	}
}

func closeFormIntPointer(value int) *int {
	return &value
}
