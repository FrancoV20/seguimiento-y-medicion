package project

import (
	"time"
)

func NewProject(name string, members []string, startDate, endDate time.Time) (Project, error) {
	if name == "" {
		return Project{}, ErrEmptyName
	}
	if len(members) == 0 {
		return Project{}, ErrMissingMembers
	}
	if startDate.IsZero() || endDate.IsZero() || !endDate.After(startDate) {
		return Project{}, ErrInvalidPeriod
	}

	return Project{
		Name:      name,
		Members:   members,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    StatusActive,
	}, nil
}
