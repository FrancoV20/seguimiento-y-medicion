// Command server inicia la aplicación web (HU-20a, T022).
//
//	go run ./src/cmd/server
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/FrancoV20/seguimiento-y-medicion/db"
	"github.com/FrancoV20/seguimiento-y-medicion/src/sprint"
	"github.com/FrancoV20/seguimiento-y-medicion/src/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Getenv, os.Stderr))
}

// run arranca el servidor y devuelve el código de salida del proceso.
func run(ctx context.Context, getenv func(string) string, stderr io.Writer) int {
	logger := slog.New(slog.NewTextHandler(stderr, nil))
	fail := func(msg string) int {
		fmt.Fprintln(stderr, "No se pudo iniciar el servidor:", msg)
		return 1
	}

	cfg, err := web.LoadConfig(getenv)
	if err != nil {
		return fail(err.Error())
	}

	srv := web.NewServer(logger)
	// La protección contra envíos de otro origen va primero (HU-20b, T025).
	srv.Use(web.CrossOriginGuard(srv.Renderer()))
	ln, err := srv.Listen(cfg.Addr)
	if err != nil {
		return fail(err.Error())
	}

	database, err := web.OpenDatabase(ctx, cfg, web.DefaultRetryPolicy(), logger)
	if err != nil {
		_ = ln.Close()
		return fail(err.Error())
	}

	expected, err := db.LatestVersion()
	if err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	warning, err := web.CheckSchema(ctx, web.NewPostgreSQLSchemaVersionReader(database), expected)
	if err != nil {
		logger.Error("verificación de esquema", "detalle", cfg.Redact(fmt.Sprint(unwrapAll(err))))
		_ = ln.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	if warning != "" {
		logger.Warn(warning)
	}
	// Contexto de trabajo: proyecto e integrante activos (HU-20b, T039).
	web.RegisterContext(srv, web.NewPostgreSQLContextStore(database))

	sprintStore, err := sprint.NewPostgreSQLSprintStore(cfg.DatabaseURL)
	if err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	sprintService := sprint.NewService(sprintStore)
	sprintCatalog := web.NewPostgreSQLSprintCatalog(database)
	sprintActions := sprintActionsAdapter{service: sprintService, store: sprintStore}
	if err := web.RegisterSprints(srv, sprintCatalog, sprintActions); err != nil {
		_ = ln.Close()
		_ = sprintStore.Close()
		_ = database.Close()
		return fail(err.Error())
	}

	if err := srv.Serve(ctx, ln, func() {
		_ = sprintStore.Close()
		_ = database.Close()
	}); err != nil {
		return fail(err.Error())
	}
	return 0
}

type sprintActionsAdapter struct {
	service sprint.Service
	store   sprint.SprintStore
}

func (a sprintActionsAdapter) CreateSprint(
	ctx context.Context,
	request sprint.SprintCreationRequest,
	project sprint.Project,
	stories []sprint.UserStory,
) (sprint.SprintCreationResult, error) {
	return a.service.CreateSprint(ctx, request, project, stories)
}

func (a sprintActionsAdapter) AssignStories(ctx context.Context, sprintID int64, storyIDs []int64) error {
	return a.store.AssignStories(ctx, sprintID, storyIDs)
}

func (a sprintActionsAdapter) RemoveStory(ctx context.Context, sprintID, storyID int64) error {
	return a.store.RemoveStory(ctx, sprintID, storyID)
}

// unwrapAll devuelve el error más interno, para registrar el detalle técnico.
func unwrapAll(err error) error {
	for {
		u, ok := err.(interface{ Unwrap() error })
		if !ok || u.Unwrap() == nil {
			return err
		}
		err = u.Unwrap()
	}
}
