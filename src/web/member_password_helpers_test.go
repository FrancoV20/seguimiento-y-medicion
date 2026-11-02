package web

import (
	"database/sql"
	"net/url"
	"strconv"
	"testing"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
)

// HU-27: auxiliares para que las pruebas se identifiquen con contraseña. Las contraseñas se guardan
// con 1.000 iteraciones para que las pruebas sean rápidas.

const testMemberPassword = "clave-de-prueba-2026"

// setTestPassword configura testMemberPassword para el integrante.
func setTestPassword(t *testing.T, db *sql.DB, memberID int64) {
	t.Helper()
	hash, err := auth.HashPasswordIterations(testMemberPassword, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE members SET password_hash = $2 WHERE id = $1`, memberID, hash); err != nil {
		t.Fatalf("no se pudo configurar la contraseña de prueba: %v", err)
	}
}

// memberLoginValues es el envío de "Identificarme" con la contraseña de prueba.
func memberLoginValues(id string) url.Values {
	return url.Values{"integrante_id": {id}, FieldPassword: {testMemberPassword}}
}

// loginAs configura la contraseña de prueba del integrante y devuelve el envío para identificarse.
func loginAs(t *testing.T, db *sql.DB, memberID int64) url.Values {
	t.Helper()
	setTestPassword(t, db, memberID)
	return memberLoginValues(strconv.FormatInt(memberID, 10))
}
