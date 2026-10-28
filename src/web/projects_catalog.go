package web

import (
	"context"
	"time"
)

// ProjectRow es una fila de la lista de proyectos.
type ProjectRow struct {
	ID          int64
	Name        string
	StartDate   time.Time
	EndDate     time.Time
	MemberCount int64
}

// MemberOption es un integrante disponible para el formulario.
type MemberOption struct {
	ID          int64
	DisplayName string
}

// ProjectCatalog proporciona los datos de lectura de la pantalla de proyectos.
type ProjectCatalog interface {
	ListProjects(ctx context.Context) ([]ProjectRow, error)
	ListMembers(ctx context.Context) ([]MemberOption, error)
	// ProjectForEdit lee los datos actuales de un proyecto para modificarlo (HU-11). Devuelve
	// NotFound si no existe y Unavailable si falla la base.
	ProjectForEdit(ctx context.Context, id int64) (ProjectEdit, error)
}

// ProjectEdit son los datos actuales de un proyecto para el formulario de modificación (HU-11).
type ProjectEdit struct {
	ID        int64
	Name      string
	StartDate time.Time
	EndDate   time.Time
	MemberIDs []int64 // ordenados
}
