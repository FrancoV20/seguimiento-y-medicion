package web

import (
	"errors"
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

const closePath = "/cierre"

// RegisterClose registra temporalmente las rutas de cierre para permitir ejecutar las pruebas RED.
func RegisterClose(server *Server, _ *sprint.CloseSprintService, _ BoardCatalog) error {
	if server == nil {
		return errors.New("web server is required")
	}
	notImplemented := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// TODO(HU-15 T019): implementar GET y POST /cierre.
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	})
	return server.Register(
		Section{Name: "Cierre de Sprint", Path: closePath, RequiresProject: true},
		Routes{
			"GET " + closePath:  notImplemented,
			"POST " + closePath: notImplemented,
		},
	)
}
