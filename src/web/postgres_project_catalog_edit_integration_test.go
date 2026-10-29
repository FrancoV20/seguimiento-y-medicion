package web

import (
	"context"
	"reflect"
	"testing"
)

// HU-11, T034: ProjectForEdit contra PostgreSQL real (fixture de HU-21).

func TestPostgresProjectForEdit_ConIntegrantes(t *testing.T) {
	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)
	got, err := NewPostgreSQLProjectCatalog(db).ProjectForEdit(context.Background(), fx.projectA)
	if err != nil {
		t.Fatalf("ProjectForEdit devolvió error: %v", err)
	}
	want := fx.projects[fx.projectA]
	ids := []int64{fx.alphaOne, fx.beta}
	if ids[0] > ids[1] {
		ids[0], ids[1] = ids[1], ids[0]
	}
	if got.ID != fx.projectA || got.Name != want.name || !sameCalendarDate(got.StartDate, want.startDate) ||
		!sameCalendarDate(got.EndDate, want.endDate) || !reflect.DeepEqual(got.MemberIDs, ids) {
		t.Errorf("datos inesperados: %+v (integrantes esperados %v)", got, ids)
	}
}

func TestPostgresProjectForEdit_SinIntegrantes(t *testing.T) {
	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)
	got, err := NewPostgreSQLProjectCatalog(db).ProjectForEdit(context.Background(), fx.projectC)
	if err != nil || got.MemberIDs == nil || len(got.MemberIDs) != 0 {
		t.Errorf("se esperaba una lista vacía de integrantes, se obtuvo %+v, %v", got, err)
	}
}

func TestPostgresProjectForEdit_Inexistente(t *testing.T) {
	db := openContextTestDB(t)
	_, err := NewPostgreSQLProjectCatalog(db).ProjectForEdit(context.Background(), MaxID)
	if KindOf(err) != KindNotFound {
		t.Errorf("se esperaba NotFound, se obtuvo %v", err)
	}
}
