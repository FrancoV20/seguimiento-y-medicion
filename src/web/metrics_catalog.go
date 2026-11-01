package web

import (
	"context"
	"time"
)

// FinishedSprint es una fila de la lista de Sprints finalizados de la pantalla "Métricas del
// Sprint" (HU-25, T005).
type FinishedSprint struct {
	ID        int64
	Number    int64
	Goal      string
	StartDate time.Time
	EndDate   time.Time
}

// MetricsCatalog lee los Sprints del proyecto activo para la pantalla "Métricas del Sprint".
// Solo lectura: no modifica Sprints ni historias. Todas las lecturas se filtran por projectID.
type MetricsCatalog interface {
	// FinishedSprints devuelve los Sprints en estado "Finalizado" del proyecto, del más reciente
	// (número mayor) al más antiguo. Sin Sprints finalizados devuelve una lista vacía, sin error.
	FinishedSprints(ctx context.Context, projectID int64) ([]FinishedSprint, error)

	// ActiveSprintNumber devuelve el número del Sprint activo del proyecto; found es false si no
	// tiene ninguno.
	ActiveSprintNumber(ctx context.Context, projectID int64) (number int64, found bool, err error)
}
