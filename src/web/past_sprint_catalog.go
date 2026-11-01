package web

import (
	"context"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// Tipos de "Sprints anteriores" (HU-16, T008). Las historias salen del registro de cierre de
// HU-15 (sprint_story_results) y la velocidad de HU-07: la pantalla no recalcula nada.

// PastSprintSummary es un Sprint finalizado de la lista.
type PastSprintSummary struct {
	ID        int64
	Number    int
	Goal      string
	StartDate time.Time
	EndDate   time.Time
}

// PastSprintStory es una fila del registro de cierre de HU-15.
type PastSprintStory struct {
	StoryID     int64
	Title       string // título actual de la historia
	StoryPoints *int   // al cerrar; nil = "Sin estimar"
	FinalStatus string // "En Sprint", "En progreso" o "Hecho"
}

// PastSprintDetail es un Sprint finalizado con su registro de cierre, ordenado por StoryID.
type PastSprintDetail struct {
	PastSprintSummary
	Stories []PastSprintStory // vacío si el Sprint no tiene registro
}

// PastSprintCatalog lee los Sprints finalizados del proyecto (solo lectura).
type PastSprintCatalog interface {
	// FinishedSprints devuelve los Sprints 'Finalizado' del proyecto, por número descendente.
	FinishedSprints(ctx context.Context, projectID int64) ([]PastSprintSummary, error)
	// SprintDetail devuelve NotFound si el Sprint no existe, es de otro proyecto o no está 'Finalizado'.
	SprintDetail(ctx context.Context, projectID, sprintID int64) (PastSprintDetail, error)
}

// PastSprintMetrics es lo que HU-16 usa de HU-07; *metrics.Service la cumple.
type PastSprintMetrics interface {
	CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
}

var _ PastSprintMetrics = (*metrics.Service)(nil)
