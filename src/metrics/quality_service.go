package metrics

import (
	"context"
	"errors"
	"fmt"
)

// QualityService calcula la calidad y el avance de un Sprint finalizado (HU-10, T009). No guarda nada.
type QualityService struct {
	store QualityStore
}

// NewQualityService crea el servicio con el almacén indicado.
func NewQualityService(store QualityStore) *QualityService {
	return &QualityService{store: store}
}

// SprintQuality devuelve las métricas de calidad del Sprint del proyecto. Devuelve ErrSprintNotFound
// si el Sprint no existe en el proyecto, ErrSprintNotFinished si todavía no está "Finalizado" y, ante
// cualquier otra falla de lectura, un error que envuelve la causa.
func (s *QualityService) SprintQuality(ctx context.Context, projectID, sprintID int64) (SprintQuality, error) {
	if s == nil || s.store == nil {
		return SprintQuality{}, errors.New("quality store is required")
	}
	if projectID <= 0 || sprintID <= 0 {
		return SprintQuality{}, ErrSprintNotFound
	}

	data, err := s.store.LoadSprintQuality(ctx, projectID, sprintID)
	if errors.Is(err, ErrSprintNotFound) {
		return SprintQuality{}, ErrSprintNotFound
	}
	if err != nil {
		return SprintQuality{}, fmt.Errorf("load sprint quality %d: %w", sprintID, err)
	}
	if data.Status != "Finalizado" {
		return SprintQuality{}, fmt.Errorf("%w: %d", ErrSprintNotFinished, sprintID)
	}
	return calculateSprintQuality(data), nil
}
