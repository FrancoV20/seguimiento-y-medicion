package web

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// Integración de PostgreSQLQualityCatalog (HU-10, T020) contra PostgreSQL real, con proyectos y
// Sprints propios que se borran al terminar.

type qualityCatalogFixture struct {
	db                         *sql.DB
	alfaID, betaID, vacioID    int64
	soloFinalizadosID          int64
	alfa1, alfa2, alfa3, alfa4 int64 // Sprints 1, 2 y 3 "Finalizado"; Sprint 4 "Activo"
	beta1                      int64
	soloFin1, soloFin2         int64
}

func newQualityCatalogFixture(t *testing.T) *qualityCatalogFixture {
	t.Helper()
	db := openContextTestDB(t)
	f := &qualityCatalogFixture{db: db}
	stamp := time.Now().UnixNano()
	var projectIDs []int64
	t.Cleanup(func() {
		for _, id := range projectIDs {
			_, _ = db.Exec(`DELETE FROM sprints WHERE project_id = $1`, id)
			_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, id)
		}
	})
	insertProject := func(name string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO projects (name, start_date, end_date, status)
			VALUES ($1, DATE '2026-01-01', DATE '2026-12-31', 'Activo') RETURNING id`,
			fmt.Sprintf("HU10 catálogo %s %d", name, stamp)).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear el proyecto %q: %v", name, err)
		}
		projectIDs = append(projectIDs, id)
		return id
	}
	insertSprint := func(projectID int64, number int, status string) int64 {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, 'Sprint HU-10', DATE '2026-10-01', DATE '2026-10-15', $3) RETURNING id`,
			projectID, number, status).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear el Sprint %d: %v", number, err)
		}
		return id
	}

	f.alfaID = insertProject("Alfa")
	f.betaID = insertProject("Beta")
	f.vacioID = insertProject("sin Sprints")
	f.soloFinalizadosID = insertProject("solo finalizados")
	// Se insertan desordenados para que el orden lo dé la consulta y no el id.
	f.alfa2 = insertSprint(f.alfaID, 2, "Finalizado")
	f.alfa1 = insertSprint(f.alfaID, 1, "Finalizado")
	f.alfa4 = insertSprint(f.alfaID, 4, "Activo")
	f.alfa3 = insertSprint(f.alfaID, 3, "Finalizado")
	f.beta1 = insertSprint(f.betaID, 1, "Finalizado")
	f.soloFin1 = insertSprint(f.soloFinalizadosID, 1, "Finalizado")
	f.soloFin2 = insertSprint(f.soloFinalizadosID, 2, "Finalizado")
	return f
}

func TestPostgresQualityCatalog(t *testing.T) {
	f := newQualityCatalogFixture(t)
	catalog := NewPostgreSQLQualityCatalog(f.db)
	ctx := context.Background()

	t.Run("solo los Sprints Finalizado, del de mayor número al de menor", func(t *testing.T) {
		list, err := catalog.QualitySprints(ctx, f.alfaID)
		if err != nil {
			t.Fatalf("QualitySprints devolvió error: %v", err)
		}
		want := []QualitySprintOption{{ID: f.alfa3, Number: 3}, {ID: f.alfa2, Number: 2}, {ID: f.alfa1, Number: 1}}
		if len(list.Finished) != len(want) {
			t.Fatalf("Finished = %+v; se esperaban los Sprints 3, 2 y 1", list.Finished)
		}
		for i, option := range list.Finished {
			if option != want[i] {
				t.Errorf("posición %d: %+v; se esperaba %+v", i, option, want[i])
			}
			if option.ID == f.alfa4 {
				t.Errorf("el Sprint activo no debía figurar entre los finalizados: %+v", option)
			}
		}
	})

	t.Run("ActiveNumber es el número del Sprint activo", func(t *testing.T) {
		list, err := catalog.QualitySprints(ctx, f.alfaID)
		if err != nil {
			t.Fatalf("QualitySprints devolvió error: %v", err)
		}
		if list.ActiveNumber != 4 {
			t.Errorf("ActiveNumber = %d; se esperaba 4", list.ActiveNumber)
		}
	})

	t.Run("ActiveNumber es 0 si no hay Sprint activo", func(t *testing.T) {
		list, err := catalog.QualitySprints(ctx, f.soloFinalizadosID)
		if err != nil {
			t.Fatalf("QualitySprints devolvió error: %v", err)
		}
		if list.ActiveNumber != 0 {
			t.Errorf("ActiveNumber = %d; se esperaba 0", list.ActiveNumber)
		}
		want := []QualitySprintOption{{ID: f.soloFin2, Number: 2}, {ID: f.soloFin1, Number: 1}}
		if len(list.Finished) != 2 || list.Finished[0] != want[0] || list.Finished[1] != want[1] {
			t.Errorf("Finished = %+v; se esperaba %+v", list.Finished, want)
		}
	})

	t.Run("nada de otro proyecto", func(t *testing.T) {
		alfa, err := catalog.QualitySprints(ctx, f.alfaID)
		if err != nil {
			t.Fatalf("QualitySprints(Alfa) devolvió error: %v", err)
		}
		for _, option := range alfa.Finished {
			if option.ID == f.beta1 {
				t.Errorf("apareció el Sprint de Beta en la lista de Alfa: %+v", option)
			}
		}
		beta, err := catalog.QualitySprints(ctx, f.betaID)
		if err != nil {
			t.Fatalf("QualitySprints(Beta) devolvió error: %v", err)
		}
		if len(beta.Finished) != 1 || beta.Finished[0] != (QualitySprintOption{ID: f.beta1, Number: 1}) || beta.ActiveNumber != 0 {
			t.Errorf("Beta = %+v; se esperaba solo su Sprint 1 y ningún activo (el Sprint 4 activo es de Alfa)", beta)
		}
	})

	t.Run("proyecto sin Sprints", func(t *testing.T) {
		list, err := catalog.QualitySprints(ctx, f.vacioID)
		if err != nil {
			t.Fatalf("un proyecto sin Sprints no debe ser un error: %v", err)
		}
		if len(list.Finished) != 0 || list.ActiveNumber != 0 {
			t.Errorf("proyecto sin Sprints = %+v; se esperaba una lista vacía y ActiveNumber 0", list)
		}
	})

	t.Run("un error de la base devuelve Unavailable", func(t *testing.T) {
		closed := openContextTestDB(t)
		_ = closed.Close()
		_, err := NewPostgreSQLQualityCatalog(closed).QualitySprints(ctx, f.alfaID)
		if err == nil || KindOf(err) != KindUnavailable {
			t.Errorf("error = %v (tipo %v); se esperaba un error Unavailable", err, KindOf(err))
		}
	})
}
