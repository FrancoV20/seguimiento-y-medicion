package backlog

import (
	"context"
	"errors"
	"testing"
)

// Integración de PostgreSQLEstimateStore (HU-17, T006). Usa newEstimateFixture de
// update_estimated_hours_test.go.

func TestPostgreSQLEstimateStore_FindEstimateLeeHorasYBloqueo(t *testing.T) {
	f := newEstimateFixture(t)
	ctx := context.Background()
	cases := []struct {
		name     string
		storyID  int64
		hours    *float64
		finished bool
	}{
		{"sin estimar", f.sinEstimar, nil, false},
		{"estimada", f.estimada, floatPtr(8), false},
		{"en Sprint activo", f.enActivo, floatPtr(4), false},
		{"en Sprint finalizado", f.finalizada, floatPtr(5), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state, err := f.store.FindEstimate(ctx, f.projectID, c.storyID)
			if err != nil {
				t.Fatalf("no se esperaba error: %v", err)
			}
			if (state.Hours == nil) != (c.hours == nil) || state.Hours != nil && *state.Hours != *c.hours {
				t.Errorf("Hours = %v; se esperaba %v", state.Hours, c.hours)
			}
			if state.InFinishedSprint != c.finished {
				t.Errorf("InFinishedSprint = %t; se esperaba %t", state.InFinishedSprint, c.finished)
			}
		})
	}
}

func TestPostgreSQLEstimateStore_FindEstimateAislaPorProyecto(t *testing.T) {
	f := newEstimateFixture(t)
	ctx := context.Background()
	if _, err := f.store.FindEstimate(ctx, f.projectID, f.ajena); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("historia de otro proyecto: error = %v; se esperaba ErrStoryNotFound", err)
	}
	if _, err := f.store.FindEstimate(ctx, f.projectID, 9223372036854775807); !errors.Is(err, ErrStoryNotFound) {
		t.Errorf("historia inexistente: error = %v; se esperaba ErrStoryNotFound", err)
	}
	if state, err := f.store.FindEstimate(ctx, f.otherID, f.ajena); err != nil || state.Hours == nil || *state.Hours != 6 {
		t.Errorf("desde su propio proyecto: state=%+v err=%v; se esperaban 6 horas", state, err)
	}
}

func TestPostgreSQLEstimateStore_UpdateEstimateCambiaSoloEstimatedHours(t *testing.T) {
	f := newEstimateFixture(t)
	before := f.storySnapshot(t, f.estimada)
	changed, err := f.store.UpdateEstimate(context.Background(), f.projectID, f.estimada, 9.75)
	if err != nil || !changed {
		t.Fatalf("UpdateEstimate = (%t, %v); se esperaba (true, nil)", changed, err)
	}
	if got := f.hoursText(t, f.estimada); got != "9.75" {
		t.Errorf("estimated_hours = %q; se esperaba 9.75", got)
	}
	if after := f.storySnapshot(t, f.estimada); after != before {
		t.Errorf("cambiaron otras columnas: antes=%s después=%s", before, after)
	}
	if got := f.hoursText(t, f.sinEstimar); got != "" {
		t.Errorf("cambió otra historia del proyecto: %q", got)
	}
}

func TestPostgreSQLEstimateStore_UpdateEstimateNoCambiaSprintFinalizadoNiOtroProyecto(t *testing.T) {
	f := newEstimateFixture(t)
	ctx := context.Background()
	cases := []struct {
		name    string
		storyID int64
		want    string
	}{
		{"Sprint finalizado", f.finalizada, "5.00"},
		{"otro proyecto", f.ajena, "6.00"},
		{"inexistente", 9223372036854775807, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			changed, err := f.store.UpdateEstimate(ctx, f.projectID, c.storyID, 1)
			if err != nil || changed {
				t.Errorf("UpdateEstimate = (%t, %v); se esperaba (false, nil)", changed, err)
			}
			if c.want != "" {
				if got := f.hoursText(t, c.storyID); got != c.want {
					t.Errorf("estimated_hours = %q; se esperaba %q sin cambios", got, c.want)
				}
			}
		})
	}
}

func TestPostgreSQLEstimateStore_UpdateEstimateVerificaElBloqueoAlGuardar(t *testing.T) {
	f := newEstimateFixture(t)
	ctx := context.Background()
	// La lectura todavía ve la historia editable...
	state, err := f.store.FindEstimate(ctx, f.projectID, f.enActivo)
	if err != nil || state.InFinishedSprint {
		t.Fatalf("FindEstimate = (%+v, %v); se esperaba una historia editable", state, err)
	}
	// ...pero el Sprint se finaliza antes de guardar: la misma sentencia debe excluirla.
	if _, err := f.db.Exec(`UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, f.activeSprintID); err != nil {
		t.Fatalf("no se pudo finalizar el Sprint: %v", err)
	}
	changed, err := f.store.UpdateEstimate(ctx, f.projectID, f.enActivo, 11)
	if err != nil || changed {
		t.Errorf("UpdateEstimate = (%t, %v); se esperaba (false, nil) con el Sprint ya finalizado", changed, err)
	}
	if got := f.hoursText(t, f.enActivo); got != "4.00" {
		t.Errorf("estimated_hours = %q; se esperaba 4.00 sin cambios", got)
	}
}
