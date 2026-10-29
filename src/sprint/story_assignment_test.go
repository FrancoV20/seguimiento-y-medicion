package sprint

import (
	"errors"
	"strings"
	"testing"
)

// FIX-HU-04 (T004, T006): el mensaje indica a qué Sprint está asignada la historia.
func TestAssignStoryToSprint_MensajeIndicaElSprintDeLaHistoria(t *testing.T) {
	story := UserStory{ID: 2, ProjectID: 3, Status: EnSprint, ActiveSprintID: 9, ActiveSprintNumber: 3}

	err := AssignStoryToSprint(&story, 12)

	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("se esperaba ErrActiveSprintConflict, se obtuvo: %v", err)
	}
	if !strings.Contains(err.Error(), "ya está asignada al Sprint 3") || strings.Contains(err.Error(), "otro Sprint") {
		t.Errorf("el mensaje debe decir 'ya está asignada al Sprint 3': %q", err.Error())
	}
}

func TestRemoveStoryFromSprint_LimpiaElNumeroDeSprint(t *testing.T) {
	story := UserStory{ID: 4, ProjectID: 3, Status: EnSprint, ActiveSprintID: 9, ActiveSprintNumber: 3}

	if err := RemoveStoryFromSprint(&story, 9); err != nil {
		t.Fatalf("quitar la historia falló: %v", err)
	}
	if story.Status != Pendiente || story.ActiveSprintID != 0 || story.ActiveSprintNumber != 0 {
		t.Errorf("al quitarla debía quedar Pendiente y sin Sprint: %+v", story)
	}
}

func TestAssignStoryToSprintUpdatesPendingStory(t *testing.T) {
	story := UserStory{ID: 7, ProjectID: 3, Status: Pendiente}
	if err := AssignStoryToSprint(&story, 12); err != nil {
		t.Fatalf("AssignStoryToSprint() devolvió error: %v", err)
	}
	if story.Status != EnSprint || story.ActiveSprintID != 12 {
		t.Fatalf("la historia no quedó asignada correctamente: %+v", story)
	}
}

func TestAssignStoryToSprintRejectsUnavailableStoryWithoutMutation(t *testing.T) {
	tests := []struct {
		name      string
		story     UserStory
		sprintID  int64
		wantError error
	}{
		{
			name:      "estado distinto de pendiente",
			story:     UserStory{ID: 1, ProjectID: 3, Status: InProgress},
			sprintID:  12,
			wantError: ErrStoryNotAvailable,
		},
		{
			name:      "ya pertenece a otro Sprint",
			story:     UserStory{ID: 2, ProjectID: 3, Status: EnSprint, ActiveSprintID: 9},
			sprintID:  12,
			wantError: ErrActiveSprintConflict,
		},
		{
			name:      "Sprint destino inválido",
			story:     UserStory{ID: 3, ProjectID: 3, Status: Pendiente},
			sprintID:  0,
			wantError: ErrStoryNotAvailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.story
			err := AssignStoryToSprint(&tt.story, tt.sprintID)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantError)
			}
			if tt.story != before {
				t.Fatalf("la historia cambió ante el error: antes=%+v después=%+v", before, tt.story)
			}
		})
	}
}

func TestValidateStoryForProjectRejectsDifferentProject(t *testing.T) {
	err := validateStoryForProject(UserStory{ID: 5, ProjectID: 4, Status: Pendiente}, 3)
	if !errors.Is(err, ErrStoryNotAvailable) {
		t.Fatalf("error = %v, se esperaba historia no disponible", err)
	}
}

func TestRemoveStoryFromSprintReturnsStoryToPending(t *testing.T) {
	story := UserStory{ID: 7, ProjectID: 3, Status: EnSprint, ActiveSprintID: 12}
	if err := RemoveStoryFromSprint(&story, 12); err != nil {
		t.Fatalf("RemoveStoryFromSprint() devolvió error: %v", err)
	}
	if story.Status != Pendiente || story.ActiveSprintID != 0 {
		t.Fatalf("la historia no volvió a Pendiente: %+v", story)
	}
}

func TestRemoveStoryFromSprintRejectsNonMatchingSprintWithoutMutation(t *testing.T) {
	story := UserStory{ID: 7, ProjectID: 3, Status: EnSprint, ActiveSprintID: 12}
	before := story
	err := RemoveStoryFromSprint(&story, 13)
	if !errors.Is(err, ErrStoryNotAvailable) {
		t.Fatalf("error = %v, se esperaba historia no disponible", err)
	}
	if story != before {
		t.Fatalf("la historia cambió ante el error: antes=%+v después=%+v", before, story)
	}
}

func TestCanRemoveStoryReportsEligibilityAndBlockedReason(t *testing.T) {
	tests := []struct {
		name       string
		status     StoryStatus
		hasEffort  bool
		canRemove  bool
		wantReason string
	}{
		{name: "En Sprint sin horas", status: EnSprint, canRemove: true},
		{name: "en progreso", status: InProgress, wantReason: "Ya está en progreso"},
		{name: "hecha", status: Done, wantReason: "Ya está hecha"},
		{name: "con horas registradas", status: EnSprint, hasEffort: true, wantReason: "Tiene horas registradas"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			canRemove, reason := CanRemoveStory(test.status, test.hasEffort)
			if canRemove != test.canRemove || reason != test.wantReason {
				t.Errorf("CanRemoveStory(%q, %t) = (%t, %q); se esperaba (%t, %q)",
					test.status, test.hasEffort, canRemove, reason, test.canRemove, test.wantReason)
			}
		})
	}
}
