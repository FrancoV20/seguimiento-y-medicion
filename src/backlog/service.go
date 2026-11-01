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

	fields, err := validateStoryFields(req.Title, req.Description, req.Priority, req.AcceptanceCriteria, req.StoryPoints)
	if err != nil {
		return nil, err
	}
	if !validEstimatedHours(req.EstimatedHours) {
		return nil, ErrInvalidEstimatedHours
	}

	var estimatedHours *float64
	if req.EstimatedHours != nil {
		h := *req.EstimatedHours
		estimatedHours = &h
	}

	story := UserStory{
		ProjectID:          req.ProjectID,
		Title:              fields.Title,
		Description:        fields.Description,
		Priority:           fields.Priority,
		AcceptanceCriteria: fields.Criteria,
		StoryPoints:        fields.StoryPoints,
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

// storyFields son los datos de una historia ya validados, recortados y con los criterios numerados.
type storyFields struct {
	Title       string
	Description string
	Priority    Priority
	Criteria    []AcceptanceCriterion
	StoryPoints *int
}

// validateStoryFields es la validación común del alta (HU-02) y de la edición (HU-13): recorta el
// título, la descripción y los criterios, descarta los criterios vacíos, numera los demás desde 1 y
// devuelve el primer error en este orden: título, descripción, prioridad, criterios y Story Points.
func validateStoryFields(title, description string, priority Priority, criteria []string, storyPoints *int) (storyFields, error) {
	fields := storyFields{
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		Priority:    priority,
	}
	if fields.Title == "" {
		return storyFields{}, ErrTitleRequired
	}
	if fields.Description == "" {
		return storyFields{}, ErrDescriptionRequired
	}
	if !priority.IsValid() {
		return storyFields{}, ErrInvalidPriority
	}
	fields.Criteria = make([]AcceptanceCriterion, 0, len(criteria))
	for _, c := range criteria {
		if c = strings.TrimSpace(c); c != "" {
			fields.Criteria = append(fields.Criteria, AcceptanceCriterion{Position: len(fields.Criteria) + 1, Content: c})
		}
	}
	if len(fields.Criteria) == 0 {
		return storyFields{}, ErrAcceptanceCriteriaRequired
	}
	if err := ValidateStoryPoints(storyPoints); err != nil {
		return storyFields{}, err
	}
	if storyPoints != nil {
		v := *storyPoints
		fields.StoryPoints = &v
	}
	return fields, nil
}
