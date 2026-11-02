package test

import (
	"database/sql"
	"net/url"
	"strconv"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
)

// HU-27: auxiliares para que los escenarios se identifiquen con contraseña. Las contraseñas se
// guardan con 1.000 iteraciones para que las pruebas sean rápidas.

const claveDePrueba = "clave-de-prueba-2026"

// configurarClave guarda la contraseña del integrante, como lo hace el comando set-password.
func configurarClave(t *testing.T, db *sql.DB, id int64, clave string) {
	t.Helper()
	hash, err := auth.HashPasswordIterations(clave, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE members SET password_hash = $2 WHERE id = $1`, id, hash); err != nil {
		t.Fatalf("no se pudo configurar la contraseña: %v", err)
	}
}

// formIdentificacion es el envío de "Identificarme".
func formIdentificacion(id, clave string) url.Values {
	return url.Values{"integrante_id": {id}, "contrasena": {clave}}
}

// identificacionDePrueba configura claveDePrueba para el integrante y devuelve el envío.
func identificacionDePrueba(t *testing.T, db *sql.DB, id int64) url.Values {
	t.Helper()
	configurarClave(t, db, id, claveDePrueba)
	return formIdentificacion(strconv.FormatInt(id, 10), claveDePrueba)
}
