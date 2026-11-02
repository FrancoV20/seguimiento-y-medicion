package web

import (
	"database/sql"
	"testing"
)

// HU-28, T014: auxiliar común para que las pruebas se identifiquen como Scrum Master.

// setTestScrumMaster configura la contraseña de prueba del integrante (HU-27) y le da el rol
// "Scrum Master" en el proyecto. Falla si el integrante no pertenece al proyecto.
func setTestScrumMaster(t *testing.T, db *sql.DB, projectID, memberID int64) {
	t.Helper()
	setTestPassword(t, db, memberID)
	res, err := db.Exec(`UPDATE project_members SET role = 'Scrum Master' WHERE project_id = $1 AND member_id = $2`,
		projectID, memberID)
	if err != nil {
		t.Fatalf("no se pudo asignar el rol Scrum Master de prueba: %v", err)
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		t.Fatalf("el integrante %d no pertenece al proyecto %d (filas: %d, error: %v)", memberID, projectID, n, err)
	}
}
