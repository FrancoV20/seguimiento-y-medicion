package test

import (
	"math/big"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

func TestMetricsBDD_SprintVelocityAndEffortDeviation(t *testing.T) {
	t.Run("Escenario 1: cálculo de desviación de esfuerzo y velocidad", func(t *testing.T) {
		t.Run("Given un Sprint finalizado con historias completas y pendientes", func(t *testing.T) {
			store := &metrics.TestMetricsStore{
				SprintData: metrics.SprintData{
					ID:     "bdd-sprint-1",
					Status: "Finalizado",
					Stories: []metrics.CompletedStory{
						{
							ID:             "bdd-story-1",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(5),
							EstimatedHours: bddRat(t, "8"),
							ActualHours:    bddRat(t, "10"),
						},
						{
							ID:             "bdd-story-2",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(3),
							EstimatedHours: bddRat(t, "12"),
							ActualHours:    bddRat(t, "15"),
						},
						{
							ID:             "bdd-story-pending",
							Status:         "Pendiente",
							StoryPoints:    bddIntPointer(13),
							EstimatedHours: bddRat(t, "100"),
							ActualHours:    bddRat(t, "200"),
						},
					},
				},
			}

			t.Run("When el sistema calcula las métricas", func(t *testing.T) {
				got, err := metrics.NewService(store).CalculateMetrics("bdd-sprint-1")

				t.Run("Then muestra velocidad y desviación calculadas", func(t *testing.T) {
					if err != nil {
						t.Fatalf("CalculateMetrics() error = %v", err)
					}
					if got.Velocity != 8 {
						t.Errorf("velocidad = %d, want 8 Story Points", got.Velocity)
					}
					assertBDDRatEqual(t, "esfuerzo estimado", got.EstimatedHoursTotal, "20")
					assertBDDRatEqual(t, "esfuerzo real", got.ActualHoursTotal, "25")
					if got.DeviationPercentage != "25.00" {
						t.Errorf("desviación = %q, want %q", got.DeviationPercentage, "25.00")
					}
				})
			})
		})
	})
}

func TestMetricsBDD_SprintWithoutEstimatedEffort(t *testing.T) {
	t.Run("Escenario 2: Sprint sin esfuerzo estimado", func(t *testing.T) {
		t.Run("Given un Sprint finalizado sin horas estimadas", func(t *testing.T) {
			store := &metrics.TestMetricsStore{
				SprintData: metrics.SprintData{
					ID:     "bdd-sprint-no-estimate",
					Status: "Finalizado",
					Stories: []metrics.CompletedStory{
						{
							ID:             "bdd-no-estimate-1",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(5),
							EstimatedHours: nil,
							ActualHours:    bddRat(t, "7"),
						},
						{
							ID:             "bdd-no-estimate-2",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(3),
							EstimatedHours: nil,
							ActualHours:    bddRat(t, "4"),
						},
					},
				},
			}

			t.Run("When el sistema calcula la desviación", func(t *testing.T) {
				got, err := metrics.NewService(store).CalculateMetrics("bdd-sprint-no-estimate")

				t.Run("Then indica No disponible sin error de división por cero", func(t *testing.T) {
					if err != nil {
						t.Fatalf("CalculateMetrics() error = %v", err)
					}
					if got.DeviationPercentage != "No disponible" {
						t.Errorf("desviación = %q, want %q", got.DeviationPercentage, "No disponible")
					}
				})
			})
		})
	})
}

func TestMetricsBDD_PartialEffortCalculation(t *testing.T) {
	t.Run("Escenario 3: cálculo parcial de desviación por datos incompletos", func(t *testing.T) {
		t.Run("Given un Sprint con datos completos solo en una de tres historias", func(t *testing.T) {
			store := &metrics.TestMetricsStore{
				SprintData: metrics.SprintData{
					ID:     "bdd-sprint-partial",
					Status: "Finalizado",
					Stories: []metrics.CompletedStory{
						{
							ID:             "bdd-complete",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(5),
							EstimatedHours: bddRat(t, "8"),
							ActualHours:    bddRat(t, "10"),
						},
						{
							ID:             "bdd-missing-estimate",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(3),
							EstimatedHours: nil,
							ActualHours:    bddRat(t, "7"),
						},
						{
							ID:             "bdd-missing-actual",
							Status:         "Terminada",
							StoryPoints:    bddIntPointer(2),
							EstimatedHours: bddRat(t, "6"),
							ActualHours:    nil,
						},
					},
				},
			}

			t.Run("When el sistema calcula la desviación con historias completas", func(t *testing.T) {
				got, err := metrics.NewService(store).CalculateMetrics("bdd-sprint-partial")

				t.Run("Then informa el conteo exacto del cálculo parcial", func(t *testing.T) {
					if err != nil {
						t.Fatalf("CalculateMetrics() error = %v", err)
					}
					if !got.IsPartial {
						t.Error("IsPartial = false, want true")
					}
					if got.CalculationUsedCount != 1 {
						t.Errorf("historias utilizadas = %d, want 1", got.CalculationUsedCount)
					}
					if got.CalculationTotalCount != 3 {
						t.Errorf("historias totales = %d, want 3", got.CalculationTotalCount)
					}
					if got.Warning != "Cálculo parcial: basado en 1 de 3 historias" {
						t.Errorf("alerta = %q, want %q", got.Warning, "Cálculo parcial: basado en 1 de 3 historias")
					}
				})
			})
		})
	})
}

func bddIntPointer(value int) *int {
	return &value
}

func bddRat(t *testing.T, value string) *big.Rat {
	t.Helper()

	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		t.Fatalf("invalid rational number %q", value)
	}
	return rat
}

func assertBDDRatEqual(t *testing.T, label string, got *big.Rat, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %s", label, want)
		return
	}

	wantRat, ok := new(big.Rat).SetString(want)
	if !ok {
		t.Fatalf("invalid expected rational number %q", want)
	}
	if got.Cmp(wantRat) != 0 {
		t.Errorf("%s = %s, want %s", label, got.RatString(), wantRat.RatString())
	}
}
