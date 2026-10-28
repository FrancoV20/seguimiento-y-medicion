package web

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// T009 y T010: adaptador sobre el servicio real de HU-03. HU-03 ya ofrece la validación que
// devuelve todas las violaciones (effort.ValidateInput), así que T010 solo agrega esta prueba.

var effortTestNow = time.Date(2026, 10, 20, 15, 0, 0, 0, time.Local)

func effortClock() time.Time { return effortTestNow }

func newTestEffortActions(store *effort.TestStore) EffortActions {
	return NewEffortActions(effort.NewService(store, effortClock), effortClock)
}

func TestEffortActions_Validate_TodasLasViolacionesJuntasConSuCampo(t *testing.T) {
	actions := newTestEffortActions(effort.NewTestStore())

	got := actions.Validate(EffortSubmission{StoryID: 1, MemberID: 0, Activity: "   ", Hours: "0"})
	want := []EffortViolation{
		{Field: EffortFieldHours, Message: effort.ErrInvalidHours.Error()},
		{Field: EffortFieldActivity, Message: effort.ErrActivityRequired.Error()},
		{Field: EffortFieldDate, Message: effort.ErrDateRequired.Error()},
		{Field: "", Message: effort.ErrMemberRequired.Error()},
	}
	if len(got) != len(want) {
		t.Fatalf("se esperaban %d violaciones, se obtuvieron %+v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("violación %d: se esperaba %+v, se obtuvo %+v", i, want[i], got[i])
		}
	}
}

func TestEffortActions_Validate_FechaFuturaEsDeHU03(t *testing.T) {
	actions := newTestEffortActions(effort.NewTestStore())
	tomorrow := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	got := actions.Validate(EffortSubmission{StoryID: 1, MemberID: 1, Date: tomorrow, Activity: "x", Hours: "1"})
	if violationFor(got, EffortFieldDate) != effort.ErrDateInFuture.Error() {
		t.Errorf("se esperaba el mensaje de fecha futura de HU-03, se obtuvo %+v", got)
	}
	today := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if got := actions.Validate(EffortSubmission{StoryID: 1, MemberID: 1, Date: today, Activity: "x", Hours: "1"}); len(got) != 0 {
		t.Errorf("una carga válida no debe tener violaciones, se obtuvo %+v", got)
	}
}

func TestEffortActions_Register(t *testing.T) {
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	valid := EffortSubmission{StoryID: 10, MemberID: 7, Date: day, Activity: " Diseño  de pantalla ", Hours: "1,5"}

	t.Run("registra una carga válida", func(t *testing.T) {
		store := effort.NewTestStore()
		store.AddStory("10", "1", true)
		store.AddMember("7", "1")
		violations, err := newTestEffortActions(store).Register(context.Background(), valid)
		if err != nil || len(violations) != 0 {
			t.Fatalf("se esperaba registrar sin errores, se obtuvo %+v, %v", violations, err)
		}
		entries := store.Entries()
		if len(entries) != 1 {
			t.Fatalf("se esperaba una carga guardada, hay %d", len(entries))
		}
		e := entries[0]
		if e.StoryID != "10" || e.MemberID != "7" || e.Activity != "Diseño de pantalla" || e.Hours != 150 || !e.WorkDate.Equal(day) {
			t.Errorf("carga guardada inesperada: %+v", e)
		}
	})

	t.Run("integrante faltante lo rechaza HU-03", func(t *testing.T) {
		store := effort.NewTestStore()
		store.AddStory("10", "1", true)
		sub := valid
		sub.MemberID = 0
		violations, err := newTestEffortActions(store).Register(context.Background(), sub)
		if err != nil {
			t.Fatalf("no se esperaba error de guardado: %v", err)
		}
		if len(violations) != 1 || violations[0] != (EffortViolation{Field: "", Message: effort.ErrMemberRequired.Error()}) {
			t.Errorf("se esperaba el error general de integrante de HU-03, se obtuvo %+v", violations)
		}
		if store.SaveCalls != 0 {
			t.Errorf("no debe intentar guardar, SaveCalls = %d", store.SaveCalls)
		}
	})

	t.Run("historia fuera del Sprint activo la rechaza HU-03 junto a historia", func(t *testing.T) {
		store := effort.NewTestStore()
		store.AddStory("10", "1", false)
		store.AddMember("7", "1")
		violations, err := newTestEffortActions(store).Register(context.Background(), valid)
		if err != nil {
			t.Fatalf("no se esperaba error de guardado: %v", err)
		}
		if violationFor(violations, EffortFieldStory) != effort.ErrStoryNotInSprint.Error() {
			t.Errorf("se esperaba el mensaje de HU-03 junto a historia, se obtuvo %+v", violations)
		}
		if len(store.Entries()) != 0 {
			t.Error("no debe guardarse nada")
		}
	})

	t.Run("falla de guardado sin detalles internos", func(t *testing.T) {
		store := effort.NewTestStore()
		store.AddStory("10", "1", true)
		store.AddMember("7", "1")
		store.SaveErr = errors.New(`pq: relation "effort_logs" violates check constraint`)
		violations, err := newTestEffortActions(store).Register(context.Background(), valid)
		if !errors.Is(err, ErrEffortNotSaved) || len(violations) != 0 {
			t.Fatalf("se esperaba ErrEffortNotSaved, se obtuvo %+v, %v", violations, err)
		}
		if err.Error() != ErrEffortNotSaved.Error() {
			t.Errorf("el error no debe llevar detalles internos: %q", err.Error())
		}
	})

	t.Run("falla al consultar la historia", func(t *testing.T) {
		store := effort.NewTestStore()
		store.QueryErr = errors.New("conexión perdida")
		_, err := newTestEffortActions(store).Register(context.Background(), valid)
		if !errors.Is(err, ErrEffortNotSaved) {
			t.Errorf("se esperaba ErrEffortNotSaved, se obtuvo %v", err)
		}
	})
}
