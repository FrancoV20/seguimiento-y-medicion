package web

// MEJORA-01, Escenario 7 (specs/mejora-01-diseno/inicio.md): datos de la pantalla de Inicio. Las
// secciones del menú se agrupan en Planificar, Ejecutar y Medir, cada una con una descripción corta y
// el nombre de su ícono. Los SVG están en templates/home.html ("home-icon"): el código no marca HTML
// como seguro (regla de TestEscape_NingunCodigoMarcaDatosComoSeguros).

// homeItem es una sección del Inicio.
type homeItem struct {
	Name        string
	Path        string
	Description string
	Icon        string // nombre del ícono de templates/home.html
}

// homeGroup es un grupo de secciones del Inicio.
type homeGroup struct {
	Title    string
	Subtitle string
	Items    []homeItem
}

type homeSectionInfo struct {
	path, description, icon string
}

// homeLayout define los grupos y el orden de sus secciones.
var homeLayout = []struct {
	title, subtitle string
	sections        []homeSectionInfo
}{
	{"Planificar", "Antes y al comenzar cada Sprint", []homeSectionInfo{
		{"/proyectos", "Crear y modificar proyectos", "carpeta"},
		{"/backlog", "Historias, prioridades y criterios", "lista"},
		{"/estimaciones", "Horas estimadas por historia", "reloj"},
		{"/planning-poker", "Votar Story Points en equipo", "cartas"},
		{"/sprints", "Crear el Sprint y asignar historias", "ciclo"},
	}},
	{"Ejecutar", "Durante el Sprint", []homeSectionInfo{
		{"/tablero", "En Sprint, En progreso y Hecho", "columnas"},
		{"/esfuerzo", "Horas trabajadas por integrante", "lapiz"},
		{"/defectos", "Registrar y resolver defectos", "alerta"},
		{"/cierre", "Cerrar el Sprint y guardar sus datos", "bandera"},
	}},
	{"Medir", "Al cerrar y en la Review", []homeSectionInfo{
		{"/dashboard", "El estado del proyecto de un vistazo", "tablero-grilla"},
		{"/estado", "Avance y Sprint en curso", "indicador"},
		{"/metricas", "Velocidad y desviación de esfuerzo", "barras"},
		{"/calidad", "Historias completadas y defectos", "torta"},
		{"/sprints-anteriores", "Historial de cada Sprint", "historial"},
		{"/reportes", "Exportar en PDF", "documento"},
	}},
}

const (
	homeOtherTitle    = "Otras secciones"
	homeOtherSubtitle = "Secciones adicionales"
	homeOtherIcon     = "punto"
)

// homeIconNames son los íconos que dibuja la plantilla.
var homeIconNames = map[string]bool{
	"carpeta": true, "lista": true, "reloj": true, "cartas": true, "ciclo": true, "columnas": true,
	"lapiz": true, "alerta": true, "bandera": true, "tablero-grilla": true, "indicador": true,
	"barras": true, "torta": true, "historial": true, "documento": true, homeOtherIcon: true,
}

// homeGroups agrupa las secciones registradas (las del menú, sin Inicio). Un grupo sin secciones no
// se devuelve, y una sección sin grupo va a "Otras secciones", al final.
func homeGroups(menu []MenuItem) []homeGroup {
	registered := make(map[string]MenuItem, len(menu))
	for _, it := range menu {
		if it.Path != "/" {
			registered[it.Path] = it
		}
	}
	placed := make(map[string]bool, len(registered))
	var groups []homeGroup
	for _, g := range homeLayout {
		group := homeGroup{Title: g.title, Subtitle: g.subtitle}
		for _, sec := range g.sections {
			it, ok := registered[sec.path]
			if !ok {
				continue
			}
			placed[sec.path] = true
			group.Items = append(group.Items, homeItem{Name: it.Name, Path: it.Path, Description: sec.description, Icon: sec.icon})
		}
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	}
	other := homeGroup{Title: homeOtherTitle, Subtitle: homeOtherSubtitle}
	for _, it := range menu {
		if it.Path == "/" || placed[it.Path] {
			continue
		}
		other.Items = append(other.Items, homeItem{Name: it.Name, Path: it.Path, Description: "Abrir " + it.Name, Icon: homeOtherIcon})
	}
	if len(other.Items) > 0 {
		groups = append(groups, other)
	}
	return groups
}
