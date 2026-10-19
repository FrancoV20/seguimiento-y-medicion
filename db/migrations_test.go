package db

import (
	"testing"
	"testing/fstest"
)

func TestLatestVersion_DevuelveLaMayorVersionDeLosUp(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/001_create_projects.up.sql":   {},
		"migrations/001_create_projects.down.sql": {},
		"migrations/004_create_sprints.up.sql":    {},
		"migrations/007_create_metrics.up.sql":    {},
		"migrations/009_solo_down.down.sql":       {},
	}

	got, err := latestVersion(fsys)

	if err != nil || got != 7 {
		t.Errorf("se esperaba versión 7, se obtuvo %d (error: %v)", got, err)
	}
}

func TestLatestVersion_NombreInvalidoFalla(t *testing.T) {
	fsys := fstest.MapFS{"migrations/abc_sin_numero.up.sql": {}}

	if _, err := latestVersion(fsys); err == nil {
		t.Error("se esperaba un error por nombre de migración inválido")
	}
}

func TestLatestVersion_LeeLasMigracionesDelRepositorio(t *testing.T) {
	got, err := LatestVersion()

	if err != nil || got == 0 {
		t.Errorf("se esperaba una versión mayor a 0, se obtuvo %d (error: %v)", got, err)
	}
}
