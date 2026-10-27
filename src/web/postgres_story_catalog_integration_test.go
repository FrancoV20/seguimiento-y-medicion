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

func TestPostgreSQLStoryCatalogListsOnlyProjectStoriesInDescendingOrder(t *testing.T) {
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
	if got[0].ID != nullValuesID || got[1].ID != firstID {
		t.Errorf("story IDs = [%d, %d], want descending IDs [%d, %d]",
			got[0].ID, got[1].ID, nullValuesID, firstID)
	}
	if got[0].Title != "HU22 Catalog missing estimates" {
		t.Errorf("newest story title = %q, want %q", got[0].Title, "HU22 Catalog missing estimates")
	}
	if got[0].Priority != "Media" || got[0].Status != "Pendiente" {
		t.Errorf("newest story priority/status = %q/%q, want Media/Pendiente", got[0].Priority, got[0].Status)
	}
	if got[0].StoryPoints != nil {
		t.Errorf("newest story StoryPoints = %v, want nil", got[0].StoryPoints)
	}
	if got[0].EstimatedHours != nil {
		t.Errorf("newest story EstimatedHours = %v, want nil", got[0].EstimatedHours)
	}
	if got[1].StoryPoints == nil || *got[1].StoryPoints != pointValue {
		t.Errorf("older story StoryPoints = %v, want %d", got[1].StoryPoints, pointValue)
	}
	if got[1].EstimatedHours == nil || *got[1].EstimatedHours != hourValue {
		t.Errorf("older story EstimatedHours = %v, want %v", got[1].EstimatedHours, hourValue)
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
