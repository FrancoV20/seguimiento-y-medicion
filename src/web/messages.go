package web

// Textos compartidos de la interfaz y de las páginas de error (HU-20a, T005).
const (
	TitleNotFound = "Página no encontrada"
	MsgNotFound   = "La dirección que buscás no existe o ya no está disponible."

	TitleMethodNotAllowed = "Acción no permitida"
	MsgMethodNotAllowed   = "Esta acción no está permitida en esta dirección."

	TitleInternal = "Error inesperado"
	MsgInternal   = "Ocurrió un error inesperado. Intentá nuevamente en unos minutos."

	TitleUnavailable = "Servicio no disponible"
	MsgUnavailable   = "El servicio no está disponible temporalmente"

	LinkBackHome = "Volver al inicio"

	MsgNoProjectSelected = "Sin proyecto seleccionado"

	TitleForbidden = "Solicitud rechazada"
	MsgForbidden   = "La solicitud se rechazó porque no se originó en esta aplicación."

	TitleSelectProject     = "Elegir proyecto"
	MsgProjectInvalid      = "El proyecto elegido no existe o no es válido. Elegí uno de la lista."
	MsgProjectGone         = "El proyecto que tenías seleccionado ya no existe. Elegí otro."
	MsgNoProjects          = "Aún no hay proyectos"
	LinkCreateFirstProject = "Crear el primer proyecto"

	TitleSelectMember     = "Identificarme"
	MsgChooseProjectFirst = "Primero elegí un proyecto."
	MsgMemberInvalid      = "El integrante elegido no pertenece al proyecto. Elegí uno de la lista."
	MsgMemberGone         = "Tu identificación ya no es válida: el integrante no pertenece al proyecto activo. Elegí otro."
	MsgNoMembers          = "Este proyecto no tiene integrantes"
)
