package web

import (
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
