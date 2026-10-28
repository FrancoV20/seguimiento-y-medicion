package poker

import "strconv"

// Series es la serie de votación: la misma de Fibonacci que acepta HU-02 para los Story Points.
var Series = []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}

// InSeries indica si v es un valor de la serie.
func InSeries(v int) bool {
	for _, s := range Series {
		if s == v {
			return true
		}
	}
	return false
}

// ParseValue lee el valor enviado. Vacío devuelve ErrValueRequired; cualquier texto que no sea
// un número canónico de la serie ("05", " 5", "5.0", "4") devuelve ErrValueNotInSeries.
func ParseValue(raw string) (int, error) {
	if raw == "" {
		return 0, ErrValueRequired
	}
	n, err := strconv.Atoi(raw)
	if err != nil || strconv.Itoa(n) != raw || !InSeries(n) {
		return 0, ErrValueNotInSeries
	}
	return n, nil
}
