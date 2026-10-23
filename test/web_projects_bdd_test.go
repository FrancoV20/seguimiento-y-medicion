package test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

type webProjectsBDDApp struct {
	db            *sql.DB
	server        *httptest.Server
	memberID      int64
	projectPrefix string
}

type noMembersBDDProjectCatalog struct{ web.ProjectCatalog }

func (noMembersBDDProjectCatalog) ListMembers(_ context.Context) ([]web.MemberOption, error) {
	return []web.MemberOption{}, nil
}

func newWebProjectsBDDApp(t *testing.T, noMembers bool) *webProjectsBDDApp {
	t.Helper()

	databaseURL := os.Getenv(web.EnvDatabaseURL)
	if databaseURL == "" {
		t.Skip("DATABASE_URL no definida: se omite el escenario con PostgreSQL")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("no se pudo abrir PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}

	stamp := time.Now().UnixNano()
	memberName := fmt.Sprintf("HU21 BDD integrante %d", stamp)
	projectPrefix := fmt.Sprintf("HU21 BDD %d", stamp)
	var memberID int64
	if err := db.QueryRow(
		`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, memberName,
	).Scan(&memberID); err != nil {
		t.Fatalf("no se pudo crear el integrante BDD: %v", err)
	}

	catalog := web.ProjectCatalog(web.NewPostgreSQLProjectCatalog(db))
	if noMembers {
		catalog = noMembersBDDProjectCatalog{ProjectCatalog: catalog}
	}
	srv := web.NewServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	if err := web.RegisterProjects(srv, project.NewProjectService(project.NewPostgreSQLProjectStore(db)), catalog); err != nil {
		t.Fatalf("no se pudo registrar la sección Proyectos: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	app := &webProjectsBDDApp{db: db, server: ts, memberID: memberID}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM projects WHERE name LIKE $1`, projectPrefix+" %")
		_, _ = db.Exec(`DELETE FROM members WHERE id = $1`, memberID)
	})
	app.projectPrefix = projectPrefix
	return app
}

func webProjectsBDDPost(t *testing.T, app *webProjectsBDDApp, name, start, end string) (*http.Response, string) {
	t.Helper()
	form := url.Values{
		web.ProjectNameField:      {name},
		web.ProjectStartDateField: {start},
		web.ProjectEndDateField:   {end},
		web.ProjectMembersField:   {fmt.Sprint(app.memberID)},
	}
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.PostForm(app.server.URL+"/proyectos", form)
	if err != nil {
		t.Fatalf("POST /proyectos: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	return resp, string(body)
}

func countBDDProject(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE name = $1`, name).Scan(&count); err != nil {
		t.Fatalf("no se pudo consultar el proyecto BDD: %v", err)
	}
	return count
}

func TestWebProjects_CreateValidProject_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false)
	name := app.projectPrefix + " alta"

	resp, _ := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-24")

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/proyectos" {
		t.Fatalf("al crear se esperaba 303 a /proyectos, se obtuvo %d a %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	page, err := http.Get(app.server.URL + resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("GET /proyectos: %v", err)
	}
	defer page.Body.Close()
	body, err := io.ReadAll(page.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la página de proyectos: %v", err)
	}
	if !strings.Contains(string(body), web.MsgProjectCreated) || !strings.Contains(string(body), name) {
		t.Errorf("la página debía mostrar el éxito y el proyecto; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 1 {
		t.Errorf("se esperaba un proyecto creado, se encontraron %d", count)
	}
}

func TestWebProjects_EqualDatesAreRejected_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, false)
	name := app.projectPrefix + " fechas iguales"

	resp, body := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-23")

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("las fechas iguales debían rechazarse con 422, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, project.ErrInvalidPeriod.Error()) || !strings.Contains(body, web.ProjectEndDateField) {
		t.Errorf("se esperaba el error de fechas de HU-01 junto a fecha_fin; respuesta: %s", body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse el proyecto con fechas iguales, se encontraron %d", count)
	}
}

func TestWebProjects_NoAvailableMembersPreventsCreation_BDD(t *testing.T) {
	app := newWebProjectsBDDApp(t, true)
	name := app.projectPrefix + " sin integrantes"

	resp, body := webProjectsBDDPost(t, app, name, "2026-10-23", "2026-10-24")

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("sin integrantes disponibles se esperaba 422, se obtuvo %d", resp.StatusCode)
	}
	if !strings.Contains(body, web.MsgNoMembersAvailable) {
		t.Errorf("se esperaba %q; respuesta: %s", web.MsgNoMembersAvailable, body)
	}
	if count := countBDDProject(t, app.db, name); count != 0 {
		t.Errorf("no debía persistirse un proyecto sin integrantes disponibles, se encontraron %d", count)
	}
}
