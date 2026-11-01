package backlog

// Edición de una historia del Product Backlog (HU-13, T024). Los errores ErrStoryNotFound y
// ErrStoryInFinishedSprint los define HU-17 (update_estimated_hours.go) y HU-13 los reutiliza.

// UpdateStoryRequest son los datos que el Scrum Master cambia de una historia. El estado, las horas
// estimadas, el Sprint y la fecha de creación no se editan.
type UpdateStoryRequest struct {
	ProjectID          int64
	StoryID            int64
	Title              string
	Description        string
	Priority           Priority
	AcceptanceCriteria []string
	StoryPoints        *int // nil = sin estimar
}

// UpdateStoryResult es el resultado de una edición exitosa.
type UpdateStoryResult struct {
	Story   *UserStory
	Warning string // WarnEstimationPending si la historia queda sin Story Points
}
