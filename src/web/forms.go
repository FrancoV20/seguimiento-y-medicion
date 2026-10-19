package web

// MsgOperationFailed es el texto único cuando falla un guardado (HU-20b, T006).
const MsgOperationFailed = "La operación no pudo completarse"

// FormView lleva a la plantilla los valores enviados y el error de cada campo.
type FormView struct {
	Values map[string]string
	Errors map[string]string
}