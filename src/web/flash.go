package web

import (
	"context"
	"net/http"
	"net/url"
	"unicode/utf8"
)

// CookieFlash es la cookie del mensaje de una sola vista (HU-20b).
const CookieFlash = "flash"

// FlashKind es el tipo de un mensaje de una sola vista.
type FlashKind string

const (
	FlashSuccess FlashKind = "success"
	FlashError   FlashKind = "error"
	FlashWarning FlashKind = "warning"
)

const (
	flashMaxAge  = 60  // segundos que vive la cookie
	flashMaxText = 300 // caracteres máximos del texto
)

// Flash es un mensaje de una sola vista. El texto lo define siempre el servidor.
type Flash struct {
	Kind FlashKind
	Text string
}

func (k FlashKind) valid() bool {
	return k == FlashSuccess || k == FlashError || k == FlashWarning
}

// Class es la clase CSS del mensaje.
func (f Flash) Class() string { return "alert alert-" + string(f.Kind) }

// Label es la etiqueta de texto que acompaña al color.
func (f Flash) Label() string {
	switch f.Kind {
	case FlashSuccess:
		return "Éxito"
	case FlashWarning:
		return "Advertencia"
	default:
		return "Error"
	}
}

// Role es el rol ARIA: los errores se anuncian con prioridad.
func (f Flash) Role() string {
	if f.Kind == FlashError {
		return "alert"
	}
	return "status"
}

// SetFlash guarda un mensaje para la próxima página que se muestre. Ignora tipos
// desconocidos y textos vacíos, y recorta los textos muy largos.
func SetFlash(w http.ResponseWriter, r *http.Request, kind FlashKind, text string) {
	if !kind.valid() || text == "" {
		return
	}
	if utf8.RuneCountInString(text) > flashMaxText {
		text = string([]rune(text)[:flashMaxText])
	}
	c := newContextCookie(r, CookieFlash, url.Values{"k": {string(kind)}, "t": {text}}.Encode())
	c.MaxAge = flashMaxAge
	http.SetCookie(w, c)
}

// flashFromRequest lee el mensaje de la cookie. La cookie no está firmada, así que se
// acepta solo un tipo conocido y un texto acotado; el texto siempre se muestra escapado.
func flashFromRequest(r *http.Request) *Flash {
	c, err := r.Cookie(CookieFlash)
	if err != nil {
		return nil
	}
	v, err := url.ParseQuery(c.Value)
	if err != nil {
		return nil
	}
	kind, text := FlashKind(v.Get("k")), v.Get("t")
	if !kind.valid() || text == "" || !utf8.ValidString(text) || utf8.RuneCountInString(text) > flashMaxText {
		return nil
	}
	return &Flash{Kind: kind, Text: text}
}

type pendingFlashKey struct{}

// withPendingFlash lleva una advertencia detectada en esta solicitud (por ejemplo, un
// proyecto que ya no existe) para mostrarla en la misma respuesta.
func withPendingFlash(ctx context.Context, f *Flash) context.Context {
	return context.WithValue(ctx, pendingFlashKey{}, f)
}

func pendingFlashFrom(ctx context.Context) *Flash {
	f, _ := ctx.Value(pendingFlashKey{}).(*Flash)
	return f
}
