package web

import (
	"context"
	"errors"
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
	return newHU22BacklogAppWithStore(t, func(app *ctxApp) backlog.BacklogStore {
		return backlog.NewPostgreSQLBacklogStore(app.db)
	})
}

func newHU22BacklogAppWithStore(t *testing.T, storeFactory func(*ctxApp) backlog.BacklogStore) *hu22BacklogApp {
	return newHU22BacklogAppWithDependencies(t, storeFactory, func(app *ctxApp) StoryCatalog {
		return NewPostgreSQLStoryCatalog(app.db)
	})
}

func newHU22BacklogAppWithDependencies(
	t *testing.T,
	storeFactory func(*ctxApp) backlog.BacklogStore,
	catalogFactory func(*ctxApp) StoryCatalog,
) *hu22BacklogApp {
	t.Helper()

	app := newCtxApp(t, nil)
	store := storeFactory(app)
	catalog := catalogFactory(app)
	if err := RegisterBacklog(
		app.srv,
		backlog.NewBacklogService(store),
		catalog,
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

type hu22FailingBacklogStore struct {
	createCalls int
}

func (s *hu22FailingBacklogStore) ProjectIsActive(int64) (bool, error) {
	return true, nil
}

func (s *hu22FailingBacklogStore) CreateStory(_ int64, _ backlog.UserStory) (backlog.UserStory, error) {
	s.createCalls++
	return backlog.UserStory{}, errors.New("simulated database failure")
}

// UpdateStory completa el puerto ampliado por HU-13 (T027).
func (s *hu22FailingBacklogStore) UpdateStory(context.Context, int64, backlog.UserStory) error {
	return errors.New("simulated database failure")
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

func hu22AssertNoInternalDetails(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{
		"simulated database failure",
		"SELECT ",
		"INSERT INTO ",
		"UPDATE ",
		"DELETE FROM ",
		"backlog_handlers.go",
		"src\\web\\",
		"/src/web/",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("response leaked internal detail %q", forbidden)
		}
	}
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

func TestBacklogHandler_ValidationErrorsReturn422AndPreserveSubmittedValues(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)

	tests := []struct {
		name         string
		field        string
		value        string
		message      string
		mutate       func(url.Values)
		wantRetained string
	}{
		{
			name:    "no acceptance criteria",
			field:   BacklogFieldCriterion,
			message: backlog.ErrAcceptanceCriteriaRequired.Error(),
			mutate: func(values url.Values) {
				values[BacklogFieldCriterion] = []string{"", "   "}
			},
			wantRetained: `value="   "`,
		},
		{
			name:    "empty title",
			field:   BacklogFieldTitle,
			value:   "   ",
			message: backlog.ErrTitleRequired.Error(),
			mutate: func(values url.Values) {
				values.Set(BacklogFieldTitle, "   ")
			},
			wantRetained: `value="   "`,
		},
		{
			name:    "empty description",
			field:   BacklogFieldDescription,
			value:   "   ",
			message: backlog.ErrDescriptionRequired.Error(),
			mutate: func(values url.Values) {
				values.Set(BacklogFieldDescription, "   ")
			},
			wantRetained: ">   </textarea>",
		},
		{
			name:    "invalid priority",
			field:   BacklogFieldPriority,
			value:   "Urgente",
			message: backlog.ErrInvalidPriority.Error(),
			mutate: func(values url.Values) {
				values.Set(BacklogFieldPriority, "Urgente")
			},
			wantRetained: "Urgente",
		},
		{
			name:    "invalid Story Points",
			field:   BacklogFieldStoryPoints,
			value:   "4",
			message: MsgInvalidStoryPoints,
			mutate: func(values url.Values) {
				values.Set(BacklogFieldStoryPoints, "4")
			},
			wantRetained: "value=\"4\"",
		},
		{
			name:    "non numeric Story Points",
			field:   BacklogFieldStoryPoints,
			value:   "abc",
			message: MsgInvalidStoryPoints,
			mutate: func(values url.Values) {
				values.Set(BacklogFieldStoryPoints, "abc")
			},
			wantRetained: "abc",
		},
	}

	for _, invalid := range []string{"4", "0", "-1", "abc"} {
		tests = append(tests, struct {
			name         string
			field        string
			value        string
			message      string
			mutate       func(url.Values)
			wantRetained string
		}{
			name:         "Story Points " + invalid,
			field:        BacklogFieldStoryPoints,
			value:        invalid,
			message:      MsgInvalidStoryPoints,
			wantRetained: invalid,
			mutate: func(values url.Values) {
				values.Set(BacklogFieldStoryPoints, invalid)
			},
		})
	}
	for _, invalid := range []string{"0", "-1", "abc", "1,555"} {
		tests = append(tests, struct {
			name         string
			field        string
			value        string
			message      string
			mutate       func(url.Values)
			wantRetained string
		}{
			name:         "estimated hours " + invalid,
			field:        BacklogFieldEstimatedHours,
			value:        invalid,
			message:      MsgInvalidEstimatedHours,
			wantRetained: `value="` + invalid + `"`,
			mutate: func(values url.Values) {
				values.Set(BacklogFieldEstimatedHours, invalid)
			},
		})
	}
	for _, invalid := range []string{"10000", "9999,995"} {
		tests = append(tests, struct {
			name         string
			field        string
			value        string
			message      string
			mutate       func(url.Values)
			wantRetained string
		}{
			name:         "estimated hours above maximum " + invalid,
			field:        BacklogFieldEstimatedHours,
			value:        invalid,
			message:      MsgHoursTooLarge,
			wantRetained: `value="` + invalid + `"`,
			mutate: func(values url.Values) {
				values.Set(BacklogFieldEstimatedHours, invalid)
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := hu22ValidValues("HU22 invalid " + tt.name)
			tt.mutate(values)

			resp, body := app.submit(t, values)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST status = %d, want 422; body=%s", resp.StatusCode, body)
			}
			if !strings.Contains(body, tt.message) {
				t.Errorf("response does not contain HU-02 error %q", tt.message)
			}
			if !strings.Contains(body, tt.wantRetained) {
				t.Errorf("response did not retain submitted value %q", tt.wantRetained)
			}
			hu22AssertNoInternalDetails(t, body)
			if got := hu22CountStories(t, app, app.fx.projectA); got != 0 {
				t.Errorf("invalid submission created %d stories, want 0", got)
			}
		})
	}
}

func TestBacklogHandler_StoreFailureReturns500AndKeepsFormValues(t *testing.T) {
	store := &hu22FailingBacklogStore{}
	app := newHU22BacklogAppWithStore(t, func(*ctxApp) backlog.BacklogStore {
		return store
	})
	app.chooseProject(t, app.fx.projectA)
	values := hu22ValidValues("HU22 persistence failure")
	values.Set(BacklogFieldDescription, "Descripción que debe conservarse")
	values[BacklogFieldCriterion] = []string{"Criterio que debe conservarse"}

	resp, body := app.submit(t, values)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("POST status = %d, want 500; body=%s", resp.StatusCode, body)
	}
	for _, want := range []string{
		MsgOperationFailed,
		`value="HU22 persistence failure"`,
		">Descripción que debe conservarse</textarea>",
		`value="Criterio que debe conservarse"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("failure response does not contain retained value/message %q", want)
		}
	}
	if strings.Contains(body, MsgStoryCreated) || strings.Contains(body, MsgStoryCreatedMissingPoints) {
		t.Error("persistence failure response contains a success message")
	}
	hu22AssertNoInternalDetails(t, body)
	if store.createCalls != 1 {
		t.Errorf("store CreateStory calls = %d, want 1", store.createCalls)
	}
	if got := hu22CountStories(t, app, app.fx.projectA); got != 0 {
		t.Errorf("failed persistence created %d stories, want 0", got)
	}
}

func TestBacklogHandler_PostWithoutValidActiveProjectRedirectsAndDoesNotSave(t *testing.T) {
	app := newHU22BacklogApp(t)
	tests := []struct {
		name    string
		cookies []*http.Cookie
	}{
		{name: "no project selected"},
		{name: "malformed project cookie", cookies: []*http.Cookie{{Name: CookieProject, Value: "invalid", Path: "/"}}},
		{name: "deleted project", cookies: []*http.Cookie{idCookie(CookieProject, math.MaxInt64)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := app.ctxApp.client(false)
			if len(tt.cookies) > 0 {
				app.setCookies(t, client, tt.cookies...)
			}
			values := hu22ValidValues("HU22 unauthorized " + tt.name)
			values.Set("proyecto_id", id64(app.fx.projectA))

			resp, _ := app.post(t, client, "/backlog", values)
			defer resp.Body.Close()

			wantLocation := "/contexto/proyecto?volver_a=%2Fbacklog"
			if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != wantLocation {
				t.Fatalf("POST status/location = %d/%q, want 303 %q",
					resp.StatusCode, resp.Header.Get("Location"), wantLocation)
			}
			selectionResp, selectionBody := app.get(t, client, "/contexto/proyecto")
			defer selectionResp.Body.Close()
			if selectionResp.StatusCode != http.StatusOK {
				t.Fatalf("project selection status = %d, want 200", selectionResp.StatusCode)
			}
			if !strings.Contains(selectionBody, MsgProjectRequired) {
				t.Errorf("project selection page does not show %q", MsgProjectRequired)
			}
			for _, projectID := range []int64{app.fx.projectA, app.fx.projectB} {
				if got := hu22CountStories(t, app, projectID); got != 0 {
					t.Errorf("unauthorized POST created %d stories in project %d", got, projectID)
				}
			}
		})
	}
}

// HU-13 (T037): antes "NewestFirst"; con FR-001, a igual prioridad la más antigua va primero.
func TestBacklogHandler_GetListsActiveProjectStoriesInPriorityOrder(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)

	insertStory := func(projectID int64, title string, points *int, hours *float64) int64 {
		t.Helper()
		var id int64
		err := app.db.QueryRow(`
						INSERT INTO user_stories (
							project_id, title, description, priority, story_points, estimated_hours, status
						)
						VALUES ($1, $2, 'HU-22 Backlog GET fixture', 'Media', $3, $4, 'Pendiente')
						RETURNING id`, projectID, title, points, hours).Scan(&id)
		if err != nil {
			t.Fatalf("create GET fixture story: %v", err)
		}
		return id
	}

	points := 3
	partialHours := 1.5
	oldestTitle := fmt.Sprintf("HU22 GET oldest %d", app.fx.projectA)
	newestTitle := fmt.Sprintf("HU22 GET newest %d", app.fx.projectA)
	otherTitle := fmt.Sprintf("HU22 GET other project %d", app.fx.projectB)
	oldestID := insertStory(app.fx.projectA, oldestTitle, &points, nil)
	newestID := insertStory(app.fx.projectA, newestTitle, nil, &partialHours)
	insertStory(app.fx.projectB, otherTitle, &points, &partialHours)

	resp, body := app.get(t, app.client, "/backlog")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /backlog status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	for _, column := range []string{
		"Identificador",
		"Título",
		"Prioridad",
		"Story Points",
		"Horas estimadas",
		"Estado",
	} {
		if !strings.Contains(body, column) {
			t.Errorf("Backlog table is missing column %q", column)
		}
	}
	for _, want := range []string{
		id64(newestID),
		newestTitle,
		"Media",
		"Sin estimar",
		"1,5",
		"Pendiente",
		id64(oldestID),
		oldestTitle,
		"3",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Backlog table does not contain %q", want)
		}
	}
	tableRows := extractHU22TableRows(t, body)
	if strings.Count(tableRows, "Sin estimar") != 2 {
		t.Errorf("Backlog should show 'Sin estimar' for both missing estimates; table has %d occurrences",
			strings.Count(tableRows, "Sin estimar"))
	}
	if strings.Contains(body, otherTitle) {
		t.Errorf("Backlog leaked a story from the inactive project: %q", otherTitle)
	}
	if strings.Index(body, oldestTitle) >= strings.Index(body, newestTitle) {
		t.Errorf("Backlog stories are not ordered oldest first within a priority: %q should precede %q", oldestTitle, newestTitle)
	}
}

func extractHU22TableRows(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "<tbody>")
	end := strings.Index(body, "</tbody>")
	if start < 0 || end < start {
		t.Fatal("Backlog response does not contain a complete table body")
	}
	return body[start:end]
}

func TestBacklogHandler_GetShowsEmptyStateWhenProjectHasNoStories(t *testing.T) {
	app := newHU22BacklogApp(t)
	app.chooseProject(t, app.fx.projectA)

	resp, body := app.get(t, app.client, "/backlog")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /backlog status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := strings.Count(body, "Aún no hay historias"); got != 1 {
		t.Errorf("empty-state text occurred %d times, want exactly once", got)
	}
	if strings.Contains(body, "<table") {
		t.Error("empty Backlog rendered a stories table")
	}
	if !strings.Contains(body, `action="/backlog"`) || !strings.Contains(body, `name="titulo"`) {
		t.Error("empty Backlog did not render the story creation form")
	}
}

type hu22FailingStoryCatalog struct {
	err error
}

func (c hu22FailingStoryCatalog) ListStories(context.Context, int64) ([]StoryRow, error) {
	return nil, c.err
}

// StoryForEdit completa el catálogo ampliado por HU-13 (T056).
func (c hu22FailingStoryCatalog) StoryForEdit(context.Context, int64, int64) (StoryEdit, error) {
	return StoryEdit{}, c.err
}

func TestBacklogHandler_GetCatalogFailureReturns500WithoutPartialBacklog(t *testing.T) {
	const internalFailure = "synthetic catalog query failure"
	app := newHU22BacklogAppWithDependencies(
		t,
		func(ctx *ctxApp) backlog.BacklogStore {
			return backlog.NewPostgreSQLBacklogStore(ctx.db)
		},
		func(*ctxApp) StoryCatalog {
			return hu22FailingStoryCatalog{err: errors.New(internalFailure)}
		},
	)
	app.chooseProject(t, app.fx.projectA)

	resp, body := app.get(t, app.client, "/backlog")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("GET /backlog status = %d, want 500; body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, TitleInternal) || !strings.Contains(body, MsgInternal) {
		t.Errorf("catalog failure did not render the generic internal error page: %s", body)
	}
	if strings.Contains(body, "<table") || strings.Contains(body, "Crear historia") {
		t.Error("catalog failure rendered a partial Backlog page")
	}
	if strings.Contains(body, internalFailure) {
		t.Error("catalog failure leaked internal error detail to the response")
	}
}
