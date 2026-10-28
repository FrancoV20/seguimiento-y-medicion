package backlog

import (
	"errors"
	"testing"
)

func floatPtr(v float64) *float64 { return &v }

func TestCreateStory_ConHorasEstimadasLasConserva(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	req := validRequest()
	req.EstimatedHours = floatPtr(8.5)

	result, err := NewBacklogService(store).CreateStory(req)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}
	if got := result.Story.EstimatedHours; got == nil || *got != 8.5 {
		t.Errorf("se esperaban 8.5 horas estimadas, se obtuvo %v", got)
	}
}

func TestCreateStory_SinHorasEstimadasSePermite(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	req := validRequest()
	req.EstimatedHours = nil

	result, err := NewBacklogService(store).CreateStory(req)
	if err != nil {
		t.Fatalf("las horas estimadas son opcionales, se obtuvo: %v", err)
	}
	if result.Story.EstimatedHours != nil {
		t.Errorf("la ausencia de horas estimadas debe conservarse, se obtuvo %v", *result.Story.EstimatedHours)
	}
}

func TestCreateStory_HorasEstimadasInvalidasSeRechazan(t *testing.T) {
	for _, h := range []float64{0, -1, 10000} {
		store := NewInMemoryBacklogStore(1)
		req := validRequest()
		req.EstimatedHours = floatPtr(h)

		result, err := NewBacklogService(store).CreateStory(req)
		if !errors.Is(err, ErrInvalidEstimatedHours) {
			t.Errorf("%v horas: se esperaba ErrInvalidEstimatedHours, se obtuvo: %v", h, err)
		}
		if result != nil || store.CreateCalls != 0 {
			t.Errorf("%v horas: la historia no debe registrarse", h)
		}
	}
}

func TestCreateStory_HorasEstimadasConMasDeDosDecimalesSeRechazan(t *testing.T) {
	for _, h := range []float64{1.555, 9999.995} {
		store := NewInMemoryBacklogStore(1)
		req := validRequest()
		req.EstimatedHours = floatPtr(h)

		result, err := NewBacklogService(store).CreateStory(req)
		if !errors.Is(err, ErrInvalidEstimatedHours) {
			t.Errorf("%v horas: se esperaba ErrInvalidEstimatedHours, se obtuvo: %v", h, err)
		}
		if result != nil || store.CreateCalls != 0 {
			t.Errorf("%v horas: la historia no debe registrarse", h)
		}
	}
}

func TestCreateStory_HorasEstimadasHasta9999ConDosDecimalesSePermiten(t *testing.T) {
	store := NewInMemoryBacklogStore(1)
	req := validRequest()
	req.EstimatedHours = floatPtr(9999.99)

	result, err := NewBacklogService(store).CreateStory(req)
	if err != nil {
		t.Fatalf("9999,99 horas son válidas, se obtuvo: %v", err)
	}
	if result == nil || result.Story.EstimatedHours == nil || *result.Story.EstimatedHours != 9999.99 {
		t.Errorf("se esperaban 9999,99 horas conservadas, se obtuvo: %+v", result)
	}
}
