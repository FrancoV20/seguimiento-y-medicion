package backlog

import (
	"math"
	"strings"
)

const (
	MsgStoryCreated       = "Historia registrada en el Product Backlog."
	WarnEstimationPending = "La historia no tiene Story Points: falta completar la estimación."
)

// BacklogService implementa el caso de uso de alta de historias.
type BacklogService struct {
	store BacklogStore
}

// NewBacklogService crea el servicio con el store indicado.
func NewBacklogService(store BacklogStore) *BacklogService {
	return &BacklogService{store: store}
}

// maxEstimatedHours es el máximo que admite NUMERIC(6,2) sin redondeo ni overflow.
const maxEstimatedHours = 9999.99

// validEstimatedHours acepta nil o un valor positivo con hasta dos decimales dentro del máximo.
func validEstimatedHours(h *float64) bool {
	if h == nil {
		return true
	}
	value := *h
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > maxEstimatedHours {
		return false
	}
	return math.Round(value*100)/100 == value
}

// CreateStory valida la solicitud y registra la historia en estado Pendiente (T017, T018, T025, T037).
// Valida todo antes de llamar al store, así un error no deja historias a medias.
func (s *BacklogService) CreateStory(req CreateStoryRequest) (*CreateStoryResult, error) {
	active, err := s.store.ProjectIsActive(req.ProjectID)
	if err != nil {
		return nil, ErrPersistence
	}
	if !active {
		return nil, ErrProjectNotActive
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		return nil, ErrDescriptionRequired
	}
	if !req.Priority.IsValid() {
		return nil, ErrInvalidPriority
	}
	criteria := make([]AcceptanceCriterion, 0, len(req.AcceptanceCriteria))
	for _, c := range req.AcceptanceCriteria {
		if c = strings.TrimSpace(c); c != "" {
			criteria = append(criteria, AcceptanceCriterion{Position: len(criteria) + 1, Content: c})
		}
	}
	if len(criteria) == 0 {
		return nil, ErrAcceptanceCriteriaRequired
	}

	if err := ValidateStoryPoints(req.StoryPoints); err != nil {
		return nil, err
	}
	if !validEstimatedHours(req.EstimatedHours) {
		return nil, ErrInvalidEstimatedHours
	}

	var storyPoints *int
	if req.StoryPoints != nil {
		v := *req.StoryPoints
		storyPoints = &v
	}
	var estimatedHours *float64
	if req.EstimatedHours != nil {
		h := *req.EstimatedHours
		estimatedHours = &h
	}

	story := UserStory{
		ProjectID:          req.ProjectID,
		Title:              title,
		Description:        description,
		Priority:           req.Priority,
		AcceptanceCriteria: criteria,
		StoryPoints:        storyPoints,
		EstimatedHours:     estimatedHours,
		Status:             StatusPending,
	}

	created, err := s.store.CreateStory(req.ProjectID, story)
	if err != nil {
		return nil, ErrPersistence
	}

	result := &CreateStoryResult{Story: &created, SuccessMessage: MsgStoryCreated}
	if created.StoryPoints == nil {
		result.Warning = WarnEstimationPending
	}
	return result, nil
}
