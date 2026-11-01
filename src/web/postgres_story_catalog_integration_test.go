package web

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func openStoryCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()

	return openContextTestDB(t)
}

// HU-13 (T030): antes "InDescendingOrder"; con FR-001, a igual prioridad la más antigua va primero.
func TestPostgreSQLStoryCatalogListsOnlyProjectStoriesInPriorityOrder(t *testing.T) {
	db := openStoryCatalogTestDB(t)
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	projectNames := []string{
		fmt.Sprintf("HU22 Catalog A %d", stamp),
		fmt.Sprintf("HU22 Catalog B %d", stamp),
		fmt.Sprintf("HU22 Catalog Empty %d", stamp),
	}
	projectIDs := make([]int64, 0, len(projectNames))

	t.Cleanup(func() {
		if len(projectIDs) == 0 {
			return
		}
		placeholders := "$1"
		for i := 2; i <= len(projectIDs); i++ {
			placeholders += fmt.Sprintf(", $%d", i)
		}
		args := make([]any, len(projectIDs))
		for i, id := range projectIDs {
			args[i] = id
		}
		if _, err := db.ExecContext(ctx,
			`DELETE FROM user_stories WHERE project_id IN (`+placeholders+`)`, args...); err != nil {
			t.Errorf("delete catalog test stories: %v", err)
		}
		if _, err := db.ExecContext(ctx,
			`DELETE FROM projects WHERE id IN (`+placeholders+`)`, args...); err != nil {
			t.Errorf("delete catalog test projects: %v", err)
		}
	})

	for _, name := range projectNames {
		var id int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO projects (name, start_date, end_date)
			VALUES ($1, CURRENT_DATE, CURRENT_DATE + 1)
			RETURNING id`, name).Scan(&id)
		if err != nil {
			t.Fatalf("create catalog test project: %v", err)
		}
		projectIDs = append(projectIDs, id)
	}

	insertStory := func(projectID int64, title string, points *int, hours *float64) int64 {
		t.Helper()
		var id int64
		err := db.QueryRowContext(ctx, `
			INSERT INTO user_stories (
				project_id, title, description, priority, story_points, estimated_hours
			)
			VALUES ($1, $2, 'Catalog integration fixture', 'Media', $3, $4)
			RETURNING id`, projectID, title, points, hours).Scan(&id)
		if err != nil {
			t.Fatalf("create catalog test story: %v", err)
		}
		return id
	}

	pointValue := 5
	hourValue := 1.5
	firstID := insertStory(projectIDs[0], "HU22 Catalog first", &pointValue, &hourValue)
	nullValuesID := insertStory(projectIDs[0], "HU22 Catalog missing estimates", nil, nil)
	otherProjectID := insertStory(projectIDs[1], "HU22 Catalog other project", &pointValue, &hourValue)

	catalog := NewPostgreSQLStoryCatalog(db)
	got, err := catalog.ListStories(ctx, projectIDs[0])
	if err != nil {
		t.Fatalf("ListStories(project A) error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListStories(project A) returned %d rows, want 2: %+v", len(got), got)
	}
	if got[0].ID != firstID || got[1].ID != nullValuesID {
		t.Errorf("story IDs = [%d, %d], want oldest first [%d, %d]",
			got[0].ID, got[1].ID, firstID, nullValuesID)
	}
	if got[1].Title != "HU22 Catalog missing estimates" {
		t.Errorf("newest story title = %q, want %q", got[1].Title, "HU22 Catalog missing estimates")
	}
	if got[1].Priority != "Media" || got[1].Status != "Pendiente" {
		t.Errorf("newest story priority/status = %q/%q, want Media/Pendiente", got[1].Priority, got[1].Status)
	}
	if got[1].StoryPoints != nil {
		t.Errorf("newest story StoryPoints = %v, want nil", got[1].StoryPoints)
	}
	if got[1].EstimatedHours != nil {
		t.Errorf("newest story EstimatedHours = %v, want nil", got[1].EstimatedHours)
	}
	if got[0].StoryPoints == nil || *got[0].StoryPoints != pointValue {
		t.Errorf("older story StoryPoints = %v, want %d", got[0].StoryPoints, pointValue)
	}
	if got[0].EstimatedHours == nil || *got[0].EstimatedHours != hourValue {
		t.Errorf("older story EstimatedHours = %v, want %v", got[0].EstimatedHours, hourValue)
	}
	for _, row := range got {
		if row.ID == otherProjectID {
			t.Errorf("ListStories(project A) included story %d from project B", row.ID)
		}
	}

	empty, err := catalog.ListStories(ctx, projectIDs[2])
	if err != nil {
		t.Fatalf("ListStories(empty project) error = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("ListStories(empty project) returned %d rows, want 0: %+v", len(empty), empty)
	}

}

// hu13CatalogProject crea un proyecto para las pruebas del catálogo de HU-13 y lo borra al final,
// junto con sus historias y Sprints.
func hu13CatalogProject(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, CURRENT_DATE, CURRENT_DATE + 30) RETURNING id`,
		fmt.Sprintf("%s %d", name, time.Now().UnixNano())).Scan(&id); err != nil {
		t.Fatalf("create HU-13 catalog project: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM sprints WHERE project_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM projects WHERE id = $1`, id)
	})
	return id
}

// hu13CatalogStory crea una historia con la prioridad y la fecha de creación indicadas.
func hu13CatalogStory(t *testing.T, db *sql.DB, projectID int64, title, priority, createdAt string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`INSERT INTO user_stories (project_id, title, description, priority, created_at)
		VALUES ($1, $2, 'HU-13 catalog fixture', $3, $4::timestamptz) RETURNING id`,
		projectID, title, priority, createdAt).Scan(&id); err != nil {
		t.Fatalf("create HU-13 catalog story: %v", err)
	}
	return id
}

func TestPostgreSQLStoryCatalog_OrdenPorPrioridadYAntiguedad(t *testing.T) {
	db := openStoryCatalogTestDB(t)
	project := hu13CatalogProject(t, db, "HU13 Catalog orden")
	// Creadas en un orden distinto del esperado; «Perfil» y «Login» empatan en created_at.
	reportes := hu13CatalogStory(t, db, project, "Reportes", "Baja", "2026-09-01 10:00:00+00")
	login := hu13CatalogStory(t, db, project, "Login", "Media", "2026-09-02 10:00:00+00")
	perfil := hu13CatalogStory(t, db, project, "Perfil", "Media", "2026-09-02 10:00:00+00")
	pagos := hu13CatalogStory(t, db, project, "Pagos", "Alta", "2026-09-03 10:00:00+00")
	antigua := hu13CatalogStory(t, db, project, "Antigua", "Media", "2026-08-01 10:00:00+00")

	got, err := NewPostgreSQLStoryCatalog(db).ListStories(context.Background(), project)
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	want := []int64{pagos, antigua, login, perfil, reportes}
	if len(got) != len(want) {
		t.Fatalf("ListStories devolvió %d historias; se esperaban %d", len(got), len(want))
	}
	for i, row := range got {
		if row.ID != want[i] {
			t.Errorf("posición %d: historia %d (%s); se esperaba %d", i, row.ID, row.Title, want[i])
		}
	}
}

func TestPostgreSQLStoryCatalog_MarcaSprintFinalizado(t *testing.T) {
	db := openStoryCatalogTestDB(t)
	project := hu13CatalogProject(t, db, "HU13 Catalog Sprint")
	finished := hu13CatalogStory(t, db, project, "Pagos", "Alta", "2026-09-01 10:00:00+00")
	active := hu13CatalogStory(t, db, project, "Login", "Alta", "2026-09-02 10:00:00+00")
	backlogOnly := hu13CatalogStory(t, db, project, "Reportes", "Alta", "2026-09-03 10:00:00+00")
	for _, s := range []struct {
		number int
		status string
		story  int64
	}{{1, "Finalizado", finished}, {2, "Activo", active}} {
		var sprintID int64
		if err := db.QueryRow(`INSERT INTO sprints (project_id, number, goal, start_date, end_date, status)
			VALUES ($1, $2, 'Objetivo', CURRENT_DATE, CURRENT_DATE + 14, $3) RETURNING id`,
			project, s.number, s.status).Scan(&sprintID); err != nil {
			t.Fatalf("create HU-13 catalog sprint: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO sprint_story_assignments (sprint_id, story_id) VALUES ($1, $2)`, sprintID, s.story); err != nil {
			t.Fatalf("assign HU-13 catalog story: %v", err)
		}
	}

	got, err := NewPostgreSQLStoryCatalog(db).ListStories(context.Background(), project)
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	marks := map[int64]bool{}
	for _, row := range got {
		marks[row.ID] = row.InFinishedSprint
	}
	if len(got) != 3 || !marks[finished] || marks[active] || marks[backlogOnly] {
		t.Errorf("InFinishedSprint = %v; solo «Pagos» (%d), del Sprint finalizado, debía estar marcada", marks, finished)
	}
}
