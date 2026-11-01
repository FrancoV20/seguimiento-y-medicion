package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// HU-09, T006: estructura del PDF, tablas, saltos de página y pie (research.md §5 y §6).

func footer(page, total int) string { return fmt.Sprintf("Página %d de %d", page, total) }

func sampleDocument(t *testing.T) *Document {
	t.Helper()
	d := New()
	d.Text(Bold, 16, "Reporte del Sprint")
	d.Text(Regular, 10, "Login (v2) – Año")
	d.Space(6)
	err := d.Table(
		[]Column{{Title: "Historia", Width: 165}, {Title: "Estado", Width: 135}},
		[][]string{
			{"Login", "Hecho"},
			{"Reportes", "En progreso – volvió al Backlog con un texto largo que no entra en la columna"},
		},
	)
	if err != nil {
		t.Fatalf("Table: %v", err)
	}
	return d
}

func TestPDF_Estructura(t *testing.T) {
	d := sampleDocument(t)
	b, err := d.Bytes(footer)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}

	if !bytes.HasPrefix(b, []byte("%PDF-1.4\n")) {
		t.Errorf("el PDF debe empezar con %%PDF-1.4, empieza con %q", b[:min(len(b), 12)])
	}
	if !bytes.HasSuffix(b, []byte("%%EOF\n")) {
		t.Errorf("el PDF debe terminar con %%%%EOF")
	}
	for _, want := range []string{
		"/MediaBox [0 0 595.28 841.89]",
		"/BaseFont /Helvetica /Encoding /WinAnsiEncoding",
		"/BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding",
		fmt.Sprintf("/Count %d", d.PageCount()),
	} {
		if !bytes.Contains(b, []byte(want)) {
			t.Errorf("falta %q en el PDF", want)
		}
	}

	p := parsePDF(t, b)
	if p.pages != d.PageCount() || p.pages != 1 {
		t.Errorf("se esperaba 1 página (PageCount %d), el PDF tiene %d", d.PageCount(), p.pages)
	}

	again, err := sampleDocument(t).Bytes(footer)
	if err != nil || !bytes.Equal(b, again) {
		t.Error("el mismo documento debe dar exactamente los mismos bytes")
	}

	text, err := ExtractText(b)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	for _, want := range []string{
		"Reporte del Sprint",
		"Login (v2) – Año",
		"Reportes En progreso – volvió al Backlog con un texto largo que no entra en la columna",
		"Página 1 de 1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("ExtractText no contiene %q:\n%s", want, text)
		}
	}
	if !bytes.Contains(b, Literal("Login (v2) – Año")) {
		t.Error("el texto debe escribirse codificado en WinAnsi y escapado")
	}

	t.Run("errores de tabla", func(t *testing.T) {
		wide := New()
		err := wide.Table([]Column{{"A", 300}, {"B", 196}}, [][]string{{"a", "b"}})
		if !errors.Is(err, ErrTableTooWide) {
			t.Errorf("columnas de 496 pt: se esperaba ErrTableTooWide, se obtuvo %v", err)
		}
		if _, err := wide.Bytes(footer); !errors.Is(err, ErrTableTooWide) {
			t.Errorf("Bytes debe devolver el error guardado, se obtuvo %v", err)
		}

		short := New()
		err = short.Table([]Column{{"A", 100}, {"B", 100}}, [][]string{{"a", "b"}, {"c"}})
		if !errors.Is(err, ErrRowColumnCount) {
			t.Errorf("fila de 1 celda: se esperaba ErrRowColumnCount, se obtuvo %v", err)
		}
		if _, err := short.Bytes(footer); !errors.Is(err, ErrRowColumnCount) {
			t.Errorf("Bytes debe devolver el error guardado, se obtuvo %v", err)
		}
	})

	t.Run("ExtractText sin xref", func(t *testing.T) {
		if _, err := ExtractText([]byte("%PDF-1.4\nnada")); err == nil {
			t.Error("se esperaba un error sin xref ni flujos")
		}
	})
}

func TestPDF_SaltoDePagina(t *testing.T) {
	columns := []Column{{"Historia", 165}, {"Estado", 135}, {"Story Points", 55}, {"Horas estimadas", 70}, {"Horas reales", 70}}
	rows := make([][]string, 0, 120)
	for i := 1; i <= 120; i++ {
		title := fmt.Sprintf("Historia %d", i)
		if i == 60 {
			title = strings.Repeat("Titulo muy largo ", 18)[:299] + "x" // 300 caracteres
		}
		rows = append(rows, []string{title, "Hecho", "2", "4", "4"})
	}
	d := New()
	d.Text(Bold, 16, "Reporte del Sprint")
	if err := d.Table(columns, rows); err != nil {
		t.Fatalf("Table: %v", err)
	}
	b, err := d.Bytes(footer)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	p := parsePDF(t, b)
	if p.pages < 2 || p.pages != d.PageCount() {
		t.Fatalf("se esperaban varias páginas (PageCount %d), el PDF tiene %d", d.PageCount(), p.pages)
	}

	const right = 595.28 - 50
	for i, content := range p.contents {
		ops := textOps(t, content)
		page := i + 1
		var hasHeader, hasRow, hasFooter bool
		for _, op := range ops {
			text := string(op.raw)
			switch {
			case op.font == Bold && text == "Historia":
				hasHeader = true
			case strings.HasPrefix(text, "Historia ") || strings.HasPrefix(text, "T"):
				hasRow = true
			}
			if text == string(encodeWinAnsi(fmt.Sprintf("Página %d de %d", page, p.pages))) {
				hasFooter = true
				continue
			}
			if op.y < 60-0.001 {
				t.Errorf("página %d: la línea %q está en y=%.2f, debajo del margen inferior", page, text, op.y)
			}
			if w := textWidth(op.font, op.size, op.raw); op.x+w > right+0.01 {
				t.Errorf("página %d: la línea %q termina en %.2f, más allá del margen derecho", page, text, op.x+w)
			}
		}
		if hasRow && !hasHeader {
			t.Errorf("página %d: tiene filas pero no repite el encabezado de la tabla", page)
		}
		if !hasFooter {
			t.Errorf("página %d: falta el pie \"Página %d de %d\"", page, page, p.pages)
		}
	}

	text, err := ExtractText(b)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	for _, want := range []string{"Historia 1 Hecho", "Historia 120 Hecho", rows[59][0]} {
		if !strings.Contains(text, want) {
			t.Errorf("ExtractText no contiene %q", want)
		}
	}
}
