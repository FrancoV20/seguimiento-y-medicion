package effort

import (
	"context"
	"time"
)

// Service coordina el registro de esfuerzo: valida la solicitud, consulta el contexto de la
// historia y guarda el registro a través de un EffortStore.
type Service struct {
	store EffortStore
	now   func() time.Time
}

// NewService crea el servicio. now es el reloj: se inyecta para que las pruebas de fechas
// sean deterministas. Si es nil se usa time.Now.
func NewService(store EffortStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

// LogEffort registra un esfuerzo sobre una historia.
//
// Valida todos los campos y reúne todos los errores antes de guardar (FR-010). Si hay al
// menos uno, no se guarda nada. El mensaje de éxito y el acumulado solo se devuelven después
// de que el registro quedó persistido (FR-012).
func (s *Service) LogEffort(ctx context.Context, req LogEffortRequest) LogEffortResult {
	input, errs := ValidateInput(req, s.now())

	accepts, err := s.store.StoryAcceptsEffort(ctx, req.StoryID)
	if err != nil {
		return LogEffortResult{StorageError: ErrStorage}
	}
	if !accepts {
		errs = append(errs, FieldError{Field: FieldStory, Err: ErrStoryNotInSprint})
	}

	if len(errs) > 0 {
		return LogEffortResult{ValidationErrors: errs}
	}

	entry, total, err := s.store.SaveEntry(ctx, EffortEntry{
		StoryID:  req.StoryID,
		MemberID: req.MemberID,
		WorkDate: req.WorkDate,
		Activity: input.Activity,
		Hours:    input.Hours,
	})
	if err != nil {
		// Se informa un error genérico: nunca se exponen detalles internos de la base (FR-013).
		return LogEffortResult{StorageError: ErrStorage}
	}

	return LogEffortResult{
		Entry:            &entry,
		AccumulatedHours: total,
		SuccessMessage:   SuccessMessage,
	}
}
