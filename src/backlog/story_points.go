package backlog

// validStoryPoints es la escala Fibonacci admitida (T008).
var validStoryPoints = map[int]bool{
	1: true, 2: true, 3: true, 5: true, 8: true,
	13: true, 21: true, 34: true, 55: true, 89: true,
}

// ValidateStoryPoints acepta nil (historia sin estimar) o un valor de la escala.
func ValidateStoryPoints(sp *int) error {
	if sp == nil {
		return nil
	}
	if !validStoryPoints[*sp] {
		return ErrInvalidStoryPoints
	}
	return nil
}
