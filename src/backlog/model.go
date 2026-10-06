package backlog

import "time"

// Priority es la prioridad de una historia de usuario.
type Priority string

const (
	PriorityHigh   Priority = "Alta"
	PriorityMedium Priority = "Media"
	PriorityLow    Priority = "Baja"
)

// IsValid indica si la prioridad pertenece al conjunto permitido.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityHigh, PriorityMedium, PriorityLow:
		return true
	}
	return false
}

// StoryStatus es el estado de una historia de usuario.
type StoryStatus string

// StatusPending es el estado inicial de toda historia creada por HU-02.
const StatusPending StoryStatus = "Pendiente"

// AcceptanceCriterion es un criterio de aceptación de una historia.
type AcceptanceCriterion struct {
	Position int
	Content  string
}

// UserStory es una historia del Product Backlog.
type UserStory struct {
	ID                 int64
	ProjectID          int64
	Title              string
	Description        string
	Priority           Priority
	AcceptanceCriteria []AcceptanceCriterion
	StoryPoints        *int // nil = sin estimación
	Status             StoryStatus
	CreatedAt          time.Time
}

// ProductBacklog es la colección de historias de un proyecto.
type ProductBacklog struct {
	ProjectID int64
	Stories   []UserStory
}

// CreateStoryRequest son los datos que ingresa el Scrum Master.
type CreateStoryRequest struct {
	ProjectID          int64
	Title              string
	Description        string
	Priority           Priority
	AcceptanceCriteria []string
	StoryPoints        *int
}

// CreateStoryResult es el resultado de un alta exitosa.
type CreateStoryResult struct {
	Story          *UserStory
	SuccessMessage string
	Warning        string // alerta de estimación pendiente si no hay Story Points
}
