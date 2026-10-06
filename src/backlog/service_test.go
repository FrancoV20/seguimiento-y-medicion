package backlog

import (
	"errors"
	"testing"
)

func validRequest() CreateStoryRequest {
	return CreateStoryRequest{
		ProjectID:          1,
		Title:              "  Alta de proyecto  ",
		Description:        "Como usuario quiero crear un proyecto",
		Priority:           PriorityHigh,
		AcceptanceCriteria: []string{"Dado un proyecto válido, cuando lo creo, entonces se registra"},
		StoryPoints:        intPtr(5),
	}
}

// T012: alta válida, asociación al proyecto y estado Pendiente.
func TestCreateStory_AltaValidaQuedaPendienteYAsociadaAlProyecto(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	service := NewBacklogService(store)

	result, err := service.CreateStory(validRequest())

	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}
	if len(store.Stories) != 1 {
		t.Fatalf("se esperaba 1 historia en el backlog, hay %d", len(store.Stories))
	}
	story := result.Story
	if story.Status != StatusPending {
		t.Errorf("estado esperado %q, se obtuvo %q", StatusPending, story.Status)
	}
	if story.ProjectID != 1 {
		t.Errorf("proyecto esperado 1, se obtuvo %d", story.ProjectID)
	}
	if story.Title != "Alta de proyecto" {
		t.Errorf("el título debe guardarse sin espacios sobrantes, se obtuvo %q", story.Title)
	}
	if result.SuccessMessage != MsgStoryCreated {
		t.Errorf("se esperaba la confirmación %q", MsgStoryCreated)
	}
	if result.Warning != "" {
		t.Errorf("con Story Points no debe haber alerta, se obtuvo %q", result.Warning)
	}
}

// T013: sin Story Points se permite el alta, con alerta y ausencia explícita.
func TestCreateStory_SinStoryPointsSeCreaConAlerta(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	req := validRequest()
	req.StoryPoints = nil

	result, err := NewBacklogService(store).CreateStory(req)

	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}
	if result.Story.StoryPoints != nil {
		t.Errorf("la historia debe conservar la ausencia de estimación")
	}
	if result.Warning != WarnEstimationPending {
		t.Errorf("se esperaba la alerta %q, se obtuvo %q", WarnEstimationPending, result.Warning)
	}
}

func TestCreateStory_StoryPointsInvalidosNoMutanElBacklog(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	req := validRequest()
	req.StoryPoints = intPtr(4)

	_, err := NewBacklogService(store).CreateStory(req)

	if !errors.Is(err, ErrInvalidStoryPoints) {
		t.Errorf("se esperaba ErrInvalidStoryPoints, se obtuvo: %v", err)
	}
	if store.CreateCalls != 0 {
		t.Errorf("no debe llamarse al store ante un error de validación")
	}
}

func TestCreateStory_ProyectoInactivoOInexistenteSeRechaza(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	req := validRequest()
	req.ProjectID = 99

	_, err := NewBacklogService(store).CreateStory(req)

	if !errors.Is(err, ErrProjectNotActive) {
		t.Errorf("se esperaba ErrProjectNotActive, se obtuvo: %v", err)
	}
	if len(store.Stories) != 0 {
		t.Errorf("no debe agregarse ninguna historia")
	}
}

func TestCreateStory_CamposObligatoriosVaciosSeRechazan(t *testing.T) {
	casos := []struct {
		nombre    string
		modificar func(*CreateStoryRequest)
		esperado  error
	}{
		{"título vacío", func(r *CreateStoryRequest) { r.Title = "   " }, ErrTitleRequired},
		{"descripción vacía", func(r *CreateStoryRequest) { r.Description = "" }, ErrDescriptionRequired},
		{"prioridad inválida", func(r *CreateStoryRequest) { r.Priority = "Urgente" }, ErrInvalidPriority},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			store := NewInMemoryBacklogStore(1)
			req := validRequest()
			c.modificar(&req)

			_, err := NewBacklogService(store).CreateStory(req)

			if !errors.Is(err, c.esperado) {
				t.Errorf("se esperaba %v, se obtuvo: %v", c.esperado, err)
			}
			if len(store.Stories) != 0 {
				t.Errorf("no debe agregarse ninguna historia")
			}
		})
	}
}

func TestCreateStory_FallaDePersistenciaNoInformaExito(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	store.FailOnCreate = true

	result, err := NewBacklogService(store).CreateStory(validRequest())

	if !errors.Is(err, ErrPersistence) {
		t.Errorf("se esperaba ErrPersistence, se obtuvo: %v", err)
	}
	if result != nil {
		t.Errorf("ante una falla de persistencia no debe devolverse resultado exitoso")
	}
}
