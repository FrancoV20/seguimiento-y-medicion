package web

// Backlog form field names and user-facing messages shared by handlers and templates.
const (
	BacklogFieldTitle          = "titulo"
	BacklogFieldDescription    = "descripcion"
	BacklogFieldPriority       = "prioridad"
	BacklogFieldStoryPoints    = "story_points"
	BacklogFieldEstimatedHours = "horas_estimadas"
	BacklogFieldCriterion      = "criterio"
	BacklogFieldAction         = "accion"

	MsgStoryCreated              = "Historia creada correctamente"
	MsgStoryCreatedMissingPoints = "Historia creada correctamente. Falta completar la estimación de Story Points"
	MsgInvalidStoryPoints        = "Los Story Points no son un valor válido"
	MsgHoursTooLarge             = "Las horas estimadas no pueden superar 9999,99"
)
