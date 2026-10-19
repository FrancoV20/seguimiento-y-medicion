package web

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// T012: verificación del esquema al iniciar (FR-005).

type fakeSchemaReader struct {
	status SchemaStatus
	err    error
}

func (r fakeSchemaReader) ReadSchemaStatus(context.Context) (SchemaStatus, error) {
	return r.status, r.err
}

func TestCheckSchema(t *testing.T) {
	const esperada = 7
	casos := []struct {
		nombre       string
		lector       fakeSchemaReader
		debeFallar   bool
		debeAdvertir bool
		textos       []string
	}{
		{"versión menor", fakeSchemaReader{status: SchemaStatus{Version: 4}}, true, false, []string{"desactualizado", "4", "7", "migrate", "up"}},
		{"sin tabla de migraciones", fakeSchemaReader{status: SchemaStatus{Version: 0}}, true, false, []string{"desactualizado", "migrate", "up"}},
		{"estado dirty", fakeSchemaReader{status: SchemaStatus{Version: 7, Dirty: true}}, true, false, []string{"dirty", "equipo", "force"}},
		{"versión igual", fakeSchemaReader{status: SchemaStatus{Version: 7}}, false, false, nil},
		{"versión mayor", fakeSchemaReader{status: SchemaStatus{Version: 9}}, false, true, []string{"9", "7"}},
		{"error de lectura", fakeSchemaReader{err: errors.New("relation x: SELECT secreto")}, true, false, []string{"esquema"}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			warning, err := CheckSchema(context.Background(), c.lector, esperada)

			if c.debeFallar != (err != nil) {
				t.Fatalf("debeFallar=%v, se obtuvo error: %v", c.debeFallar, err)
			}
			if c.debeAdvertir != (warning != "") {
				t.Fatalf("debeAdvertir=%v, se obtuvo advertencia: %q", c.debeAdvertir, warning)
			}
			texto := warning
			if err != nil {
				texto = err.Error()
				if strings.Contains(texto, "SELECT") {
					t.Errorf("el mensaje no debe incluir consultas internas: %q", texto)
				}
			}
			for _, esperado := range c.textos {
				if !strings.Contains(texto, esperado) {
					t.Errorf("el mensaje debe contener %q: %q", esperado, texto)
				}
			}
		})
	}
}
