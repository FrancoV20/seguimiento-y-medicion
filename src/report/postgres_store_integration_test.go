package report

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
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

// T039: LoadProjectReport contra PostgreSQL real.
func TestPostgresReportStore_Proyecto(t *testing.T) {
	db := openReportTestDB(t)
	f := newReportFixture(t, db)
	store := NewPostgreSQLStore(db)
	ctx := context.Background()

	t.Run("«Alfa»", func(t *testing.T) {
		data, err := store.LoadProjectReport(ctx, f.projects["Alfa"])
		if err != nil {
			t.Fatalf("LoadProjectReport: %v", err)
		}
		if data.ID != f.projects["Alfa"] || data.Name != "Alfa" ||
			dateText(data.StartDate) != "18/08/2026" || dateText(data.EndDate) != "30/11/2026" {
			t.Errorf("encabezado = %d %q %v %v", data.ID, data.Name, data.StartDate, data.EndDate)
		}
		if want := []string{"Ana", "Beto"}; !reflect.DeepEqual(data.Members, want) {
			t.Errorf("integrantes = %v, se esperaba %v", data.Members, want)
		}
		if want := (StoryCounts{Pending: 1, InSprint: 1, InProgress: 1, Done: 4}); data.Stories != want {
			t.Errorf("historias por estado = %+v, se esperaba %+v", data.Stories, want)
		}
		wantSprints := []ProjectSprint{
			{ID: f.sprints["Alfa 1"], Number: 1, Status: "Finalizado"},
			{ID: f.sprints["Alfa 2"], Number: 2, Status: "Finalizado"},
			{ID: f.sprints["Alfa 3"], Number: 3, Status: "Activo"},
		}
		if !reflect.DeepEqual(data.Sprints, wantSprints) {
			t.Errorf("Sprints = %+v, se esperaba %+v", data.Sprints, wantSprints)
		}
		if data.EstimatedTotal != 3800 || data.WithoutEstimate != 2 {
			t.Errorf("horas estimadas %d y sin estimar %d; se esperaba 3800 y 2", data.EstimatedTotal, data.WithoutEstimate)
		}
		if want := []MemberHours{{"Ana", 2300}, {"Beto", 700}}; !reflect.DeepEqual(data.MemberHours, want) || data.RegisteredTotal != 3000 {
			t.Errorf("esfuerzo = %+v (total %d), se esperaba %+v (total 3000)", data.MemberHours, data.RegisteredTotal, want)
		}
		if data.DefectsOpen != 2 || data.DefectsResolved != 1 {
			t.Errorf("defectos: %d abiertos y %d resueltos; se esperaba 2 y 1", data.DefectsOpen, data.DefectsResolved)
		}
		wantOpen := []OpenDefect{
			{Description: "La foto de perfil no se guarda", Severity: "Media", StoryTitle: "Perfil", DetectionSprint: 2},
			{Description: "El PDF sale en blanco", Severity: "Crítica", StoryTitle: "Exportar", DetectionSprint: 3},
		}
		if !reflect.DeepEqual(data.OpenDefects, wantOpen) {
			t.Errorf("abiertos = %+v, se esperaba %+v", data.OpenDefects, wantOpen)
		}
		for _, m := range data.MemberHours {
			if m.Name == "Caro" {
				t.Error("aparecen horas de Caro, que solo trabajó en «Delta»")
			}
		}
	})

	t.Run("«Beta» vacío", func(t *testing.T) {
		data, err := store.LoadProjectReport(ctx, f.projects["Beta"])
		if err != nil {
			t.Fatalf("LoadProjectReport: %v", err)
		}
		if !reflect.DeepEqual(data.Members, []string{"Ana"}) || data.Stories != (StoryCounts{}) || len(data.Sprints) != 0 ||
			data.EstimatedTotal != 0 || data.WithoutEstimate != 0 || data.RegisteredTotal != 0 || len(data.MemberHours) != 0 ||
			data.DefectsOpen != 0 || data.DefectsResolved != 0 || len(data.OpenDefects) != 0 {
			t.Errorf("«Beta» debía venir vacío con su integrante Ana: %+v", data)
		}
	})

	t.Run("proyecto inexistente", func(t *testing.T) {
		if _, err := store.LoadProjectReport(ctx, 2147483647); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("se esperaba ErrProjectNotFound, se obtuvo %v", err)
		}
	})
}

// T039: cada fila del reporte del proyecto coincide con HU-07, HU-08, HU-10 y HU-12 reales.
func TestProjectReport_Coherencia(t *testing.T) {
	db := openReportTestDB(t)
	f := newReportFixture(t, db)
	metricsService := metrics.NewService(metrics.NewPostgreSQLMetricsStore(db))
	qualityService := metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(db))
	service := NewService(NewPostgreSQLStore(db), metricsService, qualityService, clock)
	ctx := context.Background()

	r, err := service.BuildProject(ctx, f.projects["Alfa"])
	if err != nil {
		t.Fatalf("BuildProject: %v", err)
	}
	var velocities []int
	for _, row := range r.Sprints {
		if row.Active {
			continue
		}
		id := f.sprints[fmt.Sprintf("Alfa %d", row.Number)]
		q, err := qualityService.SprintQuality(ctx, f.projects["Alfa"], id)
		if err != nil {
			t.Fatalf("SprintQuality: %v", err)
		}
		m, err := metricsService.CalculateMetrics(id)
		if err != nil {
			t.Fatalf("CalculateMetrics: %v", err)
		}
		if !reflect.DeepEqual(*row.Quality, q) || row.Metrics.Velocity != m.Velocity ||
			row.Metrics.DeviationPercentage != m.DeviationPercentage || row.Metrics.IsPartial != m.IsPartial {
			t.Errorf("Sprint %d: la fila no coincide con HU-07 y HU-10", row.Number)
		}
		velocities = append(velocities, m.Velocity)
	}
	if want := metrics.SummarizeVelocity(velocities); r.Velocity != want || want.Total != 13 {
		t.Errorf("velocidad = %+v, se esperaba %+v", r.Velocity, want)
	}
	if want := string(project.StatusOn(r.Data.StartDate, r.Data.EndDate, fixedNow)); r.Status != want {
		t.Errorf("estado = %q, se esperaba el de HU-12 %q", r.Status, want)
	}
	if want := project.ProgressPercentage(4, 7); r.Progress != want {
		t.Errorf("avance = %q, se esperaba el de HU-12 %q", r.Progress, want)
	}
	// HU-07 guarda en sprint_metrics; la limpieza de los datos de prueba lo borra.
}
