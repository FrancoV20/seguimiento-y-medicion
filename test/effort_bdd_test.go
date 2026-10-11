package test

import (
	"context"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// Escenarios BDD de HU-03 (Issue #4). Cada escenario sigue el formato Given-When-Then y usa
// las reglas reales de validación, aritmética de horas y servicio; solo el almacenamiento es
// el store en memoria de pruebas.

var today = time.Date(2026, time.October, 10, 15, 0, 0, 0, time.UTC)

// givenActiveSprintStory prepara: un Product Builder con una historia en el Sprint activo.
func givenActiveSprintStory() (*effort.Service, *effort.TestStore) {
	store := effort.NewTestStore()
	store.Now = func() time.Time { return today }
	store.AddStory("HU-03", "P1", true)
	store.AddMember("M1", "P1")
	return effort.NewService(store, func() time.Time { return today }), store
}

func request(activity, hours string) effort.LogEffortRequest {
	return effort.LogEffortRequest{
		StoryID:  "HU-03",
		MemberID: "M1",
		WorkDate: time.Date(2026, time.October, 10, 0, 0, 0, 0, time.UTC),
		Activity: activity,
		Hours:    hours,
	}
}

// Escenario: Carga de esfuerzo diario.
//
//	Given un Product Builder con una historia asignada en el Sprint activo
//	When ingresa la fecha actual, la actividad "Desarrollo de endpoint en Go" y "4" horas
//	Then el sistema suma esas 4 horas al esfuerzo real acumulado de la historia
func TestScenarioDailyEffortLog(t *testing.T) {
	// Given
	service, store := givenActiveSprintStory()

	// When
	result := service.LogEffort(context.Background(), request("Desarrollo de endpoint en Go", "4"))

	// Then
	if !result.Succeeded() {
		t.Fatalf("el registro debía tener éxito: %+v", result)
	}
	if result.AccumulatedHours != 400 {
		t.Errorf("el acumulado debía ser 4 horas (400), fue %d", result.AccumulatedHours)
	}
	accumulated, err := store.AccumulatedHours(context.Background(), "HU-03")
	if err != nil || accumulated != 400 {
		t.Errorf("el acumulado consultado debía ser 400, fue %d (err=%v)", accumulated, err)
	}
}

// Escenario: Carga de esfuerzo sobre una historia con horas previas.
//
//	Given una historia que ya tiene horas acumuladas
//	When se registra un nuevo esfuerzo válido sobre ella
//	Then el acumulado es la suma de todos los registros válidos, sin sobrescribir los anteriores
func TestScenarioEffortAddsToPreviousHours(t *testing.T) {
	// Given
	service, store := givenActiveSprintStory()
	if r := service.LogEffort(context.Background(), request("Desarrollo de endpoint en Go", "4")); !r.Succeeded() {
		t.Fatalf("el escenario previo debía registrar 4 horas: %+v", r)
	}

	// When
	result := service.LogEffort(context.Background(), request("Revisión de código", "2,5"))

	// Then
	if !result.Succeeded() {
		t.Fatalf("el registro debía tener éxito: %+v", result)
	}
	if result.AccumulatedHours != 650 {
		t.Errorf("el acumulado debía ser 6,5 horas (650), fue %d", result.AccumulatedHours)
	}
	if n := len(store.Entries()); n != 2 {
		t.Errorf("debían conservarse los 2 registros, hay %d", n)
	}
}
