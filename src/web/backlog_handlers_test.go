package web

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
)

type hu22BacklogApp struct {
	*ctxApp
	client *http.Client
}

func newHU22BacklogApp(t *testing.T) *hu22BacklogApp {
	t.Helper()

	app := newCtxApp(t, nil)
	if err := RegisterBacklog(
		app.srv,
		backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(app.db)),
		NewPostgreSQLStoryCatalog(app.db),
	); err != nil {
		t.Fatalf("register Backlog routes: %v", err)
	}
	t.Cleanup(func() {
		if _, err := app.db.Exec(`DELETE FROM user_stories WHERE project_id IN ($1, $2)`,
			app.fx.projectA, app.fx.projectB); err != nil {
			t.Errorf("delete HU-22 test stories: %v", err)
		}
	})

	return &hu22BacklogApp{ctxApp: app, client: app.client(false)}
}

func (a *hu22BacklogApp) chooseProject(t *testing.T, projectID int64) {
	t.Helper()
	resp, _ := a.selectProject(t, a.client, id64(projectID), "/backlog")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
		t.Fatalf("select project: status/location = %d/%q, want 303 /backlog",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

func (a *hu22BacklogApp) submit(t *testing.T, values url.Values) (*http.Response, string) {
	t.Helper()
	return a.post(t, a.client, "/backlog", values)
}

func hu22ValidValues(title string) url.Values {
	return url.Values{
		BacklogFieldTitle:          {title},
		BacklogFieldDescription:    {"Descripción de la historia " + title},
		BacklogFieldPriority:       {"Alta"},
		BacklogFieldStoryPoints:    {"5"},
		BacklogFieldEstimatedHours: {"8"},
		BacklogFieldCriterion:      {"La historia cumple su criterio"},
	}
}

func hu22CountStories(t *testing.T, app *hu22BacklogApp, projectID int64) int {
	t.Helper()
	var count int
	if err := app.db.QueryRow(
		`SELECT COUNT(*) FROM user_stories WHERE project_id = $1`, projectID,
	).Scan(&count); err != nil {
		t.Fatalf("count project stories: %v", err)
	}
	return count
}

func hu22AssertCreatedStory(t *testing.T, app *hu22BacklogApp, projectID int64, title string, points *int, hours *float64) {
	t.Helper()
	var (
		gotProject int64
		gotTitle   string
		gotStatus  string
		gotPoints  *int
		gotHours   *float64
	)
	err := app.db.QueryRow(`
		SELECT project_id, title, status, story_points, estimated_hours
		FROM user_stories WHERE title = $1`, title,
	).Scan(&gotProject, &gotTitle, &gotStatus, &gotPoints, &gotHours)
	if err != nil {
		t.Fatalf("read created story %q: %v", title, err)
	}
	if gotProject != projectID {
		t.Errorf("created project ID = %d, want active project %d", gotProject, projectID)
	}
	if gotTitle != title {
		t.Errorf("created title = %q, want %q", gotTitle, title)
	}
	if gotStatus != string(backlog.StatusPending) {
		t.Errorf("created status = %q, want %q", gotStatus, backlog.StatusPending)
	}
	if !equalOptionalInt(gotPoints, points) {
		t.Errorf("created Story Points = %v, want %v", gotPoints, points)
	}
	if !equalOptionalFloat(gotHours, hours) {
		t.Errorf("created estimated hours = %v, want %v", gotHours, hours)
	}
}

func TestBacklogHandler_CreateStoryRedirectsAndShowsFlashAndCreatedValues(t *testing.T) {
	app := newHU22BacklogApp(t)
	projectID := app.fx.projectB
	app.chooseProject(t, projectID)
	values := hu22ValidValues(fmt.Sprintf("HU22 success %d", app.fx.projectB))

	resp, _ := app.submit(t, values)

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
		t.Fatalf("POST status/location = %d/%q, want 303 /backlog",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	hu22AssertCreatedStory(t, app, projectID, values.Get(BacklogFieldTitle),
		backlogFormIntPointer(5), backlogFormFloatPointer(8))
	if hu22CountStories(t, app, app.fx.projectA) != 0 {
		t.Fatal("story was also created in the non-active project")
	}

	getResp, body := app.get(t, app.client, "/backlog")
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /backlog status = %d, want 200", getResp.StatusCode)
	}
	if got := strings.Count(body, MsgStoryCreated); got != 1 {
		t.Errorf("success flash occurred %d times, want exactly once: %q", got, body)
	}
	for _, want := range []string{
		values.Get(BacklogFieldTitle), "Alta", "5", "8", "Pendiente",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Backlog response does not contain %q", want)
		}
	}
}

func TestBacklogHandler_AddAndRemoveCriteriaPreservesValuesAndDoesNotSave(t *testing.T) {
	tests := []struct {
		name       string
		values     url.Values
		wantFields []string
		wantCount  int
	}{
		{
			name: "add criterion",
			values: url.Values{
				BacklogFieldTitle:     {"Título conservado"},
				BacklogFieldCriterion: {"Criterio original"},
				BacklogFieldAction:    {"agregar_criterio"},
			},
			wantFields: []string{"Título conservado", "Criterio original"},
			wantCount:  2,
		},
		{
			name: "remove selected criterion",
			values: url.Values{
				BacklogFieldTitle:     {"Otro título conservado"},
				BacklogFieldCriterion: {"Criterio uno", "Criterio dos", "Criterio tres"},
				BacklogFieldAction:    {"quitar_criterio:1"},
			},
			wantFields: []string{"Otro título conservado", "Criterio uno", "Criterio tres"},
			wantCount:  2,
		},
		{
			name: "cannot remove last criterion",
			values: url.Values{
				BacklogFieldTitle:     {"Título con criterio final"},
				BacklogFieldCriterion: {"Único criterio"},
				BacklogFieldAction:    {"quitar_criterio:0"},
			},
			wantFields: []string{"Título con criterio final", "Único criterio"},
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newHU22BacklogApp(t)
			app.chooseProject(t, app.fx.projectA)

			resp, body := app.submit(t, tt.values)

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("criteria action status = %d, want 200", resp.StatusCode)
			}
			if got := strings.Count(body, `name="criterio"`); got != tt.wantCount {
				t.Errorf("rendered criterion field count = %d, want %d", got, tt.wantCount)
			}
			for _, want := range tt.wantFields {
				if !strings.Contains(body, want) {
					t.Errorf("form did not preserve %q", want)
				}
			}
			if got := hu22CountStories(t, app, app.fx.projectA); got != 0 {
				t.Errorf("criteria action created %d stories, want 0", got)
			}
		})
	}
}

func TestBacklogHandler_EnterSubmitsSaveAction(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)
	values := hu22ValidValues("HU22 Enter saves")
	values.Del(BacklogFieldAction)

	resp, _ := app.submit(t, values)

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
		t.Fatalf("POST without action status/location = %d/%q, want 303 /backlog",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := hu22CountStories(t, app, app.fx.projectA); got != 1 {
		t.Errorf("stories created by Enter = %d, want 1", got)
	}
}

func TestBacklogHandler_IgnoresBlankAcceptanceCriteria(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)
	values := hu22ValidValues("HU22 blank criteria")
	values[BacklogFieldCriterion] = []string{"Criterio conservado", "", "   "}

	resp, _ := app.submit(t, values)

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST status = %d, want 303", resp.StatusCode)
	}
	var criteria []string
	rows, err := app.db.Query(`
		SELECT ac.content FROM acceptance_criteria ac
		JOIN user_stories us ON us.id = ac.story_id
		WHERE us.title = $1 ORDER BY ac.position`, values.Get(BacklogFieldTitle))
	if err != nil {
		t.Fatalf("query created criteria: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var criterion string
		if err := rows.Scan(&criterion); err != nil {
			t.Fatalf("scan created criterion: %v", err)
		}
		criteria = append(criteria, criterion)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate created criteria: %v", err)
	}
	if len(criteria) != 1 || criteria[0] != "Criterio conservado" {
		t.Errorf("saved criteria = %#v, want only the nonblank criterion", criteria)
	}
}

func TestBacklogHandler_MissingStoryPointsCreatesStoryWithWarning(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)
	values := hu22ValidValues("HU22 missing Story Points")
	values.Set(BacklogFieldStoryPoints, "")

	resp, _ := app.submit(t, values)

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
		t.Fatalf("POST status/location = %d/%q, want 303 /backlog",
			resp.StatusCode, resp.Header.Get("Location"))
	}
	hu22AssertCreatedStory(t, app, app.fx.projectA, values.Get(BacklogFieldTitle), nil, backlogFormFloatPointer(8))
	_, body := app.get(t, app.client, "/backlog")
	if got := strings.Count(body, MsgStoryCreatedMissingPoints); got != 1 {
		t.Errorf("missing-Story-Points warning occurred %d times, want exactly once", got)
	}
	if !strings.Contains(body, "Sin estimar") {
		t.Error("Backlog did not display Sin estimar for missing Story Points")
	}
}

func TestBacklogHandler_MissingEstimatedHoursCreatesStoryWithoutWarning(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)
	values := hu22ValidValues("HU22 missing estimated hours")
	values.Set(BacklogFieldEstimatedHours, "")

	resp, _ := app.submit(t, values)

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST status = %d, want 303", resp.StatusCode)
	}
	hu22AssertCreatedStory(t, app, app.fx.projectA, values.Get(BacklogFieldTitle), backlogFormIntPointer(5), nil)
	_, body := app.get(t, app.client, "/backlog")
	if !strings.Contains(body, MsgStoryCreated) {
		t.Errorf("success message %q not shown", MsgStoryCreated)
	}
	if strings.Contains(body, MsgStoryCreatedMissingPoints) {
		t.Error("missing-estimated-hours response incorrectly showed a missing-Story-Points warning")
	}
	if !strings.Contains(body, "Sin estimar") {
		t.Error("Backlog did not display Sin estimar for missing estimated hours")
	}
}

func TestBacklogHandler_ParsesAndDisplaysValidEstimatedHours(t *testing.T) {
	tests := []struct {
		input string
		want  float64
		text  string
	}{
		{input: "8", want: 8, text: "8"},
		{input: "1,5", want: 1.5, text: "1,5"},
		{input: "1.5", want: 1.5, text: "1,5"},
		{input: "9999,99", want: 9999.99, text: "9999,99"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			app := newHU22BacklogApp(t)
			app.chooseProject(t, app.fx.projectA)
			values := hu22ValidValues("HU22 hours " + tt.input)
			values.Set(BacklogFieldEstimatedHours, tt.input)

			resp, _ := app.submit(t, values)

			if resp.StatusCode != http.StatusSeeOther {
				t.Fatalf("POST status = %d, want 303", resp.StatusCode)
			}
			hours := tt.want
			hu22AssertCreatedStory(t, app, app.fx.projectA, values.Get(BacklogFieldTitle),
				backlogFormIntPointer(5), &hours)
			_, body := app.get(t, app.client, "/backlog")
			if !strings.Contains(body, tt.text) {
				t.Errorf("Backlog does not display estimated hours %q", tt.text)
			}
			if math.Abs(hours-tt.want) > 0.000001 {
				t.Errorf("fixture hours = %v, want exact %v", hours, tt.want)
			}
		})
	}
}

func TestBacklogHandler_GetAfterPostDoesNotDuplicateOrRepeatFlash(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)
	values := hu22ValidValues("HU22 refresh after post")

	postResp, _ := app.submit(t, values)
	if postResp.StatusCode != http.StatusSeeOther || postResp.Header.Get("Location") != "/backlog" {
		t.Fatalf("POST status/location = %d/%q, want 303 /backlog",
			postResp.StatusCode, postResp.Header.Get("Location"))
	}

	firstResp, firstBody := app.get(t, app.client, "/backlog")
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("first GET status = %d, want 200", firstResp.StatusCode)
	}
	if got := strings.Count(firstBody, MsgStoryCreated); got != 1 {
		t.Errorf("first GET success flash count = %d, want 1", got)
	}

	secondResp, secondBody := app.get(t, app.client, "/backlog")
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("second GET status = %d, want 200", secondResp.StatusCode)
	}
	if strings.Contains(secondBody, MsgStoryCreated) {
		t.Error("success flash was repeated after refresh")
	}
	if got := hu22CountStories(t, app, app.fx.projectA); got != 1 {
		t.Errorf("story count after refreshing GET = %d, want 1", got)
	}
}
