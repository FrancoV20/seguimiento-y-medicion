package project

import (
	"fmt"
	"time"
)

// DateStatus es el estado de un proyecto según sus fechas y la fecha de hoy (HU-12).
type DateStatus string

const (
	StatusNotStarted DateStatus = "No iniciado"
	StatusInProgress DateStatus = "En curso"
	StatusFinished   DateStatus = "Finalizado"
)

// StatusOn calcula el estado comparando días del calendario, no instantes: de cada fecha
// toma año, mes y día en su propia zona. Los dos bordes (inicio y fin) cuentan como "En curso".
// No lee el reloj: la fecha de hoy la recibe quien llama.
func StatusOn(start, end, today time.Time) DateStatus {
	startDay, endDay, todayDay := dayKey(start), dayKey(end), dayKey(today)
	switch {
	case todayDay < startDay:
		return StatusNotStarted
	case todayDay > endDay:
		return StatusFinished
	default:
		return StatusInProgress
	}
}

// dayKey convierte una fecha en un entero comparable (aaaammdd) usando su propio día de calendario.
func dayKey(t time.Time) int {
	year, month, day := t.Date()
	return year*10000 + int(month)*100 + day
}

// ProgressPercentage devuelve done ÷ total × 100 con dos decimales, mitad hacia arriba y punto
// decimal ("50.00"). Sin historias (total <= 0) devuelve "0.00". Usa enteros para evitar errores
// de punto flotante en las mitades exactas (1 de 32 = 3,125 → "3.13").
func ProgressPercentage(done, total int) string {
	if total <= 0 {
		return "0.00"
	}
	hundredths := (done*10000*2 + total) / (2 * total)
	return fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100)
}
