package defect

// Severity es el nivel de severidad de un defecto.
type Severity string

const (
	SeverityCritical Severity = "Crítica"
	SeverityHigh     Severity = "Alta"
	SeverityMedium   Severity = "Media"
	SeverityLow      Severity = "Baja"
)

// Status es el estado de un defecto.
type Status string

const (
	StatusOpen     Status = "Abierto"
	StatusResolved Status = "Resuelto"
)

// RegisterInput contiene los datos necesarios para registrar un defecto.
type RegisterInput struct {
	ProjectID   int64
	StoryID     int64
	Description string
	Severity    Severity
}

// Registered contiene los datos del defecto que se acaba de registrar.
type Registered struct {
	ID           int64
	SprintNumber int64
}
