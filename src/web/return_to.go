package web

import "strings"

// SanitizeReturnTo devuelve raw si es un destino de regreso seguro; si no, "/".
// Es seguro si empieza con una sola "/", no contiene "//", "\", espacios ni caracteres de
// control, y su primer segmento coincide con el de una sección registrada (HU-20b, T051).
func SanitizeReturnTo(raw string, sections []Section) string {
	const home = "/"
	if raw == "" || raw[0] != '/' || strings.Contains(raw, "//") || strings.Contains(raw, `\`) {
		return home
	}
	for _, r := range raw {
		if r <= ' ' || r == 0x7f {
			return home
		}
	}
	seg := firstSegment(raw)
	if seg == "" {
		return home
	}
	for _, s := range sections {
		if firstSegment(s.Path) == seg {
			return raw
		}
	}
	return home
}

// firstSegment devuelve el primer segmento de una ruta ("/backlog/12?x=1" -> "backlog").
func firstSegment(p string) string {
	p = strings.TrimPrefix(p, "/")
	if i := strings.IndexAny(p, "/?#"); i >= 0 {
		p = p[:i]
	}
	return p
}