package web

import (
	"context"
	"reflect"
	"testing"
)

// HU-08 (T010) [E1, E2, E9, E10]: el catálogo del Dashboard lee, solo del proyecto pedido, las
// historias por estado, los defectos abiertos y los Sprints finalizados (por número ascendente).
func TestPostgresDashboardCatalog(t *testing.T) {
	db := openContextTestDB(t)
	f := newDashboardFixture(t, db)
	catalog := NewPostgreSQLDashboardCatalog(db)
	ctx := context.Background()

	numbers := func(sprints []DashboardSprint) []int {
		out := make([]int, 0, len(sprints))
		for _, s := range sprints {
			out = append(out, s.Number)
		}
		return out
	}

	cases := []struct {
		name        string
		projectID   int64
		wantStories DashboardStoryCounts
		wantDefects int
		wantNumbers []int
	}{
		{name: "Alfa", projectID: f.alfa, wantStories: DashboardStoryCounts{Pending: 2, InSprint: 1, InProgress: 2, Done: 3}, wantDefects: 3, wantNumbers: []int{1, 2}},
		{name: "Beta (otro proyecto)", projectID: f.beta, wantStories: DashboardStoryCounts{Pending: 1, Done: 1}, wantDefects: 1, wantNumbers: []int{1}},
		{name: "Vacío", projectID: f.empty, wantStories: DashboardStoryCounts{}, wantDefects: 0, wantNumbers: []int{}},
		{name: "Sin Sprint", projectID: f.noSprint, wantStories: DashboardStoryCounts{Pending: 2}, wantDefects: 0, wantNumbers: []int{}},
		{name: "Solo activo: el Sprint activo no es finalizado", projectID: f.onlyActive, wantStories: DashboardStoryCounts{Pending: 1, InSprint: 1}, wantDefects: 0, wantNumbers: []int{}},
		{name: "Doce: del 1 al 12", projectID: f.twelve, wantStories: DashboardStoryCounts{Done: 12}, wantDefects: 0, wantNumbers: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{name: "Sin historias", projectID: f.noStories, wantStories: DashboardStoryCounts{}, wantDefects: 0, wantNumbers: []int{1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := catalog.DashboardSummary(ctx, tc.projectID)
			if err != nil {
				t.Fatalf("DashboardSummary(%s) devolvió error: %v", tc.name, err)
			}
			if got.Stories != tc.wantStories {
				t.Errorf("historias por estado = %+v; se esperaba %+v", got.Stories, tc.wantStories)
			}
			if got.OpenDefects != tc.wantDefects {
				t.Errorf("defectos abiertos = %d; se esperaba %d", got.OpenDefects, tc.wantDefects)
			}
			if gotNumbers := numbers(got.Finished); !reflect.DeepEqual(gotNumbers, tc.wantNumbers) {
				t.Errorf("Sprints finalizados = %v; se esperaba %v", gotNumbers, tc.wantNumbers)
			}
		})
	}

	// Los Sprints finalizados traen su identificador real, para pedirle a HU-10 y HU-07 sus datos.
	alfa, err := catalog.DashboardSummary(ctx, f.alfa)
	if err != nil {
		t.Fatalf("DashboardSummary(Alfa) devolvió error: %v", err)
	}
	want := []DashboardSprint{{ID: f.alfaSprint1, Number: 1}, {ID: f.alfaSprint2, Number: 2}}
	if !reflect.DeepEqual(alfa.Finished, want) {
		t.Errorf("Sprints finalizados de Alfa = %+v; se esperaba %+v", alfa.Finished, want)
	}
}

// Ante una falla de la base, el catálogo devuelve un error y no un resumen parcial.
func TestPostgresDashboardCatalog_ErrorDeLectura(t *testing.T) {
	db := openContextTestDB(t)
	catalog := NewPostgreSQLDashboardCatalog(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := catalog.DashboardSummary(ctx, 1)

	if err == nil {
		t.Fatal("DashboardSummary con el contexto cancelado debía fallar")
	}
	if !reflect.DeepEqual(got, DashboardSummary{}) {
		t.Errorf("con error no debe devolver datos parciales: %+v", got)
	}
}
