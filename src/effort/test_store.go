package effort

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TestStore es un EffortStore en memoria para las pruebas unitarias del servicio.
// No reemplaza a PostgreSQLEffortStore: la integración con PostgreSQL se prueba aparte.
type TestStore struct {
	mu sync.Mutex

	stories map[string]testStory       // historias conocidas
	members map[string]map[string]bool // integrante -> proyectos a los que pertenece
	entries []EffortEntry              // registros guardados

	// SaveErr, si no es nil, hace que SaveEntry falle sin guardar nada.
	SaveErr error
	// QueryErr, si no es nil, hace que las consultas de contexto fallen.
	QueryErr error
	// SaveCalls cuenta cuántas veces se llamó a SaveEntry (incluye las que fallan).
	SaveCalls int
	// Now devuelve la fecha de creación de los registros; por defecto time.Now.
	Now func() time.Time
}

type testStory struct {
	projectID      string
	inActiveSprint bool
}

// NewTestStore crea un store vacío.
func NewTestStore() *TestStore {
	return &TestStore{
		stories: make(map[string]testStory),
		members: make(map[string]map[string]bool),
		Now:     time.Now,
	}
}

// AddStory registra una historia de un proyecto, con o sin Sprint activo.
func (s *TestStore) AddStory(storyID, projectID string, inActiveSprint bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stories[storyID] = testStory{projectID: projectID, inActiveSprint: inActiveSprint}
}

// AddMember registra que un integrante pertenece a un proyecto.
func (s *TestStore) AddMember(memberID, projectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.members[memberID] == nil {
		s.members[memberID] = make(map[string]bool)
	}
	s.members[memberID][projectID] = true
}

// Entries devuelve una copia de los registros guardados.
func (s *TestStore) Entries() []EffortEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]EffortEntry(nil), s.entries...)
}

func (s *TestStore) StoryAcceptsEffort(_ context.Context, storyID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.QueryErr != nil {
		return false, s.QueryErr
	}
	story, ok := s.stories[storyID]
	return ok && story.inActiveSprint, nil
}

func (s *TestStore) MemberBelongsToStoryProject(_ context.Context, memberID, storyID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.QueryErr != nil {
		return false, s.QueryErr
	}
	story, ok := s.stories[storyID]
	if !ok {
		return false, nil
	}
	return s.members[memberID][story.projectID], nil
}

func (s *TestStore) SaveEntry(_ context.Context, entry EffortEntry) (EffortEntry, Hours, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SaveCalls++
	if s.SaveErr != nil {
		return EffortEntry{}, 0, s.SaveErr
	}
	entry.ID = fmt.Sprintf("entry-%d", len(s.entries)+1)
	entry.CreatedAt = s.Now()
	s.entries = append(s.entries, entry)
	return entry, s.sumLocked(entry.StoryID), nil
}

func (s *TestStore) AccumulatedHours(_ context.Context, storyID string) (Hours, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.QueryErr != nil {
		return 0, s.QueryErr
	}
	return s.sumLocked(storyID), nil
}

// sumLocked suma las horas de una historia; el llamador debe tener tomado el mutex.
func (s *TestStore) sumLocked(storyID string) Hours {
	var total Hours
	for _, e := range s.entries {
		if e.StoryID == storyID {
			total = total.Add(e.Hours)
		}
	}
	return total
}

// Garantiza en compilación que TestStore cumple el contrato.
var _ EffortStore = (*TestStore)(nil)
