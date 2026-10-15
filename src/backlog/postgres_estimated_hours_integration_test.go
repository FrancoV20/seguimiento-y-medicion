package backlog

import (
	"database/sql"
	"testing"
)

// Integración de horas estimadas (FR-014) contra PostgreSQL.
// Usa openTestDB y crearProyectoActivo de postgres_backlog_store_integration_test.go.

func leerHorasEstimadas(t *testing.T, db *sql.DB, storyID int64) sql.NullFloat64 {
	t.Helper()
	var h sql.NullFloat64
	if err := db.QueryRow(`SELECT estimated_hours::float8 FROM user_stories WHERE id = $1`, storyID).Scan(&h); err != nil {
		t.Fatalf("no se pudieron leer las horas estimadas: %v", err)
	}
	return h
}

func TestPostgreSQLBacklogStore_PersisteHorasEstimadas(t *testing.T) {
	db := openTestDB(t)
	projectID := crearProyectoActivo(t, db)
	req := validRequest()
	req.ProjectID = projectID
	req.EstimatedHours = floatPtr(12.5)

	result, err := NewBacklogService(NewPostgreSQLBacklogStore(db)).CreateStory(req)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}

	if h := leerHorasEstimadas(t, db, result.Story.ID); !h.Valid || h.Float64 != 12.5 {
		t.Errorf("se esperaban 12.5 horas estimadas, se obtuvo %+v", h)
	}
}

func TestPostgreSQLBacklogStore_SinHorasEstimadasGuardaNull(t *testing.T) {
	db := openTestDB(t)
	projectID := crearProyectoActivo(t, db)
	req := validRequest()
	req.ProjectID = projectID
	req.EstimatedHours = nil

	result, err := NewBacklogService(NewPostgreSQLBacklogStore(db)).CreateStory(req)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}

	if h := leerHorasEstimadas(t, db, result.Story.ID); h.Valid {
		t.Errorf("sin horas estimadas debe guardarse NULL, se obtuvo %v", h.Float64)
	}
}
