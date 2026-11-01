package web

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
)

// HU-25: datos de prueba de "Métricas del Sprint". Cada prueba crea sus propios proyectos,
// historias, Sprints, horas y cierres (con el servicio real de HU-15) y los borra al terminar.

// metricsFixture es un proyecto de prueba con un integrante para cargar horas.
type metricsFixture struct {
	t         *testing.T
	db        *sql.DB
	ProjectID int64
	memberID  int64
}

// newMetricsProject crea un proyecto y, al terminar, borra sus horas, historias y el proyecto,
// en ese orden: user_stories.project_id y effort_logs.story_id no tienen borrado en cascada.
// Los Sprints, sus asignaciones, sus resultados y sus métricas caen en cascada con el proyecto.
func newMetricsProject(t *testing.T, db *sql.DB, label string) *metricsFixture {
	t.Helper()
	f := &metricsFixture{t: t, db: db}
	err := db.QueryRow(`
		INSERT INTO projects (name, start_date, end_date, status)
		VALUES ($1, DATE '2026-09-01', DATE '2026-12-31', 'Activo')
		RETURNING id`,
		fmt.Sprintf("HU25 métricas %s %d", label, time.Now().UnixNano()),
	).Scan(&f.ProjectID)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba %q: %v", label, err)
	}
	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM effort_logs WHERE story_id IN (SELECT id FROM user_stories WHERE project_id = $1)`,
			`DELETE FROM user_stories WHERE project_id = $1`,
			`DELETE FROM projects WHERE id = $1`,
		} {
			if _, err := db.Exec(query, f.ProjectID); err != nil {
				t.Errorf("no se pudo limpiar el proyecto de prueba %d: %v", f.ProjectID, err)
			}
		}
	})
	return f
}

// activeSprint crea un Sprint activo con número, objetivo y fechas (aaaa-mm-dd) dados.
func (f *metricsFixture) activeSprint(number int, goal, start, end string) int64 {
	f.t.Helper()
	var id int64
	err := f.db.QueryRow(`
		INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
		VALUES ($1, $2, $3, $4::date, $5::date, 'Activo')
		RETURNING id`,
		f.ProjectID, number, goal, start, end,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("no se pudo crear el Sprint %d de prueba: %v", number, err)
	}
	return id
}

// story crea una historia asignada al Sprint con Story Points, estado y horas estimadas
// (estimated vacío = sin horas estimadas).
func (f *metricsFixture) story(sprintID int64, title string, points int, status sprint.StoryStatus, estimated string) int64 {
	f.t.Helper()
	var estimatedHours sql.NullString
	if estimated != "" {
		estimatedHours = sql.NullString{String: estimated, Valid: true}
	}
	var id int64
	err := f.db.QueryRow(`
		INSERT INTO user_stories (project_id, title, description, priority, story_points, status,
		                          estimated_hours, active_sprint_id)
		VALUES ($1, $2, 'Historia para probar las métricas', 'Media', $3, $4, $5::numeric, $6)
		RETURNING id`,
		f.ProjectID, title, points, string(status), estimatedHours, sprintID,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("no se pudo crear la historia de prueba %q: %v", title, err)
	}
	if _, err := f.db.Exec(
		`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, id,
	); err != nil {
		f.t.Fatalf("no se pudo asignar la historia %q al Sprint: %v", title, err)
	}
	return id
}

// effort registra horas reales en la historia (HU-03), con un integrante propio del fixture.
func (f *metricsFixture) effort(storyID int64, hours string) {
	f.t.Helper()
	if f.memberID == 0 {
		if err := f.db.QueryRow(
			`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
			fmt.Sprintf("Integrante HU25 %d", time.Now().UnixNano()),
		).Scan(&f.memberID); err != nil {
			f.t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
		}
		memberID, db, t := f.memberID, f.db, f.t
		// Se registra después del proyecto, así que corre antes: borra sus horas y el integrante.
		t.Cleanup(func() {
			if _, err := db.Exec(`DELETE FROM effort_logs WHERE member_id = $1`, memberID); err != nil {
				t.Errorf("no se pudieron limpiar las horas de prueba: %v", err)
			}
			if _, err := db.Exec(`DELETE FROM members WHERE id = $1`, memberID); err != nil {
				t.Errorf("no se pudo limpiar el integrante de prueba: %v", err)
			}
		})
	}
	if _, err := f.db.Exec(`
		INSERT INTO effort_logs (story_id, member_id, work_date, activity, hours)
		VALUES ($1, $2, DATE '2026-10-05', 'Trabajo de HU-25', $3::numeric)`,
		storyID, f.memberID, hours,
	); err != nil {
		f.t.Fatalf("no se pudieron registrar %s horas en la historia %d: %v", hours, storyID, err)
	}
}

// close cierra el Sprint con el servicio real de HU-15 y el validador de HU-07 de main.
func (f *metricsFixture) close(sprintID int64) {
	f.t.Helper()
	service := sprint.NewCloseSprintService(sprint.NewPostgreSQLSprintCloser(f.db), closeHandlerValidator{})
	if _, err := service.CloseSprint(context.Background(), f.ProjectID, sprintID); err != nil {
		f.t.Fatalf("HU-15 no pudo cerrar el Sprint %d de prueba: %v", sprintID, err)
	}
}

// finishedSprint crea un Sprint activo, opcionalmente con historias, y lo cierra con HU-15.
func (f *metricsFixture) finishedSprint(number int, goal, start, end string) int64 {
	f.t.Helper()
	id := f.activeSprint(number, goal, start, end)
	f.close(id)
	return id
}

// hu07NoData es el texto de HU-07 para una desviación sin horas estimadas ("No disponible").
func hu07NoData(t *testing.T) string {
	t.Helper()
	text, err := metrics.CalculateDeviation(nil, nil)
	if err != nil {
		t.Fatalf("HU-07 CalculateDeviation(nil, nil) devolvió error: %v", err)
	}
	return text
}

// hu07PartialWarning es el aviso que arma HU-07 para un cálculo con used de total historias.
// Se obtiene del servicio real de HU-07 con su store de prueba en memoria, sin reescribir el texto.
func hu07PartialWarning(t *testing.T, used, total int) string {
	t.Helper()
	points := 3
	stories := make([]metrics.CompletedStory, 0, total)
	for i := 0; i < total; i++ {
		story := metrics.CompletedStory{ID: int64(i + 1), Status: "Hecho", StoryPoints: &points}
		if i < used {
			story.EstimatedHours = big.NewRat(1, 1)
			story.ActualHours = big.NewRat(1, 1)
		}
		stories = append(stories, story)
	}
	store := &metrics.TestMetricsStore{SprintData: metrics.SprintData{ID: 1, Status: "Finalizado", Stories: stories}}
	got, err := metrics.NewService(store).CalculateMetrics(1)
	if err != nil {
		t.Fatalf("HU-07 no pudo armar el aviso de cálculo parcial: %v", err)
	}
	if got.Warning == "" {
		t.Fatalf("HU-07 no devolvió aviso para %d de %d historias", used, total)
	}
	return got.Warning
}
