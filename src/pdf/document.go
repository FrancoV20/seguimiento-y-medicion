// Package pdf genera documentos PDF 1.4 simples (texto, tablas, saltos de página y pie) con la
// biblioteca estándar y las fuentes estándar Helvetica y Helvetica-Bold (HU-09, research.md §2).
// No conoce el dominio: lo usan los reportes de src/report.
package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Font es una de las dos fuentes estándar del documento.
type Font int

const (
	Regular Font = iota // Helvetica      (/F1)
	Bold                // Helvetica-Bold (/F2)
)

// Column es una columna de tabla: título y ancho en puntos.
type Column struct {
	Title string
	Width float64
}

var (
	ErrTableTooWide   = errors.New("pdf: table columns exceed the page width")
	ErrRowColumnCount = errors.New("pdf: row does not match the table columns")
)

// Medidas de la página A4 vertical (research.md §6).
const (
	pageWidth    = 595.28
	pageHeight   = 841.89
	marginLeft   = 50.0
	marginTop    = 50.0
	marginBottom = 60.0
	contentWidth = 495.0
	lineFactor   = 1.25 // interlineado = 1,25 × tamaño
	tableSize    = 9.0
	cellPadding  = 2.0 // relleno de cada lado de la celda (4 pt en total)
	footerSize   = 9.0
	footerY      = 30.0
)

// Document acumula las operaciones de dibujo de cada página. Bytes recién al final conoce el total
// de páginas, agrega el pie y serializa (research.md §5).
type Document struct {
	pages []*bytes.Buffer
	y     float64 // borde superior del próximo renglón en la página actual
	err   error
}

// New crea un documento A4 vertical con una página vacía.
func New() *Document {
	d := &Document{}
	d.newPage()
	return d
}

// PageCount devuelve la cantidad de páginas del documento.
func (d *Document) PageCount() int { return len(d.pages) }

func (d *Document) newPage() {
	d.pages = append(d.pages, &bytes.Buffer{})
	d.y = pageHeight - marginTop
}

func (d *Document) current() *bytes.Buffer { return d.pages[len(d.pages)-1] }

// fits indica si entra un bloque de height puntos en la página actual.
func (d *Document) fits(height float64) bool { return d.y-height >= marginBottom-0.001 }

// Text agrega un párrafo cortado al ancho útil; si una línea no entra, abre una página nueva.
func (d *Document) Text(font Font, size float64, text string) {
	lineHeight := size * lineFactor
	for _, line := range wrapText(font, size, contentWidth, text) {
		if !d.fits(lineHeight) {
			d.newPage()
		}
		d.drawText(font, size, marginLeft, d.y-size, line)
		d.y -= lineHeight
	}
}

// Space agrega un espacio vertical; si no entra, el próximo contenido empieza en otra página.
func (d *Document) Space(points float64) {
	if !d.fits(points) {
		d.newPage()
		return
	}
	d.y -= points
}

// Table dibuja una tabla con el encabezado en Bold 9 y las filas en Regular 9. Cada celda se corta
// al ancho de su columna; si una fila no entra, sigue en otra página y repite el encabezado. Las
// columnas que no entran en el ancho útil o una fila con otra cantidad de celdas son un error, que
// también queda guardado para Bytes.
func (d *Document) Table(columns []Column, rows [][]string) error {
	if d.err != nil {
		return d.err
	}
	total := 0.0
	for _, c := range columns {
		total += c.Width
	}
	if total > contentWidth+0.001 {
		d.err = fmt.Errorf("%w: %.2f pt", ErrTableTooWide, total)
		return d.err
	}
	for i, row := range rows {
		if len(row) != len(columns) {
			d.err = fmt.Errorf("%w: fila %d con %d celdas para %d columnas", ErrRowColumnCount, i+1, len(row), len(columns))
			return d.err
		}
	}

	header := make([]string, len(columns))
	for i, c := range columns {
		header[i] = c.Title
	}
	headerCells := wrapCells(Bold, columns, header)
	headerHeight := cellsHeight(headerCells)

	firstHeight := 0.0
	if len(rows) > 0 {
		firstHeight = cellsHeight(wrapCells(Regular, columns, rows[0]))
	}
	if !d.fits(headerHeight + firstHeight) {
		d.newPage()
	}
	d.drawHeader(columns, headerCells, headerHeight)

	lineHeight := tableSize * lineFactor
	for _, row := range rows {
		cells := wrapCells(Regular, columns, row)
		height := cellsHeight(cells)
		if !d.fits(height) {
			if height <= pageHeight-marginTop-marginBottom-headerHeight {
				d.newPage()
				d.drawHeader(columns, headerCells, headerHeight)
			} else {
				d.drawTallRow(columns, cells, headerCells, headerHeight, lineHeight)
				continue
			}
		}
		d.drawRow(columns, cells, 0, maxLines(cells))
		d.y -= height
		d.drawRule(tableWidth(columns))
	}
	return nil
}

// wrapCells corta el texto de cada celda al ancho de su columna menos el relleno.
func wrapCells(font Font, columns []Column, row []string) [][][]byte {
	cells := make([][][]byte, len(columns))
	for i, c := range columns {
		cells[i] = wrapText(font, tableSize, c.Width-2*cellPadding, row[i])
	}
	return cells
}

func maxLines(cells [][][]byte) int {
	n := 1
	for _, c := range cells {
		if len(c) > n {
			n = len(c)
		}
	}
	return n
}

func cellsHeight(cells [][][]byte) float64 {
	return float64(maxLines(cells))*tableSize*lineFactor + 2*cellPadding
}

// drawHeader dibuja el encabezado con fondo gris y avanza.
func (d *Document) drawHeader(columns []Column, cells [][][]byte, height float64) {
	fmt.Fprintf(d.current(), "0.9 g %s %s %s %s re f 0 g\n",
		num(marginLeft), num(d.y-height), num(tableWidth(columns)), num(height))
	x := marginLeft
	lineHeight := tableSize * lineFactor
	for i, c := range columns {
		for k, line := range cells[i] {
			d.drawText(Bold, tableSize, x+cellPadding, d.y-cellPadding-tableSize-float64(k)*lineHeight, line)
		}
		x += c.Width
	}
	d.y -= height
}

// drawRow dibuja las líneas [from, to) de cada celda, celda por celda, desde el borde actual.
func (d *Document) drawRow(columns []Column, cells [][][]byte, from, to int) {
	lineHeight := tableSize * lineFactor
	x := marginLeft
	for i, c := range columns {
		for k := from; k < to && k < len(cells[i]); k++ {
			d.drawText(Regular, tableSize, x+cellPadding, d.y-cellPadding-tableSize-float64(k-from)*lineHeight, cells[i][k])
		}
		x += c.Width
	}
}

// drawTallRow corta entre páginas, línea por línea, una fila más alta que una página entera.
func (d *Document) drawTallRow(columns []Column, cells, headerCells [][][]byte, headerHeight, lineHeight float64) {
	total := maxLines(cells)
	for k := 0; k < total; {
		available := int((d.y - marginBottom - 2*cellPadding) / lineHeight)
		if available <= 0 {
			d.newPage()
			d.drawHeader(columns, headerCells, headerHeight)
			continue
		}
		end := min(k+available, total)
		d.drawRow(columns, cells, k, end)
		d.y -= float64(end-k)*lineHeight + 2*cellPadding
		k = end
		if k < total {
			d.newPage()
			d.drawHeader(columns, headerCells, headerHeight)
		}
	}
	d.drawRule(tableWidth(columns))
}

// drawRule dibuja una línea fina gris en el borde actual, a lo ancho de la tabla.
func (d *Document) drawRule(width float64) {
	fmt.Fprintf(d.current(), "0.8 G 0.5 w %s %s m %s %s l S 0 G\n",
		num(marginLeft), num(d.y), num(marginLeft+width), num(d.y))
}

func tableWidth(columns []Column) float64 {
	total := 0.0
	for _, c := range columns {
		total += c.Width
	}
	return total
}

// drawText escribe una línea ya codificada con su línea base en (x, y).
func (d *Document) drawText(font Font, size, x, y float64, encoded []byte) {
	writeText(d.current(), font, size, x, y, encoded)
}

func writeText(buf *bytes.Buffer, font Font, size, x, y float64, encoded []byte) {
	fmt.Fprintf(buf, "BT /F%d %s Tf %s %s Td (", int(font)+1, num(size), num(x), num(y))
	buf.Write(escapeLiteral(encoded))
	buf.WriteString(") Tj ET\n")
}

// Bytes agrega en cada página el pie footer(página, total) centrado y devuelve el PDF serializado.
// Devuelve un error si una operación previa dejó un error guardado.
func (d *Document) Bytes(footer func(page, total int) string) ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	contents := make([][]byte, len(d.pages))
	for i, page := range d.pages {
		var buf bytes.Buffer
		buf.Write(page.Bytes())
		if footer != nil {
			text := encodeWinAnsi(footer(i+1, len(d.pages)))
			x := (pageWidth - textWidth(Regular, footerSize, text)) / 2
			writeText(&buf, Regular, footerSize, x, footerY, text)
		}
		contents[i] = buf.Bytes()
	}
	return serialize(contents), nil
}

// num escribe un número con hasta dos decimales y sin ceros sobrantes ("50", "791.89").
func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}
