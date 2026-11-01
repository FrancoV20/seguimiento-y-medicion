package report

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
)

// PostgreSQLStore implementa Store contra PostgreSQL (HU-09, research.md §7). Solo lee, en una
// transacción de solo lectura con REPEATABLE READ para que todas las partes salgan del mismo momento.
type PostgreSQLStore struct {
	db *sql.DB
}

var _ Store = (*PostgreSQLStore)(nil)

// NewPostgreSQLStore crea el almacén sobre una conexión abierta.
func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

var readOnly = &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead}

const sprintHeaderQuery = `
	SELECT p.name, s.number, s.goal, s.start_date, s.end_date, s.status
	FROM sprints s
	JOIN projects p ON p.id = s.project_id
	WHERE s.id = $1 AND s.project_id = $2`

// Fuente de las historias: el registro de cierre de HU-15 si el Sprint está 'Finalizado' o las
// asignaciones de HU-04 si está 'Activo'.
const (
	finishedStoryIDs = `SELECT story_id FROM sprint_story_results WHERE sprint_id = $1`
	activeStoryIDs   = `SELECT story_id FROM sprint_story_assignments WHERE sprint_id = $1`
)

const actualHoursColumn = `COALESCE((SELECT ROUND(SUM(el.hours) * 100)::bigint FROM effort_logs el WHERE el.story_id = us.id), 0)`

const finishedStoriesQuery = `
	SELECT us.id, us.title, r.final_status, r.story_points,
	       ROUND(us.estimated_hours * 100)::bigint, ` + actualHoursColumn + `
	FROM sprint_story_results r
	JOIN user_stories us ON us.id = r.story_id
	WHERE r.sprint_id = $1
	ORDER BY us.id`

const activeStoriesQuery = `
	SELECT us.id, us.title, us.status, us.story_points,
	       ROUND(us.estimated_hours * 100)::bigint, ` + actualHoursColumn + `
	FROM sprint_story_assignments a
	JOIN user_stories us ON us.id = a.story_id
	WHERE a.sprint_id = $1
	ORDER BY us.id`

const memberHoursQuery = `
	SELECT m.display_name, ROUND(SUM(el.hours) * 100)::bigint
	FROM effort_logs el
	JOIN members m ON m.id = el.member_id
	WHERE el.story_id IN (%s)
	GROUP BY m.id, m.display_name
	ORDER BY m.display_name, m.id`

const sprintDefectsQuery = `
	SELECT d.description, d.severity, d.status, us.title, COALESCE(rs.number, 0)
	FROM defects d
	JOIN user_stories us ON us.id = d.story_id
	LEFT JOIN sprints rs ON rs.id = d.resolution_sprint_id
	WHERE d.project_id = $2 AND d.detection_sprint_id = $1
	ORDER BY d.id`

// LoadSprintReport lee el encabezado, las historias, el esfuerzo por integrante y los defectos del
// Sprint del proyecto. Devuelve ErrSprintNotFound si el Sprint no existe en el proyecto; cualquier
// otra falla, envuelta.
func (s *PostgreSQLStore) LoadSprintReport(ctx context.Context, projectID, sprintID int64) (Data, error) {
	tx, err := s.db.BeginTx(ctx, readOnly)
	if err != nil {
		return Data{}, fmt.Errorf("sprint report %d: %w", sprintID, err)
	}
	defer tx.Rollback()

	data := Data{Stories: []Story{}, Members: []MemberHours{}, Defects: []Defect{}}
	h := &data.Sprint
	err = tx.QueryRowContext(ctx, sprintHeaderQuery, sprintID, projectID).
		Scan(&h.ProjectName, &h.Number, &h.Goal, &h.StartDate, &h.EndDate, &h.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return Data{}, ErrSprintNotFound
	}
	if err != nil {
		return Data{}, fmt.Errorf("sprint report %d header: %w", sprintID, err)
	}

	storiesQuery, storyIDs := activeStoriesQuery, activeStoryIDs
	if h.Status == sprintFinished {
		storiesQuery, storyIDs = finishedStoriesQuery, finishedStoryIDs
	}
	if data.Stories, err = readStories(ctx, tx, storiesQuery, sprintID); err != nil {
		return Data{}, fmt.Errorf("sprint report %d stories: %w", sprintID, err)
	}
	if data.Members, err = readMemberHours(ctx, tx, fmt.Sprintf(memberHoursQuery, storyIDs), sprintID); err != nil {
		return Data{}, fmt.Errorf("sprint report %d effort: %w", sprintID, err)
	}
	if data.Defects, err = readSprintDefects(ctx, tx, sprintID, projectID); err != nil {
		return Data{}, fmt.Errorf("sprint report %d defects: %w", sprintID, err)
	}
	if err := tx.Commit(); err != nil {
		return Data{}, fmt.Errorf("sprint report %d: %w", sprintID, err)
	}
	return data, nil
}

func readStories(ctx context.Context, tx *sql.Tx, query string, sprintID int64) ([]Story, error) {
	rows, err := tx.QueryContext(ctx, query, sprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stories := []Story{}
	for rows.Next() {
		var s Story
		var sp, estimated sql.NullInt64
		var actual int64
		if err := rows.Scan(&s.ID, &s.Title, &s.Status, &sp, &estimated, &actual); err != nil {
			return nil, err
		}
		if sp.Valid {
			p := int(sp.Int64)
			s.StoryPoints = &p
		}
		if estimated.Valid {
			e := effort.Hours(estimated.Int64)
			s.EstimatedHours = &e
		}
		s.ActualHours = effort.Hours(actual)
		stories = append(stories, s)
	}
	return stories, rows.Err()
}

func readMemberHours(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]MemberHours, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []MemberHours{}
	for rows.Next() {
		var m MemberHours
		var h int64
		if err := rows.Scan(&m.Name, &h); err != nil {
			return nil, err
		}
		m.Hours = effort.Hours(h)
		members = append(members, m)
	}
	return members, rows.Err()
}

func readSprintDefects(ctx context.Context, tx *sql.Tx, sprintID, projectID int64) ([]Defect, error) {
	rows, err := tx.QueryContext(ctx, sprintDefectsQuery, sprintID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defects := []Defect{}
	for rows.Next() {
		var d Defect
		if err := rows.Scan(&d.Description, &d.Severity, &d.Status, &d.StoryTitle, &d.ResolutionSprint); err != nil {
			return nil, err
		}
		defects = append(defects, d)
	}
	return defects, rows.Err()
}

// Consultas del reporte del proyecto (HU-09, T044).
const (
	projectHeaderQuery = `SELECT id, name, start_date, end_date FROM projects WHERE id = $1`

	projectMembersQuery = `
	SELECT m.display_name
	FROM project_members pm JOIN members m ON m.id = pm.member_id
	WHERE pm.project_id = $1
	ORDER BY m.display_name, m.id`

	projectStoriesQuery = `
	SELECT COUNT(*) FILTER (WHERE status = 'Pendiente'),
	       COUNT(*) FILTER (WHERE status = 'En Sprint'),
	       COUNT(*) FILTER (WHERE status = 'En progreso'),
	       COUNT(*) FILTER (WHERE status = 'Hecho'),
	       COALESCE(ROUND(SUM(estimated_hours) * 100)::bigint, 0),
	       COUNT(*) FILTER (WHERE estimated_hours IS NULL)
	FROM user_stories
	WHERE project_id = $1`

	projectSprintsQuery = `SELECT id, number, status FROM sprints WHERE project_id = $1 ORDER BY number`

	projectMemberHoursQuery = `
	SELECT m.display_name, ROUND(SUM(el.hours) * 100)::bigint
	FROM effort_logs el
	JOIN user_stories us ON us.id = el.story_id
	JOIN members m ON m.id = el.member_id
	WHERE us.project_id = $1
	GROUP BY m.id, m.display_name
	ORDER BY m.display_name, m.id`

	projectDefectCountsQuery = `
	SELECT COUNT(*) FILTER (WHERE status = 'Abierto'), COUNT(*) FILTER (WHERE status = 'Resuelto')
	FROM defects WHERE project_id = $1`

	projectOpenDefectsQuery = `
	SELECT d.description, d.severity, us.title, ds.number
	FROM defects d
	JOIN user_stories us ON us.id = d.story_id
	JOIN sprints ds ON ds.id = d.detection_sprint_id
	WHERE d.project_id = $1 AND d.status = 'Abierto'
	ORDER BY d.id`
)

// LoadProjectReport lee los datos propios del reporte del proyecto en una sola lectura consistente.
// Devuelve ErrProjectNotFound si el proyecto no existe; cualquier otra falla, envuelta.
func (s *PostgreSQLStore) LoadProjectReport(ctx context.Context, projectID int64) (ProjectData, error) {
	tx, err := s.db.BeginTx(ctx, readOnly)
	if err != nil {
		return ProjectData{}, fmt.Errorf("project report %d: %w", projectID, err)
	}
	defer tx.Rollback()

	data := ProjectData{Members: []string{}, Sprints: []ProjectSprint{}, MemberHours: []MemberHours{}, OpenDefects: []OpenDefect{}}
	err = tx.QueryRowContext(ctx, projectHeaderQuery, projectID).Scan(&data.ID, &data.Name, &data.StartDate, &data.EndDate)
	if errors.Is(err, sql.ErrNoRows) {
		return ProjectData{}, ErrProjectNotFound
	}
	if err != nil {
		return ProjectData{}, fmt.Errorf("project report %d header: %w", projectID, err)
	}

	if err := readProjectMembers(ctx, tx, projectID, &data); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d members: %w", projectID, err)
	}
	var estimated int64
	c := &data.Stories
	if err := tx.QueryRowContext(ctx, projectStoriesQuery, projectID).
		Scan(&c.Pending, &c.InSprint, &c.InProgress, &c.Done, &estimated, &data.WithoutEstimate); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d stories: %w", projectID, err)
	}
	data.EstimatedTotal = effort.Hours(estimated)
	if err := readProjectSprints(ctx, tx, projectID, &data); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d sprints: %w", projectID, err)
	}
	if data.MemberHours, err = readMemberHours(ctx, tx, projectMemberHoursQuery, projectID); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d effort: %w", projectID, err)
	}
	for _, m := range data.MemberHours {
		data.RegisteredTotal = data.RegisteredTotal.Add(m.Hours)
	}
	if err := tx.QueryRowContext(ctx, projectDefectCountsQuery, projectID).Scan(&data.DefectsOpen, &data.DefectsResolved); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d defects: %w", projectID, err)
	}
	if err := readOpenDefects(ctx, tx, projectID, &data); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d open defects: %w", projectID, err)
	}
	if err := tx.Commit(); err != nil {
		return ProjectData{}, fmt.Errorf("project report %d: %w", projectID, err)
	}
	return data, nil
}

func readProjectMembers(ctx context.Context, tx *sql.Tx, projectID int64, data *ProjectData) error {
	rows, err := tx.QueryContext(ctx, projectMembersQuery, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		data.Members = append(data.Members, name)
	}
	return rows.Err()
}

func readProjectSprints(ctx context.Context, tx *sql.Tx, projectID int64, data *ProjectData) error {
	rows, err := tx.QueryContext(ctx, projectSprintsQuery, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var sp ProjectSprint
		if err := rows.Scan(&sp.ID, &sp.Number, &sp.Status); err != nil {
			return err
		}
		data.Sprints = append(data.Sprints, sp)
	}
	return rows.Err()
}

func readOpenDefects(ctx context.Context, tx *sql.Tx, projectID int64, data *ProjectData) error {
	rows, err := tx.QueryContext(ctx, projectOpenDefectsQuery, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var d OpenDefect
		if err := rows.Scan(&d.Description, &d.Severity, &d.StoryTitle, &d.DetectionSprint); err != nil {
			return err
		}
		data.OpenDefects = append(data.OpenDefects, d)
	}
	return rows.Err()
}
