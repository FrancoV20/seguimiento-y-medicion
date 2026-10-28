package web

import "context"

// StoryRow is the read-only representation of a backlog story shown in the web UI.
type StoryRow struct {
	ID             int64
	Title          string
	Priority       string
	StoryPoints    *int
	EstimatedHours *float64
	Status         string
}

// StoryCatalog reads backlog stories for a project.
type StoryCatalog interface {
	ListStories(ctx context.Context, projectID int64) ([]StoryRow, error)
}
