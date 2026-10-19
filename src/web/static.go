package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFiles embed.FS

// staticHandler sirve /static/ desde los archivos embebidos, sin listar carpetas (HU-20a, T030).
func staticHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // la carpeta está embebida: solo falla por un error de programación
	}
	files := http.StripPrefix("/static/", http.FileServerFS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}
