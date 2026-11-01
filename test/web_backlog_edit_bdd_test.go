package test

import (
	"net/http"
	"strings"
	"testing"
)

// HU-13: escenarios BDD de la consulta y edición del Product Backlog, con la aplicación completa
// (contexto de HU-20b, Backlog de HU-22) contra PostgreSQL.

// hu13BDDStories carga «Login» (Media, la primera, Pendiente, 5 SP), «Pagos» (Alta, Hecho),
// «Reportes» (Baja, En Sprint) y «Perfil» (Media, después de «Login», Pendiente).
func hu13BDDStories(t *testing.T, app *hu22BDDApp) map[string]int64 {
	t.Helper()
	ids := map[string]int64{}
	for _, s := range []struct {
		title, priority, status, createdAt string
		points                             any
	}{
		{"Login", "Media", "Pendiente", "2026-09-01 10:00:00+00", 5},
		{"Pagos", "Alta", "Hecho", "2026-09-02 10:00:00+00", 3},
		{"Reportes", "Baja", "En Sprint", "2026-09-03 10:00:00+00", nil},
		{"Perfil", "Media", "Pendiente", "2026-09-04 10:00:00+00", 2},
	} {
		var id int64
		if err := app.db.QueryRow(`INSERT INTO user_stories (project_id, title, description, priority, status, story_points, created_at)
			VALUES ($1, $2, 'Descripción de '||$2, $3, $4, $5, $6::timestamptz) RETURNING id`,
			app.projectID, s.title, s.priority, s.status, s.points, s.createdAt).Scan(&id); err != nil {
			t.Fatalf("create HU-13 BDD story: %v", err)
		}
		if _, err := app.db.Exec(`INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, 1, 'Criterio de '||$2)`,
			id, s.title); err != nil {
			t.Fatalf("create HU-13 BDD criterion: %v", err)
		}
		ids[s.title] = id
	}
	return ids
}

// hu13BDDOrder devuelve los títulos de la tabla en el orden en que aparecen.
func hu13BDDOrder(body string, titles ...string) string {
	type found struct {
		title string
		at    int
	}
	var got []found
	for _, title := range titles {
		if i := strings.Index(body, "<td>"+title+"</td>"); i >= 0 {
			got = append(got, found{title, i})
		}
	}
	for i := 1; i < len(got); i++ {
		for j := i; j > 0 && got[j].at < got[j-1].at; j-- {
			got[j], got[j-1] = got[j-1], got[j]
		}
	}
	out := make([]string, len(got))
	for i, f := range got {
		out[i] = f.title
	}
	return strings.Join(out, ", ")
}

func TestWebBacklogEditBDD_ListadoOrdenado(t *testing.T) {
	t.Run("Given historias de distintas prioridades, When abro el Product Backlog, Then se listan por prioridad y antigüedad", func(t *testing.T) {
		app := newHU22BDDApp(t)
		hu13BDDStories(t, app)

		resp, body := app.request(t, http.MethodGet, "/backlog", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /backlog = %d; se esperaba 200", resp.StatusCode)
		}
		if got := hu13BDDOrder(body, "Login", "Pagos", "Reportes", "Perfil"); got != "Pagos, Login, Perfil, Reportes" {
			t.Errorf("orden = [%s]; se esperaba [Pagos, Login, Perfil, Reportes]", got)
		}
	})
}

func TestWebBacklogEditBDD_FiltroPorEstado(t *testing.T) {
	t.Run("Given historias en distintos estados, When filtro por Pendiente y después por Todos, Then veo las pendientes y luego todas", func(t *testing.T) {
		app := newHU22BDDApp(t)
		hu13BDDStories(t, app)

		_, body := app.request(t, http.MethodGet, "/backlog?estado=Pendiente", nil)
		if got := hu13BDDOrder(body, "Login", "Pagos", "Reportes", "Perfil"); got != "Login, Perfil" {
			t.Errorf("con Pendiente = [%s]; se esperaba [Login, Perfil]", got)
		}
		_, body = app.request(t, http.MethodGet, "/backlog?estado=Todos", nil)
		if got := hu13BDDOrder(body, "Login", "Pagos", "Reportes", "Perfil"); got != "Pagos, Login, Perfil, Reportes" {
			t.Errorf("con Todos = [%s]; se esperaba la lista completa", got)
		}
	})
}
