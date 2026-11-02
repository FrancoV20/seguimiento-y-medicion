// Package role tiene la regla de los roles del proyecto (HU-28): cada integrante es
// "Scrum Master" o "Integrante" en cada proyecto, y solo el Scrum Master conduce las acciones
// de planificación y facilitación.
package role

import (
	"errors"
	"strings"
)

// Role es el rol de un integrante en un proyecto, con el texto que se guarda y se muestra.
type Role string

const (
	// RoleScrumMaster puede planificar y cerrar Sprints, modificar el proyecto y conducir el
	// Planning Poker.
	RoleScrumMaster Role = "Scrum Master"
	// RoleMember es el rol por defecto de todo integrante.
	RoleMember Role = "Integrante"
)

// ErrInvalidRole indica un texto que no es exactamente uno de los dos roles.
var ErrInvalidRole = errors.New("role: rol no válido")

// ParseRole convierte un texto en un rol: recorta los espacios de los bordes y exige
// coincidencia exacta ("scrum master", "SM" o "Product Owner" no son válidos).
func ParseRole(s string) (Role, error) {
	switch r := Role(strings.TrimSpace(s)); r {
	case RoleScrumMaster, RoleMember:
		return r, nil
	default:
		return "", ErrInvalidRole
	}
}

// CanManage indica si el rol puede hacer las acciones reservadas al Scrum Master.
func CanManage(r Role) bool { return r == RoleScrumMaster }
