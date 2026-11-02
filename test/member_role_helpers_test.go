package test

import (
	"database/sql"
	"testing"
)

// HU-28, T020: auxiliar común para que los escenarios se identifiquen como Scrum Master.

// configurarScrumMaster guarda claveDePrueba para el integrante (HU-27) y le da el rol "Scrum Master"
// en el proyecto, como lo hace el comando set-role. Falla si el integrante no es del proyecto.
func configurarScrumMaster(t *testing.T, db *sql.DB, proyecto, integrante int64) {
	t.Helper()
	configurarClave(t, db, integrante, claveDePrueba)
	res, err := db.Exec(`UPDATE project_members SET role = 'Scrum Master' WHERE project_id = $1 AND member_id = $2`,
		proyecto, integrante)
	if err != nil {
		t.Fatalf("no se pudo asignar el rol Scrum Master: %v", err)
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		t.Fatalf("el integrante %d no es del proyecto %d (filas: %d, error: %v)", integrante, proyecto, n, err)
	}
}
