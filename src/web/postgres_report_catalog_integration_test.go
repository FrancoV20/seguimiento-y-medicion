package web

import (
	"context"
	"reflect"
	"testing"
)

// HU-09, T026: PostgreSQLReportCatalog contra PostgreSQL real con los datos de report_fixtures_test.go.

func TestPostgresReportCatalog(t *testing.T) {
	db := openContextTestDB(t)
	f := newReportWebFixture(t, db)
	catalog := NewPostgreSQLReportCatalog(db)
	ctx := context.Background()

	t.Run("«Alfa»: el activo primero y después los finalizados por número descendente", func(t *testing.T) {
		got, err := catalog.ReportSprints(ctx, f.projects["Alfa"])
		if err != nil {
			t.Fatalf("ReportSprints: %v", err)
		}
		want := []ReportSprintOption{
			{ID: f.sprints["Alfa 3"], Number: 3, Goal: "Exportar e informar", Active: true},
			{ID: f.sprints["Alfa 2"], Number: 2, Goal: "Gestionar perfiles y reportes"},
			{ID: f.sprints["Alfa 1"], Number: 1, Goal: "Dar de alta usuarios"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ReportSprints = %+v, se esperaba %+v", got, want)
		}
	})

	t.Run("«Beta»: lista vacía", func(t *testing.T) {
		got, err := catalog.ReportSprints(ctx, f.projects["Beta"])
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("ReportSprints = %+v, %v; se esperaba una lista vacía", got, err)
		}
	})

	t.Run("ReportSprint", func(t *testing.T) {
		got, found, err := catalog.ReportSprint(ctx, f.projects["Alfa"], f.sprints["Alfa 3"])
		want := ReportSprintOption{ID: f.sprints["Alfa 3"], Number: 3, Goal: "Exportar e informar", Active: true}
		if err != nil || !found || got != want {
			t.Errorf("ReportSprint(Alfa 3) = %+v, %v, %v; se esperaba %+v", got, found, err, want)
		}
		if _, found, err := catalog.ReportSprint(ctx, f.projects["Alfa"], f.sprints["Delta 1"]); err != nil || found {
			t.Errorf("Sprint de «Delta» con «Alfa»: found %v, err %v; se esperaba no encontrado", found, err)
		}
		if _, found, err := catalog.ReportSprint(ctx, f.projects["Alfa"], MaxID); err != nil || found {
			t.Errorf("Sprint inexistente: found %v, err %v; se esperaba no encontrado", found, err)
		}
	})
}
