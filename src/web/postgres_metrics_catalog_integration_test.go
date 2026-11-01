package web

import (
	"context"
	"testing"
)

// HU-25 (T006): PostgreSQLMetricsCatalog contra PostgreSQL real, con Sprints cerrados por HU-15.

type metricsCatalogWant struct {
	id               int64
	number           int64
	goal, start, end string
}

func TestPostgreSQLMetricsCatalog_SoloFinalizadosDelProyectoMasRecientePrimero(t *testing.T) {
	db := openContextTestDB(t)
	ctx := context.Background()

	// Dado el proyecto A con los Sprints 1, 2 y 3 finalizados y el 4 activo,
	// y el proyecto B con un Sprint finalizado.
	a := newMetricsProject(t, db, "catálogo A")
	s1 := a.finishedSprint(1, "Alta de usuarios", "2026-09-01", "2026-09-14")
	s2 := a.finishedSprint(2, "Pagos online", "2026-09-15", "2026-09-28")
	s3 := a.finishedSprint(3, "Reportes", "2026-09-29", "2026-10-12")
	a.activeSprint(4, "Notificaciones", "2026-10-13", "2026-10-26")
	b := newMetricsProject(t, db, "catálogo B")
	b1 := b.finishedSprint(1, "Sprint de otro proyecto", "2026-09-01", "2026-09-14")

	catalog := NewPostgreSQLMetricsCatalog(db)

	t.Run("lista los finalizados de A del más reciente al más antiguo, sin el activo", func(t *testing.T) {
		got, err := catalog.FinishedSprints(ctx, a.ProjectID)
		if err != nil {
			t.Fatalf("FinishedSprints(A) devolvió error: %v", err)
		}
		want := []metricsCatalogWant{
			{s3, 3, "Reportes", "2026-09-29", "2026-10-12"},
			{s2, 2, "Pagos online", "2026-09-15", "2026-09-28"},
			{s1, 1, "Alta de usuarios", "2026-09-01", "2026-09-14"},
		}
		assertFinishedSprints(t, got, want)
	})

	t.Run("informa el número del Sprint activo de A", func(t *testing.T) {
		number, found, err := catalog.ActiveSprintNumber(ctx, a.ProjectID)
		if err != nil {
			t.Fatalf("ActiveSprintNumber(A) devolvió error: %v", err)
		}
		if !found || number != 4 {
			t.Errorf("ActiveSprintNumber(A) = %d, %v; se esperaba 4, true", number, found)
		}
	})

	t.Run("el proyecto B solo ve su Sprint y no tiene activo", func(t *testing.T) {
		got, err := catalog.FinishedSprints(ctx, b.ProjectID)
		if err != nil {
			t.Fatalf("FinishedSprints(B) devolvió error: %v", err)
		}
		assertFinishedSprints(t, got, []metricsCatalogWant{
			{b1, 1, "Sprint de otro proyecto", "2026-09-01", "2026-09-14"},
		})
		number, found, err := catalog.ActiveSprintNumber(ctx, b.ProjectID)
		if err != nil {
			t.Fatalf("ActiveSprintNumber(B) devolvió error: %v", err)
		}
		if found || number != 0 {
			t.Errorf("ActiveSprintNumber(B) = %d, %v; se esperaba 0, false", number, found)
		}
	})
}

func TestPostgreSQLMetricsCatalog_ProyectoSinSprints(t *testing.T) {
	db := openContextTestDB(t)
	ctx := context.Background()

	// Dado un proyecto sin Sprints.
	p := newMetricsProject(t, db, "sin Sprints")
	catalog := NewPostgreSQLMetricsCatalog(db)

	// Cuando se leen sus Sprints, entonces la lista está vacía y no hay activo, sin error.
	got, err := catalog.FinishedSprints(ctx, p.ProjectID)
	if err != nil {
		t.Fatalf("FinishedSprints devolvió error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("FinishedSprints devolvió %d Sprints; se esperaba ninguno", len(got))
	}
	number, found, err := catalog.ActiveSprintNumber(ctx, p.ProjectID)
	if err != nil {
		t.Fatalf("ActiveSprintNumber devolvió error: %v", err)
	}
	if found || number != 0 {
		t.Errorf("ActiveSprintNumber = %d, %v; se esperaba 0, false", number, found)
	}
}

func assertFinishedSprints(t *testing.T, got []FinishedSprint, want []metricsCatalogWant) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("se obtuvieron %d Sprints finalizados; se esperaban %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.ID != w.id || g.Number != w.number || g.Goal != w.goal ||
			g.StartDate.Format("2006-01-02") != w.start || g.EndDate.Format("2006-01-02") != w.end {
			t.Errorf("Sprint en la posición %d = {id %d, número %d, %q, %s al %s}; se esperaba {id %d, número %d, %q, %s al %s}",
				i, g.ID, g.Number, g.Goal, g.StartDate.Format("2006-01-02"), g.EndDate.Format("2006-01-02"),
				w.id, w.number, w.goal, w.start, w.end)
		}
	}
}
