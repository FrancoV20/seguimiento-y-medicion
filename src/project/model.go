package project

import "time"

type Project struct {
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
