package project

import (
	"errors"
	"fmt"
)

// ErrProjectNotFound indica que el proyecto a modificar no existe (HU-11).
var ErrProjectNotFound = errors.New("project does not exist")

// MemberHasEffortError indica que se intentó quitar a un integrante con horas registradas en
// historias del proyecto (decisión del equipo: se bloquea).
type MemberHasEffortError struct {
	MemberName string
}

func (e *MemberHasEffortError) Error() string {
	return fmt.Sprintf("member %s has effort logged in this project", e.MemberName)
}

// SprintOutOfRangeError indica que el cambio de fechas deja fuera a un Sprint que estaba dentro.
type SprintOutOfRangeError struct {
	Number int
	Field  DateField
}

func (e *SprintOutOfRangeError) Error() string {
	return fmt.Sprintf("sprint %d would fall outside the project dates (%s)", e.Number, e.Field)
}
