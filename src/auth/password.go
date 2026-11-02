// Package auth tiene las reglas de las contraseñas de los integrantes y el bloqueo por intentos
// fallidos (HU-27). No depende de la base ni de la web.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultIterations son las iteraciones de PBKDF2-SHA256 de producción.
	DefaultIterations = 600000
	// MinPasswordLength y MaxPasswordLength son los límites de una contraseña nueva, en caracteres (runas).
	MinPasswordLength = 8
	MaxPasswordLength = 128

	saltLength    = 16
	keyLength     = 32
	maxIterations = 10000000
	scheme        = "pbkdf2-sha256"
)

var (
	ErrPasswordTooShort = errors.New("La contraseña debe tener al menos 8 caracteres")
	ErrPasswordTooLong  = errors.New("La contraseña no puede superar 128 caracteres")
	ErrPasswordMismatch = errors.New("Las contraseñas no coinciden")
	// ErrMalformedHash indica que el valor guardado no tiene el formato "pbkdf2-sha256$iteraciones$sal$hash".
	ErrMalformedHash = errors.New("auth: valor de contraseña guardado mal formado")
)

var encoding = base64.RawStdEncoding

// HashPassword devuelve el valor a guardar para la contraseña, con una sal aleatoria nueva y
// DefaultIterations.
func HashPassword(password string) (string, error) {
	return HashPasswordIterations(password, DefaultIterations)
}

// HashPasswordIterations es HashPassword con las iteraciones indicadas; lo usan las pruebas para
// no pagar 600.000 iteraciones. Iteraciones menores a 1 son un error.
func HashPasswordIterations(password string, iterations int) (string, error) {
	if iterations < 1 || iterations > maxIterations {
		return "", fmt.Errorf("auth: iteraciones fuera de rango: %d", iterations)
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generar la sal: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, keyLength)
	if err != nil {
		return "", fmt.Errorf("auth: derivar la clave: %w", err)
	}
	return fmt.Sprintf("%s$%d$%s$%s", scheme, iterations, encoding.EncodeToString(salt), encoding.EncodeToString(key)), nil
}

// VerifyPassword indica si la contraseña corresponde al valor guardado. Usa las iteraciones del
// valor guardado y compara en tiempo constante. Un valor mal formado devuelve ErrMalformedHash.
func VerifyPassword(password, stored string) (bool, error) {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != scheme {
		return false, ErrMalformedHash
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 || iterations > maxIterations {
		return false, ErrMalformedHash
	}
	salt, err := encoding.DecodeString(parts[2])
	if err != nil || len(salt) != saltLength {
		return false, ErrMalformedHash
	}
	want, err := encoding.DecodeString(parts[3])
	if err != nil || len(want) != keyLength {
		return false, ErrMalformedHash
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, keyLength)
	if err != nil {
		return false, fmt.Errorf("auth: derivar la clave: %w", err)
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// ValidateNewPassword aplica las reglas de una contraseña nueva: el largo (en caracteres, sin
// recortar espacios) y después que la confirmación coincida.
func ValidateNewPassword(password, confirm string) error {
	switch n := utf8.RuneCountInString(password); {
	case n < MinPasswordLength:
		return ErrPasswordTooShort
	case n > MaxPasswordLength:
		return ErrPasswordTooLong
	}
	if password != confirm {
		return ErrPasswordMismatch
	}
	return nil
}
