package pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"testing"
)

// HU-09, T006: lector mínimo de PDF para las pruebas, sin bibliotecas externas. Verifica la tabla
// xref, los desplazamientos de cada objeto, startxref y los /Length de los flujos, y devuelve la
// cantidad de páginas y el contenido de cada una.

type parsedPDF struct {
	pages    int
	contents [][]byte // flujo de contenido de cada página, en orden
}

var (
	startxrefRe = regexp.MustCompile(`startxref\n(\d+)\n%%EOF\n?$`)
	xrefHeadRe  = regexp.MustCompile(`^xref\n0 (\d+)\n`)
	objHeadRe   = regexp.MustCompile(`^(\d+) 0 obj\n`)
	lengthRe    = regexp.MustCompile(`^<< /Length (\d+) >>\nstream\n`)
)

func parsePDF(t *testing.T, b []byte) parsedPDF {
	t.Helper()
	p, err := checkPDF(b)
	if err != nil {
		t.Fatalf("PDF inválido: %v", err)
	}
	return p
}

func checkPDF(b []byte) (parsedPDF, error) {
	var p parsedPDF
	m := startxrefRe.FindSubmatch(b)
	if m == nil {
		return p, fmt.Errorf("falta startxref al final")
	}
	xrefAt, _ := strconv.Atoi(string(m[1]))
	if xrefAt <= 0 || xrefAt >= len(b) {
		return p, fmt.Errorf("startxref %d fuera del archivo", xrefAt)
	}
	head := xrefHeadRe.FindSubmatch(b[xrefAt:])
	if head == nil {
		return p, fmt.Errorf("startxref no apunta a la tabla xref")
	}
	count, _ := strconv.Atoi(string(head[1]))
	entries := b[xrefAt+len(head[0]):]
	if len(entries) < count*20 {
		return p, fmt.Errorf("la tabla xref está incompleta")
	}
	if string(entries[:20]) != "0000000000 65535 f \n" {
		return p, fmt.Errorf("la entrada 0 de la xref es %q", entries[:20])
	}
	for n := 1; n < count; n++ {
		entry := entries[n*20 : n*20+20]
		if !regexp.MustCompile(`^\d{10} 00000 n \n$`).Match(entry) {
			return p, fmt.Errorf("entrada %d de la xref mal formada: %q", n, entry)
		}
		offset, _ := strconv.Atoi(string(entry[:10]))
		obj := objHeadRe.FindSubmatch(b[offset:])
		if obj == nil || string(obj[1]) != strconv.Itoa(n) {
			return p, fmt.Errorf("la entrada %d apunta a %d, donde no empieza el objeto %d", n, offset, n)
		}
		body := b[offset+len(obj[0]):]
		if bytes.HasPrefix(body, []byte("<< /Type /Page ")) {
			p.pages++
		}
		if l := lengthRe.FindSubmatch(body); l != nil {
			length, _ := strconv.Atoi(string(l[1]))
			start := len(l[0])
			if start+length > len(body) || !bytes.HasPrefix(body[start+length:], []byte("\nendstream\nendobj\n")) {
				return p, fmt.Errorf("el /Length %d del objeto %d no coincide con su flujo", length, n)
			}
			p.contents = append(p.contents, body[start:start+length])
		}
	}
	if p.pages != len(p.contents) {
		return p, fmt.Errorf("%d páginas y %d flujos de contenido", p.pages, len(p.contents))
	}
	return p, nil
}

// textOp es una línea de texto dibujada: fuente, tamaño, posición y texto ya decodificado.
type textOp struct {
	font Font
	size float64
	x, y float64
	raw  []byte // bytes WinAnsi sin escapar
}

var textOpRe = regexp.MustCompile(`BT /F([12]) ([\d.]+) Tf ([\d.]+) ([\d.]+) Td \(((?:[^\\)]|\\.)*)\) Tj ET`)

// textOps devuelve las líneas de texto de un flujo de contenido.
func textOps(t *testing.T, content []byte) []textOp {
	t.Helper()
	var ops []textOp
	for _, m := range textOpRe.FindAllSubmatch(content, -1) {
		font := Regular
		if string(m[1]) == "2" {
			font = Bold
		}
		size, _ := strconv.ParseFloat(string(m[2]), 64)
		x, _ := strconv.ParseFloat(string(m[3]), 64)
		y, _ := strconv.ParseFloat(string(m[4]), 64)
		ops = append(ops, textOp{font: font, size: size, x: x, y: y, raw: unescapeLiteral(m[5])})
	}
	return ops
}
