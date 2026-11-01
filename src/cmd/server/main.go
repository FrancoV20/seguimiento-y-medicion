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
	"github.com/FrancoV20/seguimiento-y-medicion/src/backlog"
	"github.com/FrancoV20/seguimiento-y-medicion/src/defect"
	"github.com/FrancoV20/seguimiento-y-medicion/src/effort"
	"github.com/FrancoV20/seguimiento-y-medicion/src/metrics"
	"github.com/FrancoV20/seguimiento-y-medicion/src/poker"
	"github.com/FrancoV20/seguimiento-y-medicion/src/project"
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
	// Clave de las cookies de contexto (HU-26): antes de abrir el puerto, para no arrancar con
	// una clave corta. La clave nunca se registra.
	secret, temporary, err := web.LoadCookieSecret(getenv)
	if err != nil {
		return fail(err.Error())
	}
	if temporary {
		logger.Warn(web.MsgTemporaryCookieSecret)
	}
	web.SetCookieSecret(secret)

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

	// Inicialización y Registro de Backlog (HU-22)
	backlogService := backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(database))
	storyCatalog := web.NewPostgreSQLStoryCatalog(database)
	if err := web.RegisterBacklog(srv, backlogService, storyCatalog); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	// Estimaciones: horas estimadas por historia (HU-17, T019).
	estimateService := backlog.NewEstimateService(backlog.NewPostgreSQLEstimateStore(database))
	if err := web.RegisterEstimates(srv, estimateService, web.NewPostgreSQLEstimateCatalog(database)); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail("no se pudo registrar la sección Estimaciones")
	}
	// Inicialización y Registro de Proyectos (HU-21)
	projectService := project.NewProjectService(project.NewPostgreSQLProjectStore(database))
	projectCatalog := web.NewPostgreSQLProjectCatalog(database)
	if err := web.RegisterProjects(srv, projectService, projectCatalog); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail("no se pudo registrar la sección Proyectos")
	}
	// Tablero: cambio de estado de las historias del Sprint activo (HU-14, T022).
	if err := web.RegisterBoard(srv,
		sprint.NewStatusService(sprint.NewPostgreSQLStoryStatusStore(database)),
		web.NewPostgreSQLBoardCatalog(database)); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	// Planning Poker: rondas de votación y votos ocultos (HU-05a, T043).
	if err := web.RegisterPlanningPoker(srv,
		poker.NewService(poker.NewPostgreSQLStore(database)),
		web.NewPostgreSQLPokerCatalog(database)); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail(err.Error())
	}

	// Registro de esfuerzo: registra con el servicio de HU-03 y lee con su catálogo (HU-24, T022).
	effortService := effort.NewService(effort.NewPostgreSQLEffortStore(database), nil)
	effortActions := web.NewEffortActions(effortService, nil)
	if err := web.RegisterEffort(srv, effortActions, web.NewPostgreSQLEffortCatalog(database), nil); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail("no se pudo registrar la sección Registro de esfuerzo")
	}

	// Calidad y avance: métricas de cada Sprint finalizado, de solo lectura (HU-10, T030).
	qualityService := metrics.NewQualityService(metrics.NewPostgreSQLQualityStore(database))
	if err := web.RegisterQuality(srv, qualityService, web.NewPostgreSQLQualityCatalog(database)); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail("no se pudo registrar la sección Calidad y avance")
	}

	defectService := defect.NewService(defect.NewPostgreSQLStore(database))
	defectResolution := defect.NewResolutionService(defect.NewPostgreSQLResolutionStore(database))
	defectCatalog := web.NewPostgreSQLDefectCatalog(database)
	if err := web.RegisterDefectsWithResolution(srv, defectService, defectResolution, defectCatalog); err != nil {
		_ = ln.Close()
		_ = database.Close()
		return fail("no se pudo registrar la sección Defectos")
	}

	// Sprints: creación, asignación y remoción de historias del Sprint activo (HU-23, T018).
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

	// Estado del proyecto: estado por fechas, avance y Sprint en curso (HU-12).
	if err := web.RegisterProjectStatus(srv, web.NewPostgreSQLProjectStatusCatalog(database),
		web.NewPostgreSQLBoardCatalog(database), nil); err != nil {
		_ = ln.Close()
		_ = sprintStore.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	// Cierre de Sprint: valida con HU-07 y persiste el cierre atómico (HU-15).
	closeService := sprint.NewCloseSprintService(
		sprint.NewPostgreSQLSprintCloser(database),
		sprintClosureValidator{},
	)
	if err := web.RegisterClose(srv, closeService, web.NewPostgreSQLBoardCatalog(database)); err != nil {
		_ = ln.Close()
		_ = sprintStore.Close()
		_ = database.Close()
		return fail(err.Error())
	}
	// Métricas del Sprint: velocidad y desviación de HU-07 de los Sprints finalizados (HU-25, T019).
	if err := web.RegisterMetrics(srv, web.NewPostgreSQLMetricsCatalog(database),
		web.NewHU07MetricsSource(database)); err != nil {
		_ = ln.Close()
		_ = sprintStore.Close()
		_ = database.Close()
		return fail("no se pudo registrar la sección Métricas del Sprint")
	}

	if err := srv.Serve(ctx, ln, func() {
		_ = sprintStore.Close()
		_ = database.Close()
	}); err != nil {
		return fail(err.Error())
	}
	return 0
}

// sprintClosureValidator aplica las reglas de HU-07 a las historias que el cierre bloqueó (HU-15).
type sprintClosureValidator struct{}

func (sprintClosureValidator) ValidateSprintClosure(_ context.Context, stories []sprint.ClosingStory) error {
	data := metrics.SprintData{Stories: make([]metrics.CompletedStory, 0, len(stories))}
	for _, story := range stories {
		data.Stories = append(data.Stories, metrics.CompletedStory{
			ID:          story.ID,
			Status:      backlog.StoryStatus(story.Status),
			StoryPoints: story.StoryPoints,
			ActualHours: story.ActualHours,
		})
	}
	return metrics.ValidateSprintClosure(data)
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
