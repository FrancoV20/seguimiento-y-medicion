package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

const (
	MsgSprintClosed        = "Sprint %d cerrado correctamente. Historias devueltas al Backlog: %d"
	MsgSprintAlreadyClosed = "El Sprint ya está cerrado"
	MsgSprintNotFound      = "El Sprint no existe"
	MsgCloseBlocked        = "El Sprint no se puede cerrar"
	MsgNoSprintToClose     = "No hay un Sprint activo para cerrar"
	closeSprintIDField     = "sprint_id"
)

// CloseSprintForm contiene los datos enviados para cerrar el Sprint mostrado.
type CloseSprintForm struct {
	SprintID int64
}

// CloseSprintViewData contiene el resumen previo o el estado vacío de la pantalla.
type CloseSprintViewData struct {
	HasActiveSprint bool
	Summary         CloseSummary
	NoActiveSprint  string
}

// CloseSummary resume el estado actual del Sprint activo antes de confirmarlo.
type CloseSummary struct {
	SprintID        int64
	Label           string
	Goal            string
	StartDate       string
	EndDate         string
	Completed       []BoardStory
	ToReturn        []BoardStory
	PlannedPoints   int
	CompletedPoints int
}

// buildCloseSprintViewData carga el Sprint activo del proyecto y arma su resumen.
func buildCloseSprintViewData(
	ctx context.Context,
	catalog BoardCatalog,
	projectID int64,
) (CloseSprintViewData, error) {
	if catalog == nil {
		return CloseSprintViewData{}, errors.New("board catalog is required")
	}
	board, found, err := catalog.ActiveBoard(ctx, projectID)
	if err != nil {
		return CloseSprintViewData{}, err
	}
	if !found {
		return CloseSprintViewData{NoActiveSprint: MsgNoSprintToClose}, nil
	}
	return CloseSprintViewData{
		HasActiveSprint: true,
		Summary:         buildCloseSummary(board),
	}, nil
}

// parseCloseSprintForm lee el identificador del Sprint enviado por el formulario.
func parseCloseSprintForm(r *http.Request) (CloseSprintForm, error) {
	raw := r.PostFormValue(closeSprintIDField)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return CloseSprintForm{}, sprint.ErrSprintNotFound
	}
	return CloseSprintForm{SprintID: id}, nil
}

// buildCloseSummary divide las historias y calcula los Story Points desde el tablero.
func buildCloseSummary(board BoardView) CloseSummary {
	summary := CloseSummary{
		SprintID:  board.ID,
		Label:     board.Label,
		Goal:      board.Goal,
		StartDate: board.StartDate.Format("02/01/2006"),
		EndDate:   board.EndDate.Format("02/01/2006"),
	}
	stories := make([]BoardStory, 0)
	for _, group := range board.Groups {
		for _, story := range group.Stories {
			switch story.Status {
			case sprint.Done:
				summary.Completed = append(summary.Completed, story)
			case sprint.EnSprint, sprint.InProgress:
				summary.ToReturn = append(summary.ToReturn, story)
			default:
				continue
			}
			stories = append(stories, story)
		}
	}
	summary.PlannedPoints, summary.CompletedPoints = BoardPoints(stories)
	return summary
}

func closeSprintSuccessMessage(sprintNumber int64, returnedCount int) string {
	return fmt.Sprintf(MsgSprintClosed, sprintNumber, returnedCount)
}

func closeSprintErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, sprint.ErrSprintNotFound):
		return http.StatusUnprocessableEntity, MsgSprintNotFound
	case errors.Is(err, sprint.ErrSprintAlreadyClosed):
		return http.StatusConflict, MsgSprintAlreadyClosed
	case errors.Is(err, sprint.ErrCloseBlocked):
		detail := strings.TrimSpace(strings.TrimPrefix(err.Error(), sprint.ErrCloseBlocked.Error()+":"))
		if detail == "" {
			detail = MsgCloseBlocked
		}
		return http.StatusConflict, detail
	default:
		return http.StatusInternalServerError, MsgOperationFailed
	}
}
