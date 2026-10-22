package sprint

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// FIX-HU-04: un solo Sprint activo por proyecto (T002) y estados de las historias (T004).

func finishMemorySprint(t *testing.T, s *memorySprintStore, sprintID int64) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.sprints[sprintID]
	if !ok {
		t.Fatalf("no existe el Sprint %d para finalizarlo", sprintID)
	}
	sp.Status = SprintFinished
	s.sprints[sprintID] = sp
}

func memorySprintsOf(s *memorySprintStore, projectID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, sp := range s.sprints {
		if sp.ProjectID == projectID {
			n++
		}
	}
	return n
}

func memoryStoreWith(projectIDs []int64, storyProject int64, storyIDs ...int64) *memorySprintStore {
	store := newMemorySprintStore()
	for _, id := range projectIDs {
		store.addProject(Project{ID: id, Status: "Activo"})
	}
	for _, id := range storyIDs {
		store.addStory(UserStory{ID: id, ProjectID: storyProject, Status: Pendiente})
	}
	return store
}

func createWithService(t *testing.T, store *memorySprintStore, projectID int64, storyIDs ...int64) (SprintCreationResult, error) {
	t.Helper()
	stories := make([]UserStory, 0, len(storyIDs))
	for _, id := range storyIDs {
		st, _ := store.story(id)
		stories = append(stories, st)
	}
	req := validCreationRequest(storyIDs)
	req.ProjectID = projectID
	return NewService(store).CreateSprint(context.Background(), req, Project{ID: projectID, Status: "Activo"}, stories)
}

// Escenario: Un solo Sprint activo por proyecto.
func TestServicio_SegundoSprintActivoSeRechazaConSuNumero(t *testing.T) {
	store := memoryStoreWith([]int64{1}, 1, 801, 802)
	first, err := createWithService(t, store, 1, 801)
	if err != nil || first.Sprint.DisplayID != "Sprint 1" {
		t.Fatalf("el primer Sprint debía crearse como Sprint 1: %+v (error: %v)", first.Sprint, err)
	}

	_, err = createWithService(t, store, 1, 802)

	var active *ActiveSprintExistsError
	if !errors.Is(err, ErrActiveSprintExists) || !errors.As(err, &active) || active.Number != 1 {
		t.Fatalf("se esperaba ErrActiveSprintExists con el Sprint 1, se obtuvo: %v", err)
	}
	if !strings.Contains(err.Error(), "Sprint 1") {
		t.Errorf("el mensaje debe indicar el Sprint activo: %q", err.Error())
	}
	if n := memorySprintsOf(store, 1); n != 1 {
		t.Errorf("no debía crearse otro Sprint, hay %d", n)
	}
	if st, _ := store.story(802); st.Status != Pendiente || st.ActiveSprintID != 0 {
		t.Errorf("la historia del intento rechazado no debía asignarse: %+v", st)
	}
}

// Escenario: Un rechazo no deja huecos en la numeración.
func TestServicio_RechazoNoDejaHuecosEnLaNumeracion(t *testing.T) {
	store := memoryStoreWith([]int64{1}, 1)
	first, err := createWithService(t, store, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createWithService(t, store, 1); !errors.Is(err, ErrActiveSprintExists) {
		t.Fatalf("el segundo intento debía rechazarse, se obtuvo: %v", err)
	}
	finishMemorySprint(t, store, first.Sprint.ID)

	next, err := createWithService(t, store, 1)

	if err != nil || next.Sprint.DisplayID != "Sprint 2" {
		t.Errorf("después del rechazo se esperaba Sprint 2, se obtuvo %q (error: %v)", next.Sprint.DisplayID, err)
	}
}

// Escenario: Sprint nuevo cuando todos los anteriores están finalizados.
func TestServicio_ConTodosLosSprintsFinalizadosSeCreaUnoActivo(t *testing.T) {
	store := memoryStoreWith([]int64{1}, 1)
	for i := 0; i < 2; i++ {
		created, err := createWithService(t, store, 1)
		if err != nil {
			t.Fatalf("Sprint %d: %v", i+1, err)
		}
		finishMemorySprint(t, store, created.Sprint.ID)
	}

	third, err := createWithService(t, store, 1)

	if err != nil || third.Sprint.DisplayID != "Sprint 3" || third.Sprint.Status != SprintActive {
		t.Errorf("se esperaba Sprint 3 activo, se obtuvo %+v (error: %v)", third.Sprint, err)
	}
}

// Escenario: La regla es por proyecto.
func TestServicio_LaReglaEsPorProyecto(t *testing.T) {
	store := memoryStoreWith([]int64{1, 2}, 1)
	if _, err := createWithService(t, store, 1); err != nil {
		t.Fatal(err)
	}

	other, err := createWithService(t, store, 2)

	if err != nil || other.Sprint.DisplayID != "Sprint 1" {
		t.Errorf("el proyecto B debía poder crear su Sprint 1, se obtuvo %+v (error: %v)", other.Sprint, err)
	}
}

// La validación de dominio también rechaza si el proyecto informa un Sprint activo.
func TestCreateSprint_ProyectoConSprintActivoSeRechazaSinTocarHistorias(t *testing.T) {
	project := Project{ID: 1, Status: "Activo", ActiveSprintNumber: 4}
	stories := []UserStory{{ID: 901, ProjectID: 1, Status: Pendiente}}
	before := stories[0]

	_, err := CreateSprint(validCreationRequest([]int64{901}), project, stories)

	var active *ActiveSprintExistsError
	if !errors.As(err, &active) || active.Number != 4 {
		t.Fatalf("se esperaba ErrActiveSprintExists con el Sprint 4, se obtuvo: %v", err)
	}
	if stories[0] != before {
		t.Errorf("la historia no debía cambiar: %+v", stories[0])
	}
}

// Escenario modificado: Asignación exitosa deja las historias "En Sprint".
func TestServicio_AsignacionDejaLasHistoriasEnSprint(t *testing.T) {
	store := memoryStoreWith([]int64{1}, 1, 811, 812, 813)

	created, err := createWithService(t, store, 1, 811, 812, 813)

	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{811, 812, 813} {
		if st, _ := store.story(id); st.Status != EnSprint || st.ActiveSprintID != created.Sprint.ID {
			t.Errorf("la historia %d debía quedar En Sprint del Sprint creado: %+v", id, st)
		}
	}
}

// Escenario reescrito: Intento de asignar una historia que ya está asignada a un Sprint.
func TestServicio_HistoriaYaAsignadaInformaElSprint(t *testing.T) {
	store := memoryStoreWith([]int64{1}, 1, 821)
	first, err := createWithService(t, store, 1, 821)
	if err != nil {
		t.Fatal(err)
	}
	finishMemorySprint(t, store, first.Sprint.ID)
	second, err := createWithService(t, store, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = store.AssignStories(context.Background(), second.Sprint.ID, []int64{821})

	if !errors.Is(err, ErrActiveSprintConflict) || !strings.Contains(err.Error(), "ya está asignada al Sprint 1") {
		t.Errorf("se esperaba 'ya está asignada al Sprint 1', se obtuvo: %v", err)
	}
}

// Escenario reescrito: Quitar una historia y volver a asignarla.
func TestServicio_QuitarYVolverAAsignarEnElSprintActivo(t *testing.T) {
	store := memoryStoreWith([]int64{1}, 1, 831)
	created, err := createWithService(t, store, 1, 831)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := store.RemoveStory(ctx, created.Sprint.ID, 831); err != nil {
		t.Fatalf("quitar la historia falló: %v", err)
	}
	if st, _ := store.story(831); st.Status != Pendiente || st.ActiveSprintID != 0 {
		t.Fatalf("al quitarla debía volver a Pendiente: %+v", st)
	}
	if err := store.AssignStories(ctx, created.Sprint.ID, []int64{831}); err != nil {
		t.Fatalf("volver a asignarla debía permitirse: %v", err)
	}
	if st, _ := store.story(831); st.Status != EnSprint || st.ActiveSprintID != created.Sprint.ID {
		t.Errorf("al volver a asignarla debía quedar En Sprint: %+v", st)
	}
}
