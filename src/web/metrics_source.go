package web

import (
	"context"
	"database/sql"
)

// SprintMetrics son las métricas de un Sprint finalizado tal como las calcula HU-07 (HU-25, T009).
// La capa web no las recalcula ni las redondea.
type SprintMetrics struct {
	Velocity     int    // Story Points de las historias "Hecho" (Velocity de HU-07)
	Deviation    string // DeviationPercentage de HU-07: "20.00", "-8.50", "0.00" o "No disponible"
	PartialUsed  int    // X: CalculationUsedCount de HU-07
	PartialTotal int    // Y: CalculationTotalCount de HU-07
	Partial      bool   // IsPartial de HU-07
	PartialAlert string // Warning de HU-07: "Cálculo parcial: basado en X de Y historias"
}

// MetricsSource entrega las métricas de un Sprint finalizado del proyecto.
type MetricsSource interface {
	Metrics(ctx context.Context, projectID, sprintID int64) (SprintMetrics, error)
}

// hu07MetricsSource es el adaptador sobre el servicio de HU-07.
type hu07MetricsSource struct {
	db *sql.DB
}

// NewHU07MetricsSource crea el adaptador sobre el servicio de HU-07 y la base abierta.
func NewHU07MetricsSource(db *sql.DB) MetricsSource {
	return &hu07MetricsSource{db: db}
}

// Metrics pide a HU-07 las métricas del Sprint.
func (s *hu07MetricsSource) Metrics(ctx context.Context, projectID, sprintID int64) (SprintMetrics, error) {
	return SprintMetrics{}, errMetricsNotImplemented
}
