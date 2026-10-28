package web

import (
	"context"
	"errors"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// EstimateActions es la operación de HU-17 que usa la pantalla; la satisface *backlog.EstimateService.
type EstimateActions interface {
	UpdateEstimatedHours(ctx context.Context, projectID, storyID int64, input string) (backlog.EstimateResult, error)
}

var _ EstimateActions = (*backlog.EstimateService)(nil)

// RegisterEstimates registra la sección "Estimaciones" (HU-17). Los handlers GET y POST
// /estimaciones todavía no están implementados (T017): la sección figura en el menú pero sus rutas
// responden 404.
func RegisterEstimates(s *Server, actions EstimateActions, catalog EstimateCatalog) error {
	if s == nil {
		return errors.New("web server is required")
	}
	if actions == nil {
		return errors.New("estimate service is required")
	}
	if catalog == nil {
		return errors.New("estimate catalog is required")
	}
	return s.Register(Section{Name: "Estimaciones", Path: "/estimaciones", RequiresProject: true}, Routes{})
}
