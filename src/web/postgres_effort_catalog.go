package web

import (
	"context"
	"database/sql"
	"errors"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// PostgreSQLEffortCatalog implementa EffortCatalog sobre PostgreSQL (HU-24, T007). Solo lee:
// el registro de horas lo hace el servicio de HU-03.
//
// Una historia pertenece al Sprint activo cuando user_stories.active_sprint_id apunta a un
// Sprint 'Activo' del proyecto: es la misma regla con la que HU-03 acepta cargas.
type PostgreSQLEffortCatalog struct {
	db *sql.DB
}

var _ EffortCatalog = (*PostgreSQLEffortCatalog)(nil)

// NewPostgreSQLEffortCatalog crea el catálogo a partir de una conexión ya abierta.
func NewPostgreSQLEffortCatalog(db *sql.DB) *PostgreSQLEffortCatalog {
	return &PostgreSQLEffortCatalog{db: db}
}

// Sprint activo del proyecto; con más de uno (no debería pasar por FIX-HU-04) el de mayor número.
const effortActiveSprintQuery = `
SELECT id, number FROM sprints
WHERE project_id = $1 AND status = 'Activo'
ORDER BY number DESC
LIMIT 1`

// Horas acumuladas en centésimas, sin pasar por punto flotante (igual que HU-03).
const effortSprintStoriesQuery = `
SELECT us.id, us.title, us.status,
       COALESCE((SELECT SUM(el.hours) * 100 FROM effort_logs el WHERE el.story_id = us.id), 0)::bigint
FROM user_stories us
WHERE us.active_sprint_id = $1 AND us.project_id = $2
ORDER BY us.id`

// La historia solo se devuelve si está en el Sprint activo de mayor número del proyecto.
const effortStoryInActiveSprintQuery = `
SELECT us.title
FROM user_stories us
JOIN sprints s ON s.id = us.active_sprint_id
WHERE us.id = $1 AND us.project_id = $2 AND s.project_id = $2 AND s.status = 'Activo'
  AND s.number = (SELECT MAX(number) FROM sprints WHERE project_id = $2 AND status = 'Activo')`

const effortHistoryQuery = `
SELECT m.display_name, el.work_date, el.activity, (el.hours * 100)::bigint
FROM effort_logs el
JOIN members m ON m.id = el.member_id
WHERE el.story_id = $1
ORDER BY el.work_date DESC, el.id DESC`

// ActiveSprintStories devuelve el Sprint activo con sus historias, o nil si no hay.
func (c *PostgreSQLEffortCatalog) ActiveSprintStories(ctx context.Context, projectID int64) (*EffortSprint, error) {
	var sprintID int64
	sprint := &EffortSprint{Stories: []EffortStoryRow{}}
	err := c.db.QueryRowContext(ctx, effortActiveSprintQuery, projectID).Scan(&sprintID, &sprint.Number)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, effortSprintStoriesQuery, sprintID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row EffortStoryRow
		var hours int64
		if err := rows.Scan(&row.ID, &row.Title, &row.Status, &hours); err != nil {
			return nil, err
		}
		row.ActualHours = effort.Hours(hours)
		sprint.Stories = append(sprint.Stories, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sprint, nil
}

// StoryHistory devuelve el historial de la historia, o nil si no es del Sprint activo del proyecto.
func (c *PostgreSQLEffortCatalog) StoryHistory(ctx context.Context, projectID, storyID int64) (*EffortHistory, error) {
	h := &EffortHistory{StoryID: storyID, Entries: []EffortEntryRow{}}
	err := c.db.QueryRowContext(ctx, effortStoryInActiveSprintQuery, storyID, projectID).Scan(&h.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := c.db.QueryContext(ctx, effortHistoryQuery, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e EffortEntryRow
		var hours int64
		if err := rows.Scan(&e.MemberName, &e.Date, &e.Activity, &hours); err != nil {
			return nil, err
		}
		e.Hours = effort.Hours(hours)
		h.Total = h.Total.Add(e.Hours)
		h.Entries = append(h.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return h, nil
}
