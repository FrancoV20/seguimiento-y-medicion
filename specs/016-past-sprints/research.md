# Research: HU-16 – Consulta de Sprints Anteriores

## Datos verificados en el código (main `3988eed`, 2026-10-28)

Verificado con `git grep -n`:

- **HU-07 (`src/metrics`)**:
  - `NewService(store MetricsStore) *Service` (`service.go:11`);
  - `(*Service).CalculateMetrics(sprintID int64) (SprintMetrics, error)` (`service.go:15`), sin `context`;
  - `NewPostgreSQLMetricsStore(db *sql.DB)` (`postgres_metrics_store.go:13`);
  - `SprintMetrics.Velocity int` (`model.go:26`).

  `CalculateMetrics` hace esto:
  1. Lee el Sprint por `id`, sin filtrar por proyecto.
  2. Rechaza un Sprint no "Finalizado" con `ErrSprintNotFinished`.
  3. Valida con `ValidateSprintClosure`: si una historia "Hecho" no tiene Story Points válidos, devuelve
     `ClosureValidationError`.
  4. Suma la velocidad con las historias que siguen en `sprint_story_assignments` (`postgres_metrics_store.go:52`),
     con su estado y sus Story Points **actuales**; solo cuentan las "Hecho".
  5. Guarda el resultado con `INSERT … ON CONFLICT (sprint_id) DO UPDATE` en `sprint_metrics`
     (`postgres_metrics_store.go:133`).

  Los errores de lectura y de guardado se envuelven en `metricsStoreError`, que tiene `Unwrap`: el error del driver
  llega a `IsDatabaseUnavailable`. `src/metrics` importa solo `src/backlog`, y `src/web` todavía no importa
  `src/metrics`; importarlo no crea un ciclo.
- **HU-15**: no está en el código. `git grep -n "sprint_story_results"` no encuentra nada en `src/`, `test/` ni
  `db/`. Lo que sigue sale de `specs/015-sprint-close/data-model.md` (T001: HU-15):
  - la tabla `sprint_story_results(sprint_id BIGINT, story_id BIGINT, final_status VARCHAR(20), story_points
    INTEGER, created_at)`, con `PRIMARY KEY (sprint_id, story_id)`, `ON DELETE CASCADE` hacia `sprints` y
    `user_stories`, y `final_status` en 'En Sprint', 'En progreso' o 'Hecho';
  - el cierre inserta una fila por cada historia asignada, con su estado y sus Story Points;
  - borra las asignaciones de las historias no "Hecho" y las pasa a 'Pendiente' con `active_sprint_id = NULL`;
  - pone el Sprint en 'Finalizado';
  - el número de la migración se decide al implementar HU-15 (el último de `main` más uno).
- **Esquema en `main`**:
  - `sprints(id BIGSERIAL, project_id, number, goal, start_date DATE, end_date DATE, status)`, con `status` en
    'Activo' o 'Finalizado' y `UNIQUE (project_id, number)` (migración 004);
  - `sprint_story_assignments(sprint_id, story_id)` con `UNIQUE (story_id)`;
  - `user_stories(id BIGSERIAL, project_id, title, story_points, status, …)`;
  - `sprint_metrics(sprint_id INT PK → sprints ON DELETE CASCADE, velocity, …)` (migración 007).
- **Web (HU-20a/HU-20b)**:
  - `Server.Register(Section, Routes)` (`server.go:52`) protege con `RequireContext`, y `Routes` es
    `map[string]http.Handler` (`server.go:48`);
  - `Section{Name, Path, RequiresProject, RequiresMember}` (`sections.go:10`); el menú resalta la sección también en
    las direcciones que empiezan con `Path + "/"` (`sections.go:66`), así que `/sprints-anteriores/5` resalta
    "Sprints anteriores";
  - `parseID(s string) int64` (`context.go:92`) devuelve 0 para vacío, no numérico, cero, negativo, con signo, con
    ceros a la izquierda o mayor que `MaxID` (2147483647);
  - `projectIDFrom(r)` (`board_handlers.go:46`), `ActiveContextFrom` (`render.go:52`);
  - `Renderer.Page/PageStatus` (`render.go:102`, `107`); `Renderer.Fail` (`errors.go:115`) responde 404 con
    `NotFound`, 503 con `IsDatabaseUnavailable` y 500 en los demás casos;
  - `NotFound` (`errors.go:47`), `KindOf` (`errors.go:56`);
  - `Server.route` responde 405 con la página de HU-20a cuando el patrón existe con otro método;
  - `returnDestination` (`context_middleware.go:100`) vuelve a `r.URL.RequestURI()` en un `GET`;
  - `layout.html` no tiene formularios ni `<script>`.
- **`sprint.Done`** = "Hecho" (`src/sprint/model.go:50`).
- No existen en `src/` ni en `test/`: la ruta `/sprints-anteriores`, el texto "Sprints anteriores" ni nombres
  `PastSprint*`/`pastSprint*`. Las secciones "Sprints" (HU-23) y "Métricas" (HU-25) todavía no están en `main.go`.
  "/sprints-anteriores" no empieza con "/sprints/", así que no se cruza con el resaltado del menú de HU-23.
- **`main.go`**: registra cada sección con un bloque; el último hoy es el de HU-24 (`RegisterEffort`) y no importa
  `src/metrics`.
- **CI** (`.github/workflows/ci.yml`): `gofmt -l .`, `go vet ./...`, migraciones `up` y
  `go test ./... -p 1 -race -coverprofile`.
- **HU-09** (`specs/009-pdf-report/spec.md`, "Datos de ejemplo"): fechas de «Alfa» (18/08/2026 al 30/11/2026), de los
  tres Sprints y de «Beta» (01/10/2026 al 31/12/2026), y el orden de creación de las historias.

## Decisiones

### 1. Catálogo propio de solo lectura

- **Decision**: en `src/web/past_sprint_catalog.go`, la interfaz `PastSprintCatalog` con dos lecturas:
  - `FinishedSprints(ctx, projectID)`: los Sprints 'Finalizado' del proyecto, por `number DESC`;
  - `SprintDetail(ctx, projectID, sprintID)`: el Sprint (solo si es del proyecto y está 'Finalizado'; si no,
    `NotFound`) y las filas de `sprint_story_results` (T001: HU-15) unidas a `user_stories` por el título, en una
    transacción `ReadOnly` + `REPEATABLE READ`.

  Adaptador `PostgreSQLPastSprintCatalog`.
- **Rationale**: es una decisión del equipo (R9, FR-011). Por el principio VI, el único puerto nuevo con base tiene
  adaptador PostgreSQL y prueba de integración. Filtrar por proyecto y estado en la misma consulta da el 404 de los
  tres casos (inexistente, otro proyecto, en curso) sin revelar cuál fue.
- **Alternatives considered**:
  - leer las historias con HU-07 (solo ve las asignadas, y las devueltas ya no lo están);
  - un catálogo de HU-23 (todavía no está, y solo muestra el Sprint en curso);
  - agregar la lectura a `src/sprint` (edita un dominio ajeno sin necesidad: es una consulta de pantalla).

### 2. Velocidad con HU-07, detrás de una interfaz chica

- **Decision**: en `src/web`:

  ```go
  type PastSprintMetrics interface {
      CalculateMetrics(sprintID int64) (metrics.SprintMetrics, error)
  }
  ```

  `*metrics.Service` la cumple sin adaptador. El handler usa `Velocity`, y en `main.go` se pasa
  `metrics.NewService(metrics.NewPostgreSQLMetricsStore(database))`.
- **Rationale**: es una decisión del equipo (FR-009), como en HU-25. Con la interfaz, las pruebas usan un doble que
  falla (escenario 9). No se agrega un adaptador porque no hay tipos que traducir.
- **Consecuencia documentada**: cada apertura del detalle vuelve a guardar en `sprint_metrics` los mismos valores (el
  `upsert` de HU-07), como en HU-25, HU-08 y HU-09. Las pruebas borran esas filas en la limpieza.
- **Alternatives considered**:
  - sumar la velocidad con las completadas del registro (recalcula una regla de HU-07);
  - leer `sprint_metrics` directamente (puede no existir la fila si nadie la calculó).

### 3. Orden de los pasos del detalle y 404 antes de calcular

- **Decision**: `GET /sprints-anteriores/{id}` sigue estos pasos:
  1. `parseID(r.PathValue("id"))`; si da 0 → `Fail(NotFound(...))`, sin consultar la base.
  2. `catalog.SprintDetail`; con `NotFound` → 404, y con otro error → `Fail`.
  3. `metrics.CalculateMetrics(id)`; con error → `Fail`.
  4. Se arma la vista y se llama a `Page`.

  No se escribe nada antes del paso 4.
- **Rationale**: es una decisión del equipo (FR-010, FR-011). Como `CalculateMetrics` no filtra por proyecto, solo
  se llama con un Sprint que el catálogo ya validó para el proyecto activo. Si HU-07 falla, no se muestran las tablas
  sin la velocidad.
- **Alternatives considered**: calcular primero y validar después (expone o calcula Sprints ajenos).

### 4. Separar completadas y no completadas en la vista

- **Decision**: el catálogo devuelve las filas en un solo slice, ordenado por `story_id`. La vista
  (`buildPastSprintDetailView`) las separa por `FinalStatus == string(sprint.Done)` y conserva el orden.
- **Rationale**: R4 y R6. Se separan con una sola consulta, y la regla es una función pura que se prueba sin base.
- **Alternatives considered**: dos consultas filtradas (más viajes a la base, la misma información).

### 5. Coherencia entre la velocidad y el registro

- **Decision**: la pantalla muestra la velocidad de HU-07 aunque no coincida con la suma del registro.
  `TestHU16_VelocidadCoherente` comprueba que, con datos cargados como los deja HU-15, son iguales en los Sprints 1 y
  2 de «Alfa» (3 y 10) y en el Sprint 1 de «Gamma» (0).
- **Rationale**: es una decisión del equipo (no se recalcula). HU-07 lee las historias "Hecho" asignadas y el
  registro guarda las que estaban "Hecho" al cerrar. Después de HU-15 son las mismas, porque solo se borran las
  asignaciones de las no completadas y HU-13 y HU-17 bloquean la edición. La prueba cubre SC-002 y avisa si HU-15 o
  HU-07 cambian ese supuesto.
- **Alternatives considered**: mostrar un aviso si no coinciden (no lo pide ninguna historia y depende de datos
  cargados a mano).

### 6. Sprint sin registro

- **Decision**: si `SprintDetail` no trae filas, la vista muestra solo `MsgPastSprintNoRecord` y la velocidad. Ni
  "Todas…" ni "Ninguna…".
- **Rationale**: es la nota 5 del equipo. Sin registro no se sabe qué se planificó; "Ninguna historia se completó"
  sería falso si HU-07 encuentra historias "Hecho" (como en el Sprint 2 de «Gamma», con velocidad 2).

### 7. Errores

- **Decision**:
  - el adaptador devuelve `NotFound(...)` si no encuentra el Sprint y envuelve los demás errores con `%w`;
  - los errores de HU-07 llegan sin tocar a `Fail`: `ErrSprintNotFinished` o `ClosureValidationError` dan 500, y
    un error del driver da 503 porque `metricsStoreError` tiene `Unwrap`.
- **Rationale**: escenario 9 y FR-011. `Fail` ya oculta el detalle técnico.

### 8. Textos y formato

- **Decision**: en `src/web/past_sprint_view.go`:

  | Identificador | Texto |
  |---|---|
  | `TitlePastSprints` | "Sprints anteriores" |
  | `MsgPastSprintsEmpty` | "Aún no hay Sprints anteriores" |
  | `msgPastSprintLabel` | "Sprint %d" |
  | `msgPastSprintGoal` | "Objetivo: %s" |
  | `msgPastSprintDates` | "Del %s al %s" |
  | `msgPastSprintVelocity` | "Velocidad: %d Story Points" |
  | `MsgPastSprintAllCompleted` | "Todas las historias se completaron" |
  | `MsgPastSprintNoneCompleted` | "Ninguna historia se completó" |
  | `MsgPastSprintNoRecord` | "No hay registro de las historias de este Sprint" |
  | `LabelPastSprintUnestimated` | "Sin estimar" |
  | `LabelPastSprintReturned` | "volvió al Backlog" |
  | `LinkPastSprintsBack` | "Volver a Sprints anteriores" |

  También los encabezados "Completadas" y "No completadas" y las columnas "Historia", "Story Points" y "Estado al
  cerrar". Las fechas van en `02/01/2006`.
- **Rationale**: FR-015, un solo lugar. Las fechas tienen el mismo formato que `projects.html` y que HU-12.

### 9. Rutas y registro

- **Decision**: `RegisterPastSprints(s *Server, catalog PastSprintCatalog, calc PastSprintMetrics) error`:
  - con un argumento `nil` devuelve error, sin registrar;
  - registra `Section{Name: TitlePastSprints, Path: "/sprints-anteriores", RequiresProject: true}` con
    `Routes{"GET /sprints-anteriores": list, "GET /sprints-anteriores/{id}": detail}`.

  En `main.go` se agregan el import de `src/metrics` y un bloque después del último registro, con el mismo manejo de
  error que los demás.
- **Rationale**: es una decisión del equipo. El 405 lo da HU-20a porque solo hay rutas `GET`. El import es parte del
  mismo bloque; ningún otro archivo compartido cambia.

### 10. Pruebas con datos propios, cargados como los deja HU-15

- **Decision**: `src/web/past_sprint_fixtures_test.go` crea los proyectos de "Datos de ejemplo" sin pasar por HU-15.
  Inserta directamente:
  - los Sprints con su estado;
  - las historias con su estado **actual** («Reportes», «Migración» y «Respaldo» en 'Pendiente', sin asignación ni
    `active_sprint_id`);
  - las asignaciones de las "Hecho" y de las del Sprint 3 en curso;
  - las filas de `sprint_story_results` (T001: HU-15) de los Sprints 1 y 2 de «Alfa» y del Sprint 1 de «Gamma».

  Limpieza en `t.Cleanup`, en este orden:
  1. `sprint_story_results` (T001: HU-15);
  2. `sprint_story_assignments`;
  3. `sprint_metrics`;
  4. `user_stories`;
  5. `sprints`;
  6. `projects`.
- **Rationale**: HU-15 todavía no está, y las pruebas no dependen de su servicio. Los datos quedan exactamente como
  dice su `data-model.md`. El CI corre con `-p 1` y hace `TRUNCATE`; cada prueba arma lo suyo. Se borra
  `sprint_metrics` porque `CalculateMetrics` la escribe al abrir el detalle. Las historias van antes que el proyecto
  porque `user_stories.project_id` no tiene cascada.
- **Alternatives considered**:
  - cerrar los Sprints con el servicio de HU-15 (acopla las pruebas a su API, que puede cambiar);
  - reutilizar las fixtures de HU-09 (no está en `main`).

### 11. Rendimiento

- **Decision**: `TestHU16_Rendimiento` arma un proyecto con 50 Sprints finalizados y, en uno de ellos, 200 filas en el
  registro (150 "Hecho" asignadas y 50 devueltas). Exige que la lista y ese detalle respondan en menos de 500 ms cada
  uno.
- **Rationale**: SC-004. La lista es una consulta. El detalle son dos consultas, más la lectura y el guardado de
  HU-07.

## Preguntas abiertas

Ninguna.

## Resultado de T001 (01/11/2026, sobre `main` en `fd82929`)

- **HU-15 está en `main`** (PR #59) con la migración `014_sprint_story_results`: columnas `sprint_id`, `story_id`,
  `final_status` y `story_points` (`NULL` = sin estimar). El `CHECK` de `final_status` admite también 'Pendiente'
  (cuatro estados); para HU-16 sigue valiendo que 'Hecho' = completada y cualquier otro = no completada.
- **El cierre de HU-15** (`PostgreSQLSprintCloser`) escribe una fila por historia del Sprint, borra la asignación de
  las que no están "Hecho" y las pasa a 'Pendiente' con `active_sprint_id = NULL`; las "Hecho" conservan su asignación
  y su `active_sprint_id`. Coincide con §10 y con los datos de prueba de `data-model.md`.
- **HU-07**: siguen `metrics.NewService`, `metrics.NewPostgreSQLMetricsStore` y `(*Service).CalculateMetrics(sprintID
  int64) (SprintMetrics, error)` con `Velocity int`. Lee las historias asignadas al Sprint, exige que esté
  'Finalizado', valida las "Hecho" (`ValidateSprintClosure`) y hace el `upsert` en `sprint_metrics`.
- **HU-20a/b y HU-14**: están `Server.Register`, `Section.RequiresProject`, `Routes`, `parseID`, `projectIDFrom`,
  `Renderer.Page`, `Renderer.Fail`, `NotFound`, `KindOf` y `sprint.Done`.
- **No existen** la ruta `/sprints-anteriores`, el texto "Sprints anteriores" ni nombres `PastSprint*` en `src/web`.
- **`main.go` ya importa `src/metrics`** (lo usa el validador de cierre de HU-15), así que T015 solo agrega el bloque
  de registro, después del de HU-25 (`RegisterMetrics`), que hoy es el último.
- HU-25 (PR #64) muestra la velocidad de HU-07 con el mismo criterio (lee y no guarda nada propio); HU-16 usa
  `CalculateMetrics` directamente, como indica la nota 3 para el equipo.
