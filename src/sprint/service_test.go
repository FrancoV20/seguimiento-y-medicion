package sprint

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCreateSprint_AssignsThreePendingStoriesAndGeneratesDisplayID(t *testing.T) {
	req := validCreationRequest([]int64{101, 102, 103})
	project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
	stories := []UserStory{
		{ID: 101, ProjectID: 1, Status: Pendiente},
		{ID: 102, ProjectID: 1, Status: Pendiente},
		{ID: 103, ProjectID: 1, Status: Pendiente},
	}

	result, err := CreateSprint(req, project, stories)
	if err != nil {
		t.Fatalf("CreateSprint() devolvió un error inesperado: %v", err)
	}
	if result.Sprint.DisplayID != "Sprint 1" {
		t.Errorf("DisplayID = %q, se esperaba %q", result.Sprint.DisplayID, "Sprint 1")
	}
	if result.Sprint.Status != SprintActive {
		t.Errorf("estado = %q, se esperaba %q", result.Sprint.Status, SprintActive)
	}
	if result.Sprint.Goal != req.Goal {
		t.Errorf("Goal = %q, se esperaba %q", result.Sprint.Goal, req.Goal)
	}
	if !reflect.DeepEqual(result.Sprint.StoryIDs, req.StoryIDs) {
		t.Errorf("StoryIDs = %v, se esperaban %v", result.Sprint.StoryIDs, req.StoryIDs)
	}
	for _, story := range stories {
		if story.Status != EnSprint {
			t.Errorf("estado de historia %d = %q, se esperaba %q", story.ID, story.Status, EnSprint)
		}
		if story.ActiveSprintID != result.Sprint.ID {
			t.Errorf("Sprint activo de historia %d = %d, se esperaba %d", story.ID, story.ActiveSprintID, result.Sprint.ID)
		}
	}
}

func TestServiceCreatesSprintWithoutStoriesAndReturnsWarning(t *testing.T) {
	store := newMemorySprintStore()
	project := Project{ID: 1, Name: "Proyecto sin historias", Status: "Activo"}
	store.addProject(project)

	result, err := NewService(store).CreateSprint(
		context.Background(),
		validCreationRequest(nil),
		project,
		nil,
	)
	if err != nil {
		t.Fatalf("la creación válida sin historias devolvió error: %v", err)
	}
	if result.Sprint.Status != SprintActive {
		t.Errorf("estado = %q, se esperaba %q", result.Sprint.Status, SprintActive)
	}
	if len(result.Sprint.StoryIDs) != 0 {
		t.Errorf("cantidad de historias = %d, se esperaba 0", len(result.Sprint.StoryIDs))
	}
	if result.Warning == "" {
		t.Fatal("la creación exitosa sin historias debe devolver una advertencia")
	}
}

func TestCreateSprint_RejectsEmptyOrWhitespaceGoal(t *testing.T) {
	tests := []struct {
		name string
		goal string
	}{
		{name: "vacío", goal: ""},
		{name: "solo espacios", goal: "   \t\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validCreationRequest([]int64{101})
			req.Goal = tt.goal
			project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
			stories := []UserStory{{ID: 101, ProjectID: 1, Status: Pendiente}}

			_, err := CreateSprint(req, project, stories)
			if !errors.Is(err, ErrGoalRequired) {
				t.Fatalf("CreateSprint() error = %v, se esperaba ErrGoalRequired", err)
			}
		})
	}
}

func TestCreateSprintRejectsInvalidPeriods(t *testing.T) {
	start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		end  time.Time
	}{
		{
			name: "fin anterior al inicio",
			end:  start.AddDate(0, 0, -1),
		},
		{
			name: "fin igual al inicio",
			end:  start,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validCreationRequest([]int64{101})
			request.StartDate = start
			request.EndDate = tt.end
			project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
			stories := []UserStory{{ID: 101, ProjectID: 1, Status: Pendiente}}
			before := stories[0]

			_, err := CreateSprint(request, project, stories)
			if !errors.Is(err, ErrInvalidSprintPeriod) {
				t.Fatalf("CreateSprint() error = %v, se esperaba ErrInvalidSprintPeriod", err)
			}
			if stories[0] != before {
				t.Fatalf("la historia cambió ante un período inválido: antes=%+v después=%+v", before, stories[0])
			}
			if request.EndDate != tt.end {
				t.Fatalf("la solicitud cambió su fecha de fin: got=%v want=%v", request.EndDate, tt.end)
			}
		})
	}
}

func TestServiceInvalidPeriodDoesNotCallStoreOrConsumeSprintNumber(t *testing.T) {
	memoryStore := newMemorySprintStore()
	project := Project{ID: 1, Status: "Activo"}
	memoryStore.addProject(project)
	memoryStore.addStory(UserStory{ID: 801, ProjectID: project.ID, Status: Pendiente})
	store := &countingSprintStore{SprintStore: memoryStore}
	service := NewService(store)
	stories := []UserStory{{ID: 801, ProjectID: project.ID, Status: Pendiente}}

	request := validCreationRequest([]int64{801})
	request.EndDate = request.StartDate
	_, err := service.CreateSprint(context.Background(), request, project, stories)
	if !errors.Is(err, ErrInvalidSprintPeriod) {
		t.Fatalf("CreateSprint() error = %v, se esperaba ErrInvalidSprintPeriod", err)
	}
	if store.createCalls != 0 {
		t.Fatalf("CreateSprint del store llamado %d veces, se esperaba 0", store.createCalls)
	}
	if stories[0].Status != Pendiente || stories[0].ActiveSprintID != 0 {
		t.Fatalf("la historia cambió ante el período inválido: %+v", stories[0])
	}

	request.EndDate = request.StartDate.AddDate(0, 0, 14)
	result, err := service.CreateSprint(context.Background(), request, project, stories)
	if err != nil {
		t.Fatalf("la creación válida posterior devolvió error: %v", err)
	}
	if result.Sprint.DisplayID != "Sprint 1" {
		t.Errorf("DisplayID posterior al rechazo = %q, se esperaba %q", result.Sprint.DisplayID, "Sprint 1")
	}
	if store.createCalls != 1 {
		t.Errorf("CreateSprint del store llamado %d veces, se esperaba 1", store.createCalls)
	}
}

func TestSprintCreationRequestDoesNotExposeGeneratedIdentifier(t *testing.T) {
	requestType := reflect.TypeOf(SprintCreationRequest{})
	for _, forbiddenField := range []string{"ID", "Number", "DisplayID"} {
		if _, exists := requestType.FieldByName(forbiddenField); exists {
			t.Errorf("SprintCreationRequest expone el campo generado %q", forbiddenField)
		}
	}
}

func TestCreateSprint_RejectsStoryAlreadyAssignedToASprint(t *testing.T) {
	req := validCreationRequest([]int64{201})
	project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
	stories := []UserStory{{
		ID:             201,
		ProjectID:      1,
		Status:         EnSprint,
		ActiveSprintID: 99,
	}}

	_, err := CreateSprint(req, project, stories)
	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("CreateSprint() error = %v, se esperaba ErrActiveSprintConflict", err)
	}
	if stories[0].Status != EnSprint || stories[0].ActiveSprintID != 99 {
		t.Fatalf("el rechazo alteró la asignación existente: %+v", stories[0])
	}
}

func TestCreateSprint_AllowsStoryMoveOnlyAfterRemovingCurrentAssignment(t *testing.T) {
	project := Project{ID: 1, Name: "Proyecto A", Status: "Activo"}
	story := UserStory{ID: 401, ProjectID: 1, Status: EnSprint, ActiveSprintID: 11}
	before := story

	_, err := CreateSprint(validCreationRequest([]int64{story.ID}), project, []UserStory{story})
	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("asignación directa = %v, se esperaba ErrActiveSprintConflict", err)
	}
	if story != before {
		t.Fatalf("la asignación directa modificó la historia: antes=%+v después=%+v", before, story)
	}

	if err := RemoveStoryFromSprint(&story, 11); err != nil {
		t.Fatalf("RemoveStoryFromSprint() devolvió error: %v", err)
	}
	if story.Status != Pendiente || story.ActiveSprintID != 0 {
		t.Fatalf("la historia no volvió a Pendiente al quitarla: %+v", story)
	}

	storiesAfterRemoval := []UserStory{story}
	result, err := CreateSprint(validCreationRequest([]int64{story.ID}), project, storiesAfterRemoval)
	if err != nil {
		t.Fatalf("la creación después de quitar la historia falló: %v", err)
	}
	if !reflect.DeepEqual(result.Sprint.StoryIDs, []int64{story.ID}) {
		t.Errorf("StoryIDs = %v, se esperaba [%d]", result.Sprint.StoryIDs, story.ID)
	}
	if storiesAfterRemoval[0].Status != EnSprint {
		t.Errorf("estado después de reasignar = %q, se esperaba %q", storiesAfterRemoval[0].Status, EnSprint)
	}
	if storiesAfterRemoval[0].ActiveSprintID != result.Sprint.ID {
		t.Errorf("Sprint activo después de reasignar = %d, se esperaba %d", storiesAfterRemoval[0].ActiveSprintID, result.Sprint.ID)
	}
}

// "Mover una historia entre Sprints" ya no aplica con un solo Sprint activo (FIX-HU-04):
// lo reemplaza TestServicio_QuitarYVolverAAsignarEnElSprintActivo en active_sprint_test.go.

func TestServicePersistsBeforeMutatingStoriesAndUsesPersistedSprintID(t *testing.T) {
	store := newMemorySprintStore()
	project := Project{ID: 1, Status: "Activo", NextSprintNumber: 5}
	store.addProject(project)
	store.addStory(UserStory{ID: 601, ProjectID: project.ID, Status: Pendiente})
	store.nextID = 30
	stories := []UserStory{{ID: 601, ProjectID: project.ID, Status: Pendiente}}

	result, err := NewService(store).CreateSprint(
		context.Background(),
		validCreationRequest([]int64{601}),
		project,
		stories,
	)
	if err != nil {
		t.Fatalf("Service.CreateSprint() devolvió error: %v", err)
	}
	if result.Sprint.ID != 30 || result.Sprint.DisplayID != "Sprint 5" {
		t.Fatalf("Sprint persistido inesperado: %+v", result.Sprint)
	}
	if stories[0].Status != EnSprint || stories[0].ActiveSprintID != result.Sprint.ID {
		t.Fatalf("la historia no recibió el ID técnico persistido: %+v", stories[0])
	}
}

func TestServiceDoesNotMutateStoriesWhenPersistenceFails(t *testing.T) {
	store := newMemorySprintStore()
	project := Project{ID: 1, Status: "Activo"}
	store.addProject(project)
	store.addStory(UserStory{ID: 701, ProjectID: project.ID, Status: EnSprint, ActiveSprintID: 99})
	stories := []UserStory{{ID: 701, ProjectID: project.ID, Status: Pendiente}}
	before := stories[0]

	_, err := NewService(store).CreateSprint(
		context.Background(),
		validCreationRequest([]int64{701}),
		project,
		stories,
	)
	if !errors.Is(err, ErrActiveSprintConflict) {
		t.Fatalf("Service.CreateSprint() error = %v, se esperaba conflicto persistido", err)
	}
	if stories[0] != before {
		t.Fatalf("la historia cambió aunque falló la persistencia: antes=%+v después=%+v", before, stories[0])
	}
}

func validCreationRequest(storyIDs []int64) SprintCreationRequest {
	return SprintCreationRequest{
		ProjectID: 1,
		Goal:      "Entregar la versión beta",
		StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		StoryIDs:  storyIDs,
	}
}

type countingSprintStore struct {
	SprintStore
	createCalls int
}

func (s *countingSprintStore) CreateSprint(ctx context.Context, sprint Sprint) (Sprint, error) {
	s.createCalls++
	return s.SprintStore.CreateSprint(ctx, sprint)
}
