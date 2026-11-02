package report

import "context"

// Store es el puerto de lectura de los datos propios de los reportes (HU-09, research.md §7).
type Store interface {
	// LoadSprintReport devuelve ErrSprintNotFound si el Sprint no existe en el proyecto.
	LoadSprintReport(ctx context.Context, projectID, sprintID int64) (Data, error)
	// LoadProjectReport devuelve ErrProjectNotFound si el proyecto no existe.
	LoadProjectReport(ctx context.Context, projectID int64) (ProjectData, error)
}
