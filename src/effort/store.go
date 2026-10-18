package effort

import "context"

// EffortStore abstrae la persistencia de los registros de esfuerzo.
//
// Las implementaciones devuelven los errores de almacenamiento tal cual; es el servicio
// quien los convierte en ErrStorage para no exponer detalles internos al usuario.
type EffortStore interface {
	// StoryAcceptsEffort indica si la historia existe y pertenece a un Sprint activo.
	StoryAcceptsEffort(ctx context.Context, storyID string) (bool, error)

	// MemberBelongsToStoryProject indica si el integrante existe y pertenece al proyecto
	// al que pertenece la historia.
	MemberBelongsToStoryProject(ctx context.Context, memberID, storyID string) (bool, error)

	// SaveEntry guarda el registro y devuelve el registro creado (con su ID y fecha de
	// creación) junto con el esfuerzo acumulado actualizado de la historia. El guardado y
	// el cálculo del acumulado deben confirmarse como una única operación: ante cualquier
	// error no debe quedar ningún registro parcial.
	SaveEntry(ctx context.Context, entry EffortEntry) (EffortEntry, Hours, error)

	// AccumulatedHours devuelve la suma de las horas de todos los registros de la historia.
	// Una historia sin registros devuelve 0.
	AccumulatedHours(ctx context.Context, storyID string) (Hours, error)
}
