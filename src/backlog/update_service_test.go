package backlog

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// HU-13, T005–T008: estados, validación compartida y servicio de edición.

func validUpdateRequest() UpdateStoryRequest {
	return UpdateStoryRequest{
		ProjectID:          1,
		StoryID:            1,
		Title:              "  Inicio de sesión  ",
		Description:        "  Como usuario quiero ingresar  ",
		Priority:           PriorityHigh,
		AcceptanceCriteria: []string{"  Acepta email  ", "   ", "Rechaza clave vacía"},
		StoryPoints:        intPtr(8),
	}
}

// storeWithLogin devuelve un almacén con «Login» (ID 1) del proyecto 1, "En Sprint" y con 4 horas.
func storeWithLogin() *InMemoryBacklogStore {
	store := NewInMemoryBacklogStore(1)
	hours := 4.0
	store.Stories = []UserStory{{
		ID: 1, ProjectID: 1, Title: "Login", Description: "d", Priority: PriorityMedium,
		AcceptanceCriteria: []AcceptanceCriterion{{Position: 1, Content: "Acepta email"}},
		StoryPoints:        intPtr(5), EstimatedHours: &hours, Status: StatusInSprint,
	}}
	return store
}

// T005
func TestStoryStatuses(t *testing.T) {
	want := []StoryStatus{"Pendiente", "En Sprint", "En progreso", "Hecho"}
	if got := StoryStatuses(); !reflect.DeepEqual(got, want) {
		t.Errorf("StoryStatuses() = %v; se esperaba %v", got, want)
	}
}

// T006
func TestValidateStoryFields(t *testing.T) {
	t.Run("recorta, descarta criterios vacíos y numera desde 1", func(t *testing.T) {
		fields, err := validateStoryFields("  Título  ", "  Descripción  ", PriorityLow,
			[]string{"", "  Primero  ", "   ", "Segundo"}, nil)
		if err != nil {
			t.Fatalf("no se esperaba error: %v", err)
		}
		if fields.Title != "Título" || fields.Description != "Descripción" || fields.Priority != PriorityLow {
			t.Errorf("datos = %q/%q/%q; se esperaban recortados", fields.Title, fields.Description, fields.Priority)
		}
		want := []AcceptanceCriterion{{Position: 1, Content: "Primero"}, {Position: 2, Content: "Segundo"}}
		if !reflect.DeepEqual(fields.Criteria, want) {
			t.Errorf("criterios = %+v; se esperaba %+v", fields.Criteria, want)
		}
		if fields.StoryPoints != nil {
			t.Errorf("Story Points = %v; se esperaba nil (sin estimar)", *fields.StoryPoints)
		}
	})

	// Cada caso tiene varios datos inválidos: el error es el primero en el orden título,
	// descripción, prioridad, criterios y Story Points.
	cases := []struct {
		name        string
		title, desc string
		priority    Priority
		criteria    []string
		points      *int
		want        error
	}{
		{"título primero", " ", " ", "Urgente", nil, intPtr(4), ErrTitleRequired},
		{"descripción después", "T", " ", "Urgente", nil, intPtr(4), ErrDescriptionRequired},
		{"prioridad después", "T", "D", "Urgente", nil, intPtr(4), ErrInvalidPriority},
		{"criterios después", "T", "D", PriorityHigh, []string{" "}, intPtr(4), ErrAcceptanceCriteriaRequired},
		{"Story Points al final", "T", "D", PriorityHigh, []string{"C"}, intPtr(4), ErrInvalidStoryPoints},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := validateStoryFields(tc.title, tc.desc, tc.priority, tc.criteria, tc.points); !errors.Is(err, tc.want) {
				t.Errorf("error = %v; se esperaba %v", err, tc.want)
			}
		})
	}
}

// T007
func TestUpdateStory_Validaciones(t *testing.T) {
	cases := []struct {
		name   string
		change func(*UpdateStoryRequest)
		want   error
	}{
		{"proyecto 0", func(r *UpdateStoryRequest) { r.ProjectID = 0 }, ErrStoryNotFound},
		{"historia 0", func(r *UpdateStoryRequest) { r.StoryID = 0 }, ErrStoryNotFound},
		{"título vacío", func(r *UpdateStoryRequest) { r.Title = "" }, ErrTitleRequired},
		{"descripción con solo espacios", func(r *UpdateStoryRequest) { r.Description = "   " }, ErrDescriptionRequired},
		{"prioridad Urgente", func(r *UpdateStoryRequest) { r.Priority = "Urgente" }, ErrInvalidPriority},
		{"criterios vacíos", func(r *UpdateStoryRequest) { r.AcceptanceCriteria = []string{"", "  "} }, ErrAcceptanceCriteriaRequired},
		{"4 Story Points", func(r *UpdateStoryRequest) { r.StoryPoints = intPtr(4) }, ErrInvalidStoryPoints},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := storeWithLogin()
			req := validUpdateRequest()
			tc.change(&req)

			if _, err := NewBacklogService(store).UpdateStory(context.Background(), req); !errors.Is(err, tc.want) {
				t.Errorf("error = %v; se esperaba %v", err, tc.want)
			}
			if store.UpdateCalls != 0 {
				t.Errorf("se llamó al almacén %d veces; no se debía llamar", store.UpdateCalls)
			}
			if store.Stories[0].Title != "Login" {
				t.Errorf("la historia cambió: título %q", store.Stories[0].Title)
			}
		})
	}
}

// T008
func TestUpdateStory_DelegaYPropaga(t *testing.T) {
	t.Run("envío válido: llega recortado y con los criterios numerados", func(t *testing.T) {
		store := storeWithLogin()
		result, err := NewBacklogService(store).UpdateStory(context.Background(), validUpdateRequest())
		if err != nil {
			t.Fatalf("no se esperaba error: %v", err)
		}
		if store.UpdateCalls != 1 {
			t.Fatalf("UpdateCalls = %d; se esperaba 1", store.UpdateCalls)
		}
		got := store.Stories[0]
		wantCriteria := []AcceptanceCriterion{{Position: 1, Content: "Acepta email"}, {Position: 2, Content: "Rechaza clave vacía"}}
		if got.Title != "Inicio de sesión" || got.Description != "Como usuario quiero ingresar" || got.Priority != PriorityHigh {
			t.Errorf("datos guardados = %q/%q/%q", got.Title, got.Description, got.Priority)
		}
		if !reflect.DeepEqual(got.AcceptanceCriteria, wantCriteria) {
			t.Errorf("criterios guardados = %+v; se esperaba %+v", got.AcceptanceCriteria, wantCriteria)
		}
		if got.StoryPoints == nil || *got.StoryPoints != 8 {
			t.Errorf("Story Points guardados = %v; se esperaba 8", got.StoryPoints)
		}
		if got.Status != StatusInSprint || got.EstimatedHours == nil || *got.EstimatedHours != 4 {
			t.Errorf("cambiaron el estado o las horas: %q/%v", got.Status, got.EstimatedHours)
		}
		if result.Warning != "" {
			t.Errorf("Warning = %q; con Story Points no se esperaba advertencia", result.Warning)
		}
	})

	t.Run("sin Story Points: advertencia de estimación pendiente", func(t *testing.T) {
		store := storeWithLogin()
		req := validUpdateRequest()
		req.StoryPoints = nil
		result, err := NewBacklogService(store).UpdateStory(context.Background(), req)
		if err != nil {
			t.Fatalf("no se esperaba error: %v", err)
		}
		if result.Warning != WarnEstimationPending {
			t.Errorf("Warning = %q; se esperaba %q", result.Warning, WarnEstimationPending)
		}
		if store.Stories[0].StoryPoints != nil {
			t.Errorf("Story Points guardados = %v; se esperaba nil", *store.Stories[0].StoryPoints)
		}
	})

	for _, domainErr := range []error{ErrStoryNotFound, ErrStoryInFinishedSprint} {
		t.Run("propaga "+domainErr.Error(), func(t *testing.T) {
			store := storeWithLogin()
			store.UpdateErr = domainErr
			if _, err := NewBacklogService(store).UpdateStory(context.Background(), validUpdateRequest()); !errors.Is(err, domainErr) {
				t.Errorf("error = %v; se esperaba %v", err, domainErr)
			}
		})
	}

	t.Run("falla del almacén: ErrPersistence", func(t *testing.T) {
		store := storeWithLogin()
		store.FailOnUpdate = true
		if _, err := NewBacklogService(store).UpdateStory(context.Background(), validUpdateRequest()); !errors.Is(err, ErrPersistence) {
			t.Errorf("error = %v; se esperaba ErrPersistence", err)
		}
	})
}
