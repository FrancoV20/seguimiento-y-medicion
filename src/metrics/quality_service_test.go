package metrics

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// T006 (HU-10): servicio QualityService con un almacén de prueba local (sin PostgreSQL).

type qualityStoreStub struct {
	data  QualityData
	err   error
	calls int

	gotProjectID, gotSprintID int64
}

func (s *qualityStoreStub) LoadSprintQuality(_ context.Context, projectID, sprintID int64) (QualityData, error) {
	s.calls++
	s.gotProjectID, s.gotSprintID = projectID, sprintID
	if s.err != nil {
		return QualityData{}, s.err
	}
	return s.data, nil
}

var _ QualityStore = (*qualityStoreStub)(nil)

func TestQualityService_SprintQuality(t *testing.T) {
	ctx := context.Background()

	t.Run("projectID o sprintID en 0 devuelve ErrSprintNotFound sin llamar al almacén", func(t *testing.T) {
		for _, ids := range [][2]int64{{0, 5}, {3, 0}, {-1, 5}, {3, -2}} {
			store := &qualityStoreStub{}
			_, err := NewQualityService(store).SprintQuality(ctx, ids[0], ids[1])
			if !errors.Is(err, ErrSprintNotFound) {
				t.Errorf("proyecto %d, Sprint %d: error = %v; se esperaba ErrSprintNotFound", ids[0], ids[1], err)
			}
			if store.calls != 0 {
				t.Errorf("proyecto %d, Sprint %d: el almacén no debía llamarse (%d llamadas)", ids[0], ids[1], store.calls)
			}
		}
	})

	t.Run("ErrSprintNotFound del almacén se devuelve sin cambios", func(t *testing.T) {
		store := &qualityStoreStub{err: ErrSprintNotFound}
		_, err := NewQualityService(store).SprintQuality(ctx, 3, 5)
		if err != ErrSprintNotFound {
			t.Errorf("error = %v; se esperaba exactamente ErrSprintNotFound", err)
		}
		if store.gotProjectID != 3 || store.gotSprintID != 5 {
			t.Errorf("el almacén recibió proyecto %d y Sprint %d; se esperaban 3 y 5", store.gotProjectID, store.gotSprintID)
		}
	})

	t.Run("un Sprint Activo devuelve ErrSprintNotFinished", func(t *testing.T) {
		store := &qualityStoreStub{data: QualityData{SprintID: 5, Status: "Activo"}}
		got, err := NewQualityService(store).SprintQuality(ctx, 3, 5)
		if !errors.Is(err, ErrSprintNotFinished) {
			t.Errorf("error = %v; se esperaba ErrSprintNotFinished", err)
		}
		if !reflect.DeepEqual(got, SprintQuality{}) {
			t.Errorf("no debía devolver métricas de un Sprint activo: %+v", got)
		}
	})

	t.Run("otra falla del almacén se envuelve conservando la causa", func(t *testing.T) {
		cause := errors.New("connection refused")
		store := &qualityStoreStub{err: fmt.Errorf("load sprint quality 5: %w", cause)}
		_, err := NewQualityService(store).SprintQuality(ctx, 3, 5)
		if err == nil || !errors.Is(err, cause) {
			t.Errorf("error = %v; se esperaba una falla que conserve la causa original", err)
		}
		if errors.Is(err, ErrSprintNotFound) || errors.Is(err, ErrSprintNotFinished) {
			t.Errorf("una falla de lectura no debía confundirse con Sprint inexistente o no finalizado: %v", err)
		}
	})

	t.Run("almacén nil devuelve un error", func(t *testing.T) {
		for name, service := range map[string]*QualityService{
			"NewQualityService(nil)": NewQualityService(nil),
			"valor cero":             {},
		} {
			_, err := service.SprintQuality(ctx, 3, 5)
			if err == nil {
				t.Errorf("%s: se esperaba un error con el almacén nil", name)
			}
		}
	})

	t.Run("un Sprint Finalizado devuelve el resultado de calculateSprintQuality", func(t *testing.T) {
		data := QualityData{
			SprintID: 5,
			Status:   "Finalizado",
			Stories: []QualityStoryResult{
				{FinalStatus: "Hecho", StoryPoints: intPointer(3)},
				{FinalStatus: "Hecho", StoryPoints: intPointer(5)},
				{FinalStatus: "En progreso", StoryPoints: intPointer(8)},
				{FinalStatus: "En Sprint", StoryPoints: nil},
			},
			DefectsDetected: 4,
			DefectsResolved: 1,
		}
		got, err := NewQualityService(&qualityStoreStub{data: data}).SprintQuality(ctx, 3, 5)
		if err != nil {
			t.Fatalf("no se esperaba error: %v", err)
		}
		want := SprintQuality{
			SprintID: 5, StoriesPlanned: 4, StoriesCompleted: 2, CompletionPercentage: "50.00",
			PointsPlanned: 16, PointsCompleted: 8, StoriesWithoutPoints: 1,
			DefectsDetected: 4, DefectsResolved: 1,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("SprintQuality()\n got  %+v\n want %+v", got, want)
		}
		if !reflect.DeepEqual(got, calculateSprintQuality(data)) {
			t.Errorf("el servicio debía devolver exactamente calculateSprintQuality(data)")
		}
	})
}
