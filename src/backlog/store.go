package backlog

// BacklogStore abstrae la persistencia del Product Backlog (T009).
type BacklogStore interface {
	// ProjectIsActive indica si el proyecto existe y está en estado Activo.
	ProjectIsActive(projectID int64) (bool, error)
	// CreateStory registra la historia y sus criterios como una unidad
	// y devuelve la historia con el ID asignado.
	CreateStory(projectID int64, story UserStory) (UserStory, error)
}
