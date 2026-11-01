package metrics

import "math/big"

// VelocitySummary resume la velocidad de los Sprints finalizados de un proyecto (HU-08).
type VelocitySummary struct {
	SprintCount int
	Total       int    // suma de los Story Points completados
	Average     string // promedio con punto y dos decimales ("10.50"), o "No disponible" sin Sprints
}

// SummarizeVelocity calcula la velocidad acumulada y el promedio por Sprint a partir de los Story
// Points completados de cada Sprint finalizado. El promedio usa el redondeo de HU-07 (mitad hacia
// arriba, dos decimales). No lee ni guarda nada.
func SummarizeVelocity(points []int) VelocitySummary {
	summary := VelocitySummary{SprintCount: len(points)}
	for _, p := range points {
		summary.Total += p
	}
	if summary.SprintCount == 0 {
		summary.Average = "No disponible"
		return summary
	}
	average := big.NewRat(int64(summary.Total), int64(summary.SprintCount))
	summary.Average = roundToTwoDecimals(average).FloatString(2)
	return summary
}
