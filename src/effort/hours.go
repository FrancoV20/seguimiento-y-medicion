package effort

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Hours representa una cantidad de horas en centésimas de hora (400 = 4 h).
// Se usa un entero para que las sumas sean exactas, sin errores de punto flotante.
type Hours int64

const (
	// MaxHours es el máximo de horas permitido en un mismo registro (24 h).
	MaxHours Hours = 2400
)

// Hasta dos decimales, con punto o coma, y signo negativo opcional.
var hoursFormat = regexp.MustCompile(`^(-?)(\d+)(?:[.,](\d{1,2}))?$`)

// ParseHours interpreta el texto ingresado por el usuario.
// Devuelve ErrInvalidHours si no es numérico, tiene más de dos decimales,
// es menor o igual a cero o supera las 24 horas.
func ParseHours(input string) (Hours, error) {
	m := hoursFormat.FindStringSubmatch(strings.TrimSpace(input))
	if m == nil {
		return 0, ErrInvalidHours
	}
	if m[1] == "-" {
		return 0, ErrInvalidHours
	}

	entero, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil || entero > int64(MaxHours)/100 {
		return 0, ErrInvalidHours
	}

	decimales := m[3]
	if len(decimales) == 1 {
		decimales += "0"
	}
	var cent int64
	if decimales != "" {
		cent, _ = strconv.ParseInt(decimales, 10, 64)
	}

	h := Hours(entero*100 + cent)
	if h <= 0 || h > MaxHours {
		return 0, ErrInvalidHours
	}
	return h, nil
}

// Add devuelve la suma exacta de dos cantidades de horas.
func (h Hours) Add(other Hours) Hours { return h + other }

// String devuelve las horas sin ceros sobrantes: 400 -> "4", 250 -> "2.5", 5 -> "0.05".
func (h Hours) String() string {
	entero := int64(h) / 100
	cent := int64(h) % 100
	switch {
	case cent == 0:
		return strconv.FormatInt(entero, 10)
	case cent%10 == 0:
		return fmt.Sprintf("%d.%d", entero, cent/10)
	default:
		return fmt.Sprintf("%d.%02d", entero, cent)
	}
}
