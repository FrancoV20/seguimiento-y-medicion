package web

import (
	"context"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// StatusChanger es lo que el Tablero necesita del dominio; lo cumple sprint.StatusService.
type StatusChanger interface {
	ChangeStatus(ctx context.Context, projectID, storyID int64, from, to, expected sprint.StoryStatus) (sprint.StatusResult, error)
}

// RegisterBoard registra la sección "Tablero" (GET /tablero y POST /tablero/estado), que
// requiere un proyecto activo (HU-14, T020 y T022).
// RED: esqueleto sin implementar.
func RegisterBoard(s *Server, changer StatusChanger, catalog BoardCatalog) error {
	notImplemented := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "HU-14: todavía no implementado", http.StatusNotImplemented)
	})
	return s.Register(
		Section{Name: "Tablero", Path: "/tablero", RequiresProject: true},
		Routes{"GET /tablero": notImplemented, "POST /tablero/estado": notImplemented},
	)
}
