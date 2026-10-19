package web

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

// Cookies de contexto (HU-20b, T007).
const (
	CookieProject = "ctx_proyecto"
	CookieMember  = "ctx_integrante"
)

// MaxID es el mayor identificador posible: las columnas id son SERIAL (INTEGER de 32 bits).
const MaxID = math.MaxInt32

// validID indica si id puede existir en la base (1..MaxID). Un valor fuera de rango se
// trata como ausencia de selección sin consultar, para no confundirlo con una caída.
func validID(id int64) bool { return id >= 1 && id <= MaxID }

// ProjectSummary es un elemento de la lista de selección de proyectos.
type ProjectSummary struct {
	ID   int64
	Name string
}

// MemberSummary es un elemento del selector de integrante.
type MemberSummary struct {
	ID   int64
	Name string
}

// ResolvedContext es el resultado de validar las cookies contra la base.
type ResolvedContext struct {
	ProjectExists bool
	ProjectName   string
	MemberBelongs bool
	MemberName    string
}

// readIDCookie lee un identificador de una cookie de contexto. Devuelve present=true si la
// cookie vino en la solicitud; el id es 0 si el valor no es un entero canónico entre 1 y MaxID
// (vacío, no numérico, negativo, muy grande, con signo o con ceros a la izquierda). El
// middleware usa present para borrar las cookies manipuladas (HU-20b, T011).
func readIDCookie(r *http.Request, name string) (id int64, present bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseInt(c.Value, 10, 64)
	if err != nil || !validID(n) || strconv.FormatInt(n, 10) != c.Value {
		return 0, true
	}
	return n, true
}

// requestIsHTTPS indica si la solicitud llegó por HTTPS.
func requestIsHTTPS(r *http.Request) bool { return r.TLS != nil }

// newContextCookie arma una cookie de sesión: sin Expires, HttpOnly, SameSite=Lax,
// Path=/ y Secure solo si la solicitud llegó por HTTPS.
func newContextCookie(r *http.Request, name, value string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r),
	}
}

// setContextCookie guarda el identificador en una cookie de sesión.
func setContextCookie(w http.ResponseWriter, r *http.Request, name string, id int64) {
	http.SetCookie(w, newContextCookie(r, name, strconv.FormatInt(id, 10)))
}

// clearContextCookie pide al navegador que borre la cookie.
func clearContextCookie(w http.ResponseWriter, r *http.Request, name string) {
	c := newContextCookie(r, name, "")
	c.MaxAge = -1
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, c)
}