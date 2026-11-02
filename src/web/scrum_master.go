package web

import (
	"net/http"
	"net/url"

	"github.com/FrancoV20/seguimiento-y-medicion/src/role"
)

// Textos de las acciones reservadas al Scrum Master (HU-28).
const (
	TitleScrumMasterOnly       = "Acción reservada al Scrum Master"
	MsgScrumMasterOnly         = "Esta acción la puede hacer solo el Scrum Master del proyecto"
	MsgNoScrumMaster           = "Este proyecto no tiene Scrum Master asignado. Pedí que se configure con el comando de roles"
	MsgScrumMasterOtherProject = "Para modificar este proyecto, elegilo como proyecto activo e identificate como su Scrum Master"
)

// RequireScrumMaster reserva la acción h al Scrum Master del proyecto activo (HU-28). Se aplica al
// registrar la ruta, detrás de RequireContext, y rechaza antes de llamar a h:
//  1. sin proyecto activo: 303 a elegir proyecto (HU-20b);
//  2. sin integrante identificado: 303 a "Identificarme", con volver_a a la sección;
//  3. con rol "Scrum Master": llama a h;
//  4. si el proyecto no tiene ningún Scrum Master: 403 con MsgNoScrumMaster;
//  5. en otro caso: 403 con MsgScrumMasterOnly.
func (s *Server) RequireScrumMaster(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac, ok := s.requireIdentified(w, r)
		if !ok {
			return
		}
		s.serveIfScrumMaster(w, r, ac, h)
	})
}

// RequireScrumMasterOfPathProject es RequireScrumMaster para POST /proyectos/{id}/modificar (HU-11):
// además, el {id} tiene que ser el proyecto activo. Si es otro, inválido o inexistente responde 403
// con MsgScrumMasterOtherProject, sin consultar el rol en ese proyecto.
func (s *Server) RequireScrumMasterOfPathProject(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac, ok := s.requireIdentified(w, r)
		if !ok {
			return
		}
		if id := parseID(r.PathValue("id")); id == 0 || id != ac.ProjectID {
			s.forbidScrumMaster(w, r, MsgScrumMasterOtherProject)
			return
		}
		s.serveIfScrumMaster(w, r, ac, h)
	})
}

// requireIdentified hace los pasos 1 y 2: sin proyecto o sin integrante, redirige como HU-20b y
// devuelve false.
func (s *Server) requireIdentified(w http.ResponseWriter, r *http.Request) (*ActiveContext, bool) {
	ac := ActiveContextFrom(r.Context())
	back := url.QueryEscape("/" + firstSegment(r.URL.Path))
	switch {
	case ac == nil || ac.ProjectID == 0:
		SetFlash(w, r, FlashWarning, MsgProjectRequired)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a="+back)
		return nil, false
	case ac.MemberID == 0:
		SetFlash(w, r, FlashWarning, MsgMemberRequired)
		RedirectSeeOther(w, r, pathSelectMember+"?volver_a="+back)
		return nil, false
	}
	return ac, true
}

// serveIfScrumMaster hace los pasos 3 a 5.
func (s *Server) serveIfScrumMaster(w http.ResponseWriter, r *http.Request, ac *ActiveContext, h http.Handler) {
	switch {
	case role.CanManage(ac.MemberRole):
		h.ServeHTTP(w, r)
	case !ac.HasScrumMaster:
		s.forbidScrumMaster(w, r, MsgNoScrumMaster)
	default:
		s.forbidScrumMaster(w, r, MsgScrumMasterOnly)
	}
}

// forbidScrumMaster responde 403 con la estructura común y un enlace al inicio.
func (s *Server) forbidScrumMaster(w http.ResponseWriter, r *http.Request, message string) {
	s.renderer.RenderError(w, r, http.StatusForbidden, TitleScrumMasterOnly, message)
}
