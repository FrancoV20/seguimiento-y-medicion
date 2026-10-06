package backlog

import "errors"

// InMemoryBacklogStore es un store controlado para pruebas unitarias (T011).
// No reemplaza las pruebas de integración contra PostgreSQL.
type InMemoryBacklogStore struct {
	ActiveProjects map[int64]bool
	Stories        []UserStory
	FailOnCreate   bool // simula una falla de persistencia
	CreateCalls    int  // cuántas veces se llamó a CreateStory
	nextID         int64
}

// NewInMemoryBacklogStore crea un store con los proyectos activos indicados.
func NewInMemoryBacklogStore(activeProjectIDs ...int64) *InMemoryBacklogStore {
	s := &InMemoryBacklogStore{ActiveProjects: map[int64]bool{}}
	for _, id := range activeProjectIDs {
		s.ActiveProjects[id] = true
	}
	return s
}

func (s *InMemoryBacklogStore) ProjectIsActive(projectID int64) (bool, error) {
	return s.ActiveProjects[projectID], nil
}

func (s *InMemoryBacklogStore) CreateStory(projectID int64, story UserStory) (UserStory, error) {
	s.CreateCalls++
	if s.FailOnCreate {
		return UserStory{}, errors.New("falla simulada de almacenamiento")
	}
	s.nextID++
	story.ID = s.nextID
	story.ProjectID = projectID
	s.Stories = append(s.Stories, story)
	return story, nil
}
