package project

import "errors"

type TestProjectStore struct {
	Projects         []*Project
	Members          map[int]bool
	ShouldFailCreate bool
	ShouldFailMember bool
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

var _ ProjectStore = (*TestProjectStore)(nil)
