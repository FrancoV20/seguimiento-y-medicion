package web

import (
	"context"

	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
)

// DefectRow es una fila de la lista de defectos.
type DefectRow struct {
	ID               int64
	Description      string
	Severity         defect.Severity
	Status           defect.Status
	StoryID          int64
	StoryTitle       string
	DetectionSprint  int64
	ResolutionSprint *int64
}

// DefectStoryOption es una historia disponible para asociar a un defecto.
type DefectStoryOption struct {
	ID     int64
	Title  string
	Status string
}

// DefectCatalog proporciona los datos de lectura de la pantalla de defectos.
type DefectCatalog interface {
	Defects(ctx context.Context, projectID int64) ([]DefectRow, error)
	StoryOptions(ctx context.Context, projectID int64) ([]DefectStoryOption, error)
	ActiveSprintNumber(ctx context.Context, projectID int64) (int64, bool, error)
}
