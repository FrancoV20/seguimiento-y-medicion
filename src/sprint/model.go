package sprint

import "time"

// Project es el proyecto propietario de los Sprints y las historias del backlog.
type Project struct {
	ID               int64
	Name             string
	Status           string
	NextSprintNumber int64
}

// SprintStatus representa el estado del ciclo de vida de un Sprint.
type SprintStatus string

const (
	SprintActive   SprintStatus = "Activo"
	SprintFinished SprintStatus = "Finalizado"
)

// Sprint es una iteración del proyecto con un identificador visible generado y no editable.
type Sprint struct {
	ID        int64
	ProjectID int64
	Number    int64
	DisplayID string
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	Status    SprintStatus
	StoryIDs  []int64
}

// SprintCreationResult contiene el Sprint creado y una advertencia no bloqueante, si corresponde.
type SprintCreationResult struct {
	Sprint  Sprint
	Warning string
}

// StoryStatus representa el estado de una historia de usuario en el Product Backlog.
type StoryStatus string

const (
	Pendiente  StoryStatus = "Pendiente"
	EnSprint   StoryStatus = "En Sprint"
	InProgress StoryStatus = "En progreso"
	Done       StoryStatus = "Hecho"
)

// UserStory es una historia del backlog que puede asignarse a un único Sprint activo.
type UserStory struct {
	ID             int64
	ProjectID      int64
	Status         StoryStatus
	ActiveSprintID int64
}

// SprintCreationRequest contiene los datos del Sprint ingresados por el usuario.
// El identificador generado del Sprint no forma parte de la solicitud.
type SprintCreationRequest struct {
	ProjectID int64
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	StoryIDs  []int64
}
