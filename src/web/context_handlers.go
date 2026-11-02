package web

import (
	"net/http"
	"net/url"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
)

const (
	pathSelectProject = "/contexto/proyecto"
	pathProjects      = "/proyectos"
	pathSelectMember  = "/contexto/integrante"
	pathLogout        = "/contexto/salir"
)

// RegisterContext registra el middleware del contexto de trabajo y la pantalla de selección
// de proyecto (HU-20b). Estas rutas no son una sección del menú. El bloqueo por intentos
// fallidos de HU-27 usa el reloj real.
func RegisterContext(s *Server, store ContextStore) {
	RegisterContextWithLockout(s, store, auth.NewLockout(nil))
}

// RegisterContextWithLockout es RegisterContext con el bloqueo indicado; lo usan las pruebas para
// simular el reloj (HU-27).
func RegisterContextWithLockout(s *Server, store ContextStore, lockout *auth.Lockout) {
	if lockout == nil {
		lockout = auth.NewLockout(nil)
	}
	h := &contextHandlers{server: s, store: store, lockout: lockout}
	s.Use(contextMiddleware(store, s.renderer))
	s.HandleFunc("GET "+pathSelectProject, h.showProjects)
	s.HandleFunc("POST "+pathSelectProject, h.selectProject)
	s.HandleFunc("GET "+pathSelectMember, h.showMembers)
	s.HandleFunc("POST "+pathSelectMember, h.selectMember)
	s.HandleFunc("POST "+pathLogout, h.logout)
}

type contextHandlers struct {
	server  *Server
	store   ContextStore
	lockout *auth.Lockout
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

// selectMember identifica al integrante con su contraseña (HU-27; el orden de las respuestas está
// en data-model.md). Un integrante que no pertenece al proyecto activo se rechaza como en HU-20b.
// Los demás rechazos vuelven al selector sin tocar la identificación previa. La contraseña nunca
// se guarda en el formulario ni se registra.
func (h *contextHandlers) selectMember(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	returnTo := h.returnTo(r.PostFormValue("volver_a"))
	ac := h.requireProject(w, r, returnTo)
	if ac == nil {
		return
	}
	id := parseID(r.PostFormValue("integrante_id"))
	password := r.PostFormValue(FieldPassword)
	cred, err := h.store.MemberCredential(r.Context(), ac.ProjectID, id)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if !cred.Belongs {
		clearContextCookie(w, r, CookieMember)
		SetFlash(w, r, FlashError, MsgMemberInvalid)
		RedirectSeeOther(w, r, pathSelectMember+"?volver_a="+url.QueryEscape(returnTo))
		return
	}
	if password == "" {
		form := NewFormView()
		form.AddError(FieldPassword, MsgPasswordRequired)
		h.renderMembers(w, r, http.StatusUnprocessableEntity, ac, id, returnTo, form, nil)
		return
	}
	if h.lockout.Locked(id) {
		h.renderMembers(w, r, http.StatusTooManyRequests, ac, id, returnTo, nil, &Flash{Kind: FlashError, Text: MsgTooManyAttempts})
		return
	}
	if cred.PasswordHash == "" {
		h.renderMembers(w, r, http.StatusUnprocessableEntity, ac, id, returnTo, nil,
			&Flash{Kind: FlashError, Text: NoPasswordMessage(cred.Name)})
		return
	}
	ok, err := auth.VerifyPassword(password, cred.PasswordHash)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if !ok {
		h.lockout.RecordFailure(id)
		h.renderMembers(w, r, http.StatusUnauthorized, ac, id, returnTo, nil, &Flash{Kind: FlashError, Text: MsgWrongCredentials})
		return
	}
	h.lockout.RecordSuccess(id)
	setContextCookie(w, r, CookieMember, id)
	SetFlash(w, r, FlashSuccess, IdentifiedMessage(cred.Name))
	RedirectSeeOther(w, r, returnTo)
}

// renderMembers vuelve a mostrar el selector con el integrante elegido marcado, el campo de la
// contraseña vacío y el error o el mensaje del rechazo.
func (h *contextHandlers) renderMembers(w http.ResponseWriter, r *http.Request, status int, ac *ActiveContext,
	chosen int64, returnTo string, form *FormView, flash *Flash) {
	members, err := h.store.ListMembers(r.Context(), ac.ProjectID)
	if err != nil {
		h.server.renderer.Fail(w, r, err)
		return
	}
	if form == nil {
		form = NewFormView()
	}
	h.server.renderer.PageStatus(w, r, status, "select_member.html", PageData{
		Title: TitleSelectMember,
		Form:  form,
		Flash: flash,
		Data: selectMemberData{
			Members: members, ActiveID: chosen, ReturnTo: returnTo,
			ProjectName: ac.ProjectName, NoMembers: MsgNoMembers,
		},
	})
}

// logout cierra la identificación (HU-27): borra solo la cookie del integrante y vuelve al destino
// saneado ("/" por defecto). Responde igual aunque no hubiera nadie identificado.
func (h *contextHandlers) logout(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	clearContextCookie(w, r, CookieMember)
	SetFlash(w, r, FlashSuccess, MsgLoggedOut)
	RedirectSeeOther(w, r, h.returnTo(r.PostFormValue("volver_a")))
}
