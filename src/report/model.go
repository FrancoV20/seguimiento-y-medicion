// Package report arma los reportes en PDF de un Sprint y de un proyecto (HU-09): lee sus datos
// propios con el puerto Store, toma las métricas de HU-07, HU-08, HU-10 y HU-12 sin recalcularlas y
// compone el documento con src/pdf.
package report

import (
	"errors"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
)

var (
	ErrSprintNotFound     = errors.New("report: sprint not found")
	ErrProjectNotFound    = errors.New("report: project not found")
	ErrInconsistentReport = errors.New("report: sprint data changed while reading")
)

// Estados de un Sprint (HU-04).
const (
	sprintActive   = "Activo"
	sprintFinished = "Finalizado"
	storyDone      = "Hecho"
)

// SprintInfo es el encabezado del Sprint.
type SprintInfo struct {
	ProjectName string
	Number      int
	Goal        string
	StartDate   time.Time
	EndDate     time.Time
	Status      string // "Activo" o "Finalizado"
}

// Story es una historia del Sprint.
type Story struct {
	ID             int64
	Title          string
	Status         string        // final (registro de cierre) o actual (en curso)
	StoryPoints    *int          // nil = sin estimar
	EstimatedHours *effort.Hours // nil = sin estimar
	ActualHours    effort.Hours  // suma de todos sus registros de esfuerzo
}

// MemberHours son las horas registradas por un integrante.
type MemberHours struct {
	Name  string // members.display_name
	Hours effort.Hours
}

// Defect es un defecto detectado en el Sprint.
type Defect struct {
	Description      string
	Severity         string
	Status           string // "Abierto" o "Resuelto"
	StoryTitle       string
	ResolutionSprint int // número; 0 = sin Sprint de resolución
}

// Data son los datos propios que lee el puerto.
type Data struct {
	Sprint  SprintInfo
	Stories []Story       // por ID ascendente
	Members []MemberHours // por nombre, después por id
	Defects []Defect      // detectados en el Sprint, por id ascendente
}

// SprintReport es el reporte armado; Render lo convierte en PDF.
type SprintReport struct {
	Data        Data
	Partial     bool                   // Sprint en curso
	Metrics     *metrics.SprintMetrics // HU-07; nil si Partial
	Quality     *metrics.SprintQuality // HU-10; nil si Partial
	GeneratedAt time.Time
}
