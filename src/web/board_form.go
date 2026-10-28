package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T019, T029): el envío del Tablero, sus textos y la traducción de errores del dominio.
// Los textos se definen en un solo lugar.

const TitleBoard = "Tablero"

// Textos de éxito de los escenarios del Issue (se respetan tal cual).
const (
	MsgStatusStarted          = "La historia pasó a En progreso"
	MsgStatusCompleted        = "La historia pasó a Hecho"
	MsgStatusCompletedNoHours = "La historia pasó a Hecho. Advertencia: no tiene horas reales registradas"
	MsgStatusReopened         = "La historia volvió a En progreso"
)

// Textos de rechazo. Los de los escenarios del Issue se respetan tal cual; el de la acción no
// válida y los códigos de respuesta los definió el equipo. Llevan el prefijo Board para no
// chocar con los de otras historias del mismo paquete.
const (
	MsgBoardStoryNotFound     = "La historia no existe"
	MsgBoardInvalidAction     = "La acción no es válida"
	MsgBoardNotInActiveSprint = "La historia no está asignada a un Sprint activo"
	MsgBoardStatusChanged     = "El estado de la historia cambió; revise el estado actual y vuelva a intentarlo"
)

// Textos de los estados vacíos del Tablero (escenarios del Issue, tal cual).
const (
	MsgBoardNoActiveSprint       = "No hay un Sprint activo"
	MsgBoardSprintWithoutStories = "El Sprint no tiene historias asignadas"
)

func boardAlreadyInStatusMessage(status sprint.StoryStatus) string {
	return fmt.Sprintf("La historia ya está en %s", status)
}

func boardTransitionNotAllowedMessage(from, to sprint.StoryStatus) string {
	return fmt.Sprintf("No se puede pasar de %s a %s", from, to)
}

// BoardForm son los datos del envío de POST /tablero/estado.
type BoardForm struct {
	StoryID int64
	Action  sprint.Action
	// Expected es el estado que la persona veía; vacío si no se envió.
	Expected sprint.StoryStatus
}

// boardFormProblem indica por qué no se pudo leer el envío.
type boardFormProblem int

const (
	boardFormValid boardFormProblem = iota
	boardFormBadStory
	boardFormBadAction
)

// parseBoardForm lee el envío. La historia debe ser un número positivo y la acción una de las
// tres conocidas. El proyecto nunca se lee del formulario.
func parseBoardForm(r *http.Request) (BoardForm, boardFormProblem) {
	_ = r.ParseForm()
	id, err := strconv.ParseInt(strings.TrimSpace(r.PostFormValue("historia_id")), 10, 64)
	if err != nil || id <= 0 {
		return BoardForm{}, boardFormBadStory
	}
	action := sprint.Action(strings.TrimSpace(r.PostFormValue("accion")))
	if _, _, known := sprint.ActionTransition(action); !known {
		return BoardForm{}, boardFormBadAction
	}
	return BoardForm{
		StoryID:  id,
		Action:   action,
		Expected: sprint.StoryStatus(strings.TrimSpace(r.PostFormValue("estado_actual"))),
	}, boardFormValid
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

// boardSuccessFlash arma el mensaje del cambio válido; con la advertencia de horas es una advertencia.
func boardSuccessFlash(result sprint.StatusResult) Flash {
	kind := FlashSuccess
	if result.To == sprint.Done && result.NoHoursWarning {
		kind = FlashWarning
	}
	return Flash{Kind: kind, Text: boardSuccessMessage(result)}
}

// boardOutcome es la respuesta que corresponde a un error del cambio de estado.
type boardOutcome struct {
	status   int   // código HTTP cuando se muestra el Tablero con el mensaje
	flash    Flash // mensaje que se muestra
	redirect bool  // verdadero: 303 a /tablero con el mensaje de una sola vista
	internal bool  // verdadero: falla de almacenamiento, se registra el detalle técnico
}

// boardOutcomeFor traduce un error del dominio a su respuesta. Cualquier error que no sea del
// dominio es una falla de almacenamiento: mensaje genérico y detalle solo en el registro.
func boardOutcomeFor(err error) boardOutcome {
	var already *sprint.AlreadyInStatusError
	var notAllowed *sprint.TransitionNotAllowedError
	switch {
	case errors.Is(err, sprint.ErrStoryNotFound):
		return boardOutcome{status: http.StatusUnprocessableEntity, flash: Flash{Kind: FlashError, Text: MsgBoardStoryNotFound}}
	case errors.Is(err, sprint.ErrNotInActiveSprint):
		return boardOutcome{status: http.StatusConflict, flash: Flash{Kind: FlashError, Text: MsgBoardNotInActiveSprint}}
	case errors.As(err, &already):
		return boardOutcome{redirect: true, flash: Flash{Kind: FlashWarning, Text: boardAlreadyInStatusMessage(already.Status)}}
	case errors.Is(err, sprint.ErrStatusChanged):
		return boardOutcome{status: http.StatusConflict, flash: Flash{Kind: FlashError, Text: MsgBoardStatusChanged}}
	case errors.As(err, &notAllowed):
		return boardOutcome{status: http.StatusConflict, flash: Flash{Kind: FlashError, Text: boardTransitionNotAllowedMessage(notAllowed.From, notAllowed.To)}}
	default:
		return boardOutcome{status: http.StatusInternalServerError, flash: Flash{Kind: FlashError, Text: MsgOperationFailed}, internal: true}
	}
}
