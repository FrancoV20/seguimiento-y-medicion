package metrics

import "context"

// QualityStore es el puerto de lectura de los datos de calidad de un Sprint (HU-10, T009).
type QualityStore interface {
	// LoadSprintQuality devuelve ErrSprintNotFound si el Sprint no existe en el proyecto.
	// Si el Sprint no está "Finalizado", devuelve solo SprintID y Status.
	LoadSprintQuality(ctx context.Context, projectID, sprintID int64) (QualityData, error)
}
