package test

import (
	"errors"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// Escenarios BDD de HU-02 (T016, T024, T027).
// Cada test sigue la estructura Dado / Cuando / Entonces de la historia de usuario.

func historiaValida() backlog.CreateStoryRequest {
	sp := 5
	return backlog.CreateStoryRequest{
		ProjectID:          1,
		Title:              "Alta de proyecto",
		Description:        "Como usuario quiero crear un proyecto",
		Priority:           backlog.PriorityHigh,
		AcceptanceCriteria: []string{"Dado un proyecto válido, cuando lo creo, entonces se registra"},
		StoryPoints:        &sp,
	}
}

// Escenario: Registro de nueva historia (T016).
func TestEscenario_RegistroDeNuevaHistoria(t *testing.T) {
	// Dado que el Scrum Master seleccionó un proyecto activo
	store := backlog.NewInMemoryBacklogStore(1)
	service := backlog.NewBacklogService(store)

	// Cuando ingresa los datos obligatorios de una historia y guarda
	result, err := service.CreateStory(historiaValida())

	// Entonces la historia se añade al Product Backlog con estado "Pendiente"
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}
	if len(store.Stories) != 1 {
		t.Fatalf("se esperaba 1 historia en el backlog, hay %d", len(store.Stories))
	}
	if store.Stories[0].Status != backlog.StatusPending {
		t.Errorf("estado esperado %q, se obtuvo %q", backlog.StatusPending, store.Stories[0].Status)
	}
	if result.SuccessMessage != backlog.MsgStoryCreated {
		t.Errorf("se esperaba la confirmación %q", backlog.MsgStoryCreated)
	}
}

// Escenario: Criterios de aceptación obligatorios (T024, T027).
func TestEscenario_CriteriosDeAceptacionObligatorios(t *testing.T) {
	// Dado que se está redactando una nueva historia en un proyecto activo
	store := backlog.NewInMemoryBacklogStore(1)
	service := backlog.NewBacklogService(store)
	req := historiaValida()
	req.AcceptanceCriteria = nil

	// Cuando se intenta guardar sin especificar al menos un criterio de aceptación
	result, err := service.CreateStory(req)

	// Entonces el sistema solicita que se agreguen y la historia no se registra
	if !errors.Is(err, backlog.ErrAcceptanceCriteriaRequired) {
		t.Errorf("se esperaba ErrAcceptanceCriteriaRequired, se obtuvo: %v", err)
	}
	if result != nil {
		t.Errorf("la historia no debe mostrarse como creada")
	}
	if len(store.Stories) != 0 {
		t.Errorf("la historia no debe agregarse al backlog, hay %d", len(store.Stories))
	}
}
