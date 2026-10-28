package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

// Campos, textos y formato de la pantalla "Estimaciones" (HU-17, T016).

const (
	pathEstimates  = "/estimaciones"
	TitleEstimates = "Estimaciones"

	EstimateFieldStoryID = "historia_id"
	EstimateFieldHours   = "horas_estimadas"

	MsgHoursRegistered   = "Horas estimadas registradas"
	MsgHoursUpdated      = "Horas estimadas actualizadas"
	MsgStoryNotFound     = "La historia no existe"
	MsgHoursLockedSprint = "La historia pertenece a un Sprint finalizado; sus horas estimadas no se pueden modificar"
)

// estimateSubmission es el envío de una fila: la historia y el texto de horas tal como llegó.
type estimateSubmission struct {
	StoryID      int64
	ValidStoryID bool
	Hours        string
}

// parseEstimateSubmission lee historia_id (numérico y positivo) y horas_estimadas sin cambios: el
// valor lo interpreta el dominio. Cualquier campo de proyecto del formulario se ignora.
func parseEstimateSubmission(r *http.Request) estimateSubmission {
	_ = r.ParseForm()
	submission := estimateSubmission{Hours: r.PostForm.Get(EstimateFieldHours)}
	id, err := strconv.ParseInt(strings.TrimSpace(r.PostForm.Get(EstimateFieldStoryID)), 10, 64)
	if err == nil && id > 0 {
		submission.StoryID = id
		submission.ValidStoryID = true
	}
	return submission
}

// formatEstimateHours muestra las horas con coma decimal y sin ceros sobrantes (1,5; 8; 9999,99).
func formatEstimateHours(hours float64) string {
	return strings.Replace(strconv.FormatFloat(hours, 'f', -1, 64), ".", ",", 1)
}

// estimateSuccessMessage distingue el primer registro de una modificación.
func estimateSuccessMessage(result backlog.EstimateResult) string {
	if result.Previous == nil {
		return MsgHoursRegistered
	}
	return MsgHoursUpdated
}
