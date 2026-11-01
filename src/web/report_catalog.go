package web

import "context"

// Tipos de "Reportes" (HU-09, T032). La pantalla solo lee: el PDF lo arma report.Service.

// ReportSprintOption es un Sprint del proyecto en la lista de "Reportes".
type ReportSprintOption struct {
	ID     int64
	Number int
	Goal   string
	Active bool
}

// ReportCatalog son las lecturas de la pantalla "Reportes" (solo lectura).
type ReportCatalog interface {
	// ReportSprints devuelve los Sprints del proyecto: el activo primero, después los finalizados por
	// número descendente. Sin Sprints devuelve una lista vacía.
	ReportSprints(ctx context.Context, projectID int64) ([]ReportSprintOption, error)
	// ReportSprint devuelve found = false si el Sprint no existe en el proyecto.
	ReportSprint(ctx context.Context, projectID, sprintID int64) (ReportSprintOption, bool, error)
}

// ReportGenerator arma los PDF; lo implementa *report.Service.
type ReportGenerator interface {
	SprintPDF(ctx context.Context, projectID, sprintID int64) ([]byte, error)
}
