package web

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestPostgreSQLProjectCatalog_ListProjects(t *testing.T) {
	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)

	rows, err := NewPostgreSQLProjectCatalog(db).ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects devolvió error: %v", err)
	}

	positions := make(map[int64]int)
	found := make(map[int64]ProjectRow)
	for i, row := range rows {
		if _, belongsToFixture := fx.projects[row.ID]; belongsToFixture {
			positions[row.ID] = i
			found[row.ID] = row
		}
	}
	if len(found) != len(fx.projects) {
		t.Fatalf("se esperaban %d proyectos de prueba, se encontraron %d: %+v", len(fx.projects), len(found), found)
	}

	for id, want := range fx.projects {
		got := found[id]
		if got.Name != want.name || !sameCalendarDate(got.StartDate, want.startDate) ||
			!sameCalendarDate(got.EndDate, want.endDate) || got.MemberCount != want.memberCount {
			t.Errorf("proyecto %d: se esperaba %+v, se obtuvo %+v", id, want, got)
		}
	}
	if !(positions[fx.projectC] < positions[fx.projectB] &&
		positions[fx.projectB] < positions[fx.projectA]) {
		t.Errorf("los proyectos propios debían estar ordenados por id descendente (C, B, A); posiciones: %v", positions)
	}
}

func TestPostgreSQLProjectCatalog_ListMembersOrdersByNameAndID(t *testing.T) {
	db := openContextTestDB(t)
	fx := newProjectCatalogFixture(t, db)

	members, err := NewPostgreSQLProjectCatalog(db).ListMembers(context.Background())
	if err != nil {
		t.Fatalf("ListMembers devolvió error: %v", err)
	}

	var gotIDs []int64
	gotNames := make(map[int64]string)
	for _, member := range members {
		if name, belongsToFixture := fx.members[member.ID]; belongsToFixture {
			gotIDs = append(gotIDs, member.ID)
			gotNames[member.ID] = member.DisplayName
			if member.DisplayName != name {
				t.Errorf("integrante %d: se esperaba el nombre %q, se obtuvo %q", member.ID, name, member.DisplayName)
			}
		}
	}
	wantIDs := []int64{fx.alphaOne, fx.alphaTwo, fx.beta}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("se esperaban %d integrantes de prueba, se encontraron %d: %v", len(wantIDs), len(gotIDs), gotNames)
	}
	for i, id := range wantIDs {
		if gotIDs[i] != id {
			t.Errorf("orden de integrantes: en la posición %d se esperaba id %d, se obtuvo %d", i, id, gotIDs[i])
		}
	}
}

func TestPostgreSQLProjectCatalog_EmptyTablesReturnEmptySlices(t *testing.T) {
	db := openContextTestDB(t)
	db.SetMaxOpenConns(1)

	for _, statement := range []string{
		`CREATE TEMP TABLE projects (id INTEGER, name VARCHAR(255), start_date DATE, end_date DATE)`,
		`CREATE TEMP TABLE project_members (project_id INTEGER, member_id INTEGER)`,
		`CREATE TEMP TABLE members (id INTEGER, display_name VARCHAR(255))`,
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatalf("no se pudo preparar el esquema PostgreSQL aislado: %v", err)
		}
	}

	catalog := NewPostgreSQLProjectCatalog(db)
	projects, err := catalog.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects devolvió error para una tabla vacía: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("se esperaba una lista de proyectos vacía, se obtuvo %+v", projects)
	}

	members, err := catalog.ListMembers(context.Background())
	if err != nil {
		t.Fatalf("ListMembers devolvió error para una tabla vacía: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("se esperaba una lista de integrantes vacía, se obtuvo %+v", members)
	}
}

type projectCatalogProjectFixture struct {
	name        string
	startDate   time.Time
	endDate     time.Time
	memberCount int64
}

type projectCatalogFixture struct {
	projectA, projectB, projectC int64
	alphaOne, alphaTwo, beta     int64
	projects                     map[int64]projectCatalogProjectFixture
	members                      map[int64]string
}

func newProjectCatalogFixture(t *testing.T, db *sql.DB) *projectCatalogFixture {
	t.Helper()

	ctx := context.Background()
	stamp := time.Now().UnixNano()
	fx := &projectCatalogFixture{
		projects: make(map[int64]projectCatalogProjectFixture),
		members:  make(map[int64]string),
	}
	t.Cleanup(func() {
		for id := range fx.projects {
			if _, err := db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id); err != nil {
				t.Errorf("no se pudo limpiar el proyecto de prueba: %v", err)
			}
		}
		for id := range fx.members {
			if _, err := db.ExecContext(ctx, `DELETE FROM members WHERE id = $1`, id); err != nil {
				t.Errorf("no se pudo limpiar el integrante de prueba: %v", err)
			}
		}
	})

	insertID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		if err := db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
			t.Fatalf("no se pudo crear un dato de prueba: %v", err)
		}
		return id
	}
	insertMember := func(name string) int64 {
		id := insertID(`INSERT INTO members (display_name) VALUES ($1) RETURNING id`, name)
		fx.members[id] = name
		return id
	}
	insertProject := func(name string, startDate, endDate time.Time, memberCount int64) int64 {
		id := insertID(
			`INSERT INTO projects (name, start_date, end_date) VALUES ($1, $2, $3) RETURNING id`,
			name, startDate, endDate,
		)
		fx.projects[id] = projectCatalogProjectFixture{
			name: name, startDate: startDate, endDate: endDate, memberCount: memberCount,
		}
		return id
	}
	linkMember := func(projectID, memberID int64) {
		t.Helper()
		if _, err := db.ExecContext(ctx,
			`INSERT INTO project_members (project_id, member_id) VALUES ($1, $2)`,
			projectID, memberID,
		); err != nil {
			t.Fatalf("no se pudo vincular el integrante de prueba: %v", err)
		}
	}

	startDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	alphaName := fmt.Sprintf("HU21 Catalog Alpha %d", stamp)
	fx.alphaOne = insertMember(alphaName)
	fx.alphaTwo = insertMember(alphaName)
	fx.beta = insertMember(fmt.Sprintf("HU21 Catalog Beta %d", stamp))

	fx.projectA = insertProject(
		fmt.Sprintf("HU21 Catalog A %d", stamp), startDate, startDate.AddDate(0, 1, 0), 2,
	)
	fx.projectB = insertProject(
		fmt.Sprintf("HU21 Catalog B %d", stamp), startDate.AddDate(0, 0, 1), startDate.AddDate(0, 1, 1), 1,
	)
	fx.projectC = insertProject(
		fmt.Sprintf("HU21 Catalog C %d", stamp), startDate.AddDate(0, 0, 2), startDate.AddDate(0, 1, 2), 0,
	)
	linkMember(fx.projectA, fx.alphaOne)
	linkMember(fx.projectA, fx.beta)
	linkMember(fx.projectB, fx.alphaTwo)

	return fx
}

func sameCalendarDate(got, want time.Time) bool {
	return got.Year() == want.Year() && got.Month() == want.Month() && got.Day() == want.Day()
}
