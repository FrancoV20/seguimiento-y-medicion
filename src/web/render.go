package web

import "context"

// ActiveContext es el proyecto e integrante activos. HU-20a define el tipo para
// que el layout compile; HU-20b lo completa desde un middleware (T008).
type ActiveContext struct {
	ProjectID   int64
	ProjectName string
	MemberID    int64
	MemberName  string
}

// MenuItem es una entrada del menú ya resuelta para una página.
type MenuItem struct {
	Name   string
	Path   string
	Active bool
}

// PageData son los datos comunes que recibe cada plantilla.
// HU-20b agrega Flash y Form.
type PageData struct {
	Title   string
	Section string
	Menu    []MenuItem
	Context *ActiveContext
	Data    any
}

type activeContextKey struct{}

// WithActiveContext devuelve un contexto que lleva ac.
func WithActiveContext(ctx context.Context, ac *ActiveContext) context.Context {
	return context.WithValue(ctx, activeContextKey{}, ac)
}

// ActiveContextFrom devuelve el ActiveContext de ctx, o nil si no hay.
func ActiveContextFrom(ctx context.Context) *ActiveContext {
	ac, _ := ctx.Value(activeContextKey{}).(*ActiveContext)
	return ac
}
