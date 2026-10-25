package web

import (
	"context"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-14 (T008): datos de lectura del Tablero y su puerto de acceso.

// actionLabels son los textos de los botones; la regla de qué acción corresponde
// a cada estado vive en sprint.NextAction.
var actionLabels = map[sprint.Action]string{
	sprint.ActionStart:    "Comenzar",
	sprint.ActionComplete: "Completar",
	sprint.ActionReopen:   "Reabrir",
}

// BoardStory es una historia del Sprint activo tal como se muestra en el Tablero.
type BoardStory struct {
	ID    int64
	Title string
	// StoryPoints es nil cuando la historia no está estimada ("Sin estimar").
	StoryPoints *int
	Status      sprint.StoryStatus
	HasEffort   bool
	// Action es la única acción válida para el estado de la historia (sprint.NextAction).
	Action      sprint.Action
	ActionLabel string
}

// BoardGroup reúne las historias de un mismo estado, ordenadas por id.
type BoardGroup struct {
	Status  sprint.StoryStatus
	Stories []BoardStory
}

// BoardView es el Sprint activo de un proyecto con sus historias agrupadas por estado.
type BoardView struct {
	ID        int64
	Number    int64
	Label     string // "Sprint N"
	Goal      string
	StartDate time.Time
	EndDate   time.Time
	// Groups tiene tres grupos, en este orden: En Sprint, En progreso y Hecho.
	Groups          []BoardGroup
	PlannedPoints   int
	CompletedPoints int
}

// BoardCatalog lee el Tablero (solo lectura).
type BoardCatalog interface {
	// ActiveBoard devuelve el Sprint activo del proyecto con sus historias. found es falso
	// cuando el proyecto no tiene Sprint activo, lo que no es un error; solo lo es una
	// falla de lectura.
	ActiveBoard(ctx context.Context, projectID int64) (view BoardView, found bool, err error)
}

// BoardPoints calcula los Story Points planificados (todas las historias) y completados
// (solo las Hecho). Una historia sin Story Points cuenta 0. Nunca se guardan (FR-017).
func BoardPoints(stories []BoardStory) (planned, completed int) {
	// RED (T008): esqueleto, se implementa en T013.
	return 0, 0
}
