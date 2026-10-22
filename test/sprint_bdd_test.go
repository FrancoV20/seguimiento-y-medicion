package test

import (
	"errors"
	"reflect"
	"strings"
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

	t.Run("bloquear historia que ya está asignada a un Sprint", func(t *testing.T) {
		// Dado que una historia ya está asignada al Sprint 2
		project := activeProject()
		story := sprint.UserStory{
			ID:                 20,
			ProjectID:          project.ID,
			Status:             sprint.EnSprint,
			ActiveSprintID:     99,
			ActiveSprintNumber: 2,
		}

		// Cuando el Scrum Master intenta asignarla a un Sprint
		_, err := sprint.CreateSprint(validSprintRequest(project.ID, []int64{story.ID}), project, []sprint.UserStory{story})

		// Entonces se bloquea e indica que ya está asignada al Sprint 2
		if !errors.Is(err, sprint.ErrActiveSprintConflict) || !strings.Contains(err.Error(), "ya está asignada al Sprint 2") {
			t.Fatalf("error = %v, se esperaba 'ya está asignada al Sprint 2'", err)
		}
	})

	t.Run("quitar una historia y volver a asignarla", func(t *testing.T) {
		// Dado una historia En Sprint del Sprint activo
		project := activeProject()
		story := sprint.UserStory{
			ID:                 30,
			ProjectID:          project.ID,
			Status:             sprint.EnSprint,
			ActiveSprintID:     31,
			ActiveSprintNumber: 1,
		}

		// Cuando la quita del Sprint
		if err := sprint.RemoveStoryFromSprint(&story, 31); err != nil {
			t.Fatalf("no se pudo quitar la historia del Sprint: %v", err)
		}

		// Entonces vuelve a Pendiente y la nueva asignación se permite sin conflicto
		if story.Status != sprint.Pendiente {
			t.Fatalf("estado al quitarla = %q, se esperaba Pendiente", story.Status)
		}
		if err := sprint.AssignStoryToSprint(&story, 31); err != nil {
			t.Fatalf("no se pudo volver a asignar la historia: %v", err)
		}
		if story.Status != sprint.EnSprint || story.ActiveSprintID != 31 {
			t.Errorf("al volver a asignarla debía quedar En Sprint: %+v", story)
		}
	})

	t.Run("un solo Sprint activo por proyecto", func(t *testing.T) {
		// Dado un proyecto que tiene el Sprint activo "Sprint 3"
		project := activeProject()
		project.ActiveSprintNumber = 3
		stories := []sprint.UserStory{{ID: 40, ProjectID: project.ID, Status: sprint.Pendiente}}

		// Cuando el Scrum Master intenta crear otro Sprint
		_, err := sprint.CreateSprint(validSprintRequest(project.ID, []int64{40}), project, stories)

		// Entonces se impide, informa el Sprint activo y no asigna historias
		if !errors.Is(err, sprint.ErrActiveSprintExists) || !strings.Contains(err.Error(), "Sprint 3") {
			t.Fatalf("error = %v, se esperaba 'ya hay un Sprint activo (Sprint 3)'", err)
		}
		if stories[0].Status != sprint.Pendiente {
			t.Errorf("la historia no debía asignarse: %+v", stories[0])
		}
	})

	t.Run("asignación exitosa deja las historias En Sprint", func(t *testing.T) {
		// Dado un proyecto sin Sprint activo y tres historias pendientes
		project := activeProject()
		stories := []sprint.UserStory{
			{ID: 50, ProjectID: project.ID, Status: sprint.Pendiente},
			{ID: 51, ProjectID: project.ID, Status: sprint.Pendiente},
			{ID: 52, ProjectID: project.ID, Status: sprint.Pendiente},
		}

		// Cuando creo el Sprint y le asigno las tres
		result, err := sprint.CreateSprint(validSprintRequest(project.ID, []int64{50, 51, 52}), project, stories)

		// Entonces quedan En Sprint de ese Sprint
		if err != nil {
			t.Fatalf("no se pudo crear el Sprint: %v", err)
		}
		for _, st := range stories {
			if st.Status != sprint.EnSprint || st.ActiveSprintNumber != result.Sprint.Number {
				t.Errorf("la historia %d debía quedar En Sprint del %s: %+v", st.ID, result.Sprint.DisplayID, st)
			}
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

	t.Run("crear Sprint sin historias y mostrar advertencia", func(t *testing.T) {
		project := activeProject()
		result, err := sprint.CreateSprint(validSprintRequest(project.ID, nil), project, nil)
		if err != nil {
			t.Fatalf("la creación de Sprint sin historias devolvió error: %v", err)
		}
		if result.Sprint.Status != sprint.SprintActive {
			t.Errorf("estado del Sprint = %q, se esperaba %q", result.Sprint.Status, sprint.SprintActive)
		}
		if len(result.Sprint.StoryIDs) != 0 {
			t.Errorf("historias asignadas = %v, se esperaba una lista vacía", result.Sprint.StoryIDs)
		}
		if result.Warning == "" {
			t.Fatal("se esperaba una advertencia no bloqueante para el Sprint sin historias")
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

	t.Run("rechazar fecha final anterior o igual a la inicial", func(t *testing.T) {
		project := activeProject()
		start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
		for _, end := range []time.Time{start.AddDate(0, 0, -1), start} {
			request := validSprintRequest(project.ID, nil)
			request.StartDate = start
			request.EndDate = end

			_, err := sprint.CreateSprint(request, project, nil)
			if !errors.Is(err, sprint.ErrInvalidSprintPeriod) {
				t.Errorf("fecha final %v: error = %v, se esperaba período inválido", end, err)
			}
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
