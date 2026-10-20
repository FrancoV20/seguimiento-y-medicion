package web

import (
	"fmt"
	"strings"
)

// Section es un apartado del menú. Cada historia registra la suya cuando se
// implementa; RequiresProject y RequiresMember los usa HU-20b (HU-20a, T007).
type Section struct {
	Name            string
	Path            string
	RequiresProject bool
	RequiresMember  bool
}

// DuplicateSectionError indica que dos secciones usan la misma dirección.
type DuplicateSectionError struct{ Path string }

func (e *DuplicateSectionError) Error() string {
	return fmt.Sprintf("la dirección %q ya está registrada por otra sección", e.Path)
}

// Registry guarda las secciones registradas en el orden en que se agregan.
type Registry struct {
	sections []Section
	paths    map[string]bool
}

// NewRegistry crea un registro vacío.
func NewRegistry() *Registry {
	return &Registry{paths: map[string]bool{}}
}

// Add registra una sección; rechaza nombre o dirección vacíos y direcciones duplicadas.
func (r *Registry) Add(s Section) error {
	s.Name = strings.TrimSpace(s.Name)
	s.Path = strings.TrimSpace(s.Path)
	if s.Name == "" || s.Path == "" || !strings.HasPrefix(s.Path, "/") {
		return fmt.Errorf("la sección necesita un nombre y una dirección que empiece con /")
	}
	if r.paths[s.Path] {
		return &DuplicateSectionError{Path: s.Path}
	}
	r.paths[s.Path] = true
	r.sections = append(r.sections, s)
	return nil
}

// Sections devuelve una copia de las secciones registradas.
func (r *Registry) Sections() []Section {
	return append([]Section(nil), r.sections...)
}

// HomeName y HomePath identifican la página de inicio, que siempre encabeza el menú.
const (
	HomeName = "Inicio"
	HomePath = "/"
)

// Menu arma el menú para la dirección actual: Inicio primero y después las
// secciones registradas, con la sección actual resaltada (HU-20a, T028).
func (r *Registry) Menu(currentPath string) []MenuItem {
	items := make([]MenuItem, 0, len(r.sections)+1)
	items = append(items, MenuItem{Name: HomeName, Path: HomePath, Active: currentPath == HomePath})
	for _, s := range r.sections {
		active := currentPath == s.Path || strings.HasPrefix(currentPath, strings.TrimSuffix(s.Path, "/")+"/")
		items = append(items, MenuItem{Name: s.Name, Path: s.Path, Active: active})
	}
	return items
}

// remove quita una sección (se usa si el registro de sus rutas falla).
func (r *Registry) remove(path string) {
	delete(r.paths, path)
	for i, s := range r.sections {
		if s.Path == path {
			r.sections = append(r.sections[:i], r.sections[i+1:]...)
			return
		}
	}
}
