package web

import (
	"slices"
	"testing"
	"time"
)

// HU-12 (T014): textos y armado de la vista de "Estado del proyecto", sin base ni servidor.

func viewDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestProjectStatusView(t *testing.T) {
	today := time.Date(2026, time.October, 28, 10, 30, 0, 0, time.Local)

	t.Run("porcentaje con coma y espacio", func(t *testing.T) {
		for raw, want := range map[string]string{
			"50.00": "50,00 %", "0.00": "0,00 %", "100.00": "100,00 %", "66.67": "66,67 %",
		} {
			if got := projectStatusPercentText(raw); got != want {
				t.Errorf("projectStatusPercentText(%q) = %q, se esperaba %q", raw, got, want)
			}
		}
	})

	t.Run("fecha dd/mm/aaaa", func(t *testing.T) {
		if got := projectStatusDateText(viewDay(2026, time.October, 1)); got != "01/10/2026" {
			t.Errorf("projectStatusDateText() = %q, se esperaba \"01/10/2026\"", got)
		}
	})

	t.Run("Alfa con Sprint en curso", func(t *testing.T) {
		summary := ProjectStatusSummary{
			Name:      "Alfa",
			StartDate: viewDay(2026, time.October, 1),
			EndDate:   viewDay(2026, time.November, 30),
			Counts:    ProjectStatusCounts{Pending: 2, InSprint: 1, InProgress: 1, Done: 4},
		}
		board := BoardView{
			Number: 2, Label: "Sprint 2", Goal: "Pagos online",
			StartDate: viewDay(2026, time.October, 19), EndDate: viewDay(2026, time.November, 1),
		}

		view := buildProjectStatusView(summary, board, true, today)

		if view.Name != "Alfa" {
			t.Errorf("Name = %q, se esperaba \"Alfa\"", view.Name)
		}
		if view.State != "Estado: En curso" {
			t.Errorf("State = %q, se esperaba \"Estado: En curso\"", view.State)
		}
		if view.Dates != "Del 01/10/2026 al 30/11/2026" {
			t.Errorf("Dates = %q", view.Dates)
		}
		if view.Progress != "4 de 8 historias completadas (50,00 %)" {
			t.Errorf("Progress = %q", view.Progress)
		}
		wantRows := []projectStatusRow{{"Pendiente", 2}, {"En Sprint", 1}, {"En progreso", 1}, {"Hecho", 4}}
		if !slices.Equal(view.Rows, wantRows) {
			t.Errorf("Rows = %v, se esperaba %v", view.Rows, wantRows)
		}
		if !view.HasSprint {
			t.Fatalf("HasSprint = false, se esperaba true")
		}
		if view.SprintLabel != "Sprint 2 en curso" || view.SprintGoal != "Objetivo: Pagos online" ||
			view.SprintDates != "Del 19/10/2026 al 01/11/2026" {
			t.Errorf("Sprint = %q / %q / %q", view.SprintLabel, view.SprintGoal, view.SprintDates)
		}
	})

	t.Run("Beta sin historias ni Sprint", func(t *testing.T) {
		summary := ProjectStatusSummary{
			Name:      "Beta",
			StartDate: viewDay(2026, time.December, 1),
			EndDate:   viewDay(2027, time.January, 31),
		}

		view := buildProjectStatusView(summary, BoardView{}, false, today)

		if view.State != "Estado: No iniciado" {
			t.Errorf("State = %q, se esperaba \"Estado: No iniciado\"", view.State)
		}
		if view.Progress != "0 de 0 historias completadas (0,00 %)" {
			t.Errorf("Progress = %q", view.Progress)
		}
		wantRows := []projectStatusRow{{"Pendiente", 0}, {"En Sprint", 0}, {"En progreso", 0}, {"Hecho", 0}}
		if !slices.Equal(view.Rows, wantRows) {
			t.Errorf("Rows = %v, se esperaba %v", view.Rows, wantRows)
		}
		if view.HasSprint || view.NoSprint != "No hay un Sprint en curso" {
			t.Errorf("HasSprint = %v, NoSprint = %q", view.HasSprint, view.NoSprint)
		}
	})
}
