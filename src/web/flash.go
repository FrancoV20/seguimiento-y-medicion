package web

// CookieFlash es la cookie del mensaje de una sola vista (HU-20b, T006).
const CookieFlash = "flash"

// FlashKind es el tipo de un mensaje de una sola vista.
type FlashKind string

const (
	FlashSuccess FlashKind = "success"
	FlashError   FlashKind = "error"
	FlashWarning FlashKind = "warning"
)

// Flash es un mensaje de una sola vista. El texto lo define siempre el servidor.
type Flash struct {
	Kind FlashKind
	Text string
}
