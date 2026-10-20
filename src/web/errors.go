package web

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrorKind clasifica un fallo para elegir la respuesta HTTP (HU-20a, T006).
type ErrorKind int

const (
	// KindInternal corresponde a un fallo inesperado (500).
	KindInternal ErrorKind = iota
	// KindNotFound corresponde a un recurso o dirección inexistente (404).
	KindNotFound
	// KindUnavailable corresponde a la base de datos no disponible (503).
	KindUnavailable
)

// AppError envuelve un error con su clasificación. El detalle técnico (Err)
// solo se registra en el servidor; nunca se muestra al usuario.
type AppError struct {
	Kind ErrorKind
	Err  error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return "error de aplicación"
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error { return e.Err }

// NotFound clasifica err como "no encontrado".
func NotFound(err error) error { return &AppError{Kind: KindNotFound, Err: err} }

// Unavailable clasifica err como "servicio no disponible".
func Unavailable(err error) error { return &AppError{Kind: KindUnavailable, Err: err} }

// Internal clasifica err como fallo inesperado.
func Internal(err error) error { return &AppError{Kind: KindInternal, Err: err} }

// KindOf devuelve la clasificación de err; un error sin clasificar es interno.
func KindOf(err error) ErrorKind {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Kind
	}
	return KindInternal
}

// IsDatabaseUnavailable indica si err se debe a que la base de datos no responde
// (conexión rechazada o perdida, tiempo agotado), y no a un error de los datos (T040).
func IsDatabaseUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, sql.ErrConnDone) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Clase 08: excepción de conexión; 57P: el servidor se está apagando o reiniciando.
		return strings.HasPrefix(pgErr.Code, "08") || strings.HasPrefix(pgErr.Code, "57P")
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// PathID lee un identificador numérico positivo de la dirección. Un valor no
// numérico, negativo, cero o demasiado grande se trata como "no encontrado".
func PathID(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, NotFound(fmt.Errorf("identificador %q no válido en %s", raw, r.URL.Path))
	}
	return id, nil
}

// errorTemplate es la plantilla de las páginas de error.
const errorTemplate = "error.html"

// ErrorPageData es el contenido de una página de error.
type ErrorPageData struct {
	Message string
}

// RenderError muestra una página de error con la estructura común y un enlace
// al inicio, para cualquier código de estado (HU-20b la usa para el 403) (T038).
func (rd *Renderer) RenderError(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	rd.PageStatus(w, r, status, errorTemplate, PageData{Title: title, Data: ErrorPageData{Message: message}})
}

// Fail traduce un error de aplicación a la página que corresponde: 404 si no se
// encontró, 503 si la base no responde y 500 genérico en cualquier otro caso.
// El detalle técnico solo va al registro del servidor (T038).
func (rd *Renderer) Fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case KindOf(err) == KindNotFound:
		rd.logger.Info("recurso no encontrado", "metodo", r.Method, "ruta", r.URL.Path, "detalle", errText(err))
		rd.RenderError(w, r, http.StatusNotFound, TitleNotFound, MsgNotFound)
	case KindOf(err) == KindUnavailable || IsDatabaseUnavailable(err):
		rd.logger.Warn("la base de datos no está disponible", "metodo", r.Method, "ruta", r.URL.Path, "detalle", errText(err))
		rd.RenderError(w, r, http.StatusServiceUnavailable, TitleUnavailable, MsgUnavailable)
	default:
		rd.logger.Error("error inesperado", "metodo", r.Method, "ruta", r.URL.Path, "detalle", errText(err))
		rd.RenderError(w, r, http.StatusInternalServerError, TitleInternal, MsgInternal)
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
