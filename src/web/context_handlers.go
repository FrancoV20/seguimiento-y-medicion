package web

import (
	"net/http"
	"net/url"
)

const (
	pathSelectProject = "/contexto/proyecto"
	pathProjects      = "/proyectos"
	pathSelectMember  = "/contexto/integrante"
)

// RegisterContext registra el middleware del contexto de trabajo y la pantalla de selección
// de proyecto (HU-20b). Estas rutas no son una sección del menú.
func RegisterContext(s *Server, store ContextStore) {
	h := &contextHandlers{server: s, store: store}
	s.Use(contextMiddleware(store, s.renderer))
	s.HandleFunc("GET "+pathSelectProject, h.showProjects)
	s.HandleFunc("POST "+pathSelectProject, h.selectProject)
	s.HandleFunc("GET "+pathSelectMember, h.showMembers)
	s.HandleFunc("POST "+pathSelectMember, h.selectMember)
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

// selectMemberData son los datos de la pantalla de identificación.
type selectMemberData struct {
	Members     []MemberSummary
	ActiveID    int64
	ReturnTo    string
	ProjectName string
	NoMembers   string
}

// requireProject devuelve el contexto activo; si no hay proyecto, redirige a elegirlo y
// devuelve nil.
func (h *contextHandlers) requireProject(w http.ResponseWriter, r *http.Request, returnTo string) *ActiveContext {
	ac := ActiveContextFrom(r.Context())
	if ac == nil || ac.ProjectID == 0 {
		SetFlash(w, r, FlashWarning, MsgChooseProjectFirst)
		RedirectSeeOther(w, r, pathSelectProject+"?volver_a="+url.QueryEscape(returnTo))
		return nil
	}
	return ac
}

// showMembers muestra solo los integrantes del proyecto activo (T044).
func (h *contextHandlers) showMembers(w http.ResponseWriter, r *http.Request) {
	returnTo := h.returnTo(r.URL.Query().Get("volver_a"))
	ac := h.requireProject(w, r, returnTo)
	if ac == nil {
		return
	}
	members, err := h.store.ListMembers(r.Context(), ac.ProjectID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	h.server.renderer.Page(w, r, "select_member.html", PageData{
		Title: TitleSelectMember,
		Data: selectMemberData{
			Members: members, ActiveID: ac.MemberID, ReturnTo: returnTo,
			ProjectName: ac.ProjectName, NoMembers: MsgNoMembers,
		},
	})
}

// selectMember guarda la identificación (T044). Un integrante que no pertenece al proyecto
// activo se rechaza, se limpia la identificación y se vuelve al selector con un mensaje.
func (h *contextHandlers) selectMember(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	returnTo := h.returnTo(r.PostFormValue("volver_a"))
	ac := h.requireProject(w, r, returnTo)
	if ac == nil {
		return
	}
	id := parseID(r.PostFormValue("integrante_id"))
	resolved, err := h.store.ResolveContext(r.Context(), ac.ProjectID, id)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if !resolved.MemberBelongs {
		clearContextCookie(w, r, CookieMember)
		SetFlash(w, r, FlashError, MsgMemberInvalid)
		RedirectSeeOther(w, r, pathSelectMember+"?volver_a="+url.QueryEscape(returnTo))
		return
	}
	setContextCookie(w, r, CookieMember, id)
	RedirectSeeOther(w, r, returnTo)
}
