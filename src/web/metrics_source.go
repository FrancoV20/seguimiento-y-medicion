package web

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
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

// hu07MetricsSource es el adaptador sobre el servicio real de HU-07.
type hu07MetricsSource struct {
	service *metrics.Service
}

// NewHU07MetricsSource crea el adaptador sobre el servicio de HU-07. Lee con el almacenamiento
// PostgreSQL de HU-07 y no guarda nada: la pantalla no modifica datos (SC-006).
func NewHU07MetricsSource(db *sql.DB) MetricsSource {
	store := readOnlyMetricsStore{reader: metrics.NewPostgreSQLMetricsStore(db)}
	return &hu07MetricsSource{service: metrics.NewService(store)}
}

// Metrics pide a HU-07 las métricas del Sprint y traduce su resultado, sin calcular ni redondear.
// HU-07 no recibe el proyecto: el handler solo pregunta por Sprints que el catálogo devolvió para
// el proyecto activo.
func (s *hu07MetricsSource) Metrics(_ context.Context, _ int64, sprintID int64) (SprintMetrics, error) {
	result, err := s.service.CalculateMetrics(sprintID)
	if err != nil {
		return SprintMetrics{}, fmt.Errorf("métricas de HU-07 del Sprint %d: %w", sprintID, err)
	}
	return SprintMetrics{
		Velocity:     result.Velocity,
		Deviation:    result.DeviationPercentage,
		PartialUsed:  result.CalculationUsedCount,
		PartialTotal: result.CalculationTotalCount,
		Partial:      result.IsPartial,
		PartialAlert: result.Warning,
	}, nil
}

// readOnlyMetricsStore lee los datos del Sprint con el store de HU-07 y descarta el guardado de la
// copia en sprint_metrics que hace CalculateMetrics (decisión de HU-25: las métricas no se guardan).
type readOnlyMetricsStore struct {
	reader *metrics.PostgreSQLMetricsStore
}

func (s readOnlyMetricsStore) LoadSprintData(sprintID int64) (metrics.SprintData, error) {
	return s.reader.LoadSprintData(sprintID)
}

func (readOnlyMetricsStore) SaveMetrics(metrics.SprintMetrics) error { return nil }
