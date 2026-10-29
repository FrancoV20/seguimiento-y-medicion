package project

import "time"

// StatusActive es el estado con el que nace todo proyecto (FIX-HU-21).
// Debe coincidir con el que usan la base, el Product Backlog y los defectos.
const StatusActive = "Activo"

type Project struct {
	ID        int
	Name      string
	Members   []string
	StartDate time.Time
	EndDate   time.Time
	Status    string
}

type Member struct {
	ID   string
	Name string
}

type ProjectCreationRequest struct {
	Name      string
	Members   []string
	StartDate time.Time
	EndDate   time.Time
}

type ProjectCreationResult struct {
	Project *Project
	Message string
}
