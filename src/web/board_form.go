package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T019): el envío del Tablero y sus textos. Los textos se definen en un solo lugar.

const TitleBoard = "Tablero"

// Textos de éxito de los escenarios del Issue (se respetan tal cual).
const (
	MsgStatusStarted          = "La historia pasó a En progreso"
	MsgStatusCompleted        = "La historia pasó a Hecho"
	MsgStatusCompletedNoHours = "La historia pasó a Hecho. Advertencia: no tiene horas reales registradas"
	MsgStatusReopened         = "La historia volvió a En progreso"
)

// BoardForm son los datos del envío de POST /tablero/estado.
type BoardForm struct {
	StoryID int64
	Action  sprint.Action
	// Expected es el estado que la persona veía; vacío si no se envió.
	Expected sprint.StoryStatus
}

// parseBoardForm lee el envío. ok es falso si la historia no es un número positivo o la
// acción no es una de las tres conocidas. El proyecto nunca se lee del formulario.
func parseBoardForm(r *http.Request) (form BoardForm, ok bool) {
	_ = r.ParseForm()
	id, err := strconv.ParseInt(strings.TrimSpace(r.PostFormValue("historia_id")), 10, 64)
	if err != nil || id <= 0 {
		return BoardForm{}, false
	}
	action := sprint.Action(strings.TrimSpace(r.PostFormValue("accion")))
	if _, _, known := sprint.ActionTransition(action); !known {
		return BoardForm{}, false
	}
	return BoardForm{
		StoryID:  id,
		Action:   action,
		Expected: sprint.StoryStatus(strings.TrimSpace(r.PostFormValue("estado_actual"))),
	}, true
}

// boardSuccessMessage elige el mensaje de éxito; la advertencia de horas viaja en el mismo mensaje.
func boardSuccessMessage(result sprint.StatusResult) string {
	switch {
	case result.To == sprint.Done && result.NoHoursWarning:
		return MsgStatusCompletedNoHours
	case result.To == sprint.Done:
		return MsgStatusCompleted
	case result.From == sprint.Done:
		return MsgStatusReopened
	default:
		return MsgStatusStarted
	}
}