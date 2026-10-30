package poker

// maxLevelGap es la mayor distancia de niveles que todavía no es divergencia (R2).
const maxLevelGap = 2

// VoteSpread es el rango de los votos emitidos en una ronda (HU-05b). Es el contrato que
// reutiliza HU-18.
type VoteSpread struct {
	Low, High int  // menor y mayor valor emitido
	Divergent bool // level(High) - level(Low) > maxLevelGap
}

// Spread calcula el rango y la divergencia de los valores emitidos en una ronda.
// Sin valores devuelve ErrNoVotes; un valor fuera de Series, ErrValueNotInSeries.
// No ordena ni modifica values.
func Spread(values []int) (VoteSpread, error) {
	if len(values) == 0 {
		return VoteSpread{}, ErrNoVotes
	}
	low, high := -1, -1 // niveles del menor y del mayor valor
	for _, v := range values {
		l := level(v)
		if l == 0 {
			return VoteSpread{}, ErrValueNotInSeries
		}
		if low == -1 || l < low {
			low = l
		}
		if l > high {
			high = l
		}
	}
	return VoteSpread{
		Low:       Series[low-1],
		High:      Series[high-1],
		Divergent: high-low > maxLevelGap,
	}, nil
}

// level es la posición de v en Series más uno (1 → 1, 89 → 10); 0 si v no está en la serie.
func level(v int) int {
	for i, s := range Series {
		if s == v {
			return i + 1
		}
	}
	return 0
}
