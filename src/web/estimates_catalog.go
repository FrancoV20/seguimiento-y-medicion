package web

import (
	"context"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// EstimateLockedReason es el motivo que se muestra cuando una historia está en un Sprint
// finalizado y sus horas estimadas no se pueden modificar (HU-17).
const EstimateLockedReason = "Sprint finalizado"

// EstimateRow presenta una historia en la pantalla "Estimaciones" (HU-17, T007).
type EstimateRow struct {
	ID             int64
	Title          string
	Status         backlog.StoryStatus
	StoryPoints    *int     // nil: "Sin estimar"
	EstimatedHours *float64 // nil: "Sin estimar"
	ActualHours    float64  // suma de effort_logs; 0 si no hay registros
	Locked         bool     // la historia está en un Sprint finalizado
	LockedReason   string   // EstimateLockedReason cuando Locked
}

// EstimateCatalog expone la lectura de la pantalla "Estimaciones".
type EstimateCatalog interface {
	// ListEstimates devuelve las historias del proyecto por id descendente; sin historias no es un error.
	ListEstimates(ctx context.Context, projectID int64) ([]EstimateRow, error)
}
