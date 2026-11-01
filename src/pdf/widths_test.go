package pdf

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// HU-09, T005: anchos de Helvetica y corte de líneas (research.md §4, data-model.md).

func nearly(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func TestPDF_Anchos(t *testing.T) {
	if got := textWidth(Regular, 10, encodeWinAnsi("Página")); !nearly(got, 31.13) {
		t.Errorf("textWidth(Regular, 10, \"Página\") = %.4f, se esperaba 31,13", got)
	}
	if got := textWidth(Bold, 10, encodeWinAnsi("Página")); !nearly(got, 32.79) {
		t.Errorf("textWidth(Bold, 10, \"Página\") = %.4f, se esperaba 32,79", got)
	}
	if got := textWidth(Regular, 9, nil); got != 0 {
		t.Errorf("el ancho de un texto vacío debe ser 0, se obtuvo %.4f", got)
	}
}

func TestPDF_CorteDeLineas(t *testing.T) {
	t.Run("texto que entra", func(t *testing.T) {
		lines := wrapText(Regular, 10, 200, "Login (v2)")
		if len(lines) != 1 || !bytes.Equal(lines[0], encodeWinAnsi("Login (v2)")) {
			t.Fatalf("se esperaba una línea igual al texto, se obtuvo %q", lines)
		}
	})

	t.Run("varias palabras", func(t *testing.T) {
		text := "Como Scrum Master quiero exportar un reporte en PDF con el estado de un proyecto o Sprint"
		const width = 120.0
		lines := wrapText(Regular, 10, width, text)
		if len(lines) < 2 {
			t.Fatalf("se esperaban varias líneas, se obtuvo %d", len(lines))
		}
		var words []string
		for _, line := range lines {
			if w := textWidth(Regular, 10, line); w > width+0.001 {
				t.Errorf("la línea %q mide %.2f y supera %.2f", line, w, width)
			}
			words = append(words, strings.Fields(string(line))...)
		}
		if got := strings.Join(words, " "); got != string(encodeWinAnsi(text)) {
			t.Errorf("se perdieron palabras: %q", got)
		}
	})

	t.Run("palabra más ancha que el ancho", func(t *testing.T) {
		word := strings.Repeat("M", 40)
		const width = 50.0
		lines := wrapText(Bold, 9, width, word)
		if len(lines) < 2 {
			t.Fatalf("la palabra debía cortarse, se obtuvo %d línea", len(lines))
		}
		var joined []byte
		for _, line := range lines {
			if w := textWidth(Bold, 9, line); w > width+0.001 {
				t.Errorf("el trozo %q mide %.2f y supera %.2f", line, w, width)
			}
			joined = append(joined, line...)
		}
		if string(joined) != word {
			t.Errorf("la palabra cortada no coincide: %q", joined)
		}
	})

	t.Run("texto vacío", func(t *testing.T) {
		lines := wrapText(Regular, 10, 100, "")
		if len(lines) != 1 || len(lines[0]) != 0 {
			t.Fatalf("se esperaba una línea vacía, se obtuvo %q", lines)
		}
	})
}
