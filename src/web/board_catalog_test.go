package web

import (
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T008): BoardPoints es una función pura; los puntos nunca se guardan (FR-017).

func boardPuntos(n int) *int { return &n }

func TestBoardPoints_PlanificadosYCompletados(t *testing.T) {
	casos := []struct {
		nombre       string
		historias    []BoardStory
		planificados int
		completados  int
	}{
		{"sin historias", nil, 0, 0},
		{"planificados suman todas y completados solo las Hecho", []BoardStory{
			{Status: sprint.EnSprint, StoryPoints: boardPuntos(3)},
			{Status: sprint.InProgress, StoryPoints: boardPuntos(5)},
			{Status: sprint.Done, StoryPoints: boardPuntos(8)},
		}, 16, 8},
		{"una historia sin Story Points cuenta 0", []BoardStory{
			{Status: sprint.Done, StoryPoints: nil},
			{Status: sprint.Done, StoryPoints: boardPuntos(5)},
			{Status: sprint.EnSprint, StoryPoints: nil},
		}, 5, 5},
		{"En Sprint y En progreso no suman a completados", []BoardStory{
			{Status: sprint.EnSprint, StoryPoints: boardPuntos(3)},
			{Status: sprint.InProgress, StoryPoints: boardPuntos(5)},
		}, 8, 0},
		{"todas Hecho: completados igual a planificados", []BoardStory{
			{Status: sprint.Done, StoryPoints: boardPuntos(2)},
			{Status: sprint.Done, StoryPoints: boardPuntos(13)},
		}, 15, 15},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			planificados, completados := BoardPoints(c.historias)

			if planificados != c.planificados || completados != c.completados {
				t.Errorf("BoardPoints() = (%d, %d), se esperaba (%d, %d)",
					planificados, completados, c.planificados, c.completados)
			}
		})
	}
}
