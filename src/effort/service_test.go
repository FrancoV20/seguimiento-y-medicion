package effort

import (
	"context"
	"errors"
	"testing"
	"time"
)

// newTestService arma un servicio con un store en memoria y el reloj fijo en fixedNow.
// La historia S1 está en un Sprint activo del proyecto P1 y el integrante M1 pertenece a P1.
func newTestService(t *testing.T) (*Service, *TestStore) {
	t.Helper()
	store := NewTestStore()
	store.Now = func() time.Time { return fixedNow }
	store.AddStory("S1", "P1", true)
	store.AddMember("M1", "P1")
	service := NewService(store, func() time.Time { return fixedNow })
	return service, store
}

// T013: registrar 4 horas suma 4 horas al esfuerzo acumulado de la historia.
func TestLogEffortRegistersHoursAndAccumulates(t *testing.T) {
	service, store := newTestService(t)

	result := service.LogEffort(context.Background(), validRequest())

	if !result.Succeeded() {
		t.Fatalf("el registro debía tener éxito: %+v", result)
	}
	if result.AccumulatedHours != 400 {
		t.Errorf("AccumulatedHours = %d, se esperaba 400", result.AccumulatedHours)
	}
	if result.SuccessMessage != SuccessMessage {
		t.Errorf("SuccessMessage = %q, se esperaba %q", result.SuccessMessage, SuccessMessage)
	}
	if len(result.ValidationErrors) != 0 || result.StorageError != nil {
		t.Errorf("no se esperaban errores: %+v", result)
	}

	entry := result.Entry
	if entry.StoryID != "S1" || entry.MemberID != "M1" ||
		entry.Activity != "Desarrollo de endpoint en Go" || entry.Hours != 400 {
		t.Errorf("el registro creado no coincide con la solicitud: %+v", entry)
	}

	saved := store.Entries()
	if len(saved) != 1 {
		t.Fatalf("debía quedar 1 registro guardado, hay %d", len(saved))
	}
	if saved[0].ID == "" {
		t.Errorf("el registro guardado debe tener un ID")
	}
}

func TestLogEffortSuccessiveEntriesAccumulate(t *testing.T) {
	service, store := newTestService(t)
	ctx := context.Background()

	first := validRequest() // 4 horas
	second := validRequest()
	second.Hours = "2,5"
	second.Activity = "Revisión de código"

	if r := service.LogEffort(ctx, first); !r.Succeeded() || r.AccumulatedHours != 400 {
		t.Fatalf("el primer registro debía dar 400, resultado: %+v", r)
	}
	r := service.LogEffort(ctx, second)
	if !r.Succeeded() {
		t.Fatalf("el segundo registro debía tener éxito: %+v", r)
	}
	if r.AccumulatedHours != 650 {
		t.Errorf("AccumulatedHours = %d, se esperaba 650 (6,5 h)", r.AccumulatedHours)
	}

	saved := store.Entries()
	if len(saved) != 2 {
		t.Fatalf("debían quedar 2 registros independientes, hay %d", len(saved))
	}
	if saved[0].Hours != 400 || saved[1].Hours != 250 {
		t.Errorf("las horas de cada registro deben conservarse: %d y %d", saved[0].Hours, saved[1].Hours)
	}
	if saved[0].ID == saved[1].ID {
		t.Errorf("cada registro debe tener su propio ID")
	}
}

func TestLogEffortRejectsStoryWithoutActiveSprint(t *testing.T) {
	cases := []struct {
		name    string
		storyID string
	}{
		{"historia sin Sprint activo", "S2"},
		{"historia inexistente", "S999"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, store := newTestService(t)
			store.AddStory("S2", "P1", false)

			req := validRequest()
			req.StoryID = c.storyID
			result := service.LogEffort(context.Background(), req)

			if result.Succeeded() || result.SuccessMessage != "" {
				t.Fatalf("el registro debía ser rechazado, sin mensaje de éxito: %+v", result)
			}
			if !errors.Is(errorFor(result.ValidationErrors, FieldStory), ErrStoryNotInSprint) {
				t.Errorf("se esperaba ErrStoryNotInSprint en %q, se obtuvo %v", FieldStory, result.ValidationErrors)
			}
			if store.SaveCalls != 0 || len(store.Entries()) != 0 {
				t.Errorf("no debía guardarse nada (SaveCalls=%d, registros=%d)", store.SaveCalls, len(store.Entries()))
			}
		})
	}
}

func TestLogEffortRejectsInvalidHoursWithoutChangingState(t *testing.T) {
	invalid := []string{"-1", "0", "25", "24,01", "1,555", "", "abc"}
	for _, hours := range invalid {
		t.Run("horas "+hours, func(t *testing.T) {
			service, store := newTestService(t)
			ctx := context.Background()

			// Given: la historia ya tiene 4 horas acumuladas.
			if r := service.LogEffort(ctx, validRequest()); !r.Succeeded() {
				t.Fatalf("la carga previa debía registrarse: %+v", r)
			}
			before, _ := store.AccumulatedHours(ctx, "S1")

			// When: se intenta registrar horas inválidas.
			req := validRequest()
			req.Hours = hours
			result := service.LogEffort(ctx, req)

			// Then: se rechaza con el mensaje exacto y no cambia nada.
			if result.Succeeded() || result.SuccessMessage != "" {
				t.Fatalf("la carga debía ser rechazada, sin mensaje de éxito: %+v", result)
			}
			got := errorFor(result.ValidationErrors, FieldHours)
			if !errors.Is(got, ErrInvalidHours) {
				t.Fatalf("se esperaba ErrInvalidHours en %q, se obtuvo %v", FieldHours, result.ValidationErrors)
			}
			if got.Error() != "El valor de horas no es válido" {
				t.Errorf("mensaje = %q", got.Error())
			}
			if store.SaveCalls != 1 {
				t.Errorf("SaveEntry solo debía llamarse por la carga previa, se llamó %d veces", store.SaveCalls)
			}
			after, _ := store.AccumulatedHours(ctx, "S1")
			if after != before {
				t.Errorf("el acumulado cambió: antes %d, después %d", before, after)
			}
			if len(store.Entries()) != 1 {
				t.Errorf("debía seguir habiendo 1 registro, hay %d", len(store.Entries()))
			}
		})
	}
}

// Los límites válidos: 24 horas exactas y menos de una hora se aceptan.
func TestLogEffortAcceptsBoundaryHours(t *testing.T) {
	cases := []struct {
		hours string
		want  Hours
	}{
		{"24", 2400},
		{"0,5", 50},
		{"0.25", 25},
		{"1", 100},
		{"7,75", 775},
		{"8", 800},
		{"23,99", 2399},
	}
	for _, c := range cases {
		t.Run("horas "+c.hours, func(t *testing.T) {
			service, _ := newTestService(t)
			req := validRequest()
			req.Hours = c.hours
			result := service.LogEffort(context.Background(), req)
			if !result.Succeeded() || result.AccumulatedHours != c.want {
				t.Errorf("se esperaba éxito con acumulado %d, resultado: %+v", c.want, result)
			}
		})
	}
}

// assertRejected verifica que la carga fue rechazada por un único campo con el mensaje exacto,
// sin mensaje de éxito y sin haber guardado nada (T041).
func assertRejected(t *testing.T, result LogEffortResult, store *TestStore, field string, want error, wantMessage string) {
	t.Helper()
	if result.Succeeded() || result.SuccessMessage != "" {
		t.Fatalf("la carga debía ser rechazada, sin mensaje de éxito: %+v", result)
	}
	got := errorFor(result.ValidationErrors, field)
	if !errors.Is(got, want) {
		t.Fatalf("se esperaba %v en %q, se obtuvo %v", want, field, result.ValidationErrors)
	}
	if got.Error() != wantMessage {
		t.Errorf("mensaje = %q, se esperaba %q", got.Error(), wantMessage)
	}
	if store.SaveCalls != 0 || len(store.Entries()) != 0 {
		t.Errorf("no debía guardarse nada (SaveCalls=%d, registros=%d)", store.SaveCalls, len(store.Entries()))
	}
	if total, _ := store.AccumulatedHours(context.Background(), "S1"); total != 0 {
		t.Errorf("el acumulado de la historia no debía cambiar (0), fue %d", total)
	}
}

// T031: actividad vacía o solo espacios.
func TestLogEffortRejectsEmptyActivity(t *testing.T) {
	for _, activity := range []string{"", "   ", "\t\n "} {
		t.Run("actividad vacía", func(t *testing.T) {
			service, store := newTestService(t)
			req := validRequest()
			req.Activity = activity
			result := service.LogEffort(context.Background(), req)
			assertRejected(t, result, store, FieldActivity, ErrActivityRequired,
				"La descripción de la actividad es obligatoria")
		})
	}
}

// T032: fecha omitida y fecha posterior a la actual, con reloj inyectado.
func TestLogEffortRejectsMissingOrFutureDate(t *testing.T) {
	cases := []struct {
		name        string
		date        time.Time
		want        error
		wantMessage string
	}{
		{"fecha omitida", time.Time{}, ErrDateRequired, "La fecha es requerida"},
		{"fecha de mañana", time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC), ErrDateInFuture,
			"La fecha no puede ser posterior a la fecha actual"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, store := newTestService(t)
			req := validRequest()
			req.WorkDate = c.date
			result := service.LogEffort(context.Background(), req)
			assertRejected(t, result, store, FieldDate, c.want, c.wantMessage)
		})
	}
}

// T032: el resultado depende del reloj inyectado, no de la hora real.
func TestLogEffortUsesInjectedClock(t *testing.T) {
	store := NewTestStore()
	store.AddStory("S1", "P1", true)
	store.AddMember("M1", "P1")
	later := time.Date(2026, time.October, 12, 9, 0, 0, 0, time.UTC)
	service := NewService(store, func() time.Time { return later })

	req := validRequest()
	req.WorkDate = time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC) // futura para fixedNow, pasada para later
	if result := service.LogEffort(context.Background(), req); !result.Succeeded() {
		t.Errorf("con el reloj en el 12 de octubre, el 11 debía ser válido: %+v", result)
	}
}

// T033: integrante omitido, inexistente o ajeno al proyecto de la historia.
func TestLogEffortRejectsInvalidMember(t *testing.T) {
	cases := []struct {
		name     string
		memberID string
	}{
		{"integrante omitido", ""},
		{"integrante en blanco", "   "},
		{"integrante inexistente", "M999"},
		{"integrante de otro proyecto", "M2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, store := newTestService(t)
			store.AddMember("M2", "P2") // existe, pero no pertenece a P1
			req := validRequest()
			req.MemberID = c.memberID
			result := service.LogEffort(context.Background(), req)
			assertRejected(t, result, store, FieldMember, ErrMemberRequired,
				"Se debe especificar el integrante responsable")
		})
	}
}

// T034: varios campos inválidos a la vez se informan todos juntos y no se guarda nada.
func TestLogEffortReportsAllInvalidFieldsTogether(t *testing.T) {
	service, store := newTestService(t)
	req := LogEffortRequest{StoryID: "S1", Hours: "0"} // horas, actividad, fecha e integrante inválidos

	result := service.LogEffort(context.Background(), req)

	if result.Succeeded() || result.SuccessMessage != "" {
		t.Fatalf("la carga debía ser rechazada: %+v", result)
	}
	want := []string{FieldHours, FieldActivity, FieldDate, FieldMember}
	if len(result.ValidationErrors) != len(want) {
		t.Fatalf("se esperaban %d errores juntos, se obtuvo %v", len(want), result.ValidationErrors)
	}
	for i, field := range want {
		if result.ValidationErrors[i].Field != field {
			t.Errorf("error %d: campo %q, se esperaba %q", i, result.ValidationErrors[i].Field, field)
		}
	}
	if store.SaveCalls != 0 || len(store.Entries()) != 0 {
		t.Errorf("no debía guardarse nada (SaveCalls=%d, registros=%d)", store.SaveCalls, len(store.Entries()))
	}
}

// errDB simula un error interno de la base de datos con detalles que nunca deben llegar al usuario.
var errDB = errors.New("pq: connection refused to 10.0.0.5:5432 (detalle interno)")

// T042 y T047: una falla del almacenamiento informa que la operación no pudo completarse,
// sin mensaje de éxito, sin registro y sin cambiar el acumulado.
func TestLogEffortReportsStorageFailureWithoutFalseSuccess(t *testing.T) {
	cases := []struct {
		name       string
		breakStore func(*TestStore)
	}{
		{"falla al guardar", func(s *TestStore) { s.SaveErr = errDB }},
		{"falla al consultar el contexto", func(s *TestStore) { s.QueryErr = errDB }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, store := newTestService(t)
			ctx := context.Background()

			// Given: la historia ya tiene 4 horas acumuladas.
			if r := service.LogEffort(ctx, validRequest()); !r.Succeeded() {
				t.Fatalf("la carga previa debía registrarse: %+v", r)
			}

			// When: se registra una carga válida, pero el almacenamiento falla.
			c.breakStore(store)
			second := validRequest()
			second.Activity = "Revisión de código"
			result := service.LogEffort(ctx, second)
			store.SaveErr, store.QueryErr = nil, nil // se repara la base para poder consultar

			// Then: error genérico, sin éxito, sin registro y sin detalles internos.
			if result.Succeeded() || result.SuccessMessage != "" || result.Entry != nil {
				t.Fatalf("no debía haber éxito ni registro: %+v", result)
			}
			if len(result.ValidationErrors) != 0 {
				t.Errorf("una falla de almacenamiento no es un error de validación: %v", result.ValidationErrors)
			}
			if !errors.Is(result.StorageError, ErrStorage) {
				t.Fatalf("se esperaba ErrStorage, se obtuvo %v", result.StorageError)
			}
			if result.StorageError.Error() != "La operación no pudo completarse" {
				t.Errorf("mensaje = %q", result.StorageError.Error())
			}
			if errors.Is(result.StorageError, errDB) {
				t.Errorf("el error interno de la base no debe quedar expuesto")
			}

			// And: el acumulado y los registros siguen igual.
			if total, _ := store.AccumulatedHours(ctx, "S1"); total != 400 {
				t.Errorf("el acumulado debía seguir en 400, fue %d", total)
			}
			if len(store.Entries()) != 1 {
				t.Errorf("debía seguir habiendo 1 registro, hay %d", len(store.Entries()))
			}
		})
	}
}

// FR-015 y SC-006: el acumulado es la suma de los registros guardados y es reproducible.
func TestAccumulatedHoursEqualsSumOfEntriesAndIsReproducible(t *testing.T) {
	service, store := newTestService(t)
	ctx := context.Background()

	for _, hours := range []string{"4", "2,5", "0,05", "1,25"} {
		req := validRequest()
		req.Hours = hours
		if r := service.LogEffort(ctx, req); !r.Succeeded() {
			t.Fatalf("la carga de %s horas debía registrarse: %+v", hours, r)
		}
	}

	var sum Hours
	for _, e := range store.Entries() {
		sum = sum.Add(e.Hours)
	}
	first, _ := store.AccumulatedHours(ctx, "S1")
	second, _ := store.AccumulatedHours(ctx, "S1")

	if first != sum {
		t.Errorf("el acumulado (%d) debe ser la suma de los registros (%d)", first, sum)
	}
	if first != second {
		t.Errorf("consultar dos veces debe dar lo mismo: %d y %d", first, second)
	}
	if first != 780 {
		t.Errorf("el acumulado debía ser 7,8 horas (780), fue %d", first)
	}
}
