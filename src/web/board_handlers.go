package web

import (
	"context"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

const pathBoard = "/tablero"

// StatusChanger es lo que el Tablero necesita del dominio; lo cumple sprint.StatusService.
type StatusChanger interface {
	ChangeStatus(ctx context.Context, projectID, storyID int64, from, to, expected sprint.StoryStatus) (sprint.StatusResult, error)
}

type boardHandlers struct {
	server  *Server
	changer StatusChanger
	catalog BoardCatalog
}

// boardPageData son los datos de board.html.
type boardPageData struct {
	View  BoardView
	Found bool
}

// RegisterBoard registra la sección "Tablero" (GET /tablero y POST /tablero/estado), que
// requiere un proyecto activo (HU-14, T020 y T022). El proyecto sale siempre del contexto
// del servidor, nunca del formulario.
func RegisterBoard(s *Server, changer StatusChanger, catalog BoardCatalog) error {
	h := &boardHandlers{server: s, changer: changer, catalog: catalog}
	return s.Register(
		Section{Name: TitleBoard, Path: pathBoard, RequiresProject: true},
		Routes{
			"GET " + pathBoard:              http.HandlerFunc(h.show),
			"POST " + pathBoard + "/estado": http.HandlerFunc(h.changeStatus),
		},
	)
}

func projectIDFrom(r *http.Request) int64 {
	if ac := ActiveContextFrom(r.Context()); ac != nil {
		return ac.ProjectID
	}
	return 0
}

// render lee el Tablero del proyecto activo y lo muestra con el código y el mensaje dados.
func (h *boardHandlers) render(w http.ResponseWriter, r *http.Request, status int, flash *Flash) {
	view, found, err := h.catalog.ActiveBoard(r.Context(), projectIDFrom(r))
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	h.server.renderer.PageStatus(w, r, status, "board.html", PageData{
		Title: TitleBoard,
		Flash: flash,
		Data:  boardPageData{View: view, Found: found},
	})
}

// show muestra el Tablero del Sprint activo (GET /tablero).
func (h *boardHandlers) show(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, nil)
}

// changeStatus aplica una acción del Tablero y redirige con 303 (POST /tablero/estado).
func (h *boardHandlers) changeStatus(w http.ResponseWriter, r *http.Request) {
	failure := &Flash{Kind: FlashError, Text: MsgOperationFailed}
	form, ok := parseBoardForm(r)
	if !ok {
		// US2 (T029 y T030) distingue historia inexistente de acción no válida.
		h.render(w, r, http.StatusInternalServerError, failure)
		return
	}
	from, to, _ := sprint.ActionTransition(form.Action)
	result, err := h.changer.ChangeStatus(r.Context(), projectIDFrom(r), form.StoryID, from, to, form.Expected)
	if err != nil {
		// US2 (T029 y T030) traduce los errores del dominio; el detalle técnico solo va al registro.
		h.server.logger.Error("no se pudo cambiar el estado de la historia",
			"historia", form.StoryID, "accion", string(form.Action), "detalle", err.Error())
		h.render(w, r, http.StatusInternalServerError, failure)
		return
	}
	SetFlash(w, r, FlashSuccess, boardSuccessMessage(result))
	RedirectSeeOther(w, r, pathBoard)
}