package web

import "context"

// DashboardStoryCounts son las historias del proyecto por estado (HU-08).
type DashboardStoryCounts struct {
	Pending    int // "Pendiente"
	InSprint   int // "En Sprint"
	InProgress int // "En progreso"
	Done       int // "Hecho"
}

// DashboardSprint es un Sprint finalizado del proyecto.
type DashboardSprint struct {
	ID     int64
	Number int
}

// DashboardSummary son los datos propios del Dashboard, leídos de una sola vez.
type DashboardSummary struct {
	Stories     DashboardStoryCounts
	OpenDefects int
	Finished    []DashboardSprint // todos los Sprints finalizados, por número ascendente
}

// DashboardCatalog lee los datos propios del Dashboard (solo lectura). El Sprint en curso, la
// velocidad y la desviación salen de BoardCatalog (HU-14), QualityReader (HU-10) y MetricsSource
// (HU-25), que no se redefinen.
type DashboardCatalog interface {
	DashboardSummary(ctx context.Context, projectID int64) (DashboardSummary, error)
}
