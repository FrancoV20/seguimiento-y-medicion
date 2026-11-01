package backlog

import (
	"context"
	"errors"
)

// UpdateStory valida los datos con la misma regla que el alta y reemplaza la historia del proyecto
// (HU-13, T026). Devuelve los errores de HU-02, ErrStoryNotFound, ErrStoryInFinishedSprint o, ante
// cualquier otra falla del almacén, ErrPersistence. No revisa el estado guardado del proyecto: el
// proyecto ya lo valida el contexto de HU-20b y la búsqueda de la historia dentro de él.
func (s *BacklogService) UpdateStory(ctx context.Context, req UpdateStoryRequest) (*UpdateStoryResult, error) {
	if req.ProjectID <= 0 || req.StoryID <= 0 {
		return nil, ErrStoryNotFound
	}
	fields, err := validateStoryFields(req.Title, req.Description, req.Priority, req.AcceptanceCriteria, req.StoryPoints)
	if err != nil {
		return nil, err
	}

	story := UserStory{
		ID:                 req.StoryID,
		ProjectID:          req.ProjectID,
		Title:              fields.Title,
		Description:        fields.Description,
		Priority:           fields.Priority,
		AcceptanceCriteria: fields.Criteria,
		StoryPoints:        fields.StoryPoints,
	}
	if err := s.store.UpdateStory(ctx, req.ProjectID, story); err != nil {
		switch {
		case errors.Is(err, ErrStoryNotFound):
			return nil, ErrStoryNotFound
		case errors.Is(err, ErrStoryInFinishedSprint):
			return nil, ErrStoryInFinishedSprint
		default:
			return nil, ErrPersistence
		}
	}

	result := &UpdateStoryResult{Story: &story}
	if story.StoryPoints == nil {
		result.Warning = WarnEstimationPending
	}
	return result, nil
}
