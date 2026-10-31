package web

import (
	"context"
	"testing"
)

// HU-12 (T010): PostgreSQLProjectStatusCatalog contra PostgreSQL real, con datos propios.

func TestPostgresProjectStatusCatalog(t *testing.T) {
	db := openContextTestDB(t)
	catalog := NewPostgreSQLProjectStatusCatalog(db)
	ctx := context.Background()

	alfa := seedStatusAlfa(t, db)
	gamma := seedStatusGamma(t, db)
	beta := seedStatusBeta(t, db)

	t.Run("Alfa: nombre, fechas y cantidades", func(t *testing.T) {
		got, err := catalog.StatusSummary(ctx, alfa.ProjectID)
		if err != nil {
			t.Fatalf("StatusSummary() devolvió un error: %v", err)
		}
		if got.Name != alfa.Name {
			t.Errorf("Name = %q, se esperaba %q", got.Name, alfa.Name)
		}
		if start := got.StartDate.Format("2006-01-02"); start != "2026-10-01" {
			t.Errorf("StartDate = %s, se esperaba 2026-10-01", start)
		}
		if end := got.EndDate.Format("2006-01-02"); end != "2026-11-30" {
			t.Errorf("EndDate = %s, se esperaba 2026-11-30", end)
		}
		want := ProjectStatusCounts{Pending: 2, InSprint: 1, InProgress: 1, Done: 4}
		if got.Counts != want {
			t.Errorf("Counts = %+v, se esperaba %+v", got.Counts, want)
		}
		if got.Counts.Total() != 8 {
			t.Errorf("Total() = %d, se esperaba 8", got.Counts.Total())
		}
	})

	t.Run("Gamma: solo sus historias, sin mezclar las de Alfa", func(t *testing.T) {
		got, err := catalog.StatusSummary(ctx, gamma.ProjectID)
		if err != nil {
			t.Fatalf("StatusSummary() devolvió un error: %v", err)
		}
		want := ProjectStatusCounts{Done: 3}
		if got.Counts != want {
			t.Errorf("Counts = %+v, se esperaba %+v", got.Counts, want)
		}
		// Si se mezclaran con las de Alfa, el total sería 11.
		if got.Counts.Total() != 3 {
			t.Errorf("Total() = %d, se esperaba 3", got.Counts.Total())
		}
	})

	t.Run("Beta: sin historias, cuatro cantidades en cero", func(t *testing.T) {
		got, err := catalog.StatusSummary(ctx, beta.ProjectID)
		if err != nil {
			t.Fatalf("StatusSummary() devolvió un error: %v", err)
		}
		if got.Counts != (ProjectStatusCounts{}) || got.Counts.Total() != 0 {
			t.Errorf("Counts = %+v, se esperaba todo en cero", got.Counts)
		}
	})

	t.Run("proyecto inexistente es NotFound", func(t *testing.T) {
		_, err := catalog.StatusSummary(ctx, MaxID)
		if err == nil || KindOf(err) != KindNotFound {
			t.Errorf("StatusSummary() = %v, se esperaba un error NotFound", err)
		}
	})

	t.Run("una falla de lectura es un error y no NotFound", func(t *testing.T) {
		closed := openContextTestDB(t)
		_ = closed.Close() // toda consulta falla

		_, err := NewPostgreSQLProjectStatusCatalog(closed).StatusSummary(ctx, alfa.ProjectID)
		if err == nil {
			t.Fatalf("una falla de lectura debe devolver un error")
		}
		if KindOf(err) == KindNotFound {
			t.Errorf("una falla de lectura no debe confundirse con \"no encontrado\": %v", err)
		}
	})
}
