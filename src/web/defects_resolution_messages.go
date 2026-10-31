package web

// Textos de HU-19 para resolver y reabrir defectos, en un solo lugar. La falla de guardado usa
// MsgOperationFailed de HU-20b.
const (
	// DefectIDField es el campo oculto de los formularios "Resolver" y "Reabrir".
	DefectIDField = "defecto"

	MsgDefectResolved                  = "Defecto resuelto en el Sprint %d"
	MsgDefectReopened                  = "Defecto reabierto"
	MsgDefectResolveNoSprint           = "No hay un Sprint activo para registrar la resolución del defecto"
	MsgDefectResolutionBeforeDetection = "El Sprint de resolución no puede ser anterior al Sprint en que se detectó el defecto"
	MsgDefectNotFound                  = "El defecto no existe"
	MsgDefectAlreadyResolved           = "El defecto ya está resuelto"
	MsgDefectAlreadyOpen               = "El defecto ya está abierto"
)
