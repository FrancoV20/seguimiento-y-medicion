package sprint

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// memorySprintStore mantiene estado controlado para pruebas unitarias.
type memorySprintStore struct {
	mu       sync.RWMutex
	projects map[int64]Project
	stories  map[int64]UserStory
	sprints  map[int64]Sprint
	nextID   int64
}

var _ SprintStore = (*memorySprintStore)(nil)

func newMemorySprintStore() *memorySprintStore {
	return &memorySprintStore{
		projects: make(map[int64]Project),
		stories:  make(map[int64]UserStory),
		sprints:  make(map[int64]Sprint),
		nextID:   1,
	}
}

func (s *memorySprintStore) addProject(project Project) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[project.ID] = project
}

func (s *memorySprintStore) addStory(story UserStory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stories[story.ID] = story
}

func (s *memorySprintStore) story(storyID int64) (UserStory, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	story, exists := s.stories[storyID]
	return story, exists
}

func (s *memorySprintStore) CreateSprint(ctx context.Context, sprint Sprint) (Sprint, error) {
	if err := ctx.Err(); err != nil {
		return Sprint{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.projects[sprint.ProjectID]; !exists {
		return Sprint{}, ErrProjectNotFound
	}
	if err := validateSprint(sprint); err != nil {
		return Sprint{}, err
	}
	for _, existing := range s.sprints {
		if existing.ProjectID == sprint.ProjectID && existing.Status == SprintActive {
			return Sprint{}, &ActiveSprintExistsError{Number: existing.Number}
		}
	}
	storyIDs, err := normalizeStoryIDs(sprint.StoryIDs)
	if err != nil {
		return Sprint{}, err
	}
	candidates := make([]UserStory, 0, len(storyIDs))
	for _, storyID := range storyIDs {
		story, exists := s.stories[storyID]
		if !exists {
			return Sprint{}, fmt.Errorf("%w: historia %d inexistente", ErrStoryNotAvailable, storyID)
		}
		story = s.withSprintNumber(story)
		if err := validateStoryForProject(story, sprint.ProjectID); err != nil {
			return Sprint{}, err
		}
		candidates = append(candidates, story)
	}

	project := s.projects[sprint.ProjectID]
	number := project.NextSprintNumber
	if number <= 0 {
		number = 1
	}
	for _, existing := range s.sprints {
		if existing.ProjectID == sprint.ProjectID && existing.Number >= number {
			number = existing.Number + 1
		}
	}
	displayID, err := formatSprintID(number)
	if err != nil {
		return Sprint{}, err
	}
	sprint.ID = s.nextID
	sprint.Number = number
	sprint.DisplayID = displayID
	sprint.Goal = strings.TrimSpace(sprint.Goal)
	sprint.Status = SprintActive
	sprint.StoryIDs = append([]int64(nil), storyIDs...)

	for i := range candidates {
		if err := AssignStoryToSprint(&candidates[i], sprint.ID); err != nil {
			return Sprint{}, err
		}
		candidates[i].ActiveSprintNumber = number
	}
	for _, story := range candidates {
		s.stories[story.ID] = story
	}
	project.NextSprintNumber = number + 1
	s.projects[sprint.ProjectID] = project
	s.sprints[sprint.ID] = sprint
	s.nextID++
	return sprint, nil
}

func (s *memorySprintStore) AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sprint, exists := s.sprints[sprintID]
	if !exists || sprint.Status != SprintActive {
		return ErrStoryNotAvailable
	}
	ids, err := normalizeStoryIDs(storyIDs)
	if err != nil {
		return err
	}
	candidates := make([]UserStory, 0, len(ids))
	for _, storyID := range ids {
		story, exists := s.stories[storyID]
		if !exists {
			return fmt.Errorf("%w: historia %d inexistente", ErrStoryNotAvailable, storyID)
		}
		if story.ActiveSprintID == sprintID && story.Status == EnSprint {
			candidates = append(candidates, story)
			continue
		}
		story = s.withSprintNumber(story)
		if err := validateStoryForProject(story, sprint.ProjectID); err != nil {
			return err
		}
		candidates = append(candidates, story)
	}
	assigned := make(map[int64]struct{}, len(sprint.StoryIDs)+len(ids))
	for _, storyID := range sprint.StoryIDs {
		assigned[storyID] = struct{}{}
	}
	for i := range candidates {
		if candidates[i].ActiveSprintID != sprintID {
			if err := AssignStoryToSprint(&candidates[i], sprintID); err != nil {
				return err
			}
			candidates[i].ActiveSprintNumber = sprint.Number
		}
		assigned[candidates[i].ID] = struct{}{}
	}
	for _, story := range candidates {
		s.stories[story.ID] = story
	}
	sprint.StoryIDs = sprint.StoryIDs[:0]
	for storyID := range assigned {
		sprint.StoryIDs = append(sprint.StoryIDs, storyID)
	}
	sort.Slice(sprint.StoryIDs, func(i, j int) bool { return sprint.StoryIDs[i] < sprint.StoryIDs[j] })
	s.sprints[sprintID] = sprint
	return nil
}

// withSprintNumber completa el número visible del Sprint al que está asignada la historia,
// para los mensajes de conflicto (FIX-HU-04). Se llama con el mutex tomado.
func (s *memorySprintStore) withSprintNumber(story UserStory) UserStory {
	if assigned, ok := s.sprints[story.ActiveSprintID]; ok {
		story.ActiveSprintNumber = assigned.Number
	}
	return story
}

func (s *memorySprintStore) RemoveStory(ctx context.Context, sprintID int64, storyID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sprint, exists := s.sprints[sprintID]
	if !exists || sprint.Status != SprintActive {
		return &StoryRemovalError{StoryID: storyID}
	}
	story, exists := s.stories[storyID]
	if !exists {
		return &StoryRemovalError{StoryID: storyID}
	}
	if err := RemoveStoryFromSprint(&story, sprintID); err != nil {
		return err
	}
	s.stories[storyID] = story
	remaining := sprint.StoryIDs[:0]
	for _, assignedID := range sprint.StoryIDs {
		if assignedID != storyID {
			remaining = append(remaining, assignedID)
		}
	}
	sprint.StoryIDs = remaining
	s.sprints[sprintID] = sprint
	return nil
}
