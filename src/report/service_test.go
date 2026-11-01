package report

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// HU-09, T012: armado del reporte de un Sprint con un almacén y fuentes de HU-07 y HU-10 de prueba
// y un reloj fijo (28/10/2026 10:30).

var fixedNow = time.Date(2026, 10, 28, 10, 30, 0, 0, time.UTC)

func clock() time.Time { return fixedNow }

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func points(n int) *int { return &n }

func hours(h effort.Hours) *effort.Hours { return &h }

// Datos de ejemplo de la spec: proyecto «Alfa», integrantes Ana y Beto.

func alfaSprint1() Data {
	return Data{
		Sprint: SprintInfo{ProjectName: "Alfa", Number: 1, Goal: "Dar de alta usuarios",
			StartDate: day("2026-08-18"), EndDate: day("2026-08-31"), Status: "Finalizado"},
		Stories: []Story{{ID: 1, Title: "Alta", Status: "Hecho", StoryPoints: points(3), EstimatedHours: hours(400), ActualHours: 400}},
		Members: []MemberHours{{Name: "Ana", Hours: 400}},
		Defects: []Defect{},
	}
}

func alfaSprint1Metrics() metrics.SprintMetrics {
	return metrics.SprintMetrics{Velocity: 3, DeviationPercentage: "0.00"}
}

func alfaSprint1Quality() metrics.SprintQuality {
	return metrics.SprintQuality{StoriesPlanned: 1, StoriesCompleted: 1, CompletionPercentage: "100.00",
		PointsPlanned: 3, PointsCompleted: 3}
}

func alfaSprint2() Data {
	return Data{
		Sprint: SprintInfo{ProjectName: "Alfa", Number: 2, Goal: "Gestionar perfiles y reportes",
			StartDate: day("2026-09-01"), EndDate: day("2026-09-14"), Status: "Finalizado"},
		Stories: []Story{
			{ID: 2, Title: "Login", Status: "Hecho", StoryPoints: points(5), EstimatedHours: hours(1000), ActualHours: 1100},
			{ID: 3, Title: "Tablero", Status: "Hecho", StoryPoints: points(3), EstimatedHours: hours(600), ActualHours: 600},
			{ID: 4, Title: "Perfil", Status: "Hecho", StoryPoints: points(2), ActualHours: 400},
			{ID: 5, Title: "Reportes", Status: "En progreso", StoryPoints: points(8), EstimatedHours: hours(1200), ActualHours: 300},
		},
		Members: []MemberHours{{Name: "Ana", Hours: 1700}, {Name: "Beto", Hours: 700}},
		Defects: []Defect{
			{Description: "El login no valida la contraseña", Severity: "Alta", Status: "Resuelto", StoryTitle: "Login", ResolutionSprint: 2},
			{Description: "La foto de perfil no se guarda", Severity: "Media", Status: "Abierto", StoryTitle: "Perfil"},
		},
	}
}

func alfaSprint2Metrics() metrics.SprintMetrics {
	return metrics.SprintMetrics{Velocity: 10, DeviationPercentage: "6.25", IsPartial: true,
		Warning: "Cálculo parcial: basado en 2 de 3 historias"}
}

func alfaSprint2Quality() metrics.SprintQuality {
	return metrics.SprintQuality{StoriesPlanned: 4, StoriesCompleted: 3, CompletionPercentage: "75.00",
		PointsPlanned: 18, PointsCompleted: 10, DefectsDetected: 2, DefectsResolved: 1}
}

func alfaSprint3() Data {
	return Data{
		Sprint: SprintInfo{ProjectName: "Alfa", Number: 3, Goal: "Exportar e informar",
			StartDate: day("2026-09-15"), EndDate: day("2026-09-28"), Status: "Activo"},
		Stories: []Story{
			{ID: 6, Title: "Exportar", Status: "En progreso", StoryPoints: points(5), EstimatedHours: hours(600), ActualHours: 200},
			{ID: 7, Title: "Ayuda", Status: "En Sprint"},
		},
		Members: []MemberHours{{Name: "Ana", Hours: 200}},
		Defects: []Defect{{Description: "El PDF sale en blanco", Severity: "Crítica", Status: "Abierto", StoryTitle: "Exportar"}},
	}
}

// --- dobles de prueba ---

type fakeStore struct {
	data  Data
	err   error
	calls int

	// Reporte del proyecto (T038).
	project      ProjectData
	projectErr   error
	projectCalls int
}

func (s *fakeStore) LoadSprintReport(_ context.Context, _, _ int64) (Data, error) {
	s.calls++
	return s.data, s.err
}

func (s *fakeStore) LoadProjectReport(context.Context, int64) (ProjectData, error) {
	s.projectCalls++
	return s.project, s.projectErr
}

// fakeMetrics devuelve byID[sprintID] si está cargado y, si no, result.
type fakeMetrics struct {
	result metrics.SprintMetrics
	byID   map[int64]metrics.SprintMetrics
	err    error
	calls  int
}

func (m *fakeMetrics) CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error) {
	m.calls++
	if r, ok := m.byID[sprintID]; ok {
		return r, m.err
	}
	return m.result, m.err
}

// fakeQuality devuelve byID[sprintID] si está cargado y, si no, result.
type fakeQuality struct {
	result metrics.SprintQuality
	byID   map[int64]metrics.SprintQuality
	err    error
	calls  int
}

func (q *fakeQuality) SprintQuality(_ context.Context, _, sprintID int64) (metrics.SprintQuality, error) {
	q.calls++
	if r, ok := q.byID[sprintID]; ok {
		return r, q.err
	}
	return q.result, q.err
}

type fakes struct {
	store   *fakeStore
	metrics *fakeMetrics
	quality *fakeQuality
}

func sprint2Fakes() fakes {
	return fakes{
		store:   &fakeStore{data: alfaSprint2()},
		metrics: &fakeMetrics{result: alfaSprint2Metrics()},
		quality: &fakeQuality{result: alfaSprint2Quality()},
	}
}

func (f fakes) service() *Service { return NewService(f.store, f.metrics, f.quality, clock) }

// --- pruebas ---

func TestBuildSprintReport(t *testing.T) {
	t.Run("Sprint finalizado", func(t *testing.T) {
		f := sprint2Fakes()
		r, err := f.service().Build(context.Background(), 1, 2)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !reflect.DeepEqual(r.Data, alfaSprint2()) {
			t.Errorf("Data no coincide con lo leído:\n%+v", r.Data)
		}
		if r.Partial {
			t.Error("un Sprint finalizado no es un reporte parcial")
		}
		if r.Metrics == nil || !reflect.DeepEqual(*r.Metrics, alfaSprint2Metrics()) {
			t.Errorf("Metrics debe venir de HU-07: %+v", r.Metrics)
		}
		if r.Quality == nil || !reflect.DeepEqual(*r.Quality, alfaSprint2Quality()) {
			t.Errorf("Quality debe venir de HU-10: %+v", r.Quality)
		}
		if !r.GeneratedAt.Equal(fixedNow) {
			t.Errorf("GeneratedAt = %v, se esperaba el reloj %v", r.GeneratedAt, fixedNow)
		}
		if f.store.calls != 1 || f.metrics.calls != 1 || f.quality.calls != 1 {
			t.Errorf("llamadas: almacén %d, HU-07 %d, HU-10 %d; se esperaba una de cada una",
				f.store.calls, f.metrics.calls, f.quality.calls)
		}
	})

	t.Run("Sprint en curso", func(t *testing.T) {
		f := fakes{store: &fakeStore{data: alfaSprint3()}, metrics: &fakeMetrics{}, quality: &fakeQuality{}}
		r, err := f.service().Build(context.Background(), 1, 3)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if !r.Partial || r.Metrics != nil || r.Quality != nil {
			t.Errorf("Sprint en curso: Partial %v, Metrics %v, Quality %v", r.Partial, r.Metrics, r.Quality)
		}
		if f.metrics.calls != 0 || f.quality.calls != 0 {
			t.Errorf("no se debe llamar a HU-07 (%d) ni a HU-10 (%d) con un Sprint en curso", f.metrics.calls, f.quality.calls)
		}
	})

	t.Run("reloj nil usa la hora actual", func(t *testing.T) {
		f := sprint2Fakes()
		before := time.Now()
		r, err := NewService(f.store, f.metrics, f.quality, nil).Build(context.Background(), 1, 2)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if r.GeneratedAt.Before(before) || r.GeneratedAt.After(time.Now()) {
			t.Errorf("GeneratedAt = %v, se esperaba la hora actual", r.GeneratedAt)
		}
	})
}

func TestReportService_Errores(t *testing.T) {
	cause := errors.New("falla de prueba")
	ctx := context.Background()

	assertNoPDF := func(t *testing.T, f fakes, project, sprint int64) error {
		t.Helper()
		b, err := f.service().SprintPDF(ctx, project, sprint)
		if b != nil {
			t.Errorf("ante un error SprintPDF debe devolver nil como bytes, devolvió %d bytes", len(b))
		}
		return err
	}

	t.Run("identificadores en 0", func(t *testing.T) {
		for _, ids := range [][2]int64{{0, 2}, {1, 0}, {-1, 2}} {
			f := sprint2Fakes()
			if err := assertNoPDF(t, f, ids[0], ids[1]); !errors.Is(err, ErrSprintNotFound) {
				t.Errorf("%v: se esperaba ErrSprintNotFound, se obtuvo %v", ids, err)
			}
			if f.store.calls != 0 {
				t.Errorf("%v: no se debe consultar el almacén", ids)
			}
		}
	})

	t.Run("Sprint inexistente en el almacén", func(t *testing.T) {
		f := sprint2Fakes()
		f.store.err = ErrSprintNotFound
		if err := assertNoPDF(t, f, 1, 2); err != ErrSprintNotFound {
			t.Errorf("se esperaba ErrSprintNotFound sin cambios, se obtuvo %v", err)
		}
	})

	for name, setup := range map[string]func(f fakes){
		"falla del almacén": func(f fakes) { f.store.err = cause },
		"falla de HU-07":    func(f fakes) { f.metrics.err = cause },
		"falla de HU-10":    func(f fakes) { f.quality.err = cause },
	} {
		t.Run(name, func(t *testing.T) {
			f := sprint2Fakes()
			setup(f)
			err := assertNoPDF(t, f, 1, 2)
			if !errors.Is(err, cause) || err == cause {
				t.Errorf("se esperaba la causa envuelta, se obtuvo %v", err)
			}
		})
	}

	t.Run("historias que no coinciden con HU-10", func(t *testing.T) {
		f := sprint2Fakes()
		f.quality.result.StoriesPlanned = 5
		if err := assertNoPDF(t, f, 1, 2); !errors.Is(err, ErrInconsistentReport) {
			t.Errorf("se esperaba ErrInconsistentReport, se obtuvo %v", err)
		}
	})

	t.Run("defectos que no coinciden con HU-10", func(t *testing.T) {
		f := sprint2Fakes()
		f.quality.result.DefectsDetected = 3
		if err := assertNoPDF(t, f, 1, 2); !errors.Is(err, ErrInconsistentReport) {
			t.Errorf("se esperaba ErrInconsistentReport, se obtuvo %v", err)
		}
	})

	t.Run("dependencias nil", func(t *testing.T) {
		f := sprint2Fakes()
		for name, s := range map[string]*Service{
			"almacén": NewService(nil, f.metrics, f.quality, clock),
			"HU-07":   NewService(f.store, nil, f.quality, clock),
			"HU-10":   NewService(f.store, f.metrics, nil, clock),
		} {
			b, err := s.SprintPDF(ctx, 1, 2)
			if err == nil || b != nil {
				t.Errorf("sin %s se esperaba un error y ningún byte", name)
			}
		}
	})
}
