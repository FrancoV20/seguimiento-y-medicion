package sprint

// Action es una acción que se puede hacer sobre una historia del Sprint activo
// desde el Tablero (HU-14). El valor es el que viaja en el campo "accion".
type Action string

const (
	ActionStart    Action = "comenzar"
	ActionComplete Action = "completar"
	ActionReopen   Action = "reabrir"
)

// CanTransition indica si una historia puede pasar de un estado a otro (HU-14, T010).
// RED: esqueleto sin implementar.
func CanTransition(from, to StoryStatus) bool {
	return false
}

// NextAction devuelve la única acción válida para un estado; ok es false si no hay (HU-14, T010).
// RED: esqueleto sin implementar.
func NextAction(status StoryStatus) (action Action, ok bool) {
	return "", false
}

// ActionTransition devuelve el estado de origen y el de destino de una acción; ok es false
// si la acción no existe (HU-14, T010).
// RED: esqueleto sin implementar.
func ActionTransition(action Action) (from, to StoryStatus, ok bool) {
	return "", "", false
}
