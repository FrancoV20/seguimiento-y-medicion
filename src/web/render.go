package web

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"sync"
)

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
// Renderer.Page completa Menu, Section y Context; HU-20b agrega Flash y Form.
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

//go:embed templates/*.html
var templateFiles embed.FS

// AppName es el nombre que se muestra en el encabezado y en el título de la pestaña.
const AppName = "Software Metrics & Estimation"

// Renderer arma las páginas con el layout común (HU-20a, T028).
type Renderer struct {
	registry *Registry
	logger   *slog.Logger
	files    fs.FS // plantillas; las pruebas pueden reemplazarlas

	mu    sync.Mutex
	pages map[string]*template.Template
}

// NewRenderer crea el renderizador; el menú se arma con registry.
func NewRenderer(registry *Registry, logger *slog.Logger) *Renderer {
	return &Renderer{registry: registry, logger: logger, files: templateFiles, pages: map[string]*template.Template{}}
}

var templateFuncs = template.FuncMap{
	"appName":   func() string { return AppName },
	"noProject": func() string { return MsgNoProjectSelected },
	"backHome":  func() string { return LinkBackHome },
}

// page devuelve layout.html + la plantilla pedida, parseadas una sola vez.
func (rd *Renderer) page(name string) (*template.Template, error) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if t, ok := rd.pages[name]; ok {
		return t, nil
	}
	t, err := template.New("layout.html").Funcs(templateFuncs).ParseFS(rd.files, "templates/layout.html", path.Join("templates", name))
	if err != nil {
		return nil, fmt.Errorf("plantilla %q: %w", name, err)
	}
	rd.pages[name] = t
	return t, nil
}

// Page renderiza la plantilla name dentro del layout común. Completa el menú,
// la sección actual y el contexto activo. Renderiza a un buffer y solo envía
// la página si terminó sin error, para no mandar HTML a medias.
func (rd *Renderer) Page(w http.ResponseWriter, r *http.Request, name string, data PageData) {
	rd.PageStatus(w, r, http.StatusOK, name, data)
}

// PageStatus es como Page pero con un código de estado HTTP distinto de 200.
func (rd *Renderer) PageStatus(w http.ResponseWriter, r *http.Request, status int, name string, data PageData) {
	data.Menu = rd.registry.Menu(r.URL.Path)
	if data.Section == "" {
		for _, it := range data.Menu {
			if it.Active {
				data.Section = it.Name
			}
		}
	}
	if data.Context == nil {
		data.Context = ActiveContextFrom(r.Context())
	}

	var buf bytes.Buffer
	t, err := rd.page(name)
	if err == nil {
		err = t.ExecuteTemplate(&buf, "layout", data)
	}
	if err != nil {
		rd.logger.Error("no se pudo construir la página", "plantilla", name, "ruta", r.URL.Path, "detalle", err.Error())
		if name != errorTemplate {
			rd.RenderError(w, r, http.StatusInternalServerError, TitleInternal, MsgInternal)
			return
		}
		// Si falla la propia página de error, queda un texto plano genérico.
		http.Error(w, MsgInternal, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
