package web

import (
	"fmt"
	"strings"
)

// EnvCookieSecret es la variable de entorno con la clave de las cookies de contexto (HU-26).
const EnvCookieSecret = "COOKIE_SECRET"

// MinCookieSecretLength es el largo mínimo de la clave, en caracteres (bytes ASCII).
const MinCookieSecretLength = 32

// MsgTemporaryCookieSecret es el aviso que se registra cuando no hay clave configurada.
const MsgTemporaryCookieSecret = "COOKIE_SECRET no está definida: se usa una clave temporal y las sesiones se perderán al reiniciar"

// ShortCookieSecretError indica que la clave configurada es demasiado corta. No incluye el valor.
type ShortCookieSecretError struct{}

func (*ShortCookieSecretError) Error() string {
	return fmt.Sprintf("la variable de entorno %s debe tener al menos %d caracteres", EnvCookieSecret, MinCookieSecretLength)
}

// LoadCookieSecret lee la clave con getenv (normalmente os.Getenv). Sin clave devuelve una
// temporal aleatoria con temporary=true (E7); con menos del mínimo, ShortCookieSecretError (E8).
func LoadCookieSecret(getenv func(string) string) (secret []byte, temporary bool, err error) {
	raw := strings.TrimSpace(getenv(EnvCookieSecret))
	if raw == "" {
		return randomCookieSecret(), true, nil
	}
	if len(raw) < MinCookieSecretLength {
		return nil, false, &ShortCookieSecretError{}
	}
	return []byte(raw), false, nil
}
