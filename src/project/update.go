package project

import "time"

// ProjectUpdateRequest son los datos de la modificación tal como llegan (HU-11).
type ProjectUpdateRequest struct {
	ID        int
	Name      string   // la web lo recorta y el servicio lo vuelve a recortar
	Members   []string // identificadores de integrantes
	StartDate time.Time
	EndDate   time.Time
}

// ProjectUpdate es la modificación ya validada que recibe el almacenamiento.
type ProjectUpdate struct {
	ID        int
	Name      string
	MemberIDs []int // sin repetidos
	StartDate time.Time
	EndDate   time.Time
}

// SprintPeriod son las fechas de un Sprint del proyecto.
type SprintPeriod struct {
	Number    int
	StartDate time.Time
	EndDate   time.Time
}

// DateField indica qué fecha del proyecto deja fuera a un Sprint.
type DateField string

const (
	DateFieldStart DateField = "inicio"
	DateFieldEnd   DateField = "fin"
)
