package effort

import (
	"errors"
	"testing"
)

func TestParseHoursValid(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado Hours
	}{
		{"4", 400},
		{"0.5", 50},
		{"1,5", 150},
		{"24", 2400},
		{"24.00", 2400},
		{"0.01", 1},
		{"  2  ", 200},
		{"2.25", 225},
	}
	for _, c := range casos {
		t.Run(c.entrada, func(t *testing.T) {
			got, err := ParseHours(c.entrada)
			if err != nil {
				t.Fatalf("ParseHours(%q) devolvió error inesperado: %v", c.entrada, err)
			}
			if got != c.esperado {
				t.Errorf("ParseHours(%q) = %d, se esperaba %d", c.entrada, got, c.esperado)
			}
		})
	}
}

func TestParseHoursInvalid(t *testing.T) {
	casos := []string{
		"",                     // vacío
		"   ",                  // solo espacios
		"0",                    // cero
		"0.00",                 // cero con decimales
		"-1",                   // negativo
		"-0.5",                 // negativo decimal
		"24.01",                // apenas sobre el máximo
		"25",                   // sobre el máximo
		"100",                  // muy sobre el máximo
		"1.555",                // más de dos decimales
		"abc",                  // no numérico
		"1e2",                  // notación científica
		"4.",                   // decimal incompleto
		".5",                   // sin parte entera
		"--1",                  // doble signo
		"1,5,5",                // dos separadores
		"99999999999999999999", // desborda int64
	}
	for _, entrada := range casos {
		t.Run(entrada, func(t *testing.T) {
			got, err := ParseHours(entrada)
			if !errors.Is(err, ErrInvalidHours) {
				t.Fatalf("ParseHours(%q) = (%d, %v), se esperaba ErrInvalidHours", entrada, got, err)
			}
		})
	}
}

func TestInvalidHoursMessage(t *testing.T) {
	const esperado = "El valor de horas no es válido"
	if ErrInvalidHours.Error() != esperado {
		t.Errorf("mensaje = %q, se esperaba %q", ErrInvalidHours.Error(), esperado)
	}
}

func TestHoursAddIsExact(t *testing.T) {
	// 0,1 + 0,2 con float64 da 0.30000000000000004; con centésimas debe dar exactamente 0,30.
	got := Hours(10).Add(Hours(20))
	if got != Hours(30) {
		t.Errorf("0,10 + 0,20 = %d centésimas, se esperaba 30", got)
	}
}

func TestHoursString(t *testing.T) {
	casos := []struct {
		horas    Hours
		esperado string
	}{
		{400, "4"},
		{250, "2.5"},
		{5, "0.05"},
		{150, "1.5"},
		{2400, "24"},
		{0, "0"},
		{225, "2.25"},
	}
	for _, c := range casos {
		if got := c.horas.String(); got != c.esperado {
			t.Errorf("Hours(%d).String() = %q, se esperaba %q", c.horas, got, c.esperado)
		}
	}
}
