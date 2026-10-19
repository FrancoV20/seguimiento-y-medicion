package web

import (
	"errors"
	"strings"
	"testing"
)

// T010: configuración por variables de entorno (FR-002, SC-001).

func envFrom(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func TestLoadConfig_FaltaDatabaseURLNombraLaVariableSinValores(t *testing.T) {
	_, err := LoadConfig(envFrom(map[string]string{"ADDR": ":9999-valor-secreto"}))

	var missing *MissingEnvError
	if !errors.As(err, &missing) || missing.Name != EnvDatabaseURL {
		t.Fatalf("se esperaba MissingEnvError para %s, se obtuvo: %v", EnvDatabaseURL, err)
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("el mensaje debe nombrar la variable, se obtuvo %q", err.Error())
	}
	if strings.Contains(err.Error(), "valor-secreto") {
		t.Errorf("el mensaje no debe incluir valores de configuración: %q", err.Error())
	}
}

func TestLoadConfig_DatabaseURLSoloConEspaciosCuentaComoFaltante(t *testing.T) {
	_, err := LoadConfig(envFrom(map[string]string{"DATABASE_URL": "   "}))

	var missing *MissingEnvError
	if !errors.As(err, &missing) {
		t.Fatalf("se esperaba MissingEnvError, se obtuvo: %v", err)
	}
}

func TestLoadConfig_AddrPorDefecto(t *testing.T) {
	cfg, err := LoadConfig(envFrom(map[string]string{"DATABASE_URL": "postgres://u:p@localhost:5433/db"}))

	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo: %v", err)
	}
	if cfg.Addr != DefaultAddr || DefaultAddr != ":8080" {
		t.Errorf("ADDR por defecto debe ser :8080, se obtuvo %q", cfg.Addr)
	}
}

func TestLoadConfig_AddrConfigurado(t *testing.T) {
	cfg, err := LoadConfig(envFrom(map[string]string{"DATABASE_URL": "postgres://u:p@h/db", "ADDR": ":9090"}))

	if err != nil || cfg.Addr != ":9090" {
		t.Errorf("se esperaba ADDR :9090, se obtuvo %q (error: %v)", cfg.Addr, err)
	}
}

func TestDescribeDatabase_NoMuestraUsuarioClaveNiParametros(t *testing.T) {
	got := DescribeDatabase("postgres://sym_user:Sup3rS3cret@localhost:5433/seguimiento_y_medicion?sslmode=disable")

	if got != "localhost:5433/seguimiento_y_medicion" {
		t.Errorf("descripción inesperada: %q", got)
	}
	for _, prohibido := range []string{"Sup3rS3cret", "sym_user", "sslmode", "postgres://"} {
		if strings.Contains(got, prohibido) {
			t.Errorf("la descripción no debe contener %q: %q", prohibido, got)
		}
	}
}

func TestDescribeDatabase_URLInvalidaNoSeRepite(t *testing.T) {
	got := DescribeDatabase("::no es una url con clave Sup3rS3cret::")

	if strings.Contains(got, "Sup3rS3cret") {
		t.Errorf("una URL inválida no debe repetirse: %q", got)
	}
}

func TestConfig_RedactQuitaClaveYURLCompleta(t *testing.T) {
	url := "postgres://sym_user:Sup3rS3cret@localhost:5433/db?sslmode=disable"
	cfg := Config{DatabaseURL: url}

	got := cfg.Redact("falló " + url + " con clave Sup3rS3cret")

	if strings.Contains(got, "Sup3rS3cret") || strings.Contains(got, url) {
		t.Errorf("el texto redactado todavía expone datos sensibles: %q", got)
	}
}
