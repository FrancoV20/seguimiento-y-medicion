package web

import (
	"context"
	"strings"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// EffortStoryRow es una historia del Sprint activo con sus horas reales acumuladas (HU-24, T005).
type EffortStoryRow struct {
	ID          int64
	Title       string
	Status      string
	ActualHours effort.Hours
}

// HoursText son las horas acumuladas con coma decimal ("2,5").
func (r EffortStoryRow) HoursText() string { return FormatEffortHours(r.ActualHours) }

// EffortSprint es el Sprint activo del proyecto con sus historias, ordenadas por id.
type EffortSprint struct {
	Number  int
	Stories []EffortStoryRow
}

// Story devuelve la historia con ese id si pertenece al Sprint, o nil.
func (s *EffortSprint) Story(id int64) *EffortStoryRow {
	if s == nil {
		return nil
	}
	for i := range s.Stories {
		if s.Stories[i].ID == id {
			return &s.Stories[i]
		}
	}
	return nil
}

// EffortEntryRow es un registro de esfuerzo del historial de una historia.
type EffortEntryRow struct {
	MemberName string
	Date       time.Time
	Activity   string
	Hours      effort.Hours
}

// HoursText son las horas del registro con coma decimal.
func (e EffortEntryRow) HoursText() string { return FormatEffortHours(e.Hours) }

// EffortHistory es el historial de una historia del Sprint activo y su total.
type EffortHistory struct {
	StoryID int64
	Title   string
	Entries []EffortEntryRow
	Total   effort.Hours
}

// TotalText es el total con coma decimal.
func (h EffortHistory) TotalText() string { return FormatEffortHours(h.Total) }

// EffortCatalog es la lectura de pantalla de HU-24 (solo lectura). Todas las consultas se
// filtran por el proyecto del contexto del servidor (FR-016).
type EffortCatalog interface {
	// ActiveSprintStories devuelve el Sprint activo del proyecto con sus historias, o nil (sin
	// error) si el proyecto no tiene Sprint activo.
	ActiveSprintStories(ctx context.Context, projectID int64) (*EffortSprint, error)

	// StoryHistory devuelve el historial de la historia si pertenece al Sprint activo del
	// proyecto, o nil (sin error) si no la encuentra.
	StoryHistory(ctx context.Context, projectID, storyID int64) (*EffortHistory, error)
}

// FormatEffortHours muestra horas con coma decimal y sin ceros sobrantes: 250 -> "2,5".
func FormatEffortHours(h effort.Hours) string {
	return strings.Replace(h.String(), ".", ",", 1)
}
