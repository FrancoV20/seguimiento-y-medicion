package web

import (
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

// RegisterDefects registra la pantalla de defectos.
func RegisterDefects(s *Server, service *defect.Service, catalog DefectCatalog) error {
	_, _ = service, catalog
	notImplemented := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "implementación pendiente", http.StatusNotImplemented)
	})
	return s.Register(Section{Name: "Defectos", Path: "/defectos", RequiresProject: true}, Routes{
		"GET /defectos":  notImplemented,
		"POST /defectos": notImplemented,
	})
}
