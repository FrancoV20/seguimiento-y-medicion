package metrics

import (
	"reflect"
	"testing"
)

// T005 (HU-10): cálculo de la calidad y el avance de un Sprint finalizado a partir de los datos crudos.

func qualityStory(status string, points *int) QualityStoryResult {
	return QualityStoryResult{FinalStatus: status, StoryPoints: points}
}

func TestCalculateSprintQuality(t *testing.T) {
	done := func(points *int) QualityStoryResult { return qualityStory("Hecho", points) }
	inProgress := func(points *int) QualityStoryResult { return qualityStory("En progreso", points) }
	inSprint := func(points *int) QualityStoryResult { return qualityStory("En Sprint", points) }
	p := intPointer

	tests := []struct {
		name string
		data QualityData
		want SprintQuality
	}{
		{
			name: "3 de 4 historias completadas",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(1)), done(p(1)), done(p(1)), inProgress(p(1)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 4, StoriesCompleted: 3, CompletionPercentage: "75.00",
				PointsPlanned: 4, PointsCompleted: 3},
		},
		{
			name: "0 de 3 historias completadas",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				inProgress(p(2)), inSprint(p(3)), inSprint(p(5)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 3, StoriesCompleted: 0, CompletionPercentage: "0.00",
				PointsPlanned: 10, PointsCompleted: 0},
		},
		{
			name: "sin historias: No disponible",
			data: QualityData{SprintID: 7, Status: "Finalizado"},
			want: SprintQuality{SprintID: 7, CompletionPercentage: "No disponible"},
		},
		{
			name: "2 de 3 se redondea a 66.67",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(1)), done(p(1)), inSprint(p(1)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 3, StoriesCompleted: 2, CompletionPercentage: "66.67",
				PointsPlanned: 3, PointsCompleted: 2},
		},
		{
			name: "1 de 8 es 12.50",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(1)), inSprint(p(1)), inSprint(p(1)), inSprint(p(1)),
				inSprint(p(1)), inSprint(p(1)), inSprint(p(1)), inSprint(p(1)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 8, StoriesCompleted: 1, CompletionPercentage: "12.50",
				PointsPlanned: 8, PointsCompleted: 1},
		},
		{
			name: "1 de 3 se redondea a 33.33",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(1)), inSprint(p(1)), inProgress(p(1)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 3, StoriesCompleted: 1, CompletionPercentage: "33.33",
				PointsPlanned: 3, PointsCompleted: 1},
		},
		{
			name: "Story Points 3, 5, 8 y 2 con 3 y 5 Hecho: 18 planificados y 8 completados",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(3)), done(p(5)), inProgress(p(8)), inSprint(p(2)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 4, StoriesCompleted: 2, CompletionPercentage: "50.00",
				PointsPlanned: 18, PointsCompleted: 8},
		},
		{
			name: "una historia sin Story Points no se suma",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(3)), inSprint(nil),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 2, StoriesCompleted: 1, CompletionPercentage: "50.00",
				PointsPlanned: 3, PointsCompleted: 3, StoriesWithoutPoints: 1},
		},
		{
			name: "dos historias sin Story Points no se suman",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(p(5)), inSprint(nil), inProgress(nil),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 3, StoriesCompleted: 1, CompletionPercentage: "33.33",
				PointsPlanned: 5, PointsCompleted: 5, StoriesWithoutPoints: 2},
		},
		{
			name: "una Hecho sin Story Points cuenta como completada y sin puntos",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(nil), done(p(2)), inSprint(p(3)),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 3, StoriesCompleted: 2, CompletionPercentage: "66.67",
				PointsPlanned: 5, PointsCompleted: 2, StoriesWithoutPoints: 1},
		},
		{
			name: "todas sin Story Points: 0 planificados y 0 completados",
			data: QualityData{SprintID: 7, Status: "Finalizado", Stories: []QualityStoryResult{
				done(nil), inSprint(nil),
			}},
			want: SprintQuality{SprintID: 7, StoriesPlanned: 2, StoriesCompleted: 1, CompletionPercentage: "50.00",
				PointsPlanned: 0, PointsCompleted: 0, StoriesWithoutPoints: 2},
		},
		{
			name: "los defectos se copian sin cambios",
			data: QualityData{SprintID: 9, Status: "Finalizado", Stories: []QualityStoryResult{done(p(2))},
				DefectsDetected: 5, DefectsResolved: 3},
			want: SprintQuality{SprintID: 9, StoriesPlanned: 1, StoriesCompleted: 1, CompletionPercentage: "100.00",
				PointsPlanned: 2, PointsCompleted: 2, DefectsDetected: 5, DefectsResolved: 3},
		},
		{
			name: "defectos sin historias también se copian",
			data: QualityData{SprintID: 9, Status: "Finalizado", DefectsDetected: 2, DefectsResolved: 0},
			want: SprintQuality{SprintID: 9, CompletionPercentage: "No disponible", DefectsDetected: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSprintQuality(tt.data)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateSprintQuality()\n got  %+v\n want %+v", got, tt.want)
			}
		})
	}
}
