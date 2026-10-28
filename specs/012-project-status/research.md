# Research: HU-12 – Consulta del Estado de un Proyecto

## Datos verificados en el código (main `b7419c2`, 2026-10-28)

Verificado con `git grep -n`:

- **`src/project` (HU-01)**: archivos `errors.go`, `model.go`, `postgres_store.go`, `project.go`, `service.go`,
  `store.go`, `test_store.go` y sus pruebas.
  - `Project` tiene `StartDate` y `EndDate` (`time.Time`) y `Status string`; `NewProject` pone `Status: "Active"`, y
    la base guarda `'Activo'` por defecto. Ninguna historia lo cambia.
  - El paquete **no importa** ningún otro paquete del repositorio.
  - **No existen** `DateStatus`, `StatusOn`, `ProgressPercentage`, `StatusNotStarted`, `StatusInProgress` ni
    `StatusFinished` en `src/` ni en `test/`.
- **`src/metrics` (HU-07)** importa solo `src/backlog`. `roundToTwoDecimals(*big.Rat) *big.Rat` (mitad hacia arriba a
  dos decimales) está **sin exportar** en `calculation.go`: desde `src/project` no se puede usar sin editar un archivo
  de HU-07. Importar `src/metrics` desde `src/project` no crearía un ciclo hoy (`metrics → backlog`), pero no sirve
  porque la función no es visible.
- **HU-14** (`src/web/board_catalog.go`, `postgres_board_catalog.go`):
  - `BoardCatalog.ActiveBoard(ctx, projectID) (BoardView, found bool, err error)`;
  - `BoardView` tiene `ID`, `Number int64`, `Label` ("Sprint N"), `Goal`, `StartDate`, `EndDate`, `Groups`,
    `PlannedPoints` y `CompletedPoints`;
  - lee el Sprint con `status = 'Activo'` del proyecto; sin fila devuelve `found = false` sin error; los errores del
    driver se devuelven sin envolver;
  - `NewPostgreSQLBoardCatalog(db)`.
- **Estados de historia**: `sprint.Pendiente`, `sprint.EnSprint`, `sprint.InProgress` y `sprint.Done` en
  `src/sprint/model.go` ("Pendiente", "En Sprint", "En progreso", "Hecho"); la base los fija con
  `chk_user_stories_status` (migración 004).
- **Esquema**: `projects(id SERIAL, name, start_date DATE, end_date DATE, status)` con `end_date > start_date`
  (migración 008); `user_stories(project_id, status, …)` sin borrado en cascada desde `projects`;
  `sprints(project_id, number, goal, start_date, end_date, status)`; `sprint_story_assignments(sprint_id, story_id)`.
- **Web (HU-20a/HU-20b)**:
  - `Server.Register(Section{Name, Path, RequiresProject}, Routes)` protege con `RequireContext`;
  - `ActiveContextFrom`, `projectIDFrom(r)` (en `board_handlers.go`), `Renderer.Page/PageStatus/Fail`, `NotFound`,
    `Unavailable`, `IsDatabaseUnavailable`;
  - `returnDestination` vuelve a la misma dirección (`/estado`) después de elegir proyecto;
  - `Server.route` responde 405 con la página de HU-20a; las plantillas están embebidas (`templates/*.html`);
  - `RegisterEffort(..., now func() time.Time)` ya usa un reloj inyectable (nil = `time.Now`).
- **`main.go`** registra cada sección con un bloque; hoy el último es el de HU-24 (`RegisterEffort`), después del de
  HU-05a (`RegisterPlanningPoker`). En la copia de trabajo de Windows los archivos tienen fin de línea CRLF: no se
  corre `gofmt -w` sobre archivos existentes (nota de HU-11).
- No existe ninguna sección con la dirección `/estado` ni el texto "Estado del proyecto", ni nombres
  `ProjectStatus*`/`projectStatus*` en `src/web`.
- HU-11 edita `projects_handlers.go`, `projects_catalog.go`, `projects.html` y `projects_handlers_test.go`
  (`specs/011-project-update/plan.md`); HU-12 no toca ninguno.
- **CI** (`.github/workflows/ci.yml`): `gofmt -l .` sin salida, `go vet ./...`, migraciones `up` y
  `go test ./... -p 1 -race -coverprofile`; las pruebas de integración arman sus propios datos.
- **Planificación (PDF)**: no está en el repositorio; se usan el Issue y las notas del equipo.
- **HU-09** (`specs/009-pdf-report`) espera de HU-12 una regla de estado con fecha de referencia y una de avance que
  devuelva el porcentaje como texto ("57.14"; "0.00" sin historias), con nombres a confirmar en su T001.

## Decisiones

### 1. Reglas en un archivo nuevo de `src/project`

- **Decision**: `src/project/status.go` con:

  ```go
  type DateStatus string

  const (
      StatusNotStarted DateStatus = "No iniciado"
      StatusInProgress DateStatus = "En curso"
      StatusFinished   DateStatus = "Finalizado"
  )

  func StatusOn(start, end, today time.Time) DateStatus
  func ProgressPercentage(done, total int) string
  ```

  No se edita ningún archivo existente de `src/project`.
- **Rationale**: decisión del equipo. El estado y el avance son reglas del dominio de proyectos; puras, sin base ni
  reloj, se prueban con tablas y HU-09 las reutiliza.
- **Alternatives considered**: un método `(*Project).Status()` (edita `model.go` y choca con el campo `Status`
  existente); ponerlas en `src/web` (HU-09 no las podría usar sin importar la web).

### 2. Comparar días del calendario

- **Decision**: `StatusOn` toma de cada fecha el año, el mes y el día con `t.Date()` en su propia zona y compara esos
  días: `today < start` → `StatusNotStarted`; `start <= today <= end` → `StatusInProgress`; `today > end` →
  `StatusFinished`. Las fechas del proyecto llegan de la base como medianoche UTC (columna `DATE`); "hoy" llega como la
  hora local del servidor.
- **Rationale**: R1 (días, no horas, con los dos bordes incluidos). Comparar instantes depende de la zona: con el
  servidor en UTC-3, las 22:00 locales del 27/10 ya son el 28/10 en UTC, y un proyecto que empieza el 28/10 aparecería
  "En curso" un día antes. Comparar días del calendario evita ese corrimiento.
- **Alternatives considered**: `today.Truncate(24 * time.Hour)` (trunca en UTC, no en la zona local); comparar en la
  base con `CURRENT_DATE` (la fecha la decide el servidor de base, no el de la aplicación, y no se puede inyectar).

### 3. Avance con enteros y la regla de HU-07

- **Decision**: `ProgressPercentage(done, total)`:
  - `total <= 0` → `"0.00"`;
  - si no, centésimas = `(done × 10000 × 2 + total) ÷ (2 × total)` con división entera (mitad hacia arriba), y el
    texto es `fmt.Sprintf("%d.%02d", centésimas/100, centésimas%100)`.
- **Rationale**:
  - misma regla que HU-07 (mitad hacia arriba a dos decimales), verificada con los mismos casos que HU-10 (2 de 3 →
    "66.67", 1 de 8 → "12.50") y con un caso de mitad exacta (1 de 32 = 3,125 → "3.13");
  - `roundToTwoDecimals` no está exportada y no se edita HU-07; con enteros no hace falta `math/big`;
  - el texto con punto es el mismo contrato que `DeviationPercentage` (HU-07) y `CompletionPercentage` (HU-10): HU-09
    lo usa sin parsear una coma.
- **Alternatives considered**: importar `src/metrics` (no hay ciclo, pero la función no es visible); exportarla en
  HU-07 (edita un archivo ajeno); `float64` con `math.Round` (errores binarios en mitades exactas).

### 4. Catálogo propio con una lectura consistente

- **Decision**: `ProjectStatusCatalog.StatusSummary(ctx, projectID)` en `src/web`, con adaptador
  `PostgreSQLProjectStatusCatalog`, en una transacción `ReadOnly` + `REPEATABLE READ`:
  1. `SELECT name, start_date, end_date FROM projects WHERE id = $1` (sin fila → `NotFound`);
  2. cuatro `COUNT(*) FILTER` sobre `user_stories` del proyecto.
- **Rationale**: decisión del equipo (FR-011). Principio VI: es el único puerto nuevo y tiene adaptador PostgreSQL y
  prueba de integración. El nombre, las fechas y las cantidades salen coherentes entre sí.
- **Alternatives considered**: reutilizar `ProjectCatalog` de HU-21 (lo edita HU-11 y no trae cantidades); contar
  con el Tablero (solo ve el Sprint activo).

### 5. Sprint en curso con el Tablero de HU-14

- **Decision**: el handler recibe un `BoardCatalog` (HU-14) y usa `ActiveBoard`: con `found`, muestra `Label + " en
  curso"`, `Goal` y las fechas; sin `found`, "No hay un Sprint en curso". El Tablero se lee después del catálogo, fuera
  de su transacción.
- **Rationale**: decisión del equipo; no se recalcula ni se edita HU-14. Si cambia el Sprint activo entre las dos
  lecturas, lo peor es mostrar el Sprint nuevo junto con las cantidades de un instante antes: no hay números que
  crucen las dos fuentes.
- **Alternatives considered**: leer el Sprint activo en el catálogo propio (duplica la consulta de HU-14).

### 6. Reloj inyectable solo en la pantalla

- **Decision**: `RegisterProjectStatus(s, catalog, board, now func() time.Time)`; `now == nil` → `time.Now`. El
  handler llama a `now()` una vez por pedido y pasa el resultado a `project.StatusOn`.
- **Rationale**: decisión del equipo (contrato para HU-09: la regla no lee el reloj). Mismo patrón que
  `RegisterEffort`.

### 7. Errores

- **Decision**:
  - el adaptador envuelve los errores con `%w`; sin fila de proyecto devuelve `NotFound(...)`;
  - cualquier error del catálogo o del Tablero va a `Renderer.Fail`, que responde 404 (`NotFound`), 503
    (`IsDatabaseUnavailable`) o 500, antes de escribir nada.
- **Rationale**: escenario 10 y FR-011/FR-012. `Fail` ya oculta el detalle técnico.

### 8. Textos y formato

- **Decision**: en `src/web/project_status_view.go`:
  - `TitleProjectStatus` = "Estado del proyecto";
  - `msgProjectStatusDates` = "Del %s al %s" (dd/mm/aaaa);
  - `msgProjectStatusState` = "Estado: %s";
  - `msgProjectStatusProgress` = "%d de %d historias completadas (%s)";
  - `msgProjectStatusSprint` = "%s en curso" (con `BoardView.Label`);
  - `msgProjectStatusSprintGoal` = "Objetivo: %s";
  - `MsgProjectStatusNoSprint` = "No hay un Sprint en curso";
  - `projectStatusPercentText("50.00")` → "50,00 %".
- **Rationale**: FR-015, un solo lugar. HU-08 (todavía no en `main`) planea textos parecidos con su propio prefijo;
  cada pantalla mantiene los suyos.

### 9. Registro en `main.go`

- **Decision**: un bloque nuevo después del último registro (hoy el de HU-24), que llama a
  `web.RegisterProjectStatus(srv, web.NewPostgreSQLProjectStatusCatalog(database),
  web.NewPostgreSQLBoardCatalog(database), nil)`, con el mismo manejo de error que los demás. Se respeta el fin de
  línea del archivo y no se corre `gofmt -w` sobre él.
- **Rationale**: notas del equipo. Una segunda instancia del catálogo del Tablero no guarda estado.

### 10. Pruebas con datos propios

- **Decision**: cada prueba de integración crea los proyectos del Issue con fechas fijas y el reloj en 28/10/2026:
  - «Alfa» (01/10/2026 al 30/11/2026): Sprint 1 "Finalizado" (05/10/2026 al 18/10/2026) con 3 historias "Hecho"
    asignadas; Sprint 2 "Activo" ("Pagos online", 19/10/2026 al 01/11/2026) con 1 "Hecho", 1 "En Sprint" y 1 "En
    progreso" asignadas; 2 "Pendiente" sin asignar;
  - «Beta» (01/12/2026 al 31/01/2027), sin historias ni Sprints;
  - «Gamma» (01/08/2026 al 30/09/2026): Sprint 1 "Finalizado" (01/08/2026 al 14/08/2026) con sus 3 historias
    "Hecho";
  - «Delta» (28/10/2026 al 15/12/2026) y «Épsilon» (01/09/2026 al 28/10/2026), sin historias ni Sprints.

  Limpieza en `t.Cleanup`, en este orden: asignaciones, historias, Sprints y proyectos (las historias antes que el
  proyecto, porque `user_stories.project_id` no tiene cascada).
- **Rationale**: el CI hace `TRUNCATE` y corre con `-p 1`. Cada escenario usa su propio proyecto; el Sprint 1 de
  «Alfa» y el de «Gamma» solo existen para que el Sprint activo sea el 2 y para que las "Hecho" estén donde las deja
  HU-15.

### 11. Rendimiento

- **Decision**: `TestHU12_Rendimiento` arma un proyecto con 1000 historias (repartidas en los cuatro estados) y un
  Sprint activo con 50, y exige `GET /estado` en menos de 500 ms.
- **Rationale**: SC-004. Son tres consultas (proyecto, cantidades y Sprint) más las historias del Tablero, que HU-14
  ya lee.

## Preguntas abiertas

Ninguna.
