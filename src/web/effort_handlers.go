package web

import (
	"net/http"
	"strconv"
	"time"
)

// pathEffort es la dirección de la sección "Registro de esfuerzo" (HU-24).
const pathEffort = "/esfuerzo"

// effortMaxBody limita el tamaño del envío (64 KB): sobra para una actividad y evita que un
// envío enorme ocupe memoria o termine guardado en la base. Un envío más grande no registra nada.
const effortMaxBody = 64 << 10

// EffortSection es la sección del menú: exige proyecto activo e integrante identificado (FR-001).
var EffortSection = Section{Name: TitleEffort, Path: pathEffort, RequiresProject: true, RequiresMember: true}

// RegisterEffort registra la sección "Registro de esfuerzo" con GET y POST /esfuerzo (T022).
// now es el reloj para la fecha por defecto (nil = time.Now).
func RegisterEffort(s *Server, actions EffortActions, catalog EffortCatalog, now func() time.Time) error {
	if now == nil {
		now = time.Now
	}
	h := &effortHandlers{rd: s.renderer, actions: actions, catalog: catalog, now: now}
	return s.Register(EffortSection, Routes{
		"GET " + pathEffort:  http.HandlerFunc(h.show),
		"POST " + pathEffort: http.HandlerFunc(h.submit),
	})
}

type effortHandlers struct {
	rd      *Renderer
	actions EffortActions
	catalog EffortCatalog
	now     func() time.Time
}

// effortPageData son los datos de effort.html.
type effortPageData struct {
	Sprint     *EffortSprint  // nil: no hay Sprint activo
	History    *EffortHistory // nil: ninguna historia elegida
	SelectedID int64
	NoSprint   string
	NoStories  string
	NoEntries  string
	Errors     []string // errores generales del envío
}

func (h *effortHandlers) page(sprint *EffortSprint) effortPageData {
	return effortPageData{
		Sprint:    sprint,
		NoSprint:  MsgEffortNoSprint,
		NoStories: MsgEffortNoStories,
		NoEntries: MsgEffortNoEntries,
	}
}

// show atiende GET /esfuerzo: historias del Sprint activo, formulario con la fecha de hoy y,
// si ?historia=ID es una historia del Sprint activo del proyecto, su historial (T019, T037).
// Un ?historia inválido o ajeno se ignora (Edge Cases).
func (h *effortHandlers) show(w http.ResponseWriter, r *http.Request) {
	ac := ActiveContextFrom(r.Context())
	sprint, err := h.catalog.ActiveSprintStories(r.Context(), ac.ProjectID)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	data := h.page(sprint)
	form := NewFormView()
	form.Values[EffortFieldDate] = h.now().Format(effortDateLayout)

	if story := sprint.Story(parseID(r.URL.Query().Get(EffortFieldStory))); story != nil {
		history, err := h.catalog.StoryHistory(r.Context(), ac.ProjectID, story.ID)
		if err != nil {
			h.rd.Fail(w, r, err)
			return
		}
		if history != nil {
			data.History = history
			data.SelectedID = story.ID
			form.Values[EffortFieldStory] = strconv.FormatInt(story.ID, 10)
		}
	}
	h.rd.Page(w, r, "effort.html", PageData{Title: TitleEffort, Data: data, Form: form})
}

// submit atiende POST /esfuerzo (T020, T029, T030). El proyecto y el integrante salen del
// contexto del servidor; un campo "integrante" enviado se ignora (FR-008).
func (h *effortHandlers) submit(w http.ResponseWriter, r *http.Request) {
	ac := ActiveContextFrom(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, effortMaxBody)
	form := FormFromRequest(r, effortFields...)

	sprint, err := h.catalog.ActiveSprintStories(r.Context(), ac.ProjectID)
	if err != nil {
		h.rd.Fail(w, r, err)
		return
	}
	data := h.page(sprint)
	if sprint == nil {
		h.rd.PageStatus(w, r, http.StatusConflict, "effort.html", PageData{Title: TitleEffort, Data: data, Form: form})
		return
	}

	sub, readErrs := parseEffortForm(form.Values)
	sub.MemberID = ac.MemberID
	for field, msg := range readErrs {
		form.AddError(field, msg)
	}
	// Solo se aceptan historias del Sprint activo del proyecto activo (FR-007).
	if _, bad := form.Errors[EffortFieldStory]; !bad && sprint.Story(sub.StoryID) == nil {
		form.AddError(EffortFieldStory, MsgEffortStoryNotInSprint)
	}
	// Todas las reglas de HU-03 a la vez, un error por campo (FR-006).
	data.Errors = mergeEffortErrors(form, h.actions.Validate(sub))
	if form.HasErrors() || len(data.Errors) > 0 {
		h.rd.PageStatus(w, r, http.StatusUnprocessableEntity, "effort.html", PageData{Title: TitleEffort, Data: data, Form: form})
		return
	}

	violations, err := h.actions.Register(r.Context(), sub)
	if err != nil {
		// Falla de guardado: datos conservados, sin éxito y sin detalles internos (FR-012, FR-014).
		h.rd.PageStatus(w, r, http.StatusInternalServerError, "effort.html", PageData{
			Title: TitleEffort, Data: data, Form: form,
			Flash: &Flash{Kind: FlashError, Text: MsgOperationFailed},
		})
		return
	}
	if len(violations) > 0 {
		// Por ejemplo, el Sprint se cerró entre abrir el formulario y enviarlo (T027).
		data.Errors = mergeEffortErrors(form, violations)
		h.rd.PageStatus(w, r, http.StatusUnprocessableEntity, "effort.html", PageData{Title: TitleEffort, Data: data, Form: form})
		return
	}

	SetFlash(w, r, FlashSuccess, MsgEffortRecorded)
	RedirectSeeOther(w, r, pathEffort+"?"+EffortFieldStory+"="+strconv.FormatInt(sub.StoryID, 10))
}
