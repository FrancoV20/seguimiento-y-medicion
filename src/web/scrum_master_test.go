package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/role"
)

// HU-28, T018: el envoltorio de las acciones del Scrum Master, con un ActiveContext armado a mano
// y un handler que cuenta llamadas (escenarios 3, 5, 8, 9, 12 y 13).

// countingHandler responde 204 y cuenta cuántas veces se lo llamó.
type countingHandler struct{ calls atomic.Int32 }

func (h *countingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.calls.Add(1)
	w.WriteHeader(http.StatusNoContent)
}

func newScrumMasterTestServer() *Server {
	return NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// serveWithContext ejecuta handler con el ActiveContext dado (nil = sin contexto).
func serveWithContext(handler http.Handler, method, path string, ac *ActiveContext) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if ac != nil {
		req = req.WithContext(WithActiveContext(req.Context(), ac))
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// scrumMasterContext es «Alfa» (id 1) con Ana identificada como Scrum Master.
func scrumMasterContext() *ActiveContext {
	return &ActiveContext{ProjectID: 1, ProjectName: "Alfa", MemberID: 10, MemberName: "Ana",
		MemberRole: role.RoleScrumMaster, HasScrumMaster: true}
}

// memberContext es «Alfa» con Beto identificado como Integrante; el proyecto tiene Scrum Master.
func memberContext() *ActiveContext {
	return &ActiveContext{ProjectID: 1, ProjectName: "Alfa", MemberID: 11, MemberName: "Beto",
		MemberRole: role.RoleMember, HasScrumMaster: true}
}

// noScrumMasterContext es «Beta» (id 2) con Dani identificada; el proyecto no tiene Scrum Master.
func noScrumMasterContext() *ActiveContext {
	return &ActiveContext{ProjectID: 2, ProjectName: "Beta", MemberID: 20, MemberName: "Dani",
		MemberRole: role.RoleMember}
}

// assertForbidden comprueba el 403 con la estructura común, el título y el texto, sin detalles
// internos y sin llamar a la acción.
func assertForbidden(t *testing.T, rec *httptest.ResponseRecorder, h *countingHandler, message string) {
	t.Helper()
	body := rec.Body.String()
	if rec.Code != http.StatusForbidden {
		t.Errorf("estado = %d; se esperaba 403", rec.Code)
	}
	if !strings.Contains(body, TitleScrumMasterOnly) || !strings.Contains(body, message) {
		t.Errorf("la página 403 debía tener el título %q y el texto %q", TitleScrumMasterOnly, message)
	}
	if !strings.Contains(body, `class="app-context"`) || !strings.Contains(body, `<a href="/">`) {
		t.Error("la página 403 debía usar la estructura común con un enlace al inicio")
	}
	for _, internal := range []string{"role:", "goroutine", "sql", "panic"} {
		if strings.Contains(body, internal) {
			t.Errorf("la página 403 expone un detalle interno (%q)", internal)
		}
	}
	if n := h.calls.Load(); n != 0 {
		t.Errorf("la acción se llamó %d veces; en un rechazo no debía llamarse", n)
	}
}

func TestRequireScrumMaster_ScrumMasterPasa(t *testing.T) {
	srv := newScrumMasterTestServer()
	for _, path := range []string{"/cierre", "/sprints", "/sprints/historias", "/planning-poker/abrir"} {
		h := &countingHandler{}
		ac := scrumMasterContext()
		rec := serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, path, ac)
		if rec.Code != http.StatusNoContent || h.calls.Load() != 1 {
			t.Errorf("POST %s como Scrum Master = %d con %d llamadas; se esperaba la acción una vez", path, rec.Code, h.calls.Load())
		}
	}

	// Más de un Scrum Master: cualquiera de ellos pasa (el control mira solo el rol propio).
	h := &countingHandler{}
	other := scrumMasterContext()
	other.MemberID, other.MemberName = 12, "Caro"
	if rec := serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, "/cierre", other); rec.Code != http.StatusNoContent {
		t.Errorf("un segundo Scrum Master recibió %d; se esperaba la acción", rec.Code)
	}
}

func TestRequireScrumMaster_IntegranteRechazado(t *testing.T) {
	srv := newScrumMasterTestServer()
	h := &countingHandler{}
	rec := serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, "/cierre", memberContext())
	assertForbidden(t, rec, h, MsgScrumMasterOnly)

	// Un rol vacío (contexto armado sin rol) tampoco puede.
	empty := memberContext()
	empty.MemberRole = ""
	rec = serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, "/sprints", empty)
	assertForbidden(t, rec, h, MsgScrumMasterOnly)
}

func TestRequireScrumMaster_SinScrumMaster(t *testing.T) {
	srv := newScrumMasterTestServer()
	h := &countingHandler{}
	rec := serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, "/sprints", noScrumMasterContext())
	assertForbidden(t, rec, h, MsgNoScrumMaster)
	if strings.Contains(rec.Body.String(), MsgScrumMasterOnly) {
		t.Error("sin Scrum Master en el proyecto el texto debía ser el propio, no el genérico")
	}
}

func TestRequireScrumMaster_SinIdentificacion(t *testing.T) {
	srv := newScrumMasterTestServer()
	cases := []struct {
		path, location string
	}{
		{"/cierre", "/contexto/integrante?volver_a=%2Fcierre"},
		{"/sprints/historias/quitar", "/contexto/integrante?volver_a=%2Fsprints"},
		{"/planning-poker/estimacion", "/contexto/integrante?volver_a=%2Fplanning-poker"},
	}
	for _, tc := range cases {
		h := &countingHandler{}
		ac := memberContext()
		ac.MemberID, ac.MemberName, ac.MemberRole = 0, "", ""
		rec := serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, tc.path, ac)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != tc.location {
			t.Errorf("POST %s sin identificarse = %d a %q; se esperaba 303 a %q", tc.path, rec.Code, rec.Header().Get("Location"), tc.location)
		}
		if !strings.Contains(rec.Header().Get("Set-Cookie"), CookieFlash+"=") {
			t.Errorf("POST %s sin identificarse: se esperaba el mensaje de HU-20b para identificarse", tc.path)
		}
		if h.calls.Load() != 0 {
			t.Errorf("POST %s sin identificarse llamó a la acción", tc.path)
		}
	}
}

func TestRequireScrumMaster_SinProyecto(t *testing.T) {
	srv := newScrumMasterTestServer()
	for _, ac := range []*ActiveContext{nil, {}} {
		h := &countingHandler{}
		rec := serveWithContext(srv.RequireScrumMaster(h), http.MethodPost, "/cierre", ac)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/contexto/proyecto?volver_a=%2Fcierre" {
			t.Errorf("POST /cierre sin proyecto = %d a %q; se esperaba 303 a elegir proyecto", rec.Code, rec.Header().Get("Location"))
		}
		if !strings.Contains(rec.Header().Get("Set-Cookie"), CookieFlash+"=") {
			t.Error("sin proyecto: se esperaba el mensaje de HU-20b para elegir proyecto")
		}
		if h.calls.Load() != 0 {
			t.Error("sin proyecto se llamó a la acción")
		}
	}
}

func TestRequireScrumMasterOfPathProject_OtroProyecto(t *testing.T) {
	srv := newScrumMasterTestServer()
	const pattern = "POST /proyectos/{id}/modificar"
	// serve registra el envoltorio en un mux para que exista el valor de ruta {id}.
	serve := func(h *countingHandler, id string, ac *ActiveContext) *httptest.ResponseRecorder {
		mux := http.NewServeMux()
		mux.Handle(pattern, srv.RequireScrumMasterOfPathProject(h))
		return serveWithContext(mux, http.MethodPost, "/proyectos/"+id+"/modificar", ac)
	}

	t.Run("otro proyecto, inválido o inexistente", func(t *testing.T) {
		// Ana es Scrum Master del proyecto activo (1) y aun así no puede guardar otro.
		for _, id := range []string{"2", "2147483647", "0", "-1", "abc", "01", "2147483648"} {
			h := &countingHandler{}
			rec := serve(h, id, scrumMasterContext())
			assertForbidden(t, rec, h, MsgScrumMasterOtherProject)
		}
	})

	t.Run("el proyecto activo sigue las reglas normales", func(t *testing.T) {
		h := &countingHandler{}
		if rec := serve(h, "1", scrumMasterContext()); rec.Code != http.StatusNoContent || h.calls.Load() != 1 {
			t.Errorf("Scrum Master sobre el proyecto activo = %d con %d llamadas; se esperaba la acción", rec.Code, h.calls.Load())
		}
		h = &countingHandler{}
		assertForbidden(t, serve(h, "1", memberContext()), h, MsgScrumMasterOnly)
		h = &countingHandler{}
		assertForbidden(t, serve(h, "2", noScrumMasterContext()), h, MsgNoScrumMaster)
	})

	t.Run("primero el proyecto y la identificación", func(t *testing.T) {
		h := &countingHandler{}
		rec := serve(h, "2", nil)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/contexto/proyecto?volver_a=%2Fproyectos" {
			t.Errorf("sin proyecto = %d a %q; se esperaba 303 a elegir proyecto", rec.Code, rec.Header().Get("Location"))
		}
		ac := scrumMasterContext()
		ac.MemberID, ac.MemberName, ac.MemberRole = 0, "", ""
		rec = serve(h, "2", ac)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/contexto/integrante?volver_a=%2Fproyectos" {
			t.Errorf("sin identificarse = %d a %q; se esperaba 303 a identificarse", rec.Code, rec.Header().Get("Location"))
		}
		if h.calls.Load() != 0 {
			t.Error("sin proyecto o sin identificarse se llamó a la acción")
		}
	})
}
