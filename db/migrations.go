// Package db expone los archivos de migración embebidos para que el servidor
// conozca la versión de esquema esperada sin una segunda fuente de verdad (HU-20a, T003).
package db

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

// Migrations contiene los archivos .sql de db/migrations.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// LatestVersion devuelve la mayor versión entre los archivos .up.sql embebidos.
func LatestVersion() (uint, error) {
	return latestVersion(Migrations)
}

func latestVersion(fsys fs.FS) (uint, error) {
	files, err := fs.Glob(fsys, "migrations/*.up.sql")
	if err != nil {
		return 0, err
	}
	var latest uint
	for _, f := range files {
		name := path.Base(f)
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			return 0, fmt.Errorf("la migración %q no tiene el formato NNN_nombre.up.sql", name)
		}
		v, err := strconv.ParseUint(prefix, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("la migración %q no empieza con un número de versión", name)
		}
		if uint(v) > latest {
			latest = uint(v)
		}
	}
	if latest == 0 {
		return 0, fmt.Errorf("no se encontraron migraciones .up.sql")
	}
	return latest, nil
}
