package web

import (
	"net/http"
	"net/url"
)

const (
	pathSelectProject = "/contexto/proyecto"
	pathProjects      = "/proyectos"
)

// RegisterContext registra el middleware del contexto de trabajo y la pantalla de selección
// de proyecto (HU-20b). Estas rutas no son una sección del menú.
func RegisterContext(s *Server, store ContextStore) {
	h := &contextHandlers{server: s, store: store}
	s.Use(contextMiddleware(store, s.renderer))
	s.HandleFunc("GET "+pathSelectProject, h.showProjects)
	s.HandleFunc("POST "+pathSelectProject, h.selectProject)
}

type contextHandlers struct {
	server *Server
	store  ContextStore
}

// selectProjectData son los datos de la pantalla de selección de proyecto.
type selectProjectData struct {
	Projects    []ProjectSummary
	ActiveID    int64
	ReturnTo    string
	CanCreate   bool
	CreatePath  string
	NoProjects  string
	CreateLabel string
}

func (h *contextHandlers) returnTo(raw string) string {
	return SanitizeReturnTo(raw, h.server.registry.Sections())
}

// showProjects muestra la lista de proyectos para elegir (T037).
func (h *contextHandlers) showProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	data := selectProjectData{
		Projects:    projects,
		ReturnTo:    h.returnTo(r.URL.Query().Get("volver_a")),
		CanCreate:   h.server.registry.Has(pathProjects),
		CreatePath:  pathProjects,
		NoProjects:  MsgNoProjects,
		CreateLabel: LinkCreateFirstProject,
	}
	if ac := ActiveContextFrom(r.Context()); ac != nil {
		data.ActiveID = ac.ProjectID
	}
	h.server.renderer.Page(w, r, "select_project.html", PageData{Title: TitleSelectProject, Data: data})
}

// selectProject fija el proyecto activo y redirige con 303 (T037). Si el identificador no es
// válido o no existe, conserva la selección anterior y vuelve a la lista con un mensaje.
func (h *contextHandlers) selectProject(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	returnTo := h.returnTo(r.PostFormValue("volver_a"))
	id := parseID(r.PostFormValue("proyecto_id"))

	var resolved ResolvedContext
	if id != 0 {
		var err error
		resolved, err = h.store.ResolveContext(r.Context(), id, 0)
		if err != nil {
			h.server.renderer.Fail(w, r, err)
			return
		}
	}
	if !resolved.ProjectExists {
		SetFlash(w, r, FlashError, MsgProjectInvalid)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a="+url.QueryEscape(returnTo))
		return
	}

	// Al cambiar a otro proyecto la identificación del integrante se limpia; al volver a
	// elegir el mismo, se conserva.
	if ac := ActiveContextFrom(r.Context()); ac == nil || ac.ProjectID != id {
		if _, present := readIDCookie(r, CookieMember); present {
			clearContextCookie(w, r, CookieMember)
		}
	}
	setContextCookie(w, r, CookieProject, id)
	RedirectSeeOther(w, r, returnTo)
}