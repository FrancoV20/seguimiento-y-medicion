package pdf

import (
	"bytes"
	"errors"
	"strings"
)

// ExtractText devuelve el texto de todas las cadenas "(...) Tj" del documento, decodificado a UTF-8
// y unido con un espacio, en el orden en que se dibujan (HU-09, research.md §19). Así una celda
// cortada en varias líneas vuelve a leerse completa y una fila de tabla queda como el texto de sus
// celdas unidas con un espacio. Lo usan las pruebas de src/report, src/web y test/. Devuelve un
// error si no encuentra la tabla xref o ningún flujo de contenido.
func ExtractText(b []byte) (string, error) {
	if !bytes.Contains(b, []byte("\nxref\n")) {
		return "", errors.New("pdf: no se encontró la tabla xref")
	}
	var parts []string
	found := false
	rest := b
	for {
		start := bytes.Index(rest, []byte("stream\n"))
		if start < 0 {
			break
		}
		rest = rest[start+len("stream\n"):]
		end := bytes.Index(rest, []byte("\nendstream"))
		if end < 0 {
			return "", errors.New("pdf: flujo de contenido sin cerrar")
		}
		found = true
		parts = append(parts, streamStrings(rest[:end])...)
		rest = rest[end+len("\nendstream"):]
	}
	if !found {
		return "", errors.New("pdf: no se encontró ningún flujo de contenido")
	}
	return strings.Join(parts, " "), nil
}

// streamStrings devuelve, decodificadas, las cadenas literales seguidas de Tj de un flujo.
func streamStrings(content []byte) []string {
	var out []string
	for i := 0; i < len(content); i++ {
		if content[i] != '(' {
			continue
		}
		var raw []byte
		j := i + 1
		for ; j < len(content) && content[j] != ')'; j++ {
			if content[j] == '\\' && j+1 < len(content) {
				j++
			}
			raw = append(raw, content[j])
		}
		if bytes.HasPrefix(content[min(j+1, len(content)):], []byte(" Tj")) {
			out = append(out, decodeWinAnsi(raw))
		}
		i = j
	}
	return out
}

// unescapeLiteral quita el escape de una cadena literal ("\(" → "(").
func unescapeLiteral(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\\' && i+1 < len(b) {
			i++
		}
		out = append(out, b[i])
	}
	return out
}

// decodeWinAnsi convierte bytes WinAnsi a texto UTF-8.
func decodeWinAnsi(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if r, ok := winAnsiReverse[c]; ok {
			sb.WriteRune(r)
			continue
		}
		sb.WriteRune(rune(c))
	}
	return sb.String()
}
