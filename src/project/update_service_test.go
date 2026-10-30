package project

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// HU-11, T008-T009: UpdateProject con TestProjectStore.

func validUpdate() ProjectUpdateRequest {
	return ProjectUpdateRequest{ID: 7, Name: "Alfa", Members: []string{"1", "2"}, StartDate: day("2026-03-01"), EndDate: day("2026-06-30")}
}

func TestUpdateProject_Validaciones(t *testing.T) {
	cases := []struct {
		name   string
		change func(*ProjectUpdateRequest)
		want   error
	}{
		{"ID cero", func(r *ProjectUpdateRequest) { r.ID = 0 }, ErrProjectNotFound},
		{"nombre vacío", func(r *ProjectUpdateRequest) { r.Name = "" }, ErrEmptyName},
		{"nombre con espacios", func(r *ProjectUpdateRequest) { r.Name = "   " }, ErrEmptyName},
		{"sin integrantes", func(r *ProjectUpdateRequest) { r.Members = nil }, ErrMissingMembers},
		{"fin igual al inicio", func(r *ProjectUpdateRequest) { r.EndDate = r.StartDate }, ErrInvalidPeriod},
		{"integrante no numérico", func(r *ProjectUpdateRequest) { r.Members = []string{"abc"} }, ErrMemberNotFound},
		{"integrantes repetidos", func(r *ProjectUpdateRequest) { r.Members = []string{"1", "1"} }, ErrDuplicateMember},
	}
	for _, c := range cases {
		store := &TestProjectStore{}
		req := validUpdate()
		c.change(&req)
		err := NewProjectService(store).UpdateProject(context.Background(), req)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", c.name, c.want, err)
		}
		if len(store.Updates) != 0 {
			t.Errorf("%s: no debía llamarse a Update", c.name)
		}
	}
}

func TestUpdateProject_RecortaElNombre(t *testing.T) {
	store := &TestProjectStore{}
	req := validUpdate()
	req.Name = "  Alfa 2  "
	if err := NewProjectService(store).UpdateProject(context.Background(), req); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if store.Updates[0].Name != "Alfa 2" {
		t.Errorf("se esperaba el nombre recortado, se obtuvo %q", store.Updates[0].Name)
	}
}

func TestUpdateProject_DelegaYPropaga(t *testing.T) {
	store := &TestProjectStore{}
	req := validUpdate()
	if err := NewProjectService(store).UpdateProject(context.Background(), req); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	want := ProjectUpdate{ID: 7, Name: "Alfa", MemberIDs: []int{1, 2}, StartDate: req.StartDate, EndDate: req.EndDate}
	if len(store.Updates) != 1 || !reflect.DeepEqual(store.Updates[0], want) {
		t.Fatalf("Update recibió %+v, se esperaba %+v", store.Updates, want)
	}

	effort := &MemberHasEffortError{MemberName: "Beto"}
	sprint := &SprintOutOfRangeError{Number: 2, Field: DateFieldEnd}
	for _, domainErr := range []error{ErrProjectNotFound, effort, sprint, ErrMemberNotFound} {
		err := NewProjectService(&TestProjectStore{UpdateErr: domainErr}).UpdateProject(context.Background(), req)
		if err != domainErr {
			t.Errorf("el error de dominio %v debía devolverse sin cambios, se obtuvo %v", domainErr, err)
		}
	}
	var gotEffort *MemberHasEffortError
	err := NewProjectService(&TestProjectStore{UpdateErr: effort}).UpdateProject(context.Background(), req)
	if !errors.As(err, &gotEffort) || gotEffort.MemberName != "Beto" {
		t.Errorf("se esperaba MemberHasEffortError con Beto, se obtuvo %v", err)
	}

	err = NewProjectService(&TestProjectStore{ShouldFailUpdate: true}).UpdateProject(context.Background(), req)
	var anySprint *SprintOutOfRangeError
	if err == nil || errors.Is(err, ErrProjectNotFound) || errors.As(err, &gotEffort) || errors.As(err, &anySprint) {
		t.Errorf("una falla de almacenamiento debía envolverse sin parecer de dominio, se obtuvo %v", err)
	}
}
