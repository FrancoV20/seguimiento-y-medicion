package web

import (
	"context"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-10 (T026): datos de lectura de la pantalla "Calidad y avance" y sus puertos.

// QualitySprintOption es un Sprint finalizado de la lista.
type QualitySprintOption struct {
	ID     int64
	Number int
}

// QualitySprintList son los Sprints del proyecto que importan a la pantalla.
type QualitySprintList struct {
	Finished     []QualitySprintOption // del número mayor al menor
	ActiveNumber int                   // 0 = no hay Sprint activo
}

// QualityCatalog lee la lista de Sprints del proyecto.
type QualityCatalog interface {
	QualitySprints(ctx context.Context, projectID int64) (QualitySprintList, error)
}

// QualityReader calcula la calidad y el avance de un Sprint finalizado; lo implementa
// *metrics.QualityService.
type QualityReader interface {
	SprintQuality(ctx context.Context, projectID, sprintID int64) (metrics.SprintQuality, error)
}

var _ QualityReader = (*metrics.QualityService)(nil)
