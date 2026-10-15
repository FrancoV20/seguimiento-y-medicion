package project

// ProjectStore define el contrato de persistencia para los proyectos.
type ProjectStore interface {
	Create(p *Project) error
	MemberExists(id int) (bool, error)
}
