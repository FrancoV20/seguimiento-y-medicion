package backlog

import "context"

// BacklogStore abstrae la persistencia del Product Backlog (T009).
type BacklogStore interface {
	// ProjectIsActive indica si el proyecto existe y está en estado Activo.
	ProjectIsActive(projectID int64) (bool, error)
	// CreateStory registra la historia y sus criterios como una unidad
	// y devuelve la historia con el ID asignado.
	CreateStory(projectID int64, story UserStory) (UserStory, error)
	// UpdateStory reemplaza el título, la descripción, la prioridad, los Story Points y los criterios
	// de la historia del proyecto como una unidad (HU-13). Devuelve ErrStoryNotFound si la historia no
	// es del proyecto y ErrStoryInFinishedSprint si pertenece a un Sprint finalizado.
	UpdateStory(ctx context.Context, projectID int64, story UserStory) error
}
