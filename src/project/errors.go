package project

import "errors"

var (
	ErrEmptyName       = errors.New("project name cannot be empty")
	ErrMissingMembers  = errors.New("the member list is required")
	ErrInvalidPeriod   = errors.New("La fecha de finalización debe ser posterior a la fecha de inicio")
	ErrMemberNotFound  = errors.New("member does not exist")
	ErrDuplicateMember = errors.New("duplicate members are not allowed")
)
