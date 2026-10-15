package test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

func TestSprintBDDScenarios(t *testing.T) {
	t.Run("crear Sprint con tres historias pendientes", func(t *testing.T) {
		project := activeProject()
		stories := []sprint.UserStory{
			{ID: 10, ProjectID: project.ID, Status: sprint.Pendiente},
			{ID: 11, ProjectID: project.ID, Status: sprint.Pendiente},
			{ID: 12, ProjectID: project.ID, Status: sprint.Pendiente},
		}

		result, err := sprint.CreateSprint(validSprintRequest(project.ID, []int64{10, 11, 12}), project, stories)
		if err != nil {
			t.Fatalf("no se pudo crear el Sprint: %v", err)
		}
		if result.Sprint.DisplayID != "Sprint 1" {
			t.Errorf("identificador = %q, se esperaba %q", result.Sprint.DisplayID, "Sprint 1")
		}
		if result.Sprint.Status != sprint.SprintActive {
			t.Errorf("estado del Sprint = %q, se esperaba %q", result.Sprint.Status, sprint.SprintActive)
		}
		if !reflect.DeepEqual(result.Sprint.StoryIDs, []int64{10, 11, 12}) {
			t.Errorf("historias del Sprint = %v, se esperaban [10 11 12]", result.Sprint.StoryIDs)
		}
	})

	t.Run("bloquear historia que pertenece a otro Sprint activo", func(t *testing.T) {
		project := activeProject()
		story := sprint.UserStory{
			ID:             20,
			ProjectID:      project.ID,
			Status:         sprint.EnSprint,
			ActiveSprintID: 99,
		}

		_, err := sprint.CreateSprint(validSprintRequest(project.ID, []int64{story.ID}), project, []sprint.UserStory{story})
		if !errors.Is(err, sprint.ErrActiveSprintConflict) {
			t.Fatalf("error = %v, se esperaba conflicto con Sprint activo", err)
		}
	})

	t.Run("permitir movimiento solo después de quitar la asignación", func(t *testing.T) {
		project := activeProject()
		story := sprint.UserStory{
			ID:             30,
			ProjectID:      project.ID,
			Status:         sprint.EnSprint,
			ActiveSprintID: 31,
		}
		if err := sprint.RemoveStoryFromSprint(&story, 31); err != nil {
			t.Fatalf("no se pudo quitar la historia del Sprint actual: %v", err)
		}

		result, err := sprint.CreateSprint(validSprintRequest(project.ID, []int64{story.ID}), project, []sprint.UserStory{story})
		if err != nil {
			t.Fatalf("no se pudo asignar la historia después de quitarla: %v", err)
		}
		if !reflect.DeepEqual(result.Sprint.StoryIDs, []int64{story.ID}) {
			t.Errorf("historias del nuevo Sprint = %v, se esperaba [%d]", result.Sprint.StoryIDs, story.ID)
		}
	})

	t.Run("generar identificador de Sprint y no solicitarlo al usuario", func(t *testing.T) {
		project := activeProject()
		result, err := sprint.CreateSprint(validSprintRequest(project.ID, nil), project, nil)
		if err != nil {
			t.Fatalf("no se pudo crear el Sprint: %v", err)
		}
		if result.Sprint.DisplayID != "Sprint 1" {
			t.Errorf("identificador = %q, se esperaba %q", result.Sprint.DisplayID, "Sprint 1")
		}
	})

	t.Run("rechazar Sprint Goal vacío", func(t *testing.T) {
		project := activeProject()
		request := validSprintRequest(project.ID, nil)
		request.Goal = " \t "

		_, err := sprint.CreateSprint(request, project, nil)
		if !errors.Is(err, sprint.ErrGoalRequired) {
			t.Fatalf("error = %v, se esperaba Goal obligatorio", err)
		}
	})
}

func activeProject() sprint.Project {
	return sprint.Project{ID: 1, Name: "Proyecto BDD", Status: "Activo"}
}

func validSprintRequest(projectID int64, storyIDs []int64) sprint.SprintCreationRequest {
	return sprint.SprintCreationRequest{
		ProjectID: projectID,
		Goal:      "Completar las historias comprometidas",
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		StoryIDs:  storyIDs,
	}
}
