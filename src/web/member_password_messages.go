package web

// Textos de la identificación con contraseña (HU-27). Todos viven acá.
const (
	LabelPassword       = "Contraseña"
	MsgPasswordRequired = "Ingresá tu contraseña"
	MsgWrongCredentials = "Integrante o contraseña incorrectos"
	MsgTooManyAttempts  = "Demasiados intentos fallidos. Esperá un minuto y volvé a intentar"
	MsgLoggedOut        = "Cerraste tu identificación"
	ButtonLogout        = "Salir"

	// FieldPassword es el nombre del campo de la contraseña en "Identificarme".
	FieldPassword = "contrasena"
)

// NoPasswordMessage avisa que el integrante todavía no tiene contraseña configurada.
func NoPasswordMessage(name string) string {
	return name + " todavía no tiene contraseña configurada. Pedile al Scrum Master que la configure"
}

// IdentifiedMessage confirma la identificación.
func IdentifiedMessage(name string) string {
	return "Te identificaste como " + name
}
