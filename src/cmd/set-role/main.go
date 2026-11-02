// Comando set-role: asigna el rol de un integrante en un proyecto (HU-28).
//
//	go run ./src/cmd/set-role -proyecto P -integrante N -rol "Scrum Master"
//
// El rol es "Scrum Master" o "Integrante", con esa escritura exacta. Usa DATABASE_URL. Asignar el
// rol que ya tenía se informa igual; quitarle el rol al último Scrum Master se permite.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"

	"github.com/FrancoV20/seguimiento-y-medicion/src/role"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

const (
	msgUsage              = `Uso: go run ./src/cmd/set-role -proyecto P -integrante N -rol "Scrum Master"`
	msgInvalidRole        = "El rol no es válido"
	msgProjectNotFound    = "El proyecto no existe"
	msgMemberNotInProject = "El integrante no pertenece al proyecto"
	msgDatabase           = "No se pudo acceder a la base de datos"
	msgAssigned           = "%s ahora es %s de «%s»"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Getenv))
}

// run interpreta los argumentos, valida el rol, abre la base y asigna el rol. Devuelve 0 si se
// asignó, 1 ante un rechazo o una falla y 2 ante un uso incorrecto (sin conectarse).
func run(ctx context.Context, args []string, stdout io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("set-role", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	projectArg := flags.String("proyecto", "", "identificador del proyecto")
	memberArg := flags.String("integrante", "", "identificador del integrante")
	roleArg := flags.String("rol", "", `"Scrum Master" o "Integrante"`)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		fmt.Fprintln(stdout, msgUsage)
		return 2
	}
	given := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { given[f.Name] = true })
	projectID, okProject := parseID(*projectArg)
	memberID, okMember := parseID(*memberArg)
	if !given["proyecto"] || !given["integrante"] || !given["rol"] || !okProject || !okMember {
		fmt.Fprintln(stdout, msgUsage)
		return 2
	}

	r, err := role.ParseRole(*roleArg)
	if err != nil {
		fmt.Fprintln(stdout, msgInvalidRole)
		return 1
	}

	cfg, err := web.LoadConfig(getenv)
	if err != nil {
		fmt.Fprintln(stdout, err)
		return 1
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(stdout, msgDatabase)
		return 1
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintln(stdout, msgDatabase)
		return 1
	}
	return assignRole(ctx, role.NewPostgreSQLStore(db), projectID, memberID, r, stdout)
}

// parseID acepta un entero de 1 a 2147483647 (las columnas id son SERIAL).
func parseID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 || id > math.MaxInt32 {
		return 0, false
	}
	return id, true
}

// assignRole comprueba el proyecto y después guarda el rol del integrante. Informa un solo motivo
// por vez y nunca muestra detalles internos de la base.
func assignRole(ctx context.Context, store role.Store, projectID, memberID int64, r role.Role, stdout io.Writer) int {
	projectName, err := store.ProjectName(ctx, projectID)
	if errors.Is(err, role.ErrProjectNotFound) {
		fmt.Fprintln(stdout, msgProjectNotFound)
		return 1
	}
	if err != nil {
		fmt.Fprintln(stdout, msgDatabase)
		return 1
	}

	memberName, err := store.SetMemberRole(ctx, projectID, memberID, r)
	if errors.Is(err, role.ErrMemberNotInProject) {
		fmt.Fprintln(stdout, msgMemberNotInProject)
		return 1
	}
	if err != nil {
		fmt.Fprintln(stdout, msgDatabase)
		return 1
	}
	fmt.Fprintf(stdout, msgAssigned+"\n", memberName, r, projectName)
	return 0
}
