package web

import (
	"context"
	"time"
)

// ProjectStatusCounts son las historias del proyecto por estado (HU-12).
type ProjectStatusCounts struct {
	Pending    int // "Pendiente"
	InSprint   int // "En Sprint"
	InProgress int // "En progreso"
	Done       int // "Hecho"
}

// Total es la suma de las cuatro cantidades.
func (c ProjectStatusCounts) Total() int {
	return c.Pending + c.InSprint + c.InProgress + c.Done
}

// ProjectStatusSummary son los datos propios de la pantalla "Estado del proyecto".
type ProjectStatusSummary struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Counts    ProjectStatusCounts
}

// ProjectStatusCatalog lee el proyecto y sus cantidades por estado. Solo lectura.
type ProjectStatusCatalog interface {
	// StatusSummary devuelve NotFound si el proyecto no existe.
	StatusSummary(ctx context.Context, projectID int64) (ProjectStatusSummary, error)
}
