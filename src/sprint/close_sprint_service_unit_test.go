package sprint

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Ajuste 3 de la revisión de HU-15: ramas del servicio de cierre con puertos falsos, sin base.

type fakeCloseStore struct {
	sprint      Sprint
	findErr     error
	closeErr    error
	stories     []ClosingStory
	result      CloseResult
	closeCalls  int
	checkResult error
}

func (s *fakeCloseStore) FindSprint(context.Context, int64, int64) (Sprint, error) {
	return s.sprint, s.findErr
}

// CloseSprint imita al almacenamiento real: ejecuta check con las historias "bloqueadas" y, si
// la validación falla, devuelve ese error sin cerrar.
func (s *fakeCloseStore) CloseSprint(ctx context.Context, _ int64, check ClosureCheck) (CloseResult, error) {
	s.closeCalls++
	if s.closeErr != nil {
		return CloseResult{}, s.closeErr
	}
	if err := check(ctx, s.stories); err != nil {
		s.checkResult = err
		return CloseResult{}, err
	}
	return s.result, nil
}

type fakeClosureValidator struct {
	err      error
	calls    int
	received []ClosingStory
}

func (v *fakeClosureValidator) ValidateSprintClosure(_ context.Context, stories []ClosingStory) error {
	v.calls++
	v.received = stories
	return v.err
}

func activeFakeSprint() Sprint {
	return Sprint{ID: 10, ProjectID: 1, Number: 4, Status: SprintActive}
}

func TestCloseSprintServiceUnit_ServicioIncompletoEsErrorDePersistencia(t *testing.T) {
	store := &fakeCloseStore{sprint: activeFakeSprint()}
	validator := &fakeClosureValidator{}
	cases := map[string]*CloseSprintService{
		"servicio nulo":  nil,
		"sin almacén":    NewCloseSprintService(nil, validator),
		"sin validación": NewCloseSprintService(store, nil),
	}
	for name, service := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := service.CloseSprint(context.Background(), 1, 10)
			if !errors.Is(err, ErrPersistence) {
				t.Errorf("error = %v, se esperaba ErrPersistence", err)
			}
		})
	}
	if store.closeCalls != 0 {
		t.Errorf("se intentó cerrar %d veces con un servicio incompleto", store.closeCalls)
	}
}

func TestCloseSprintServiceUnit_ErroresAlBuscarElSprint(t *testing.T) {
	cases := []struct {
		name  string
		store *fakeCloseStore
		want  error
	}{
		{"no existe", &fakeCloseStore{findErr: ErrSprintNotFound}, ErrSprintNotFound},
		{"no existe envuelto", &fakeCloseStore{findErr: errors.Join(errors.New("detalle"), ErrSprintNotFound)}, ErrSprintNotFound},
		{"falla de lectura", &fakeCloseStore{findErr: errors.New("pq: SELECT falló en sprints")}, ErrPersistence},
		{"Sprint de otro proyecto", &fakeCloseStore{sprint: Sprint{ID: 10, ProjectID: 2, Number: 1, Status: SprintActive}}, ErrSprintNotFound},
		{"Sprint finalizado", &fakeCloseStore{sprint: Sprint{ID: 10, ProjectID: 1, Number: 1, Status: SprintFinished}}, ErrSprintAlreadyClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &fakeClosureValidator{}
			_, err := NewCloseSprintService(tc.store, validator).CloseSprint(context.Background(), 1, 10)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, se esperaba %v", err, tc.want)
			}
			if err != nil && strings.Contains(err.Error(), "pq:") {
				t.Errorf("el error expone el detalle técnico de la lectura: %v", err)
			}
			if tc.store.closeCalls != 0 || validator.calls != 0 {
				t.Errorf("no debía cerrar ni validar: cierres=%d validaciones=%d", tc.store.closeCalls, validator.calls)
			}
		})
	}
}

func TestCloseSprintServiceUnit_ErroresDelCierreSeTraducen(t *testing.T) {
	cases := []struct {
		name     string
		closeErr error
		want     error
	}{
		{"desapareció durante el cierre", ErrSprintNotFound, ErrSprintNotFound},
		{"otro cierre ganó la carrera", ErrSprintAlreadyClosed, ErrSprintAlreadyClosed},
		{"falla de guardado", errors.New("record sprint 10 story results: pq: timeout"), ErrPersistence},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeCloseStore{sprint: activeFakeSprint(), closeErr: tc.closeErr}
			_, err := NewCloseSprintService(store, &fakeClosureValidator{}).CloseSprint(context.Background(), 1, 10)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, se esperaba %v", err, tc.want)
			}
			if strings.Contains(err.Error(), "pq:") {
				t.Errorf("el error expone el detalle técnico del guardado: %v", err)
			}
		})
	}
}

func TestCloseSprintServiceUnit_ValidaLasHistoriasQueEntregaElCierre(t *testing.T) {
	points := 5
	stories := []ClosingStory{
		{ID: 7, Status: Done, StoryPoints: &points},
		{ID: 8, Status: InProgress},
	}

	t.Run("validación que falla bloquea el cierre", func(t *testing.T) {
		store := &fakeCloseStore{sprint: activeFakeSprint(), stories: stories}
		validator := &fakeClosureValidator{err: errors.New("Historia 7: Story Points inválidos")}

		_, err := NewCloseSprintService(store, validator).CloseSprint(context.Background(), 1, 10)

		if !errors.Is(err, ErrCloseBlocked) {
			t.Fatalf("error = %v, se esperaba ErrCloseBlocked", err)
		}
		if !strings.Contains(err.Error(), "Historia 7: Story Points inválidos") {
			t.Errorf("el error no conserva el motivo de HU-07: %v", err)
		}
		if !errors.Is(store.checkResult, ErrCloseBlocked) {
			t.Errorf("el almacén debía recibir ErrCloseBlocked desde la validación, recibió %v", store.checkResult)
		}
		if !reflect.DeepEqual(validator.received, stories) {
			t.Errorf("la validación recibió %+v, se esperaban las historias del cierre %+v", validator.received, stories)
		}
	})

	t.Run("validación correcta cierra y usa el número del Sprint", func(t *testing.T) {
		store := &fakeCloseStore{
			sprint:  activeFakeSprint(),
			stories: stories,
			result:  CloseResult{SprintNumber: 99, CompletedCount: 1, ReturnedCount: 1},
		}
		validator := &fakeClosureValidator{}

		result, err := NewCloseSprintService(store, validator).CloseSprint(context.Background(), 1, 10)

		if err != nil {
			t.Fatalf("CloseSprint() devolvió error: %v", err)
		}
		want := CloseResult{SprintNumber: 4, CompletedCount: 1, ReturnedCount: 1}
		if result != want {
			t.Errorf("CloseSprint() = %+v, se esperaba %+v", result, want)
		}
		if validator.calls != 1 || store.closeCalls != 1 {
			t.Errorf("validaciones=%d cierres=%d, se esperaba 1 y 1", validator.calls, store.closeCalls)
		}
	})
}
