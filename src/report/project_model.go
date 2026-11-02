package report

import (
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

// Modelo del reporte del proyecto (HU-09, T043, research.md §18).

// StoryCounts son las historias del proyecto por su estado actual.
type StoryCounts struct {
	Pending, InSprint, InProgress, Done int // "Pendiente", "En Sprint", "En progreso", "Hecho"
}

// Total es la cantidad de historias del proyecto.
func (c StoryCounts) Total() int { return c.Pending + c.InSprint + c.InProgress + c.Done }

// ProjectSprint es un Sprint del proyecto.
type ProjectSprint struct {
	ID     int64
	Number int
	Status string // "Activo" o "Finalizado"
}

// OpenDefect es un defecto "Abierto" del proyecto.
type OpenDefect struct {
	Description     string
	Severity        string
	StoryTitle      string
	DetectionSprint int // número
}

// ProjectData son los datos propios del reporte del proyecto que lee el puerto.
type ProjectData struct {
	ID                 int64
	Name               string
	StartDate, EndDate time.Time
	Members            []string        // display_name de project_members, por nombre
	Stories            StoryCounts     // estado actual de todas las historias del proyecto
	Sprints            []ProjectSprint // por número ascendente
	EstimatedTotal     effort.Hours    // suma de las horas estimadas informadas
	WithoutEstimate    int             // historias sin horas estimadas
	RegisteredTotal    effort.Hours    // suma de effort_logs de las historias del proyecto
	MemberHours        []MemberHours   // por nombre, después por id
	DefectsOpen        int
	DefectsResolved    int
	OpenDefects        []OpenDefect // los "Abierto", por id
}

// ProjectSprintRow es una fila de la tabla de Sprints.
type ProjectSprintRow struct {
	Number  int
	Active  bool                   // Sprint en curso: sin Quality ni Metrics
	Quality *metrics.SprintQuality // HU-10
	Metrics *metrics.SprintMetrics // HU-07
}

// ProjectReport es el reporte del proyecto armado; RenderProject lo convierte en PDF.
type ProjectReport struct {
	Data        ProjectData
	Status      string // HU-12: "No iniciado", "En curso" o "Finalizado"
	Progress    string // HU-12: porcentaje de avance con punto, "57.14" o "0.00"
	Partial     bool   // hay un Sprint en curso
	Sprints     []ProjectSprintRow
	Velocity    metrics.VelocitySummary // HU-08
	GeneratedAt time.Time
}
