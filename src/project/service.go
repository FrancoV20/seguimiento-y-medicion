package project

import "errors"

type ProjectService struct {
	store ProjectStore
}

func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{store: store}
}

func (service *ProjectService) CreateProject(req ProjectCreationRequest) (*Project, error) {
	return nil, errors.New("not implemented")
}
