package test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

type hu22BDDApp struct {
	db        *sql.DB
	server    *httptest.Server
	client    *http.Client
	projectID int64
}

func newHU22BDDApp(t *testing.T) *hu22BDDApp {
	return newHU22BDDAppWithStore(t, nil)
}

func newHU22BDDAppWithStore(t *testing.T, store backlog.BacklogStore) *hu22BDDApp {
	t.Helper()

	databaseURL := os.Getenv(web.EnvDatabaseURL)
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite la integración BDD de HU-22")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open HU-22 BDD database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("HU-22 BDD database unavailable: %v", err)
	}

	var projectID int64
	projectName := fmt.Sprintf("HU22 BDD project %d", time.Now().UnixNano())
	if err := db.QueryRowContext(context.Background(), `
		INSERT INTO projects (name, start_date, end_date)
		VALUES ($1, CURRENT_DATE, CURRENT_DATE + 1)
		RETURNING id`, projectName).Scan(&projectID); err != nil {
		t.Fatalf("create HU-22 BDD project: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, projectID); err != nil {
			t.Errorf("delete HU-22 BDD stories: %v", err)
		}
		if _, err := db.Exec(`DELETE FROM projects WHERE id = $1`, projectID); err != nil {
			t.Errorf("delete HU-22 BDD project: %v", err)
		}
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := web.NewServer(logger)
	server.Use(web.CrossOriginGuard(server.Renderer()))
	web.RegisterContext(server, web.NewPostgreSQLContextStore(db))
	if store == nil {
		store = backlog.NewPostgreSQLBacklogStore(db)
	}
	if err := web.RegisterBacklog(
		server,
		backlog.NewBacklogService(store),
		web.NewPostgreSQLStoryCatalog(db),
	); err != nil {
		t.Fatalf("register HU-22 BDD Backlog routes: %v", err)
	}
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	app := &hu22BDDApp{db: db, server: ts, client: client, projectID: projectID}
	app.selectProject(t)
	return app
}

type hu22BDDFailingBacklogStore struct{}

func (hu22BDDFailingBacklogStore) ProjectIsActive(int64) (bool, error) {
	return true, nil
}

func (hu22BDDFailingBacklogStore) CreateStory(int64, backlog.UserStory) (backlog.UserStory, error) {
	return backlog.UserStory{}, errors.New("simulated database failure")
}

// UpdateStory completa el puerto ampliado por HU-13 (T027).
func (hu22BDDFailingBacklogStore) UpdateStory(context.Context, int64, backlog.UserStory) error {
	return errors.New("simulated database failure")
}

func (a *hu22BDDApp) request(t *testing.T, method, path string, values url.Values) (*http.Response, string) {
	t.Helper()

	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequest(method, a.server.URL+path, body)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, path, err)
	}
	if values != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if method == http.MethodPost {
		req.Header.Set("Origin", a.server.URL)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	return resp, string(responseBody)
}

func (a *hu22BDDApp) selectProject(t *testing.T) {
	t.Helper()

	resp, _ := a.request(t, http.MethodPost, "/contexto/proyecto", url.Values{
		"proyecto_id": {fmt.Sprintf("%d", a.projectID)},
		"volver_a":    {"/backlog"},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
		t.Fatalf("select active project: status/location = %d/%q, want 303 /backlog",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

func (a *hu22BDDApp) validStoryValues(title string) url.Values {
	return url.Values{
		web.BacklogFieldTitle:          {title},
		web.BacklogFieldDescription:    {"Descripción BDD"},
		web.BacklogFieldPriority:       {"Media"},
		web.BacklogFieldStoryPoints:    {"3"},
		web.BacklogFieldEstimatedHours: {"8"},
		web.BacklogFieldCriterion:      {"Criterio BDD"},
	}
}

func (a *hu22BDDApp) storyCount(t *testing.T) int {
	t.Helper()
	var count int
	if err := a.db.QueryRow(
		`SELECT COUNT(*) FROM user_stories WHERE project_id = $1`, a.projectID,
	).Scan(&count); err != nil {
		t.Fatalf("count BDD stories: %v", err)
	}
	return count
}

func TestWebBacklogBDD_CreateStory(t *testing.T) {
	t.Run("Given un proyecto activo y datos válidos, When guardo, Then veo la historia Pendiente", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues(fmt.Sprintf("HU22 BDD created %d", time.Now().UnixNano()))

		resp, _ := app.request(t, http.MethodPost, "/backlog", values)

		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
			t.Fatalf("POST status/location = %d/%q, want 303 /backlog",
				resp.StatusCode, resp.Header.Get("Location"))
		}
		_, body := app.request(t, http.MethodGet, "/backlog", nil)
		for _, want := range []string{
			web.MsgStoryCreated, values.Get(web.BacklogFieldTitle), "Media", "3", "8", "Pendiente",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("Backlog does not contain %q", want)
			}
		}
		if got := app.storyCount(t); got != 1 {
			t.Errorf("saved stories = %d, want 1", got)
		}
	})
}

func TestWebBacklogBDD_AddCriterionPreservesForm(t *testing.T) {
	t.Run("Given el formulario escrito, When agrego un criterio, Then conserva los datos y no guarda", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("Título conservado")
		values[web.BacklogFieldCriterion] = []string{"Primer criterio"}
		values.Set(web.BacklogFieldAction, "agregar_criterio")

		resp, body := app.request(t, http.MethodPost, "/backlog", values)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST add criterion status = %d, want 200", resp.StatusCode)
		}
		if got := strings.Count(body, `name="criterio"`); got != 2 {
			t.Errorf("rendered criteria fields = %d, want 2", got)
		}
		for _, want := range []string{"Título conservado", "Primer criterio"} {
			if !strings.Contains(body, want) {
				t.Errorf("form did not preserve %q", want)
			}
		}
		if got := app.storyCount(t); got != 0 {
			t.Errorf("adding a criterion created %d stories, want 0", got)
		}
	})
}

func TestWebBacklogBDD_RemoveCriterion(t *testing.T) {
	t.Run("Given varios criterios, When quito uno, Then solo desaparece ese campo", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("Título con criterios")
		values[web.BacklogFieldCriterion] = []string{"Criterio uno", "Criterio dos", "Criterio tres"}
		values.Set(web.BacklogFieldAction, "quitar_criterio:1")

		resp, body := app.request(t, http.MethodPost, "/backlog", values)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST remove criterion status = %d, want 200", resp.StatusCode)
		}
		if got := strings.Count(body, `name="criterio"`); got != 2 {
			t.Errorf("rendered criteria fields = %d, want 2", got)
		}
		for _, want := range []string{"Título con criterios", "Criterio uno", "Criterio tres"} {
			if !strings.Contains(body, want) {
				t.Errorf("form did not preserve %q", want)
			}
		}
		if strings.Contains(body, "Criterio dos") {
			t.Error("removed criterion is still present")
		}
		if got := app.storyCount(t); got != 0 {
			t.Errorf("removing a criterion created %d stories, want 0", got)
		}
	})
}

func TestWebBacklogBDD_CannotRemoveLastCriterion(t *testing.T) {
	t.Run("Given un único criterio, When lo quito, Then permanece sin guardar", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("Título con último criterio")
		values[web.BacklogFieldCriterion] = []string{"Único criterio"}
		values.Set(web.BacklogFieldAction, "quitar_criterio:0")

		resp, body := app.request(t, http.MethodPost, "/backlog", values)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST remove-last-criterion status = %d, want 200", resp.StatusCode)
		}
		if got := strings.Count(body, `name="criterio"`); got != 1 {
			t.Errorf("rendered criteria fields = %d, want 1", got)
		}
		if !strings.Contains(body, "Único criterio") {
			t.Error("the last criterion was not preserved")
		}
		if got := app.storyCount(t); got != 0 {
			t.Errorf("removing the last criterion created %d stories, want 0", got)
		}
	})
}

func TestWebBacklogBDD_BlankCriteriaAreIgnored(t *testing.T) {
	t.Run("Given criterios en blanco y uno completo, When guardo, Then solo persiste el completo", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("HU22 BDD blank criteria")
		values[web.BacklogFieldCriterion] = []string{"Criterio válido", "", "   "}

		resp, _ := app.request(t, http.MethodPost, "/backlog", values)

		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST status = %d, want 303", resp.StatusCode)
		}
		var count int
		var content string
		err := app.db.QueryRow(`
			SELECT COUNT(*), MIN(ac.content)
			FROM acceptance_criteria ac
			JOIN user_stories us ON us.id = ac.story_id
			WHERE us.project_id = $1`, app.projectID).Scan(&count, &content)
		if err != nil {
			t.Fatalf("read saved acceptance criteria: %v", err)
		}
		if count != 1 || content != "Criterio válido" {
			t.Errorf("saved criteria = count %d, content %q; want one nonblank criterion", count, content)
		}
	})
}

func TestWebBacklogBDD_StoryPointsCanBeMissing(t *testing.T) {
	t.Run("Given los demás datos válidos, When guardo sin Story Points, Then veo advertencia y Sin estimar", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("HU22 BDD no points")
		values.Set(web.BacklogFieldStoryPoints, "")

		resp, _ := app.request(t, http.MethodPost, "/backlog", values)
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST status = %d, want 303", resp.StatusCode)
		}
		_, body := app.request(t, http.MethodGet, "/backlog", nil)
		for _, want := range []string{web.MsgStoryCreatedMissingPoints, "Sin estimar"} {
			if !strings.Contains(body, want) {
				t.Errorf("Backlog response does not contain %q", want)
			}
		}
		var points sql.NullInt64
		if err := app.db.QueryRow(
			`SELECT story_points FROM user_stories WHERE project_id = $1`, app.projectID,
		).Scan(&points); err != nil {
			t.Fatalf("read missing Story Points: %v", err)
		}
		if points.Valid {
			t.Errorf("stored Story Points = %d, want NULL", points.Int64)
		}
	})
}

func TestWebBacklogBDD_EstimatedHoursAreOptional(t *testing.T) {
	t.Run("Given los demás datos válidos, When guardo sin horas, Then no hay advertencia y dice Sin estimar", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("HU22 BDD no hours")
		values.Set(web.BacklogFieldEstimatedHours, "")

		resp, _ := app.request(t, http.MethodPost, "/backlog", values)
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST status = %d, want 303", resp.StatusCode)
		}
		_, body := app.request(t, http.MethodGet, "/backlog", nil)
		if !strings.Contains(body, web.MsgStoryCreated) {
			t.Errorf("Backlog response does not contain %q", web.MsgStoryCreated)
		}
		if strings.Contains(body, web.MsgStoryCreatedMissingPoints) {
			t.Error("missing estimated hours incorrectly produced a Story Points warning")
		}
		if !strings.Contains(body, "Sin estimar") {
			t.Error("missing estimated hours are not shown as Sin estimar")
		}
		var hours sql.NullFloat64
		if err := app.db.QueryRow(
			`SELECT estimated_hours FROM user_stories WHERE project_id = $1`, app.projectID,
		).Scan(&hours); err != nil {
			t.Fatalf("read missing estimated hours: %v", err)
		}
		if hours.Valid {
			t.Errorf("stored estimated hours = %v, want NULL", hours.Float64)
		}
	})
}

func TestWebBacklogBDD_CommaDecimalHours(t *testing.T) {
	t.Run("Given horas con coma decimal, When guardo 1,5, Then el valor queda exacto y visible", func(t *testing.T) {
		app := newHU22BDDApp(t)
		values := app.validStoryValues("HU22 BDD comma hours")
		values.Set(web.BacklogFieldEstimatedHours, "1,5")

		resp, _ := app.request(t, http.MethodPost, "/backlog", values)
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST status = %d, want 303", resp.StatusCode)
		}
		_, body := app.request(t, http.MethodGet, "/backlog", nil)
		if !strings.Contains(body, "1,5") {
			t.Error("Backlog did not display the decimal using a comma")
		}
		var got sql.NullFloat64
		if err := app.db.QueryRow(
			`SELECT estimated_hours FROM user_stories WHERE project_id = $1`, app.projectID,
		).Scan(&got); err != nil {
			t.Fatalf("read comma-decimal estimated hours: %v", err)
		}
		if !got.Valid || math.Abs(got.Float64-1.5) > 0.000001 {
			t.Errorf("stored estimated hours = %v (valid %t), want 1.5", got.Float64, got.Valid)
		}
	})
}

func TestWebBacklogBDD_InvalidFormSubmissions(t *testing.T) {
	tests := []struct {
		name         string
		field        string
		value        string
		message      string
		retainedHTML string
		mutate       func(url.Values)
	}{
		{
			name:         "Given criterios sin texto, When guardo, Then recibo el error de HU-02 y no se crea la historia",
			field:        web.BacklogFieldCriterion,
			message:      backlog.ErrAcceptanceCriteriaRequired.Error(),
			retainedHTML: `value="   "`,
			mutate: func(values url.Values) {
				values[web.BacklogFieldCriterion] = []string{"", "   "}
			},
		},
		{
			name:         "Given el título vacío, When guardo, Then recibo el error junto al título",
			field:        web.BacklogFieldTitle,
			value:        "   ",
			message:      backlog.ErrTitleRequired.Error(),
			retainedHTML: `value="   "`,
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldTitle, "   ")
			},
		},
		{
			name:         "Given la descripción vacía, When guardo, Then recibo el error junto a la descripción",
			field:        web.BacklogFieldDescription,
			value:        "   ",
			message:      backlog.ErrDescriptionRequired.Error(),
			retainedHTML: ">   </textarea>",
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldDescription, "   ")
			},
		},
		{
			name:         "Given una prioridad manipulada, When guardo, Then recibo el error de HU-02",
			field:        web.BacklogFieldPriority,
			value:        "Urgente",
			message:      backlog.ErrInvalidPriority.Error(),
			retainedHTML: `value="Urgente"`,
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldPriority, "Urgente")
			},
		},
		{
			name:         "Given Story Points fuera de Fibonacci, When guardo, Then recibo el error de HU-22",
			field:        web.BacklogFieldStoryPoints,
			value:        "4",
			message:      web.MsgInvalidStoryPoints,
			retainedHTML: `value="4"`,
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldStoryPoints, "4")
			},
		},
		{
			name:         "Given Story Points no numéricos, When guardo, Then recibo el error de HU-22",
			field:        web.BacklogFieldStoryPoints,
			value:        "abc",
			message:      web.MsgInvalidStoryPoints,
			retainedHTML: `value="abc"`,
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldStoryPoints, "abc")
			},
		},
		{
			name:         "Given horas estimadas inválidas, When guardo, Then recibo el error de HU-17",
			field:        web.BacklogFieldEstimatedHours,
			value:        "1,555",
			message:      web.MsgInvalidEstimatedHours,
			retainedHTML: `value="1,555"`,
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldEstimatedHours, "1,555")
			},
		},
		{
			name:         "Given horas estimadas por encima del máximo, When guardo, Then recibo el mensaje de límite",
			field:        web.BacklogFieldEstimatedHours,
			value:        "9999,995",
			message:      web.MsgHoursTooLarge,
			retainedHTML: `value="9999,995"`,
			mutate: func(values url.Values) {
				values.Set(web.BacklogFieldEstimatedHours, "9999,995")
			},
		},
	}

	app := newHU22BDDApp(t)
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := app.validStoryValues(fmt.Sprintf("HU22 BDD invalid %d %d", time.Now().UnixNano(), i))
			tt.mutate(values)

			resp, body := app.request(t, http.MethodPost, "/backlog", values)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST status = %d, want 422; body=%s", resp.StatusCode, body)
			}
			if !strings.Contains(body, tt.message) {
				t.Errorf("response does not contain error message %q", tt.message)
			}
			if !strings.Contains(body, `id="`+tt.field+`-error"`) {
				t.Errorf("error message is not associated with field %q", tt.field)
			}
			if !strings.Contains(body, tt.retainedHTML) {
				t.Errorf("form did not retain submitted value %q", tt.retainedHTML)
			}
			if got := app.storyCount(t); got != 0 {
				t.Errorf("invalid form created %d stories, want 0", got)
			}
		})
	}
	for _, invalid := range []string{"0", "-1", "abc"} {
		t.Run("Given horas "+invalid+", When guardo, Then el valor se rechaza sin guardar", func(t *testing.T) {
			values := app.validStoryValues(fmt.Sprintf("HU22 BDD invalid hours %s %d", invalid, time.Now().UnixNano()))
			values.Set(web.BacklogFieldEstimatedHours, invalid)
			resp, body := app.request(t, http.MethodPost, "/backlog", values)

			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST status = %d, want 422; body=%s", resp.StatusCode, body)
			}
			if !strings.Contains(body, web.MsgInvalidEstimatedHours) ||
				!strings.Contains(body, `value="`+invalid+`"`) {
				t.Errorf("response did not show the hours error and retain %q", invalid)
			}
			if got := app.storyCount(t); got != 0 {
				t.Errorf("invalid hours created %d stories, want 0", got)
			}
		})
	}
	for _, invalid := range []string{"0", "-1"} {
		t.Run("Given Story Points "+invalid+", When guardo, Then el valor se rechaza sin guardar", func(t *testing.T) {
			values := app.validStoryValues(fmt.Sprintf("HU22 BDD invalid points %s %d", invalid, time.Now().UnixNano()))
			values.Set(web.BacklogFieldStoryPoints, invalid)
			resp, body := app.request(t, http.MethodPost, "/backlog", values)

			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST status = %d, want 422; body=%s", resp.StatusCode, body)
			}
			if !strings.Contains(body, web.MsgInvalidStoryPoints) ||
				!strings.Contains(body, `value="`+invalid+`"`) {
				t.Errorf("response did not show the Story Points error and submitted value %q", invalid)
			}
			if got := app.storyCount(t); got != 0 {
				t.Errorf("invalid Story Points created %d stories, want 0", got)
			}
		})
	}
	for _, invalid := range []string{"10000", "9999,995"} {
		t.Run("Given horas "+invalid+", When guardo, Then se informa el máximo permitido", func(t *testing.T) {
			values := app.validStoryValues(fmt.Sprintf("HU22 BDD above max %s %d", invalid, time.Now().UnixNano()))
			values.Set(web.BacklogFieldEstimatedHours, invalid)
			resp, body := app.request(t, http.MethodPost, "/backlog", values)

			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("POST status = %d, want 422; body=%s", resp.StatusCode, body)
			}
			if !strings.Contains(body, web.MsgHoursTooLarge) ||
				!strings.Contains(body, `value="`+invalid+`"`) {
				t.Errorf("response did not show the maximum-hours error and retain %q", invalid)
			}
			if got := app.storyCount(t); got != 0 {
				t.Errorf("over-limit hours created %d stories, want 0", got)
			}
		})
	}
}

func TestWebBacklogBDD_PersistenceFailureShowsGenericErrorAndRetainsForm(t *testing.T) {
	app := newHU22BDDAppWithStore(t, hu22BDDFailingBacklogStore{})
	values := app.validStoryValues("HU22 BDD failed persistence")
	values.Set(web.BacklogFieldDescription, "Descripción conservada")
	values[web.BacklogFieldCriterion] = []string{"Criterio conservado"}

	resp, body := app.request(t, http.MethodPost, "/backlog", values)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("POST status = %d, want 500; body=%s", resp.StatusCode, body)
	}
	for _, want := range []string{
		web.MsgOperationFailed,
		`value="HU22 BDD failed persistence"`,
		">Descripción conservada</textarea>",
		`value="Criterio conservado"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain generic error or retained value %q", want)
		}
	}
	if strings.Contains(body, web.MsgStoryCreated) {
		t.Error("persistence failure response contains a success message")
	}
	if got := app.storyCount(t); got != 0 {
		t.Errorf("failed persistence created %d stories, want 0", got)
	}
}

func TestWebBacklogBDD_BacklogListAndEmptyState(t *testing.T) {
	t.Run("Given un proyecto activo con historias, When abro Backlog, Then veo solo sus historias ordenadas por prioridad", func(t *testing.T) {
		app := newHU22BDDApp(t)
		ctx := context.Background()

		var otherProjectID int64
		if err := app.db.QueryRowContext(ctx, `
				INSERT INTO projects (name, start_date, end_date)
				VALUES ($1, CURRENT_DATE, CURRENT_DATE + 1)
				RETURNING id`, fmt.Sprintf("HU22 BDD other project %d", time.Now().UnixNano())).Scan(&otherProjectID); err != nil {
			t.Fatalf("create other BDD project: %v", err)
		}
		t.Cleanup(func() {
			if _, err := app.db.Exec(`DELETE FROM user_stories WHERE project_id = $1`, otherProjectID); err != nil {
				t.Errorf("delete other-project BDD stories: %v", err)
			}
			if _, err := app.db.Exec(`DELETE FROM projects WHERE id = $1`, otherProjectID); err != nil {
				t.Errorf("delete other BDD project: %v", err)
			}
		})

		insertStory := func(projectID int64, title string, points *int, hours *float64) int64 {
			t.Helper()
			var id int64
			err := app.db.QueryRowContext(ctx, `
					INSERT INTO user_stories (
						project_id, title, description, priority, story_points, estimated_hours, status
					)
					VALUES ($1, $2, 'BDD listado', 'Media', $3, $4, 'Pendiente')
					RETURNING id`, projectID, title, points, hours).Scan(&id)
			if err != nil {
				t.Fatalf("create BDD list story: %v", err)
			}
			return id
		}

		points := 5
		hours := 1.5
		olderTitle := fmt.Sprintf("HU22 BDD older %d", time.Now().UnixNano())
		newerTitle := fmt.Sprintf("HU22 BDD newer %d", time.Now().UnixNano())
		otherTitle := fmt.Sprintf("HU22 BDD foreign %d", time.Now().UnixNano())
		olderID := insertStory(app.projectID, olderTitle, &points, nil)
		newerID := insertStory(app.projectID, newerTitle, nil, &hours)
		insertStory(otherProjectID, otherTitle, &points, &hours)

		resp, body := app.request(t, http.MethodGet, "/backlog", nil)
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
			fmt.Sprintf("%d", newerID),
			newerTitle,
			"Media",
			"Sin estimar",
			"1,5",
			"Pendiente",
			fmt.Sprintf("%d", olderID),
			olderTitle,
			"5",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("Backlog does not contain %q", want)
			}
		}
		start := strings.Index(body, "<tbody>")
		end := strings.Index(body, "</tbody>")
		if start < 0 || end < start {
			t.Fatal("Backlog response does not contain a complete stories table")
		}
		tableRows := body[start:end]
		if strings.Count(tableRows, "Sin estimar") != 2 {
			t.Errorf("missing estimates displayed as Sin estimar %d times, want 2",
				strings.Count(tableRows, "Sin estimar"))
		}
		if strings.Contains(body, otherTitle) {
			t.Errorf("Backlog included another project's story %q", otherTitle)
		}
		// HU-13 (T037): a igual prioridad, la más antigua primero (FR-001).
		if strings.Index(body, olderTitle) >= strings.Index(body, newerTitle) {
			t.Errorf("older story %q should precede newer story %q", olderTitle, newerTitle)
		}
	})

	t.Run("Given un proyecto sin historias, When abro Backlog, Then veo el estado vacío y el formulario", func(t *testing.T) {
		app := newHU22BDDApp(t)

		var emptyProjectID int64
		if err := app.db.QueryRow(`
				INSERT INTO projects (name, start_date, end_date)
				VALUES ($1, CURRENT_DATE, CURRENT_DATE + 1)
				RETURNING id`, fmt.Sprintf("HU22 BDD empty project %d", time.Now().UnixNano())).Scan(&emptyProjectID); err != nil {
			t.Fatalf("create empty BDD project: %v", err)
		}
		t.Cleanup(func() {
			if _, err := app.db.Exec(`DELETE FROM projects WHERE id = $1`, emptyProjectID); err != nil {
				t.Errorf("delete empty BDD project: %v", err)
			}
		})
		resp, _ := app.request(t, http.MethodPost, "/contexto/proyecto", url.Values{
			"proyecto_id": {fmt.Sprintf("%d", emptyProjectID)},
			"volver_a":    {"/backlog"},
		})
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/backlog" {
			t.Fatalf("select empty project status/location = %d/%q, want 303 /backlog",
				resp.StatusCode, resp.Header.Get("Location"))
		}

		getResp, body := app.request(t, http.MethodGet, "/backlog", nil)
		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("GET empty /backlog status = %d, want 200; body=%s", getResp.StatusCode, body)
		}
		if got := strings.Count(body, "Aún no hay historias"); got != 1 {
			t.Errorf("empty-state phrase occurred %d times, want exactly once", got)
		}
		if strings.Contains(body, "<table") || !strings.Contains(body, `action="/backlog"`) {
			t.Error("empty state should hide the story table and retain the creation form")
		}
	})
}
