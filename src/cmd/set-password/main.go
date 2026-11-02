// Comando set-password: el Scrum Master configura la contraseña de un integrante (HU-27).
//
//	go run ./src/cmd/set-password -integrante N
//
// Usa DATABASE_URL, pide la contraseña dos veces por la entrada estándar y guarda solo su valor
// derivado (PBKDF2-SHA256). Si el integrante ya tenía contraseña, la reemplaza.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"

	"github.com/FrancoV20/seguimiento-y-medicion/src/auth"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

const (
	msgUsage          = "Uso: go run ./src/cmd/set-password -integrante N"
	msgDatabase       = "No se pudo acceder a la base de datos"
	msgMemberNotFound = "El integrante no existe"
	msgConfigured     = "Contraseña configurada para %s"
	promptPassword    = "Contraseña: "
	promptConfirm     = "Repetí la contraseña: "
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Getenv))
}

// run interpreta los argumentos y el entorno, abre la base y configura la contraseña. Devuelve 0
// si se configuró, 1 ante un error de datos o de la base y 2 ante un uso incorrecto.
func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("set-password", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	member := flags.String("integrante", "", "identificador del integrante")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		fmt.Fprintln(stdout, msgUsage)
		return 2
	}
	memberID, err := strconv.ParseInt(*member, 10, 64)
	if err != nil || memberID <= 0 || memberID > math.MaxInt32 {
		fmt.Fprintln(stdout, msgUsage)
		return 2
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
	return setPassword(ctx, auth.NewPostgreSQLPasswordStore(db), memberID, stdin, stdout)
}

// setPassword pide la contraseña dos veces, la valida y guarda su valor derivado. Nunca imprime lo
// que lee.
func setPassword(ctx context.Context, store auth.PasswordStore, memberID int64, stdin io.Reader, stdout io.Writer) int {
	name, err := store.MemberName(ctx, memberID)
	if errors.Is(err, auth.ErrMemberNotFound) {
		fmt.Fprintln(stdout, msgMemberNotFound)
		return 1
	}
	if err != nil {
		fmt.Fprintln(stdout, msgDatabase)
		return 1
	}

	reader := bufio.NewReader(stdin)
	fmt.Fprint(stdout, promptPassword)
	password := readLine(reader)
	fmt.Fprint(stdout, promptConfirm)
	confirm := readLine(reader)
	fmt.Fprintln(stdout)
	if err := auth.ValidateNewPassword(password, confirm); err != nil {
		fmt.Fprintln(stdout, err)
		return 1
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		fmt.Fprintln(stdout, msgDatabase)
		return 1
	}
	if err := store.SetPasswordHash(ctx, memberID, hash); err != nil {
		if errors.Is(err, auth.ErrMemberNotFound) {
			fmt.Fprintln(stdout, msgMemberNotFound)
		} else {
			fmt.Fprintln(stdout, msgDatabase)
		}
		return 1
	}
	fmt.Fprintf(stdout, msgConfigured+"\n", name)
	return 0
}

// readLine lee una línea y le quita solo los \n y \r finales; una entrada cerrada da lo leído
// hasta ahí (vacío si no hay nada).
func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}
