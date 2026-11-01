package web

import "context"

// StoryRow is the read-only representation of a backlog story shown in the web UI.
type StoryRow struct {
	ID               int64
	Title            string
	Priority         string
	StoryPoints      *int
	EstimatedHours   *float64
	Status           string
	InFinishedSprint bool // HU-13: asignada a un Sprint 'Finalizado'; no se edita
}

// StoryEdit son los datos actuales de una historia para su formulario de edición (HU-13).
type StoryEdit struct {
	ID               int64
	Title            string
	Description      string
	Priority         string
	StoryPoints      *int
	Criteria         []string // ordenados por posición
	InFinishedSprint bool
}

// StoryCatalog reads backlog stories for a project.
type StoryCatalog interface {
	// ListStories devuelve las historias del proyecto ordenadas por prioridad y antigüedad (HU-13).
	ListStories(ctx context.Context, projectID int64) ([]StoryRow, error)
	// StoryForEdit devuelve la historia del proyecto para editarla, o un error NotFound (HU-13).
	StoryForEdit(ctx context.Context, projectID, storyID int64) (StoryEdit, error)
}
