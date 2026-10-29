package project

import (
	"context"
	"errors"
)

type TestProjectStore struct {
	Projects         []*Project
	Members          map[int]bool
	ShouldFailCreate bool
	ShouldFailMember bool
	// HU-11: Updates guarda las modificaciones recibidas; ShouldFailUpdate simula una falla de
	// almacenamiento y UpdateErr devuelve un error de dominio.
	Updates          []ProjectUpdate
	ShouldFailUpdate bool
	UpdateErr        error
}

func (store *TestProjectStore) Create(p *Project) error {
	if store.ShouldFailCreate {
		return errors.New("simulated create error")
	}

	p.ID = len(store.Projects) + 1
	store.Projects = append(store.Projects, p)
	return nil
}

func (store *TestProjectStore) MemberExists(id int) (bool, error) {
	if store.ShouldFailMember {
		return false, errors.New("simulated member error")
	}

	return store.Members[id], nil
}

// Update registra la modificación o devuelve el error configurado (HU-11).
func (store *TestProjectStore) Update(_ context.Context, u ProjectUpdate) error {
	if store.ShouldFailUpdate {
		return errors.New("simulated update error")
	}
	if store.UpdateErr != nil {
		return store.UpdateErr
	}
	store.Updates = append(store.Updates, u)
	return nil
}

var _ ProjectStore = (*TestProjectStore)(nil)
