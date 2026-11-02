package web

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/role"
)

// HU-28, T014: la estructura común muestra "Nombre (Rol)" del integrante identificado (escenario 1).

// roleOverrideStore devuelve el contexto real con otro texto de rol, para simular un valor que
// ParseRole no reconoce (la base lo impide con su CHECK).
type roleOverrideStore struct {
	ContextStore
	text string
}

func (s roleOverrideStore) ResolveContext(ctx context.Context, projectID, memberID int64) (ResolvedContext, error) {
	rc, err := s.ContextStore.ResolveContext(ctx, projectID, memberID)
	if rc.MemberBelongs {
		rc.MemberRole = role.Role(s.text)
	}
	return rc, err
}

func TestContexto_RolEnLaEstructuraComun(t *testing.T) {
	t.Run("Ana, Scrum Master", func(t *testing.T) {
		app := newCtxApp(t, nil)
		setTestScrumMaster(t, app.db, app.fx.projectA, app.fx.ana)
		c := app.client(false)
		app.selectProject(t, c, id64(app.fx.projectA), "/libre")
		app.selectMember(t, c, id64(app.fx.ana), "/libre")

		resp, body := app.get(t, c, "/libre")

		want := "Integrante: <strong>" + app.fx.anaName + " (Scrum Master)</strong>"
		if resp.StatusCode != http.StatusOK || !strings.Contains(body, want) {
			t.Errorf("se esperaba %q en la estructura común (estado %d)", want, resp.StatusCode)
		}
	})

	t.Run("Beto, Integrante", func(t *testing.T) {
		app := newCtxApp(t, nil)
		setTestScrumMaster(t, app.db, app.fx.projectA, app.fx.ana)
		c := app.client(false)
		app.selectProject(t, c, id64(app.fx.projectA), "/libre")
		app.selectMember(t, c, id64(app.fx.beto), "/libre")

		_, body := app.get(t, c, "/libre")

		want := "Integrante: <strong>" + app.fx.betoName + " (Integrante)</strong>"
		if !strings.Contains(body, want) {
			t.Errorf("se esperaba %q en la estructura común", want)
		}
		if strings.Contains(body, "(Scrum Master)") {
			t.Error("Beto no es Scrum Master y la estructura común lo muestra así")
		}
	})

	t.Run("sin integrante", func(t *testing.T) {
		app := newCtxApp(t, nil)
		setTestScrumMaster(t, app.db, app.fx.projectA, app.fx.ana)
		c := app.client(false)
		app.selectProject(t, c, id64(app.fx.projectA), "/libre")

		_, body := app.get(t, c, "/libre")

		if strings.Contains(body, "Integrante: <strong>") || strings.Contains(body, "(Scrum Master)") ||
			strings.Contains(body, "(Integrante)") {
			t.Error("sin integrante identificado no debe aparecer \"Nombre (Rol)\" en la estructura común")
		}
	})

	t.Run("rol no reconocido se muestra como Integrante", func(t *testing.T) {
		app := newCtxApp(t, func(s ContextStore) ContextStore { return roleOverrideStore{ContextStore: s, text: "Product Owner"} })
		c := app.client(false)
		app.selectProject(t, c, id64(app.fx.projectA), "/libre")
		app.selectMember(t, c, id64(app.fx.beto), "/libre")

		_, body := app.get(t, c, "/libre")

		want := "Integrante: <strong>" + app.fx.betoName + " (Integrante)</strong>"
		if !strings.Contains(body, want) || strings.Contains(body, "Product Owner") {
			t.Errorf("un rol no reconocido debía mostrarse como %q", want)
		}
	})
}
