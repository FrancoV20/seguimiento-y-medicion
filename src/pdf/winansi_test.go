package pdf

import (
	"bytes"
	"testing"
)

// HU-09, T005: codificación WinAnsi y escape de cadenas literales (research.md §3, data-model.md).

func TestPDF_Texto(t *testing.T) {
	t.Run("encodeWinAnsi", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			want []byte
		}{
			{"acento", "Página", []byte{0x50, 0xE1, 0x67, 0x69, 0x6E, 0x61}},
			{"eñe, raya y euro", "Año – 5 €", []byte{0x41, 0xF1, 0x6F, 0x20, 0x96, 0x20, 0x35, 0x20, 0x80}},
			{"comillas y puntos suspensivos", "“Hola” ‘x’ …",
				[]byte{0x93, 0x48, 0x6F, 0x6C, 0x61, 0x94, 0x20, 0x91, 0x78, 0x92, 0x20, 0x85}},
			{"comillas angulares y raya larga", "«Alfa» — ok",
				[]byte{0xAB, 0x41, 0x6C, 0x66, 0x61, 0xBB, 0x20, 0x97, 0x20, 0x6F, 0x6B}},
			{"tabulación y salto de línea", "a\tb\nc", []byte{0x61, 0x20, 0x62, 0x20, 0x63}},
			{"retorno de carro", "a\r\nb", []byte{0x61, 0x20, 0x20, 0x62}},
			{"emoji y otro alfabeto", "😀 Ω", []byte{0x3F, 0x20, 0x3F}},
			{"UTF-8 inválido", "a\xffb", []byte{0x61, 0x3F, 0x62}},
			{"control", "a\x01b", []byte{0x61, 0x3F, 0x62}},
			{"vacío", "", []byte{}},
		}
		for _, c := range cases {
			if got := encodeWinAnsi(c.in); !bytes.Equal(got, c.want) {
				t.Errorf("%s: encodeWinAnsi(%q) = % X, se esperaba % X", c.name, c.in, got, c.want)
			}
		}
	})

	t.Run("escapeLiteral y Literal", func(t *testing.T) {
		want := []byte(`Login \(v2\) C:\\x`)
		if got := escapeLiteral([]byte(`Login (v2) C:\x`)); !bytes.Equal(got, want) {
			t.Errorf("escapeLiteral = %q, se esperaba %q", got, want)
		}
		if got := Literal(`Login (v2) C:\x`); !bytes.Equal(got, want) {
			t.Errorf("Literal = %q, se esperaba %q", got, want)
		}
		if got, want := Literal("Año (1)"), []byte{0x41, 0xF1, 0x6F, 0x20, '\\', '(', '1', '\\', ')'}; !bytes.Equal(got, want) {
			t.Errorf("Literal(\"Año (1)\") = % X, se esperaba % X", got, want)
		}
	})
}
