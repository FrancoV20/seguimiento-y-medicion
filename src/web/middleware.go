package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// recoverPanics atrapa los pánicos de cualquier handler o middleware: registra
// el detalle con la traza, responde la página 500 genérica y deja el servidor
// en pie. Siempre es el primer eslabón de la cadena (HU-20a, T039).
func recoverPanics(logger *slog.Logger, rd *Renderer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tw := &trackingWriter{ResponseWriter: w}
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec) // corte intencional de la conexión: lo maneja net/http
			}
			logger.Error("pánico al procesar la solicitud",
				"metodo", r.Method, "ruta", r.URL.Path, "panico", fmt.Sprint(rec), "traza", string(debug.Stack()))
			if !tw.wrote {
				rd.RenderError(w, r, http.StatusInternalServerError, TitleInternal, MsgInternal)
			}
		}()
		next.ServeHTTP(tw, r)
	})
}

// trackingWriter registra si ya se empezó a escribir la respuesta.
type trackingWriter struct {
	http.ResponseWriter
	wrote bool
}

func (t *trackingWriter) WriteHeader(code int) {
	t.wrote = true
	t.ResponseWriter.WriteHeader(code)
}

func (t *trackingWriter) Write(b []byte) (int, error) {
	t.wrote = true
	return t.ResponseWriter.Write(b)
}

// Unwrap permite que http.ResponseController llegue al ResponseWriter original.
func (t *trackingWriter) Unwrap() http.ResponseWriter { return t.ResponseWriter }
