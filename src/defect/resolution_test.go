package defect

import (
	"context"
	"errors"
	"testing"
)

// HU-19: reglas de resolución y reapertura (sin base de datos) y servicio con un almacén de prueba.

// T005 [E1]: un defecto "Abierto" con Sprint activo igual o posterior al de detección se puede resolver.
func TestCheckResolve_OK(t *testing.T) {
	cases := []struct {
		name              string
		detection, active int64
	}{
		{name: "mismo Sprint", detection: 3, active: 3},
		{name: "Sprint posterior", detection: 3, active: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckResolve(StatusOpen, tc.detection, tc.active, true); err != nil {
				t.Errorf("CheckResolve(Abierto, detección %d, activo %d) = %v; se esperaba sin error",
					tc.detection, tc.active, err)
			}
		})
	}
}

// T006 [E2]: sin Sprint activo no se puede resolver.
func TestCheckResolve_SinSprint(t *testing.T) {
	err := CheckResolve(StatusOpen, 3, 0, false)
	if !errors.Is(err, ErrNoActiveSprint) {
		t.Errorf("CheckResolve sin Sprint activo = %v; se esperaba ErrNoActiveSprint", err)
	}
}

// T007 [E4]: el Sprint de resolución no puede ser anterior al de detección.
func TestCheckResolve_Anterior(t *testing.T) {
	err := CheckResolve(StatusOpen, 3, 2, true)
	if !errors.Is(err, ErrResolutionBeforeDetection) {
		t.Errorf("CheckResolve(detección 3, activo 2) = %v; se esperaba ErrResolutionBeforeDetection", err)
	}
}

// T008 [E7]: un defecto ya resuelto no se vuelve a resolver, aunque no haya Sprint activo o el
// Sprint activo sea anterior: el estado se informa primero.
func TestCheckResolve_YaResuelto(t *testing.T) {
	cases := []struct {
		name      string
		active    int64
		hasActive bool
	}{
		{name: "con Sprint activo", active: 3, hasActive: true},
		{name: "sin Sprint activo", active: 0, hasActive: false},
		{name: "Sprint activo anterior", active: 1, hasActive: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckResolve(StatusResolved, 2, tc.active, tc.hasActive)
			if !errors.Is(err, ErrAlreadyResolved) {
				t.Errorf("CheckResolve(Resuelto) = %v; se esperaba ErrAlreadyResolved", err)
			}
		})
	}
}

// T009 [E3]: un defecto "Resuelto" se puede reabrir.
func TestCheckReopen_OK(t *testing.T) {
	if err := CheckReopen(StatusResolved); err != nil {
		t.Errorf("CheckReopen(Resuelto) = %v; se esperaba sin error", err)
	}
}

// T009 [E8]: un defecto "Abierto" no se puede reabrir.
func TestCheckReopen_YaAbierto(t *testing.T) {
	if err := CheckReopen(StatusOpen); !errors.Is(err, ErrAlreadyOpen) {
		t.Errorf("CheckReopen(Abierto) = %v; se esperaba ErrAlreadyOpen", err)
	}
}

// Los estados desconocidos no se resuelven ni se reabren.
func TestCheckResolveYReopen_EstadoDesconocido(t *testing.T) {
	if err := CheckResolve(Status("Cerrado"), 1, 1, true); err == nil {
		t.Error("CheckResolve con un estado desconocido debía fallar")
	}
	if err := CheckReopen(Status("Cerrado")); err == nil {
		t.Error("CheckReopen con un estado desconocido debía fallar")
	}
}

// Los errores nuevos de HU-19 son distintos entre sí y del de HU-06.
func TestResolutionErrors_SonDistintos(t *testing.T) {
	all := []error{ErrDefectNotFound, ErrResolutionBeforeDetection, ErrAlreadyResolved, ErrAlreadyOpen, ErrNoActiveSprint}
	for i, a := range all {
		if a == nil || a.Error() == "" {
			t.Fatalf("el error %d no tiene mensaje", i)
		}
		for j, b := range all {
			if i != j && errors.Is(a, b) {
				t.Errorf("los errores %q y %q no deben confundirse", a, b)
			}
		}
	}
}

// T010: servicio de resolución con un almacén de prueba.

type resolutionTestStore struct {
	resolved    Resolved
	err         error
	resolveCall int
	reopenCall  int
	projectID   int64
	defectID    int64
}

func (s *resolutionTestStore) Resolve(_ context.Context, projectID, defectID int64) (Resolved, error) {
	s.resolveCall++
	s.projectID, s.defectID = projectID, defectID
	return s.resolved, s.err
}

func (s *resolutionTestStore) Reopen(_ context.Context, projectID, defectID int64) error {
	s.reopenCall++
	s.projectID, s.defectID = projectID, defectID
	return s.err
}

// T010 [E1]: Resolve devuelve el número del Sprint de resolución del almacén.
func TestResolutionService_ResolveDevuelveElSprint(t *testing.T) {
	store := &resolutionTestStore{resolved: Resolved{DefectID: 7, SprintNumber: 3}}
	service := NewResolutionService(store)

	got, err := service.Resolve(context.Background(), 10, 7)

	if err != nil {
		t.Fatalf("Resolve() devolvió error: %v", err)
	}
	if got != (Resolved{DefectID: 7, SprintNumber: 3}) {
		t.Errorf("Resolve() = %+v; se esperaba el defecto 7 resuelto en el Sprint 3", got)
	}
	if store.resolveCall != 1 || store.projectID != 10 || store.defectID != 7 {
		t.Errorf("el almacén recibió llamadas=%d proyecto=%d defecto=%d; se esperaba 1, 10 y 7",
			store.resolveCall, store.projectID, store.defectID)
	}
}

// T010 [E3]: Reopen delega en el almacén.
func TestResolutionService_ReopenDelegaEnElAlmacen(t *testing.T) {
	store := &resolutionTestStore{}
	if err := NewResolutionService(store).Reopen(context.Background(), 10, 7); err != nil {
		t.Fatalf("Reopen() devolvió error: %v", err)
	}
	if store.reopenCall != 1 || store.projectID != 10 || store.defectID != 7 {
		t.Errorf("el almacén recibió llamadas=%d proyecto=%d defecto=%d; se esperaba 1, 10 y 7",
			store.reopenCall, store.projectID, store.defectID)
	}
}

// T010 [E2, E4, E5, E6, E7, E8]: los errores del almacén llegan sin cambios.
func TestResolutionService_PropagaLosErrores(t *testing.T) {
	saveFailure := errors.New("falla al guardar")
	for _, want := range []error{
		ErrNoActiveSprint, ErrResolutionBeforeDetection, ErrAlreadyResolved,
		ErrAlreadyOpen, ErrDefectNotFound, saveFailure,
	} {
		t.Run(want.Error(), func(t *testing.T) {
			store := &resolutionTestStore{err: want}
			service := NewResolutionService(store)

			if _, err := service.Resolve(context.Background(), 10, 7); !errors.Is(err, want) {
				t.Errorf("Resolve() = %v; se esperaba %v", err, want)
			}
			if err := service.Reopen(context.Background(), 10, 7); !errors.Is(err, want) {
				t.Errorf("Reopen() = %v; se esperaba %v", err, want)
			}
		})
	}
}

// [E5]: identificadores no válidos se tratan como defecto inexistente sin consultar el almacén.
func TestResolutionService_IdentificadoresInvalidos(t *testing.T) {
	cases := []struct {
		name                string
		projectID, defectID int64
	}{
		{name: "defecto cero", projectID: 10, defectID: 0},
		{name: "defecto negativo", projectID: 10, defectID: -1},
		{name: "proyecto cero", projectID: 0, defectID: 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &resolutionTestStore{}
			service := NewResolutionService(store)

			if _, err := service.Resolve(context.Background(), tc.projectID, tc.defectID); !errors.Is(err, ErrDefectNotFound) {
				t.Errorf("Resolve() = %v; se esperaba ErrDefectNotFound", err)
			}
			if err := service.Reopen(context.Background(), tc.projectID, tc.defectID); !errors.Is(err, ErrDefectNotFound) {
				t.Errorf("Reopen() = %v; se esperaba ErrDefectNotFound", err)
			}
			if store.resolveCall != 0 || store.reopenCall != 0 {
				t.Errorf("no se debía consultar el almacén: resolve=%d reopen=%d", store.resolveCall, store.reopenCall)
			}
		})
	}
}
