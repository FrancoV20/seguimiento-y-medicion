package sprint

import (
	"context"
	"errors"
	"testing"
)

// HU-14 (T006): servicio ChangeStatus contra PostgreSQL real con fixtures propias.

// sinEstadoEsperado: el formulario no envió estado_actual.
const sinEstadoEsperado StoryStatus = ""

// idInexistente no puede ser el id de ninguna historia de la base de pruebas.
const idInexistente int64 = 2147483647

// FR-004, FR-005, FR-007 (SC-001): cada transición permitida actualiza el estado.
func TestChangeStatus_CadaTransicionActualizaElEstado(t *testing.T) {
	casos := []struct {
		nombre   string
		from, to StoryStatus
	}{
		{"comenzar", EnSprint, InProgress},
		{"completar", InProgress, Done},
		{"reabrir", Done, InProgress},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			service, db := newStatusServiceForTest(t)
			projectID, storyID := newStoryInSprint(t, db, SprintActive, c.from)

			result, err := service.ChangeStatus(context.Background(), projectID, storyID, c.from, c.to, sinEstadoEsperado)

			if err != nil {
				t.Fatalf("ChangeStatus() devolvió error: %v", err)
			}
			if result.From != c.from || result.To != c.to {
				t.Errorf("resultado = %+v, se esperaba %q a %q", result, c.from, c.to)
			}
			wantStoryStatus(t, db, storyID, c.to)
		})
	}
}

// FR-005, FR-006 (SC-005): la advertencia de horas aparece solo al completar sin horas.
func TestChangeStatus_AdvertenciaDeHorasSoloAlCompletarSinHoras(t *testing.T) {
	casos := []struct {
		nombre            string
		from, to          StoryStatus
		conHoras          bool
		quiereAdvertencia bool
	}{
		{"completar con horas", InProgress, Done, true, false},
		{"completar sin horas", InProgress, Done, false, true},
		{"comenzar sin horas", EnSprint, InProgress, false, false},
		{"reabrir sin horas", Done, InProgress, false, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			service, db := newStatusServiceForTest(t)
			projectID, storyID := newStoryInSprint(t, db, SprintActive, c.from)
			if c.conHoras {
				addStatusEffort(t, db, storyID)
			}

			result, err := service.ChangeStatus(context.Background(), projectID, storyID, c.from, c.to, sinEstadoEsperado)

			if err != nil {
				t.Fatalf("ChangeStatus() devolvió error: %v", err)
			}
			if result.NoHoursWarning != c.quiereAdvertencia {
				t.Errorf("NoHoursWarning = %v, se esperaba %v", result.NoHoursWarning, c.quiereAdvertencia)
			}
		})
	}
}

// FR-014: una historia inexistente o de otro proyecto se rechaza y no cambia nada.
func TestChangeStatus_HistoriaInexistenteODeOtroProyecto(t *testing.T) {
	t.Run("inexistente", func(t *testing.T) {
		service, db := newStatusServiceForTest(t)
		projectID, _ := newStoryInSprint(t, db, SprintActive, EnSprint)

		_, err := service.ChangeStatus(context.Background(), projectID, idInexistente, EnSprint, InProgress, sinEstadoEsperado)

		wantDomainError(t, err, ErrStoryNotFound)
	})
	t.Run("de otro proyecto", func(t *testing.T) {
		service, db := newStatusServiceForTest(t)
		_, storyID := newStoryInSprint(t, db, SprintActive, EnSprint)
		otroProyecto := newStatusProject(t, db, "Otro proyecto de HU-14")

		_, err := service.ChangeStatus(context.Background(), otroProyecto, storyID, EnSprint, InProgress, sinEstadoEsperado)

		wantDomainError(t, err, ErrStoryNotFound)
		wantStoryStatus(t, db, storyID, EnSprint)
	})
}

// FR-008 (SC-002): una historia sin Sprint o de un Sprint finalizado se rechaza sin cambios.
func TestChangeStatus_FueraDeUnSprintActivo(t *testing.T) {
	t.Run("Pendiente sin Sprint", func(t *testing.T) {
		service, db := newStatusServiceForTest(t)
		projectID := newStatusProject(t, db, "Proyecto de HU-14")
		storyID := newStatusStory(t, db, projectID, Pendiente, 0)

		_, err := service.ChangeStatus(context.Background(), projectID, storyID, EnSprint, InProgress, sinEstadoEsperado)

		wantDomainError(t, err, ErrNotInActiveSprint)
		wantStoryStatus(t, db, storyID, Pendiente)
	})
	t.Run("en un Sprint finalizado", func(t *testing.T) {
		service, db := newStatusServiceForTest(t)
		projectID, storyID := newStoryInSprint(t, db, SprintFinished, InProgress)

		_, err := service.ChangeStatus(context.Background(), projectID, storyID, InProgress, Done, sinEstadoEsperado)

		wantDomainError(t, err, ErrNotInActiveSprint)
		wantStoryStatus(t, db, storyID, InProgress)
	})
}

// FR-009 (SC-003): pedir el estado en el que ya está no cambia nada e informa el estado.
func TestChangeStatus_YaEstaEnElEstadoPedido(t *testing.T) {
	service, db := newStatusServiceForTest(t)
	projectID, storyID := newStoryInSprint(t, db, SprintActive, Done)

	_, err := service.ChangeStatus(context.Background(), projectID, storyID, InProgress, Done, sinEstadoEsperado)

	wantDomainError(t, err, ErrAlreadyInStatus)
	var ya *AlreadyInStatusError
	if !errors.As(err, &ya) || ya.Status != Done {
		t.Errorf("el error debía informar el estado %q: %v", Done, err)
	}
	wantStoryStatus(t, db, storyID, Done)
}

// FR-011: si el estado real difiere del que la persona veía, no se cambia nada.
func TestChangeStatus_EstadoEsperadoDistintoDelReal(t *testing.T) {
	t.Run("distinto del real", func(t *testing.T) {
		service, db := newStatusServiceForTest(t)
		projectID, storyID := newStoryInSprint(t, db, SprintActive, InProgress)

		_, err := service.ChangeStatus(context.Background(), projectID, storyID, InProgress, Done, EnSprint)

		wantDomainError(t, err, ErrStatusChanged)
		wantStoryStatus(t, db, storyID, InProgress)
	})
	t.Run("igual al real se aplica", func(t *testing.T) {
		service, db := newStatusServiceForTest(t)
		projectID, storyID := newStoryInSprint(t, db, SprintActive, InProgress)

		if _, err := service.ChangeStatus(context.Background(), projectID, storyID, InProgress, Done, InProgress); err != nil {
			t.Fatalf("ChangeStatus() devolvió error: %v", err)
		}
		wantStoryStatus(t, db, storyID, Done)
	})
}

// FR-010: una transición que no es una de las tres permitidas se rechaza con origen y destino.
func TestChangeStatus_TransicionNoPermitida(t *testing.T) {
	casos := []struct {
		nombre         string
		real           StoryStatus
		from, to       StoryStatus
		wantFrom, want StoryStatus
	}{
		// Completar una historia que sigue En Sprint: del real (En Sprint) a Hecho.
		{"En Sprint a Hecho", EnSprint, InProgress, Done, EnSprint, Done},
		{"En progreso a En Sprint", InProgress, InProgress, EnSprint, InProgress, EnSprint},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			service, db := newStatusServiceForTest(t)
			projectID, storyID := newStoryInSprint(t, db, SprintActive, c.real)

			_, err := service.ChangeStatus(context.Background(), projectID, storyID, c.from, c.to, sinEstadoEsperado)

			wantDomainError(t, err, ErrTransitionNotAllowed)
			var noPermitida *TransitionNotAllowedError
			if !errors.As(err, &noPermitida) || noPermitida.From != c.wantFrom || noPermitida.To != c.want {
				t.Errorf("el error debía informar %q a %q: %v", c.wantFrom, c.want, err)
			}
			wantStoryStatus(t, db, storyID, c.real)
		})
	}
}

// Punto 5 (decisión del 24/10/2026): comenzar y reabrir comparten destino; cada acción exige su
// origen y, si no coincide, se rechaza como estado desactualizado, sin cambios.
func TestChangeStatus_AccionQueNoCorrespondeAlEstado(t *testing.T) {
	casos := []struct {
		nombre   string
		real     StoryStatus
		from, to StoryStatus
	}{
		{"comenzar sobre una historia Hecho", Done, EnSprint, InProgress},
		{"reabrir sobre una historia En Sprint", EnSprint, Done, InProgress},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			service, db := newStatusServiceForTest(t)
			projectID, storyID := newStoryInSprint(t, db, SprintActive, c.real)

			_, err := service.ChangeStatus(context.Background(), projectID, storyID, c.from, c.to, sinEstadoEsperado)

			wantDomainError(t, err, ErrStatusChanged)
			wantStoryStatus(t, db, storyID, c.real)
		})
	}
}

// Orden de validación: cuando se cumplen dos condiciones de rechazo, gana la que viene primero
// (historia del proyecto, Sprint activo, ya aplicado, estado esperado, transición permitida).
func TestChangeStatus_OrdenDeValidacion(t *testing.T) {
	casos := []struct {
		nombre       string
		sprint       SprintStatus
		real         StoryStatus
		otroProyecto bool
		from, to     StoryStatus
		expected     StoryStatus
		want         error
	}{
		{"el proyecto antes que el Sprint", SprintFinished, InProgress, true, InProgress, Done, sinEstadoEsperado, ErrStoryNotFound},
		{"el Sprint activo antes que ya aplicado", SprintFinished, Done, false, InProgress, Done, sinEstadoEsperado, ErrNotInActiveSprint},
		{"ya aplicado antes que el estado esperado", SprintActive, Done, false, InProgress, Done, EnSprint, ErrAlreadyInStatus},
		{"el estado esperado antes que la transición", SprintActive, EnSprint, false, InProgress, Done, InProgress, ErrStatusChanged},
		{"la transición antes que el origen de la acción", SprintActive, EnSprint, false, InProgress, Done, sinEstadoEsperado, ErrTransitionNotAllowed},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			service, db := newStatusServiceForTest(t)
			projectID, storyID := newStoryInSprint(t, db, c.sprint, c.real)
			if c.otroProyecto {
				projectID = newStatusProject(t, db, "Otro proyecto de HU-14")
			}

			_, err := service.ChangeStatus(context.Background(), projectID, storyID, c.from, c.to, c.expected)

			wantDomainError(t, err, c.want)
			wantStoryStatus(t, db, storyID, c.real)
		})
	}
}
