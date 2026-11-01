package pdf

// Codificación WinAnsi (Windows-1252) de las fuentes estándar y escape de cadenas literales
// (HU-09, research.md §3).

// winAnsiExtras son los caracteres de Windows-1252 entre 0x80 y 0x9F, que no coinciden con Latin-1.
var winAnsiExtras = map[rune]byte{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87, 'ˆ': 0x88,
	'‰': 0x89, 'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E, '‘': 0x91, '’': 0x92, '“': 0x93,
	'”': 0x94, '•': 0x95, '–': 0x96, '—': 0x97, '˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B,
	'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F,
}

// winAnsiReverse convierte un byte WinAnsi del rango 0x80–0x9F a su runa (lo usa ExtractText).
var winAnsiReverse = func() map[byte]rune {
	m := make(map[byte]rune, len(winAnsiExtras))
	for r, b := range winAnsiExtras {
		m[b] = r
	}
	return m
}()

// encodeWinAnsi convierte un texto UTF-8 a bytes WinAnsi. Tabulación, retorno y salto de línea pasan
// a espacio; cualquier runa que las fuentes estándar no pueden mostrar (controles, emojis, otros
// alfabetos o UTF-8 inválido) se reemplaza por "?".
func encodeWinAnsi(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			out = append(out, ' ')
		case r >= 0x20 && r <= 0x7E, r >= 0xA0 && r <= 0xFF:
			out = append(out, byte(r))
		default:
			if b, ok := winAnsiExtras[r]; ok {
				out = append(out, b)
			} else {
				out = append(out, '?')
			}
		}
	}
	return out
}

// escapeLiteral escapa "(", ")" y "\" para escribir los bytes dentro de una cadena literal del PDF.
func escapeLiteral(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == '(' || c == ')' || c == '\\' {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return out
}

// Literal devuelve el texto tal como queda dentro de una cadena del PDF (codificado y escapado).
// Lo usan las pruebas para verificar los bytes de la codificación.
func Literal(s string) []byte {
	return escapeLiteral(encodeWinAnsi(s))
}
