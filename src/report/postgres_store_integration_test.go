package report

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-09, T021 y T022: PostgreSQLStore contra PostgreSQL real con los datos de fixtures_test.go.

func loadSprint(t *testing.T, s *PostgreSQLStore, f *reportFixture, project, sprint string) Data {
	t.Helper()
	data, err := s.LoadSprintReport(context.Background(), f.projects[project], f.sprints[sprint])
	if err != nil {
		t.Fatalf("LoadSprintReport(%s, %s): %v", project, sprint, err)
	}
	return data
}

type storyRow struct {
	title     string
	status    string
	points    *int
	estimated *effort.Hours
	actual    effort.Hours
}

func storyRows(stories []Story) []storyRow {
	rows := make([]storyRow, 0, len(stories))
	for _, s := range stories {
		rows = append(rows, storyRow{s.Title, s.Status, s.StoryPoints, s.EstimatedHours, s.ActualHours})
	}
	return rows
}

func TestPostgresReportStore_Finalizado(t *testing.T) {
	db := openReportTestDB(t)
	f := newReportFixture(t, db)
	store := NewPostgreSQLStore(db)

	t.Run("Sprint 2 de «Alfa»", func(t *testing.T) {
		data := loadSprint(t, store, f, "Alfa", "Alfa 2")
		wantInfo := SprintInfo{ProjectName: "Alfa", Number: 2, Goal: "Gestionar perfiles y reportes",
			StartDate: day("2026-09-01"), EndDate: day("2026-09-14"), Status: "Finalizado"}
		if data.Sprint.ProjectName != wantInfo.ProjectName || data.Sprint.Number != wantInfo.Number ||
			data.Sprint.Goal != wantInfo.Goal || data.Sprint.Status != wantInfo.Status ||
			dateText(data.Sprint.StartDate) != "01/09/2026" || dateText(data.Sprint.EndDate) != "14/09/2026" {
			t.Errorf("encabezado = %+v, se esperaba %+v", data.Sprint, wantInfo)
		}
		wantStories := []storyRow{
			{"Login", "Hecho", points(5), hours(1000), 1100},
			{"Tablero", "Hecho", points(3), hours(600), 600},
			{"Perfil", "Hecho", points(2), nil, 400},
			{"Reportes", "En progreso", points(8), hours(1200), 300},
		}
		if got := storyRows(data.Stories); !reflect.DeepEqual(got, wantStories) {
			t.Errorf("historias = %+v, se esperaba %+v", got, wantStories)
		}
		wantMembers := []MemberHours{{"Ana", 1700}, {"Beto", 700}}
		if !reflect.DeepEqual(data.Members, wantMembers) {
			t.Errorf("integrantes = %+v, se esperaba %+v", data.Members, wantMembers)
		}
		wantDefects := []Defect{
			{Description: "El login no valida la contraseña", Severity: "Alta", Status: "Resuelto", StoryTitle: "Login", ResolutionSprint: 2},
			{Description: "La foto de perfil no se guarda", Severity: "Media", Status: "Abierto", StoryTitle: "Perfil"},
		}
		if !reflect.DeepEqual(data.Defects, wantDefects) {
			t.Errorf("defectos = %+v, se esperaba %+v", data.Defects, wantDefects)
		}
	})

	t.Run("Sprint 1 de «Gamma»: defecto resuelto en el Sprint 2 y sin esfuerzo", func(t *testing.T) {
		data := loadSprint(t, store, f, "Gamma", "Gamma 1")
		want := []Defect{{Description: "La migración pierde acentos", Severity: "Baja", Status: "Resuelto", StoryTitle: "G1", ResolutionSprint: 2}}
		if !reflect.DeepEqual(data.Defects, want) {
			t.Errorf("defectos = %+v, se esperaba %+v", data.Defects, want)
		}
		if len(data.Members) != 0 {
			t.Errorf("no debía haber integrantes con horas, se obtuvo %+v", data.Members)
		}
	})

	t.Run("Sprint 2 de «Gamma»: sin defectos detectados", func(t *testing.T) {
		if data := loadSprint(t, store, f, "Gamma", "Gamma 2"); len(data.Defects) != 0 {
			t.Errorf("no debía haber defectos detectados, se obtuvo %+v", data.Defects)
		}
	})

	t.Run("Sprint 3 de «Gamma»: finalizado sin registro de cierre", func(t *testing.T) {
		data := loadSprint(t, store, f, "Gamma", "Gamma 3")
		if data.Sprint.Status != "Finalizado" || len(data.Stories) != 0 || len(data.Members) != 0 || len(data.Defects) != 0 {
			t.Errorf("se esperaba un Sprint finalizado vacío, se obtuvo %+v", data)
		}
	})

	t.Run("Sprint 1 de «Delta»: horas de un integrante que no está en el proyecto", func(t *testing.T) {
		data := loadSprint(t, store, f, "Delta", "Delta 1")
		if want := []MemberHours{{"Caro", 100}}; !reflect.DeepEqual(data.Members, want) {
			t.Errorf("integrantes = %+v, se esperaba %+v", data.Members, want)
		}
	})
}

func TestPostgresReportStore_EnCurso(t *testing.T) {
	db := openReportTestDB(t)
	f := newReportFixture(t, db)
	data := loadSprint(t, NewPostgreSQLStore(db), f, "Alfa", "Alfa 3")

	if data.Sprint.Status != "Activo" || data.Sprint.Number != 3 {
		t.Errorf("encabezado = %+v", data.Sprint)
	}
	wantStories := []storyRow{
		{"Exportar", "En progreso", points(5), hours(600), 200},
		{"Ayuda", "En Sprint", nil, nil, 0},
	}
	if got := storyRows(data.Stories); !reflect.DeepEqual(got, wantStories) {
		t.Errorf("historias = %+v, se esperaba %+v", got, wantStories)
	}
	// Las horas de Ana en «Alta» y en el Sprint 2 no cuentan.
	if want := []MemberHours{{"Ana", 200}}; !reflect.DeepEqual(data.Members, want) {
		t.Errorf("integrantes = %+v, se esperaba %+v", data.Members, want)
	}
	want := []Defect{{Description: "El PDF sale en blanco", Severity: "Crítica", Status: "Abierto", StoryTitle: "Exportar"}}
	if !reflect.DeepEqual(data.Defects, want) {
		t.Errorf("defectos = %+v, se esperaba %+v", data.Defects, want)
	}
}

func TestPostgresReportStore_OtroProyecto(t *testing.T) {
	db := openReportTestDB(t)
	f := newReportFixture(t, db)
	store := NewPostgreSQLStore(db)
	ctx := context.Background()

	if _, err := store.LoadSprintReport(ctx, f.projects["Alfa"], f.sprints["Delta 1"]); !errors.Is(err, ErrSprintNotFound) {
		t.Errorf("Sprint de «Delta» pedido con «Alfa»: se esperaba ErrSprintNotFound, se obtuvo %v", err)
	}
	if _, err := store.LoadSprintReport(ctx, f.projects["Alfa"], 2147483647); !errors.Is(err, ErrSprintNotFound) {
		t.Errorf("Sprint inexistente: se esperaba ErrSprintNotFound, se obtuvo %v", err)
	}
	for _, key := range []string{"Alfa 1", "Alfa 2", "Alfa 3"} {
		data := loadSprint(t, store, f, "Alfa", key)
		for _, s := range data.Stories {
			if s.Title == "D1" {
				t.Errorf("%s: aparece la historia «D1» de «Delta»", key)
			}
		}
		for _, m := range data.Members {
			if m.Name == "Caro" {
				t.Errorf("%s: aparecen horas de Caro, que solo trabajó en «Delta»", key)
			}
		}
		for _, d := range data.Defects {
			if d.Description == "Defecto de Delta" {
				t.Errorf("%s: aparece el defecto de «Delta»", key)
			}
		}
	}
}

// T022: los números del reporte coinciden con HU-07 y HU-10 reales.
func TestReport_CoherenciaConHU07YHU10(t *testing.T) {
	db := openReportTestDB(t)
	f := newReportFixture(t, db)
	service := NewService(NewPostgreSQLStore(db),
		metrics.NewService(metrics.NewPostgreSQLMetricsStore(db)),
		metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db)), clock)

	r, err := service.Build(context.Background(), f.projects["Alfa"], f.sprints["Alfa 2"])
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m, q := r.Metrics, r.Quality
	if m.Velocity != 10 || m.Velocity != q.PointsCompleted {
		t.Errorf("velocidad %d y completados de HU-10 %d; se esperaba 10 en los dos", m.Velocity, q.PointsCompleted)
	}
	if m.DeviationPercentage != "6.25" || m.Warning != "Cálculo parcial: basado en 2 de 3 historias" {
		t.Errorf("desviación %q y aviso %q", m.DeviationPercentage, m.Warning)
	}
	if len(r.Data.Stories) != 4 || q.StoriesPlanned != 4 || q.CompletionPercentage != "75.00" {
		t.Errorf("historias %d, planificadas %d, avance %q", len(r.Data.Stories), q.StoriesPlanned, q.CompletionPercentage)
	}
	if q.PointsPlanned != 18 || q.PointsCompleted != 10 || q.DefectsDetected != 2 || q.DefectsResolved != 1 {
		t.Errorf("HU-10 = %+v", q)
	}
	// HU-07 guarda en sprint_metrics; la limpieza de los datos de prueba lo borra.
}
