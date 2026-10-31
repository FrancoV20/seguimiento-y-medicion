package backlog

import (
	"context"
	"errors"
	"testing"
	"time"
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

// FR-009 (revisión de PR): si se guarda una estimación mientras HU-15 está cerrando el Sprint, el
// guardado espera el cierre y después NO debe modificar una historia que quedó en un Sprint
// finalizado. Se reproduce el cierre de HU-15 en una transacción que bloquea el Sprint y sus
// historias FOR UPDATE, como PostgreSQLSprintCloser.
func TestPostgreSQLEstimateStore_GuardarDuranteElCierreDelSprintNoModificaLaHistoria(t *testing.T) {
	f := newEstimateFixture(t)
	ctx := context.Background()
	// La historia termina "Hecho": el cierre de HU-15 conserva su asignación al Sprint.
	if _, err := f.db.Exec(`UPDATE user_stories SET status = 'Hecho' WHERE id = $1`, f.enActivo); err != nil {
		t.Fatalf("no se pudo marcar la historia como Hecho: %v", err)
	}

	closing, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("no se pudo iniciar el cierre: %v", err)
	}
	defer closing.Rollback()
	if _, err := closing.Exec(`SELECT status FROM sprints WHERE id = $1 FOR UPDATE`, f.activeSprintID); err != nil {
		t.Fatalf("no se pudo bloquear el Sprint: %v", err)
	}
	if _, err := closing.Exec(`SELECT id FROM user_stories WHERE active_sprint_id = $1 FOR UPDATE`, f.activeSprintID); err != nil {
		t.Fatalf("no se pudieron bloquear las historias del Sprint: %v", err)
	}

	type outcome struct {
		result EstimateResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := f.service.UpdateEstimatedHours(ctx, f.projectID, f.enActivo, "9")
		done <- outcome{result, err}
	}()

	select {
	case got := <-done:
		t.Fatalf("el guardado debía esperar el cierre en curso; terminó antes: %+v", got)
	case <-time.After(300 * time.Millisecond):
	}

	if _, err := closing.Exec(`UPDATE sprints SET status = 'Finalizado' WHERE id = $1`, f.activeSprintID); err != nil {
		t.Fatalf("no se pudo finalizar el Sprint: %v", err)
	}
	if err := closing.Commit(); err != nil {
		t.Fatalf("no se pudo confirmar el cierre: %v", err)
	}

	select {
	case got := <-done:
		if !errors.Is(got.err, ErrStoryInFinishedSprint) {
			t.Errorf("después del cierre se esperaba ErrStoryInFinishedSprint; se obtuvo %+v, %v", got.result, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("el guardado no terminó después del cierre")
	}
	if got := f.hoursText(t, f.enActivo); got != "4.00" {
		t.Errorf("la historia del Sprint ya finalizado cambió de 4.00 a %q (FR-009)", got)
	}
}
