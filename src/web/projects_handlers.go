package web

import (
	"net/http"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

// RegisterProjects registers provisional routes until the T016 implementation.
func RegisterProjects(s *Server, _ *project.ProjectService, _ ProjectCatalog) error {
	unimplemented := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not implemented", http.StatusNotImplemented)
	})
	return s.Register(Section{Name: "Proyectos", Path: "/proyectos"}, Routes{
		"GET /proyectos":  unimplemented,
		"POST /proyectos": unimplemented,
	})
}
