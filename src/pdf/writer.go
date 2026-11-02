package pdf

import (
	"bytes"
	"fmt"
)

// serialize escribe el PDF 1.4 (HU-09, research.md §5): encabezado con línea binaria, catálogo,
// árbol de páginas, las dos fuentes, una página y un flujo de contenido por página, la tabla xref con
// los desplazamientos exactos, el trailer, startxref y %%EOF. No escribe fechas: la salida es
// determinista.
func serialize(contents [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	total := 4 + 2*len(contents) // objetos 1..total
	offsets := make([]int, total+1)
	object := func(n int, body string) {
		offsets[n] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}

	object(1, "<< /Type /Catalog /Pages 2 0 R >>")
	var kids bytes.Buffer
	for i := range contents {
		if i > 0 {
			kids.WriteByte(' ')
		}
		fmt.Fprintf(&kids, "%d 0 R", 5+2*i)
	}
	object(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids.String(), len(contents)))
	object(3, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	object(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	for i, content := range contents {
		page, stream := 5+2*i, 6+2*i
		object(page, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595.28 841.89] "+
			"/Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>", stream))

		offsets[stream] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n<< /Length %d >>\nstream\n", stream, len(content))
		buf.Write(content)
		buf.WriteString("\nendstream\nendobj\n")
	}

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", total+1)
	buf.WriteString("0000000000 65535 f \n")
	for n := 1; n <= total; n++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[n])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", total+1, xref)
	return buf.Bytes()
}
