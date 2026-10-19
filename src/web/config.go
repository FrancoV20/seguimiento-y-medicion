package web

import (
	"fmt"
	"net/url"
	"strings"
)

// Variables de entorno de la aplicación (HU-20a, T017).
const (
	EnvDatabaseURL = "DATABASE_URL"
	EnvAddr        = "ADDR"
	DefaultAddr    = ":8080"
)

// Config son los valores leídos del entorno al iniciar.
type Config struct {
	DatabaseURL string
	Addr        string
}

// MissingEnvError indica una variable obligatoria ausente. Solo nombra la variable.
type MissingEnvError struct{ Name string }

func (e *MissingEnvError) Error() string {
	return fmt.Sprintf("falta la variable de entorno obligatoria %s", e.Name)
}

// LoadConfig lee la configuración con getenv (normalmente os.Getenv).
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{
		DatabaseURL: strings.TrimSpace(getenv(EnvDatabaseURL)),
		Addr:        strings.TrimSpace(getenv(EnvAddr)),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, &MissingEnvError{Name: EnvDatabaseURL}
	}
	if cfg.Addr == "" {
		cfg.Addr = DefaultAddr
	}
	return cfg, nil
}

// DescribeDatabase devuelve "host:puerto/base" sin usuario, clave ni parámetros,
// para usar en mensajes y registros.
func DescribeDatabase(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "(dirección de base de datos no válida)"
	}
	return u.Host + u.Path
}

// Redact quita de s la URL completa de la base y su clave, si aparecen.
func (c Config) Redact(s string) string {
	if c.DatabaseURL == "" {
		return s
	}
	s = strings.ReplaceAll(s, c.DatabaseURL, DescribeDatabase(c.DatabaseURL))
	if u, err := url.Parse(c.DatabaseURL); err == nil {
		if pass, ok := u.User.Password(); ok && pass != "" {
			s = strings.ReplaceAll(s, pass, "xxxxx")
		}
	}
	return s
}
