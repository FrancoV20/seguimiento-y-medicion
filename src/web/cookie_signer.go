package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"sync/atomic"
)

// CookieSigner firma y verifica los valores de las cookies de contexto con HMAC-SHA256
// (HU-26). El formato es "<valor>.<firma>", con la firma en base64 URL sin relleno calculada
// sobre "nombre 0x00 valor": así la firma de una cookie no sirve en otra (FR-006).
type CookieSigner struct {
	key []byte
}

// NewCookieSigner crea un firmador con una copia de la clave.
func NewCookieSigner(key []byte) *CookieSigner {
	return &CookieSigner{key: append([]byte(nil), key...)}
}

func (s *CookieSigner) mac(name, value string) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(name))
	m.Write([]byte{0})
	m.Write([]byte(value))
	return m.Sum(nil)
}

// Sign devuelve el valor con su firma para la cookie name.
func (s *CookieSigner) Sign(name, value string) string {
	return value + "." + base64.RawURLEncoding.EncodeToString(s.mac(name, value))
}

// Verify separa el valor de la firma por el último punto y la comprueba en tiempo constante
// (FR-007). Devuelve ok=false si falta el punto, el valor o la firma, o si la firma no coincide.
func (s *CookieSigner) Verify(name, signed string) (string, bool) {
	i := strings.LastIndexByte(signed, '.')
	if i <= 0 || i == len(signed)-1 {
		return "", false
	}
	value, sig := signed[:i], signed[i+1:]
	// Strict rechaza codificaciones no canónicas (bits de relleno distintos de cero): cada firma
	// tiene una sola forma válida y no se puede alterar su último carácter sin invalidarla.
	got, err := base64.RawURLEncoding.Strict().DecodeString(sig)
	if err != nil || !hmac.Equal(got, s.mac(name, value)) {
		return "", false
	}
	return value, true
}

// cookieSigner es la clave del proceso. Arranca con una clave aleatoria para que el servidor y
// las pruebas funcionen sin configurar nada; main la reemplaza al arrancar (SetCookieSecret).
var cookieSigner atomic.Pointer[CookieSigner]

func init() {
	cookieSigner.Store(NewCookieSigner(randomCookieSecret()))
}

// randomCookieSecret devuelve 32 bytes aleatorios. crypto/rand.Read no falla en Go 1.24+.
func randomCookieSecret() []byte {
	b := make([]byte, MinCookieSecretLength)
	_, _ = rand.Read(b)
	return b
}

// SetCookieSecret reemplaza la clave con la que se firman y verifican las cookies de contexto.
func SetCookieSecret(secret []byte) {
	cookieSigner.Store(NewCookieSigner(secret))
}

// SignedContextValue es el valor firmado de una cookie de contexto con ese id. Lo usan las
// pruebas para armar cookies válidas.
func SignedContextValue(name string, id int64) string {
	return cookieSigner.Load().Sign(name, strconv.FormatInt(id, 10))
}

// verifyContextValue devuelve el valor de la cookie si la firma es válida.
func verifyContextValue(name, signed string) (string, bool) {
	return cookieSigner.Load().Verify(name, signed)
}
