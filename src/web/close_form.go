package web

// CloseSummary es el espacio reservado para el resumen del formulario de cierre.
type CloseSummary struct {
	Completed       []BoardStory
	ToReturn        []BoardStory
	PlannedPoints   int
	CompletedPoints int
}

// buildCloseSummary es un stub RED para mantener compilable el paquete web.
func buildCloseSummary(BoardView) CloseSummary {
	// TODO(HU-15 T018): construir el resumen de cierre desde el tablero.
	return CloseSummary{}
}
