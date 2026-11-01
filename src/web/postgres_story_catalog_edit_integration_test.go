package web

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// HU-13, T045: lectura de una historia para su formulario de edición.

func storyForEditFixture(t *testing.T) (*PostgreSQLStoryCatalog, int64, map[string]int64) {
	t.Helper()
	db := openStoryCatalogTestDB(t)
	project := hu13CatalogProject(t, db, "HU13 StoryForEdit")
	stories := map[string]int64{
		"Login":  hu13CatalogStory(t, db, project, "Login", "Media", "2026-09-01 10:00:00+00"),
		"Vacía":  hu13CatalogStory(t, db, project, "Vacía", "Baja", "2026-09-02 10:00:00+00"),
		"Pagos":  hu13CatalogStory(t, db, project, "Pagos", "Alta", "2026-09-03 10:00:00+00"),
		"Activa": hu13CatalogStory(t, db, project, "Activa", "Alta", "2026-09-04 10:00:00+00"),
	}
	if _, err := db.Exec(`UPDATE user_stories SET story_points = 5 WHERE id = $1`, stories["Login"]); err != nil {
		t.Fatal(err)
	}
	// Criterios cargados al revés de su posición, para comprobar el orden.
	for _, c := range []struct {
		position int
		content  string
	}{{2, "Rechaza clave vacía"}, {1, "Acepta email"}} {
		if _, err := db.Exec(`INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, $2, $3)`,
			stories["Login"], c.position, c.content); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []struct {
		number int
		status string
		story  int64
	}{{1, "Finalizado", stories["Pagos"]}, {2, "Activo", stories["Activa"]}} {
		var sprintID int64
		if err := db.QueryRow(`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, 'Objetivo', CURRENT_DATE, CURRENT_DATE + 14, $3) RETURNING id`,
			project, s.number, s.status).Scan(&sprintID); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, s.story); err != nil {
			t.Fatal(err)
		}
	}
	return NewPostgreSQLStoryCatalog(db), project, stories
}

func TestPostgresStoryForEdit_ConCriteriosEnOrden(t *testing.T) {
	catalog, project, stories := storyForEditFixture(t)
	got, err := catalog.StoryForEdit(context.Background(), project, stories["Login"])
	if err != nil {
		t.Fatalf("StoryForEdit: %v", err)
	}
	if got.ID != stories["Login"] || got.Title != "Login" || got.Description != "HU-13 catalog fixture" ||
		got.Priority != "Media" || got.StoryPoints == nil || *got.StoryPoints != 5 || got.InFinishedSprint {
		t.Errorf("StoryForEdit = %+v", got)
	}
	if want := []string{"Acepta email", "Rechaza clave vacía"}; !reflect.DeepEqual(got.Criteria, want) {
		t.Errorf("criterios = %v; se esperaba %v", got.Criteria, want)
	}
}

func TestPostgresStoryForEdit_SinCriterios(t *testing.T) {
	catalog, project, stories := storyForEditFixture(t)
	got, err := catalog.StoryForEdit(context.Background(), project, stories["Vacía"])
	if err != nil {
		t.Fatalf("StoryForEdit: %v", err)
	}
	if got.Title != "Vacía" || len(got.Criteria) != 0 || got.StoryPoints != nil {
		t.Errorf("StoryForEdit = %+v; se esperaba sin criterios ni Story Points", got)
	}
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != KindNotFound {
		t.Errorf("error = %v; se esperaba KindNotFound", err)
	}
}

func TestPostgresStoryForEdit_Inexistente(t *testing.T) {
	catalog, project, _ := storyForEditFixture(t)
	_, err := catalog.StoryForEdit(context.Background(), project, 2147483647)
	assertNotFound(t, err)
}

func TestPostgresStoryForEdit_OtroProyecto(t *testing.T) {
	catalog, _, stories := storyForEditFixture(t)
	other := hu13CatalogProject(t, catalog.db, "HU13 StoryForEdit otro")
	_, err := catalog.StoryForEdit(context.Background(), other, stories["Login"])
	assertNotFound(t, err)
}

func TestPostgresStoryForEdit_SprintFinalizado(t *testing.T) {
	catalog, project, stories := storyForEditFixture(t)
	for title, want := range map[string]bool{"Pagos": true, "Activa": false, "Login": false} {
		got, err := catalog.StoryForEdit(context.Background(), project, stories[title])
		if err != nil {
			t.Fatalf("StoryForEdit(%s): %v", title, err)
		}
		if got.InFinishedSprint != want {
			t.Errorf("%s: InFinishedSprint = %v; se esperaba %v", title, got.InFinishedSprint, want)
		}
	}
}
