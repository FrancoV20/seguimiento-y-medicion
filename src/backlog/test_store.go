package backlog

import (
	"context"
	"errors"
)

// InMemoryBacklogStore es un store controlado para pruebas unitarias (T011).
// No reemplaza las pruebas de integración contra PostgreSQL.
type InMemoryBacklogStore struct {
	ActiveProjects map[int64]bool
	Stories        []UserStory
	FailOnCreate   bool  // simula una falla de persistencia
	CreateCalls    int   // cuántas veces se llamó a CreateStory
	UpdateErr      error // HU-13: error que devuelve UpdateStory (por ejemplo, ErrStoryInFinishedSprint)
	FailOnUpdate   bool  // HU-13: simula una falla de persistencia al editar
	UpdateCalls    int   // HU-13: cuántas veces se llamó a UpdateStory
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

// UpdateStory reemplaza la historia con ese ID del proyecto, o devuelve ErrStoryNotFound (HU-13).
// Conserva el estado, las horas estimadas y la fecha de creación, como el almacén real.
func (s *InMemoryBacklogStore) UpdateStory(_ context.Context, projectID int64, story UserStory) error {
	s.UpdateCalls++
	if s.UpdateErr != nil {
		return s.UpdateErr
	}
	if s.FailOnUpdate {
		return errors.New("falla simulada de almacenamiento")
	}
	for i, current := range s.Stories {
		if current.ID == story.ID && current.ProjectID == projectID {
			story.ProjectID = projectID
			story.Status = current.Status
			story.EstimatedHours = current.EstimatedHours
			story.CreatedAt = current.CreatedAt
			s.Stories[i] = story
			return nil
		}
	}
	return ErrStoryNotFound
}
