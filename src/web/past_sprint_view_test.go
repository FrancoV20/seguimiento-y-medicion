package web

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// HU-16, T010 y T017: armado de las vistas de "Sprints anteriores", sin base.

func pastDay(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func TestPastSprintView(t *testing.T) {
	t.Run("fecha dd/mm/aaaa", func(t *testing.T) {
		if got := pastSprintDateText(pastDay("2026-09-01")); got != "01/09/2026" {
			t.Errorf("se esperaba 01/09/2026, se obtuvo %q", got)
		}
	})

	t.Run("lista de «Alfa»: Sprint 2 y después Sprint 1", func(t *testing.T) {
		view := buildPastSprintListView([]PastSprintSummary{
			{ID: 22, Number: 2, Goal: "Gestionar perfiles y reportes", StartDate: pastDay("2026-09-01"), EndDate: pastDay("2026-09-14")},
			{ID: 11, Number: 1, Goal: "Dar de alta usuarios", StartDate: pastDay("2026-08-18"), EndDate: pastDay("2026-08-31")},
		})
		want := []pastSprintItem{
			{Label: "Sprint 2", Goal: "Gestionar perfiles y reportes", Dates: "Del 01/09/2026 al 14/09/2026", Link: "/sprints-anteriores/22"},
			{Label: "Sprint 1", Goal: "Dar de alta usuarios", Dates: "Del 18/08/2026 al 31/08/2026", Link: "/sprints-anteriores/11"},
		}
		if len(view.Sprints) != len(want) {
			t.Fatalf("se esperaban %d Sprints, se obtuvo %+v", len(want), view.Sprints)
		}
		for i := range want {
			if view.Sprints[i] != want[i] {
				t.Errorf("Sprint %d: se esperaba %+v, se obtuvo %+v", i, want[i], view.Sprints[i])
			}
		}
		if view.Empty != "" {
			t.Errorf("con Sprints no debía haber mensaje de lista vacía, hay %q", view.Empty)
		}
	})

	t.Run("lista vacía", func(t *testing.T) {
		view := buildPastSprintListView(nil)
		if len(view.Sprints) != 0 || view.Empty != "Aún no hay Sprints anteriores" {
			t.Errorf("se esperaba «Aún no hay Sprints anteriores», se obtuvo %+v", view)
		}
	})
}

func pastPoints(n int) *int { return &n }

func TestPastSprintDetailView(t *testing.T) {
	t.Run("Story Points", func(t *testing.T) {
		if got := pastSprintPointsText(nil); got != "Sin estimar" {
			t.Errorf("sin Story Points se esperaba «Sin estimar», se obtuvo %q", got)
		}
		if got := pastSprintPointsText(pastPoints(5)); got != "5" {
			t.Errorf("se esperaba «5», se obtuvo %q", got)
		}
	})

	summary := func(number int, goal string) PastSprintSummary {
		return PastSprintSummary{ID: 9, Number: number, Goal: goal, StartDate: pastDay("2026-09-01"), EndDate: pastDay("2026-09-14")}
	}
	story := func(id int64, title string, points *int, status string) PastSprintStory {
		return PastSprintStory{StoryID: id, Title: title, StoryPoints: points, FinalStatus: status}
	}
	row := func(title, points, status string, returned bool) pastSprintRow {
		r := pastSprintRow{Title: title, StoryPoints: points, FinalStatus: status}
		if returned {
			r.Returned = "volvió al Backlog"
		}
		return r
	}
	cases := []struct {
		name     string
		detail   PastSprintDetail
		velocity int
		want     pastSprintDetailView
	}{
		{
			name: "Sprint 2 de «Alfa»",
			detail: PastSprintDetail{PastSprintSummary: summary(2, "Gestionar perfiles y reportes"), Stories: []PastSprintStory{
				story(2, "Login", pastPoints(5), "Hecho"), story(3, "Tablero", pastPoints(3), "Hecho"),
				story(4, "Perfil", pastPoints(2), "Hecho"), story(5, "Reportes", pastPoints(8), "En progreso"),
			}},
			velocity: 10,
			want: pastSprintDetailView{
				Completed: []pastSprintRow{row("Login", "5", "Hecho", false), row("Tablero", "3", "Hecho", false), row("Perfil", "2", "Hecho", false)},
				Pending:   []pastSprintRow{row("Reportes", "8", "En progreso", true)},
			},
		},
		{
			name:     "Sprint 1 de «Alfa»: todas completadas",
			detail:   PastSprintDetail{PastSprintSummary: summary(1, "Dar de alta usuarios"), Stories: []PastSprintStory{story(1, "Alta", pastPoints(3), "Hecho")}},
			velocity: 3,
			want: pastSprintDetailView{
				Completed:    []pastSprintRow{row("Alta", "3", "Hecho", false)},
				AllCompleted: "Todas las historias se completaron",
			},
		},
		{
			name: "Sprint 1 de «Gamma»: ninguna completada",
			detail: PastSprintDetail{PastSprintSummary: summary(1, "Migrar datos"), Stories: []PastSprintStory{
				story(8, "Migración", pastPoints(5), "En progreso"), story(9, "Respaldo", nil, "En Sprint"),
			}},
			velocity: 0,
			want: pastSprintDetailView{
				NoneCompleted: "Ninguna historia se completó",
				Pending:       []pastSprintRow{row("Migración", "5", "En progreso", true), row("Respaldo", "Sin estimar", "En Sprint", true)},
			},
		},
		{
			name:     "Sprint 2 de «Gamma»: sin registro de cierre",
			detail:   PastSprintDetail{PastSprintSummary: summary(2, "Auditar accesos"), Stories: []PastSprintStory{}},
			velocity: 2,
			want:     pastSprintDetailView{NoRecord: "No hay registro de las historias de este Sprint"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildPastSprintDetailView(c.detail, c.velocity)
			wantLabel := fmt.Sprintf("Sprint %d", c.detail.Number)
			if got.Label != wantLabel || got.Goal != "Objetivo: "+c.detail.Goal || got.Dates != "Del 01/09/2026 al 14/09/2026" ||
				got.Velocity != fmt.Sprintf("Velocidad: %d Story Points", c.velocity) || got.Back != "Volver a Sprints anteriores" {
				t.Errorf("encabezado inesperado: %+v", got)
			}
			if !reflect.DeepEqual(got.Completed, c.want.Completed) || !reflect.DeepEqual(got.Pending, c.want.Pending) {
				t.Errorf("tablas: se esperaba %+v / %+v, se obtuvo %+v / %+v", c.want.Completed, c.want.Pending, got.Completed, got.Pending)
			}
			if got.NoRecord != c.want.NoRecord || got.AllCompleted != c.want.AllCompleted || got.NoneCompleted != c.want.NoneCompleted {
				t.Errorf("mensajes: se esperaba %q/%q/%q, se obtuvo %q/%q/%q", c.want.NoRecord, c.want.AllCompleted, c.want.NoneCompleted,
					got.NoRecord, got.AllCompleted, got.NoneCompleted)
			}
		})
	}
}
