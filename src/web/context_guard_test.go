package web

import (
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// Pruebas de handler de US5 (HU-20b, T041 y T049): secciones que requieren contexto.
// Usan las secciones de prueba de testhelpers_test.go: /libre (sin requisitos),
// /con-proyecto (RequiresProject) y /con-integrante (RequiresMember).

func volverAQuery(ruta string) string { return "volver_a=" + url.QueryEscape(ruta) }

func TestSeccion_RequiereProyectoSinProyectoRedirigeConMensajeYVuelveAlElegir(t *testing.T) {
	app := newCtxApp(t, nil)
	c := app.client(false)

	resp, _ := app.get(t, c, "/con-proyecto")

	want := pathSelectProject + "?" + volverAQuery("/con-proyecto")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
		t.Fatalf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, resp.Header.Get("Location"))
	}
	_, pantalla := app.get(t, c, resp.Header.Get("Location"))
	if !strings.Contains(pantalla, MsgProjectRequired) || !strings.Contains(pantalla, `name="proyecto_id"`) {
		t.Errorf("debía verse %q con la lista de proyectos", MsgProjectRequired)
	}

	// Al elegir uno, vuelve a la sección que había pedido.
	resp, _ = app.selectProject(t, c, id64(app.fx.projectA), "/con-proyecto")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/con-proyecto" {
		t.Fatalf("se esperaba 303 a /con-proyecto, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	r, body := app.get(t, c, "/con-proyecto")
	if r.StatusCode != http.StatusOK || !strings.Contains(body, "<h1>Con proyecto</h1>") || !strings.Contains(body, app.fx.nameA) {
		t.Errorf("la sección debía mostrarse con el proyecto activo (estado %d)", r.StatusCode)
	}
}

func TestSeccion_ElDestinoConservaLaConsulta(t *testing.T) {
	app := newCtxApp(t, nil)
	c := app.client(false)

	resp, _ := app.get(t, c, "/con-proyecto?pestana=2")

	want := pathSelectProject + "?" + volverAQuery("/con-proyecto?pestana=2")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
		t.Fatalf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, resp.Header.Get("Location"))
	}
	resp, _ = app.selectProject(t, c, id64(app.fx.projectA), "/con-proyecto?pestana=2")
	if resp.Header.Get("Location") != "/con-proyecto?pestana=2" {
		t.Errorf("al volver debía conservarse la consulta, se obtuvo %q", resp.Header.Get("Location"))
	}
}

func TestSeccion_SinRequisitosSeMuestraSinProyecto(t *testing.T) {
	app := newCtxApp(t, nil)
	for _, ruta := range []string{"/libre", "/", "/contexto/proyecto"} {
		resp, _ := app.get(t, app.client(false), ruta)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: se esperaba 200 sin proyecto, se obtuvo %d", ruta, resp.StatusCode)
		}
	}
}

func TestSeccion_ConProyectoYSinIntegranteLaSeccionDeProyectoSeMuestra(t *testing.T) {
	app := newCtxApp(t, nil)
	c := app.client(false)
	app.selectProject(t, c, id64(app.fx.projectA), "/libre")

	if resp, _ := app.get(t, c, "/con-proyecto"); resp.StatusCode != http.StatusOK {
		t.Errorf("una sección que solo requiere proyecto no pide integrante: se obtuvo %d", resp.StatusCode)
	}
}

func TestSeccion_RequiereIntegranteGuiaPorProyectoEIdentificacionYVuelve(t *testing.T) {
	app := newCtxApp(t, nil)
	fx := app.fx
	c := app.client(false)

	// Sin proyecto: primero se pide el proyecto, y se recuerda la sección pedida.
	resp, _ := app.get(t, c, "/con-integrante")
	if want := pathSelectProject + "?" + volverAQuery("/con-integrante"); resp.Header.Get("Location") != want {
		t.Fatalf("se esperaba ir a %q, se obtuvo %q", want, resp.Header.Get("Location"))
	}

	// Con proyecto pero sin identificarse: se pide identificarse, con un mensaje.
	app.selectProject(t, c, id64(fx.projectA), "/con-integrante")
	resp, _ = app.get(t, c, "/con-integrante")
	want := pathSelectMember + "?" + volverAQuery("/con-integrante")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
		t.Fatalf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, resp.Header.Get("Location"))
	}
	_, pantalla := app.get(t, c, resp.Header.Get("Location"))
	if !strings.Contains(pantalla, MsgMemberRequired) || !strings.Contains(pantalla, fx.anaName) {
		t.Errorf("debía verse %q con los integrantes del proyecto", MsgMemberRequired)
	}

	// Al identificarse vuelve a la sección pedida, y ahora se muestra.
	resp, _ = app.selectMember(t, c, id64(fx.ana), "/con-integrante")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/con-integrante" {
		t.Fatalf("se esperaba 303 a /con-integrante, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if r, body := app.get(t, c, "/con-integrante"); r.StatusCode != http.StatusOK || !strings.Contains(body, "<h1>Con integrante</h1>") {
		t.Errorf("la sección debía mostrarse tras identificarse (estado %d)", r.StatusCode)
	}
}

func TestSeccion_EnvioSinIdentificarseNoEjecutaLaAccionYVuelveAlOrigen(t *testing.T) {
	var ejecuciones atomic.Int32
	app := newCtxApp(t, nil, func(s *Server) {
		err := s.Register(Section{Name: "Accion", Path: "/accion", RequiresMember: true}, Routes{
			"POST /accion/guardar": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ejecuciones.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}),
		})
		if err != nil {
			t.Errorf("no se pudo registrar la sección de prueba: %v", err)
		}
	})
	c := app.client(false)
	app.selectProject(t, c, id64(app.fx.projectA), "/libre")

	resp, _ := app.post(t, c, "/accion/guardar", url.Values{"dato": {"1"}})

	// Un envío vuelve a la sección de origen (por GET), no a la dirección del envío.
	want := pathSelectMember + "?" + volverAQuery("/accion")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != want {
		t.Fatalf("se esperaba 303 a %q, se obtuvo %d a %q", want, resp.StatusCode, resp.Header.Get("Location"))
	}
	if ejecuciones.Load() != 0 {
		t.Fatalf("sin identificarse la acción no debía ejecutarse")
	}

	app.selectMember(t, c, id64(app.fx.ana), "/accion")
	resp, _ = app.post(t, c, "/accion/guardar", url.Values{"dato": {"1"}})
	if resp.StatusCode != http.StatusNoContent || ejecuciones.Load() != 1 {
		t.Errorf("identificada, la acción debía ejecutarse una vez (estado %d, ejecuciones %d)", resp.StatusCode, ejecuciones.Load())
	}
}

func TestDestinoDeRegreso_ExternoOMalformadoVaAlInicio(t *testing.T) {
	app := newCtxApp(t, nil)
	invalidos := []string{
		"//otro.com", "https://otro.com", `\\otro.com`, `/\otro.com`,
		"/no-existe", "/contexto/proyecto", "javascript:alert(1)", "backlog",
	}
	for _, destino := range invalidos {
		t.Run(destino, func(t *testing.T) {
			c := app.client(false)

			resp, _ := app.selectProject(t, c, id64(app.fx.projectA), destino)
			if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
				t.Errorf("al elegir proyecto: se esperaba 303 a /, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
			}
			resp, _ = app.selectMember(t, c, id64(app.fx.ana), destino)
			if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
				t.Errorf("al identificarse: se esperaba 303 a /, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
			}
		})
	}
}