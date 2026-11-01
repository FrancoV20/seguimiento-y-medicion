package metrics

import (
	"errors"
	"math/big"
)

// Calidad y avance de un Sprint finalizado (HU-10, T008).

// ErrSprintNotFound indica que el Sprint no existe en el proyecto.
var ErrSprintNotFound = errors.New("sprint not found")

// qualityDoneStatus es el estado final que cuenta una historia como completada.
const qualityDoneStatus = "Hecho"

// qualityFinishedStatus es el estado del Sprint cuyas métricas se calculan.
const qualityFinishedStatus = "Finalizado"

// qualityNotAvailable es el porcentaje de avance cuando el Sprint no tiene historias planificadas.
const qualityNotAvailable = "No disponible"

// QualityStoryResult es una historia planificada en el Sprint con su estado al cierre.
type QualityStoryResult struct {
	FinalStatus string // estado al cierre: "Hecho", "En progreso" o "En Sprint" (el CHECK admite también "Pendiente")
	StoryPoints *int   // nil = sin estimar
}

// QualityData son los datos crudos que lee el puerto.
type QualityData struct {
	SprintID        int64
	Status          string // estado del Sprint
	Stories         []QualityStoryResult
	DefectsDetected int
	DefectsResolved int
}

// SprintQuality es el resultado del cálculo; lo consumen la web, HU-08 y HU-09.
type SprintQuality struct {
	SprintID             int64
	StoriesPlanned       int
	StoriesCompleted     int
	CompletionPercentage string // "75.00" o "No disponible"
	PointsPlanned        int
	PointsCompleted      int
	StoriesWithoutPoints int
	DefectsDetected      int
	DefectsResolved      int
}

// calculateSprintQuality calcula el avance, los Story Points y los defectos del Sprint. Las historias
// sin Story Points cuentan como planificadas (y como completadas si están "Hecho"), pero no suman
// puntos: se informan en StoriesWithoutPoints.
func calculateSprintQuality(data QualityData) SprintQuality {
	result := SprintQuality{
		SprintID:        data.SprintID,
		StoriesPlanned:  len(data.Stories),
		DefectsDetected: data.DefectsDetected,
		DefectsResolved: data.DefectsResolved,
	}
	for _, story := range data.Stories {
		done := story.FinalStatus == qualityDoneStatus
		if done {
			result.StoriesCompleted++
		}
		if story.StoryPoints == nil {
			result.StoriesWithoutPoints++
			continue
		}
		result.PointsPlanned += *story.StoryPoints
		if done {
			result.PointsCompleted += *story.StoryPoints
		}
	}

	if result.StoriesPlanned == 0 {
		result.CompletionPercentage = qualityNotAvailable
		return result
	}
	percentage := big.NewRat(int64(result.StoriesCompleted)*100, int64(result.StoriesPlanned))
	result.CompletionPercentage = roundToTwoDecimals(percentage).FloatString(2)
	return result
}
