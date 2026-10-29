package test

import (
	"database/sql"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
)

// FIX-HU-21: escenario "Un proyecto recién creado admite historias".
// Necesita DATABASE_URL y las migraciones aplicadas; sin la variable se omite.

func abrirBaseFixHU21(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := backlog.OpenPostgreSQL(url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("no se pudo conectar a la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestEscenario_ProyectoRecienCreadoAdmiteHistorias(t *testing.T) {
	db := abrirBaseFixHU21(t)

	// Dado que el Scrum Master creó un proyecto desde la aplicación
	var memberID int
	if err := db.QueryRow(
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`,
		"Integrante FIX-HU-21",
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante de prueba: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM members WHERE id = $1`, memberID) })

	start := time.Now().UTC().Truncate(24 * time.Hour)
	projects := project.NewProjectService(project.NewPostgreSQLProjectStore(db))
	created, err := projects.CreateProject(project.ProjectCreationRequest{
		Name:      "Proyecto FIX-HU-21",
		Members:   []string{strconv.Itoa(memberID)},
		StartDate: start,
		EndDate:   start.AddDate(0, 0, 30),
	})
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto: %v", err)
	}
	projectID := int64(created.Project.ID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, projectID)
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, projectID)
	})

	// Cuando agrega a ese proyecto una historia con datos válidos
	stories := backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(db))
	_, err = stories.CreateStory(backlog.CreateStoryRequest{
		ProjectID:          projectID,
		Title:              "Historia de un proyecto nuevo",
		Description:        "Como Scrum Master quiero cargar historias en un proyecto recién creado",
		Priority:           backlog.PriorityHigh,
		AcceptanceCriteria: []string{"Dado un proyecto recién creado, cuando cargo una historia, entonces se registra"},
	})

	// Entonces la historia se registra en el Product Backlog en estado "Pendiente"
	if err != nil {
		t.Fatalf("el proyecto recién creado debía admitir historias: %v", err)
	}
	var status string
	if err := db.QueryRow(
		`SELECT status FROM user_stories WHERE project_id = $1`, projectID,
	).Scan(&status); err != nil {
		t.Fatalf("no se encontró la historia registrada: %v", err)
	}
	if status != string(backlog.StatusPending) {
		t.Errorf("estado esperado %q, se obtuvo %q", backlog.StatusPending, status)
	}
}
