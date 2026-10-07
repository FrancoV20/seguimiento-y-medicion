package metrics

import (
	"fmt"
	"math/big"
)

func calculateSprintMetrics(sprint SprintData) (SprintMetrics, error) {
	metrics := SprintMetrics{
		SprintID:            sprint.ID,
		EstimatedHoursTotal: new(big.Rat),
		ActualHoursTotal:    new(big.Rat),
		DeviationPercentage: "No disponible",
	}

	for _, story := range sprint.Stories {
		if story.Status != "Terminada" {
			continue
		}

		metrics.CalculationTotalCount++
		if story.StoryPoints == nil {
			return SprintMetrics{}, CalculationError{
				StoryID: story.ID,
				Field:   "storyPoints",
				Message: ErrInvalidStoryPoints.Error(),
				Cause:   ErrInvalidStoryPoints,
			}
		}
		metrics.Velocity += *story.StoryPoints

		if story.EstimatedHours == nil || story.ActualHours == nil {
			continue
		}

		metrics.CalculationUsedCount++
		metrics.EstimatedHoursTotal.Add(metrics.EstimatedHoursTotal, story.EstimatedHours)
		metrics.ActualHoursTotal.Add(metrics.ActualHoursTotal, story.ActualHours)
	}

	if metrics.CalculationUsedCount < metrics.CalculationTotalCount {
		metrics.IsPartial = true
		metrics.Warning = fmt.Sprintf(
			"Cálculo parcial: basado en %d de %d historias",
			metrics.CalculationUsedCount,
			metrics.CalculationTotalCount,
		)
	}

	return metrics, nil
}

func CalculateDeviation(estimated, actual *big.Rat) (string, error) {
	if estimated == nil || actual == nil || estimated.Sign() <= 0 {
		return "No disponible", nil
	}

	difference := new(big.Rat).Sub(actual, estimated)
	deviation := new(big.Rat).Quo(difference, estimated)
	deviation.Mul(deviation, big.NewRat(100, 1))

	return roundToTwoDecimals(deviation).FloatString(2), nil
}

func roundToTwoDecimals(value *big.Rat) *big.Rat {
	if value == nil {
		return nil
	}

	scaledNumerator := new(big.Int).Mul(value.Num(), big.NewInt(100))
	negative := scaledNumerator.Sign() < 0
	if negative {
		scaledNumerator.Abs(scaledNumerator)
	}

	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(scaledNumerator, value.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(value.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if negative {
		quotient.Neg(quotient)
	}

	return new(big.Rat).SetFrac(quotient, big.NewInt(100))
}
