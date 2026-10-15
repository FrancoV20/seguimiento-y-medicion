package project

import "errors"

var (
	ErrEmptyName       = errors.New("project name cannot be empty")
	ErrMissingMembers  = errors.New("the member list is required")
	ErrInvalidPeriod   = errors.New("end date cannot be before start date")
	ErrMemberNotFound  = errors.New("member does not exist")
	ErrDuplicateMember = errors.New("duplicate members are not allowed")
)
