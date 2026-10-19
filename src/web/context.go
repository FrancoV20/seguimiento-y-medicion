package web

import "math"

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