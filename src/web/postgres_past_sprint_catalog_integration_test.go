package web

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// HU-16, T006: el catálogo de "Sprints anteriores" contra PostgreSQL con los datos de T005.

// pastStoryLine resume una fila del registro de cierre para comparar: título, Story Points
// (0 = sin estimar) y estado al cerrar.
type pastStoryLine struct {
	Title  string
	Points int
	Status string
}

func pastStoryLines(stories []PastSprintStory) []pastStoryLine {
	lines := []pastStoryLine{}
	for _, s := range stories {
		line := pastStoryLine{Title: s.Title, Status: s.FinalStatus}
		if s.StoryPoints != nil {
			line.Points = *s.StoryPoints
		}
		lines = append(lines, line)
	}
	return lines
}

func TestPostgresPastSprintCatalog(t *testing.T) {
	db := openContextTestDB(t)
	f := newPastSprintFixture(t, db)
	catalog := NewPostgreSQLPastSprintCatalog(db)
	ctx := context.Background()
	day := func(s string) time.Time {
		d, _ := time.Parse("2006-01-02", s)
		return d
	}

	t.Run("Sprints finalizados de «Alfa», del más reciente al más antiguo, sin el activo", func(t *testing.T) {
		got, err := catalog.FinishedSprints(ctx, f.alfa)
		if err != nil {
			t.Fatalf("FinishedSprints devolvió error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("se esperaban los Sprints 2 y 1, se obtuvo %+v", got)
		}
		want := []PastSprintSummary{
			{ID: f.sprints["Alfa 2"], Number: 2, Goal: "Gestionar perfiles y reportes", StartDate: day("2026-09-01"), EndDate: day("2026-09-14")},
			{ID: f.sprints["Alfa 1"], Number: 1, Goal: "Dar de alta usuarios", StartDate: day("2026-08-18"), EndDate: day("2026-08-31")},
		}
		for i := range want {
			g := got[i]
			if g.ID != want[i].ID || g.Number != want[i].Number || g.Goal != want[i].Goal ||
				!sameDay(g.StartDate, want[i].StartDate) || !sameDay(g.EndDate, want[i].EndDate) {
				t.Errorf("Sprint %d: se esperaba %+v, se obtuvo %+v", i, want[i], g)
			}
		}
	})

	t.Run("«Beta» sin Sprints finalizados: lista vacía sin error", func(t *testing.T) {
		got, err := catalog.FinishedSprints(ctx, f.beta)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("se esperaba una lista vacía, se obtuvo %+v, %v", got, err)
		}
	})

	t.Run("los Sprints de «Gamma» no aparecen con «Alfa»", func(t *testing.T) {
		got, _ := catalog.FinishedSprints(ctx, f.alfa)
		for _, s := range got {
			if s.ID == f.sprints["Gamma 1"] || s.ID == f.sprints["Gamma 2"] {
				t.Errorf("apareció un Sprint de «Gamma»: %+v", s)
			}
		}
	})

	details := []struct {
		name    string
		project int64
		sprint  string
		number  int
		want    []pastStoryLine
	}{
		{"detalle del Sprint 2", f.alfa, "Alfa 2", 2, []pastStoryLine{
			{"Login", 5, "Hecho"}, {"Tablero", 3, "Hecho"}, {"Perfil", 2, "Hecho"}, {"Reportes", 8, "En progreso"},
		}},
		{"detalle del Sprint 1", f.alfa, "Alfa 1", 1, []pastStoryLine{{"Alta", 3, "Hecho"}}},
		{"detalle del Sprint 1 de «Gamma»", f.gamma, "Gamma 1", 1, []pastStoryLine{
			{"Migración", 5, "En progreso"}, {"Respaldo", 0, "En Sprint"},
		}},
		{"Sprint 2 de «Gamma» sin registro de cierre", f.gamma, "Gamma 2", 2, []pastStoryLine{}},
	}
	for _, d := range details {
		t.Run(d.name, func(t *testing.T) {
			got, err := catalog.SprintDetail(ctx, d.project, f.sprints[d.sprint])
			if err != nil {
				t.Fatalf("SprintDetail devolvió error: %v", err)
			}
			if got.ID != f.sprints[d.sprint] || got.Number != d.number {
				t.Errorf("Sprint inesperado: %+v", got.PastSprintSummary)
			}
			if lines := pastStoryLines(got.Stories); !reflect.DeepEqual(lines, d.want) {
				t.Errorf("se esperaba %+v, se obtuvo %+v", d.want, lines)
			}
		})
	}

	t.Run("«Respaldo» sin estimar llega como nil", func(t *testing.T) {
		got, _ := catalog.SprintDetail(ctx, f.gamma, f.sprints["Gamma 1"])
		for _, s := range got.Stories {
			if s.Title == "Respaldo" && s.StoryPoints != nil {
				t.Errorf("«Respaldo» debía llegar sin Story Points, llegó %d", *s.StoryPoints)
			}
		}
	})

	for name, sprintID := range map[string]int64{
		"Sprint 3 en curso":                     f.sprints["Alfa 3"],
		"Sprint 1 de «Gamma» pedido con «Alfa»": f.sprints["Gamma 1"],
		"Sprint inexistente":                    2147483647,
	} {
		t.Run(name+" → NotFound", func(t *testing.T) {
			_, err := catalog.SprintDetail(ctx, f.alfa, sprintID)
			if KindOf(err) != KindNotFound {
				t.Errorf("se esperaba NotFound, se obtuvo %v", err)
			}
		})
	}
}
