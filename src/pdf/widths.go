package pdf

import "bytes"

// Anchos de Helvetica y Helvetica-Bold y corte de líneas (HU-09, research.md §4).

// helveticaWidths y helveticaBoldWidths son los anchos de cada byte WinAnsi en milésimas del tamaño
// de letra, tomados de las métricas AFM estándar de las dos fuentes.
var helveticaWidths = [256]int{
	278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, // 0x00
	278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, // 0x10
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278, // 0x20
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556, // 0x30
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778, // 0x40
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556, // 0x50
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556, // 0x60
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584, 350, // 0x70
	556, 350, 222, 556, 333, 1000, 556, 556, 333, 1000, 667, 333, 1000, 350, 611, 350, // 0x80
	350, 222, 222, 333, 333, 350, 556, 1000, 333, 1000, 500, 333, 944, 350, 500, 667, // 0x90
	278, 333, 556, 556, 556, 556, 260, 556, 333, 737, 370, 556, 584, 333, 737, 333, // 0xA0
	400, 584, 333, 333, 333, 556, 537, 278, 333, 333, 365, 556, 834, 834, 834, 611, // 0xB0
	667, 667, 667, 667, 667, 667, 1000, 722, 667, 667, 667, 667, 278, 278, 278, 278, // 0xC0
	722, 722, 778, 778, 778, 778, 778, 584, 778, 722, 722, 722, 722, 667, 667, 611, // 0xD0
	556, 556, 556, 556, 556, 556, 889, 500, 556, 556, 556, 556, 278, 278, 278, 278, // 0xE0
	556, 556, 556, 556, 556, 556, 556, 584, 611, 556, 556, 556, 556, 500, 556, 500, // 0xF0
}

var helveticaBoldWidths = [256]int{
	278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, // 0x00
	278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, 278, // 0x10
	278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278, // 0x20
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611, // 0x30
	975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778, // 0x40
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 584, 556, // 0x50
	333, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611, // 0x60
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 389, 280, 389, 584, 350, // 0x70
	556, 350, 278, 556, 500, 1000, 556, 556, 333, 1000, 667, 333, 1000, 350, 611, 350, // 0x80
	350, 278, 278, 500, 500, 350, 556, 1000, 333, 1000, 556, 333, 944, 350, 500, 667, // 0x90
	278, 333, 556, 556, 556, 556, 280, 556, 333, 737, 370, 556, 584, 333, 737, 333, // 0xA0
	400, 584, 333, 333, 333, 611, 556, 278, 333, 333, 365, 556, 834, 834, 834, 611, // 0xB0
	722, 722, 722, 722, 722, 722, 1000, 722, 667, 667, 667, 667, 278, 278, 278, 278, // 0xC0
	722, 722, 778, 778, 778, 778, 778, 584, 778, 722, 722, 722, 722, 667, 667, 611, // 0xD0
	556, 556, 556, 556, 556, 556, 889, 556, 556, 556, 556, 556, 278, 278, 278, 278, // 0xE0
	611, 611, 611, 611, 611, 611, 611, 584, 611, 611, 611, 611, 611, 556, 611, 556, // 0xF0
}

// textWidth devuelve el ancho en puntos de un texto ya codificado en WinAnsi.
func textWidth(font Font, size float64, encoded []byte) float64 {
	widths := &helveticaWidths
	if font == Bold {
		widths = &helveticaBoldWidths
	}
	total := 0
	for _, c := range encoded {
		total += widths[c]
	}
	return float64(total) * size / 1000
}

// wrapText codifica el texto y lo corta por palabras para que cada línea mida como máximo width
// puntos. Una palabra más ancha que width se corta por caracteres. Un texto vacío ocupa una línea
// vacía.
func wrapText(font Font, size, width float64, text string) [][]byte {
	encoded := encodeWinAnsi(text)
	if textWidth(font, size, encoded) <= width {
		return [][]byte{encoded}
	}
	var lines [][]byte
	var line []byte
	for _, word := range bytes.Fields(encoded) {
		if len(line) > 0 {
			candidate := append(append(append([]byte{}, line...), ' '), word...)
			if textWidth(font, size, candidate) <= width {
				line = candidate
				continue
			}
			lines = append(lines, line)
			line = nil
		}
		if textWidth(font, size, word) <= width {
			line = append([]byte{}, word...)
			continue
		}
		// Palabra más ancha que el ancho disponible: se corta por caracteres.
		for _, c := range word {
			if len(line) > 0 && textWidth(font, size, append(append([]byte{}, line...), c)) > width {
				lines = append(lines, line)
				line = nil
			}
			line = append(line, c)
		}
	}
	if len(line) > 0 || len(lines) == 0 {
		lines = append(lines, line)
	}
	return lines
}
