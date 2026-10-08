package project

import (
	"errors"
	"time"
)

type Project struct {
	Name      string
	Members   []string
	StartDate time.Time
	EndDate   time.Time
}

func NewProject(name string, members []string, startDate, endDate time.Time) (Project, error) {
	if name == "" {
		return Project{}, errors.New("project name cannot be empty")
	}
	if len(members) == 0 {
		return Project{}, errors.New("the member list is required")
	}
	if endDate.Before(startDate) {
		return Project{}, errors.New("end date cannot be before start date")
	}

	return Project{
		Name:      name,
		Members:   members,
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}
