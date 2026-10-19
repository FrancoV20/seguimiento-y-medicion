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
