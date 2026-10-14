package backlog

import (
	"database/sql"
	"os"
	"testing"
)

// Pruebas de integración de PostgreSQLBacklogStore (T015, T021).
// Requieren DATABASE_URL y las migraciones aplicadas; sin la variable se omiten.

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite la prueba de integración")
	}
	db, err := OpenPostgreSQL(url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("no se pudo conectar a la base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// crearProyectoActivo inserta un proyecto de prueba y lo borra al terminar el test.
// Ajustar las columnas cuando HU-01 defina la tabla projects definitiva.
func crearProyectoActivo(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(
		`INSERT INTO projects (name, start_date, end_date, status)
		 VALUES ($1, CURRENT_DATE, CURRENT_DATE + 30, 'Activo') RETURNING id`,
		"Proyecto de prueba HU-02",
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo crear el proyecto de prueba: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, id)
	})
	return id
}

func contarHistorias(t *testing.T, db *sql.DB, projectID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM user_stories WHERE project_id = $1`, projectID).Scan(&n); err != nil {
		t.Fatalf("no se pudieron contar las historias: %v", err)
	}
	return n
}

func TestPostgreSQLBacklogStore_PersisteHistoriaCriteriosYAsociacion(t *testing.T) {
	db := openTestDB(t)
	projectID := crearProyectoActivo(t, db)
	service := NewBacklogService(NewPostgreSQLBacklogStore(db))
	req := validRequest()
	req.ProjectID = projectID
	req.AcceptanceCriteria = []string{"Primer criterio", "Segundo criterio"}

	result, err := service.CreateStory(req)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}

	var gotProject int64
	var title, priority, status string
	var sp sql.NullInt64
	err = db.QueryRow(
		`SELECT project_id, title, priority, status, story_points FROM user_stories WHERE id = $1`,
		result.Story.ID,
	).Scan(&gotProject, &title, &priority, &status, &sp)
	if err != nil {
		t.Fatalf("la historia no quedó persistida: %v", err)
	}
	if gotProject != projectID || title != "Alta de proyecto" || priority != "Alta" || status != "Pendiente" {
		t.Errorf("datos persistidos incorrectos: proyecto=%d título=%q prioridad=%q estado=%q",
			gotProject, title, priority, status)
	}
	if !sp.Valid || sp.Int64 != 5 {
		t.Errorf("se esperaban 5 Story Points, se obtuvo %+v", sp)
	}

	var criterios int
	if err := db.QueryRow(`SELECT count(*) FROM acceptance_criteria WHERE story_id = $1`, result.Story.ID).Scan(&criterios); err != nil {
		t.Fatalf("no se pudieron contar los criterios: %v", err)
	}
	if criterios != 2 {
		t.Errorf("se esperaban 2 criterios persistidos, hay %d", criterios)
	}
}

func TestPostgreSQLBacklogStore_SinStoryPointsGuardaNull(t *testing.T) {
	db := openTestDB(t)
	projectID := crearProyectoActivo(t, db)
	req := validRequest()
	req.ProjectID = projectID
	req.StoryPoints = nil

	result, err := NewBacklogService(NewPostgreSQLBacklogStore(db)).CreateStory(req)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}

	var sp sql.NullInt64
	if err := db.QueryRow(`SELECT story_points FROM user_stories WHERE id = $1`, result.Story.ID).Scan(&sp); err != nil {
		t.Fatalf("no se pudo leer la historia: %v", err)
	}
	if sp.Valid {
		t.Errorf("la ausencia de estimación debe guardarse como NULL, se obtuvo %d", sp.Int64)
	}
}

func TestPostgreSQLBacklogStore_ErrorEnCriterioNoDejaPersistenciaParcial(t *testing.T) {
	db := openTestDB(t)
	projectID := crearProyectoActivo(t, db)
	criterios := []AcceptanceCriterion{
		{Position: 1, Content: "Criterio válido"},
		{Position: 2, Content: "   "}, // viola el CHECK de la base
	}
	story := UserStory{
		Title:              "Historia",
		Description:        "Descripción",
		Priority:           PriorityLow,
		Status:             StatusPending,
		AcceptanceCriteria: criterios,
	}

	if _, err := NewPostgreSQLBacklogStore(db).CreateStory(projectID, story); err == nil {
		t.Fatal("se esperaba un error por criterio vacío")
	}
	if n := contarHistorias(t, db, projectID); n != 0 {
		t.Errorf("la transacción debe revertirse por completo, quedaron %d historias", n)
	}
}

func TestPostgreSQLBacklogStore_ProyectoInexistenteNoEstaActivo(t *testing.T) {
	db := openTestDB(t)

	active, err := NewPostgreSQLBacklogStore(db).ProjectIsActive(-1)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}
	if active {
		t.Error("un proyecto inexistente no debe considerarse activo")
	}
}
