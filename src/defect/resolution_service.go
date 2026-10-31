package defect

import "context"

// ResolutionService coordina la resolución y la reapertura de defectos.
type ResolutionService struct {
	store ResolutionStore
}

// NewResolutionService crea el servicio sobre su puerto de almacenamiento.
func NewResolutionService(store ResolutionStore) *ResolutionService {
	return &ResolutionService{store: store}
}

// Resolve resuelve el defecto en el Sprint activo del proyecto. Un identificador no válido se
// trata como defecto inexistente sin consultar el almacenamiento.
func (s *ResolutionService) Resolve(ctx context.Context, projectID, defectID int64) (Resolved, error) {
	if projectID <= 0 || defectID <= 0 {
		return Resolved{}, ErrDefectNotFound
	}
	return s.store.Resolve(ctx, projectID, defectID)
}

// Reopen vuelve a abrir un defecto resuelto. Un identificador no válido se trata como defecto
// inexistente sin consultar el almacenamiento.
func (s *ResolutionService) Reopen(ctx context.Context, projectID, defectID int64) error {
	if projectID <= 0 || defectID <= 0 {
		return ErrDefectNotFound
	}
	return s.store.Reopen(ctx, projectID, defectID)
}
