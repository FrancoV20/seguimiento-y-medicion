package project

import "strconv"

type ProjectService struct {
	store ProjectStore
}

func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{store: store}
}

func (service *ProjectService) CreateProject(req ProjectCreationRequest) (*Project, error) {
	if req.Name == "" {
		return nil, ErrEmptyName
	}
	if len(req.Members) == 0 {
		return nil, ErrMissingMembers
	}

	memberIDs := make([]int, 0, len(req.Members))
	seenMembers := make(map[int]struct{}, len(req.Members))
	for _, memberIDString := range req.Members {
		memberID, err := strconv.Atoi(memberIDString)
		if err != nil {
			return nil, ErrMemberNotFound
		}
		if _, exists := seenMembers[memberID]; exists {
			return nil, ErrDuplicateMember
		}
		seenMembers[memberID] = struct{}{}
		memberIDs = append(memberIDs, memberID)
	}

	for _, memberID := range memberIDs {
		exists, err := service.store.MemberExists(memberID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrMemberNotFound
		}
	}

	createdProject, err := NewProject(req.Name, req.Members, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	createdProject.Members = req.Members

	project := &createdProject
	if err := service.store.Create(project); err != nil {
		return nil, err
	}

	return project, nil
}
