package report

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

// HU-09, T038: armado del reporte del proyecto con el almacén y las fuentes de prueba de T012, las
// reglas reales de HU-08 (SummarizeVelocity) y HU-12 (StatusOn, ProgressPercentage) y el reloj fijo.

// Identificadores de los Sprints de «Alfa» en los dobles de prueba.
const (
	alfaS1 int64 = 11
	alfaS2 int64 = 12
	alfaS3 int64 = 13
)

func alfaProject() ProjectData {
	return ProjectData{
		ID: 1, Name: "Alfa", StartDate: day("2026-08-18"), EndDate: day("2026-11-30"),
		Members: []string{"Ana", "Beto"},
		Stories: StoryCounts{Pending: 1, InSprint: 1, InProgress: 1, Done: 4},
		Sprints: []ProjectSprint{
			{ID: alfaS1, Number: 1, Status: "Finalizado"},
			{ID: alfaS2, Number: 2, Status: "Finalizado"},
			{ID: alfaS3, Number: 3, Status: "Activo"},
		},
		EstimatedTotal: 3800, WithoutEstimate: 2,
		RegisteredTotal: 3000,
		MemberHours:     []MemberHours{{Name: "Ana", Hours: 2300}, {Name: "Beto", Hours: 700}},
		DefectsOpen:     2, DefectsResolved: 1,
		OpenDefects: []OpenDefect{
			{Description: "La foto de perfil no se guarda", Severity: "Media", StoryTitle: "Perfil", DetectionSprint: 2},
			{Description: "El PDF sale en blanco", Severity: "Crítica", StoryTitle: "Exportar", DetectionSprint: 3},
		},
	}
}

func betaProject() ProjectData {
	return ProjectData{ID: 2, Name: "Beta", StartDate: day("2026-10-01"), EndDate: day("2026-12-31"),
		Members: []string{"Ana"}, Sprints: []ProjectSprint{}, MemberHours: []MemberHours{}, OpenDefects: []OpenDefect{}}
}

func alfaProjectFakes() fakes {
	return fakes{
		store: &fakeStore{project: alfaProject()},
		metrics: &fakeMetrics{byID: map[int64]metrics.SprintMetrics{
			alfaS1: alfaSprint1Metrics(), alfaS2: alfaSprint2Metrics()}},
		quality: &fakeQuality{byID: map[int64]metrics.SprintQuality{
			alfaS1: alfaSprint1Quality(), alfaS2: alfaSprint2Quality()}},
	}
}

func TestBuildProjectReport(t *testing.T) {
	ctx := context.Background()

	t.Run("«Alfa» con un Sprint en curso", func(t *testing.T) {
		f := alfaProjectFakes()
		r, err := f.service().BuildProject(ctx, 1)
		if err != nil {
			t.Fatalf("BuildProject: %v", err)
		}
		if !reflect.DeepEqual(r.Data, alfaProject()) {
			t.Errorf("Data no coincide con lo leído: %+v", r.Data)
		}
		if want := string(project.StatusOn(day("2026-08-18"), day("2026-11-30"), fixedNow)); r.Status != want || r.Status != "En curso" {
			t.Errorf("Status = %q, se esperaba el de HU-12 (%q)", r.Status, want)
		}
		if want := project.ProgressPercentage(4, 7); r.Progress != want || r.Progress != "57.14" {
			t.Errorf("Progress = %q, se esperaba el de HU-12 (%q)", r.Progress, want)
		}
		if !r.Partial {
			t.Error("con un Sprint en curso el reporte es parcial")
		}
		if len(r.Sprints) != 3 {
			t.Fatalf("se esperaban 3 filas de Sprints, hay %d", len(r.Sprints))
		}
		s1, s2, s3 := r.Sprints[0], r.Sprints[1], r.Sprints[2]
		if s1.Number != 1 || s1.Active || !reflect.DeepEqual(*s1.Quality, alfaSprint1Quality()) || !reflect.DeepEqual(*s1.Metrics, alfaSprint1Metrics()) {
			t.Errorf("fila del Sprint 1 = %+v", s1)
		}
		if s2.Number != 2 || s2.Active || !reflect.DeepEqual(*s2.Quality, alfaSprint2Quality()) || !reflect.DeepEqual(*s2.Metrics, alfaSprint2Metrics()) {
			t.Errorf("fila del Sprint 2 = %+v", s2)
		}
		if s3.Number != 3 || !s3.Active || s3.Quality != nil || s3.Metrics != nil {
			t.Errorf("fila del Sprint 3 = %+v; se esperaba en curso, sin métricas", s3)
		}
		if want := metrics.SummarizeVelocity([]int{3, 10}); r.Velocity != want || r.Velocity.Total != 13 || r.Velocity.Average != "6.50" {
			t.Errorf("Velocity = %+v, se esperaba %+v", r.Velocity, want)
		}
		if f.metrics.calls != 2 || f.quality.calls != 2 {
			t.Errorf("se esperaban 2 llamadas a HU-07 (%d) y a HU-10 (%d), una por Sprint finalizado", f.metrics.calls, f.quality.calls)
		}
		if !r.GeneratedAt.Equal(fixedNow) {
			t.Errorf("GeneratedAt = %v", r.GeneratedAt)
		}
	})

	t.Run("«Beta» sin Sprints", func(t *testing.T) {
		f := fakes{store: &fakeStore{project: betaProject()}, metrics: &fakeMetrics{}, quality: &fakeQuality{}}
		r, err := f.service().BuildProject(ctx, 2)
		if err != nil {
			t.Fatalf("BuildProject: %v", err)
		}
		if r.Partial || len(r.Sprints) != 0 || r.Progress != "0.00" || r.Status != "En curso" {
			t.Errorf("«Beta»: Partial %v, filas %d, Progress %q, Status %q", r.Partial, len(r.Sprints), r.Progress, r.Status)
		}
		if f.metrics.calls != 0 || f.quality.calls != 0 {
			t.Error("sin Sprints no se debe llamar a HU-07 ni a HU-10")
		}
		if r.Velocity.Average != "No disponible" {
			t.Errorf("Velocity = %+v", r.Velocity)
		}
	})
}

func TestProjectReportService_Errores(t *testing.T) {
	cause := errors.New("falla de prueba")
	ctx := context.Background()

	assertNoPDF := func(t *testing.T, f fakes, projectID int64) error {
		t.Helper()
		b, err := f.service().ProjectPDF(ctx, projectID)
		if b != nil {
			t.Errorf("ante un error ProjectPDF debe devolver nil como bytes, devolvió %d bytes", len(b))
		}
		return err
	}

	t.Run("projectID 0", func(t *testing.T) {
		f := alfaProjectFakes()
		if err := assertNoPDF(t, f, 0); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("se esperaba ErrProjectNotFound, se obtuvo %v", err)
		}
		if f.store.projectCalls != 0 {
			t.Error("no se debe consultar el almacén")
		}
	})

	t.Run("proyecto inexistente en el almacén", func(t *testing.T) {
		f := alfaProjectFakes()
		f.store.projectErr = ErrProjectNotFound
		if err := assertNoPDF(t, f, 1); err != ErrProjectNotFound {
			t.Errorf("se esperaba ErrProjectNotFound sin cambios, se obtuvo %v", err)
		}
	})

	for name, setup := range map[string]func(f fakes){
		"falla del almacén": func(f fakes) { f.store.projectErr = cause },
		"falla de HU-07":    func(f fakes) { f.metrics.err = cause },
		"falla de HU-10":    func(f fakes) { f.quality.err = cause },
	} {
		t.Run(name, func(t *testing.T) {
			f := alfaProjectFakes()
			setup(f)
			if err := assertNoPDF(t, f, 1); !errors.Is(err, cause) || err == cause {
				t.Errorf("se esperaba la causa envuelta, se obtuvo %v", err)
			}
		})
	}

	t.Run("dependencias nil", func(t *testing.T) {
		if b, err := NewService(nil, nil, nil, clock).ProjectPDF(ctx, 1); err == nil || b != nil {
			t.Error("sin dependencias se esperaba un error y ningún byte")
		}
	})
}
