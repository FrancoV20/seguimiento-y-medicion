# Research: HU-13 – Consulta y Edición del Product Backlog (web)

## Datos verificados en el código (main, 2026-10-27)

- **`src/backlog`, el dominio de HU-02**:
  - `BacklogService` tiene solo `CreateStory(req CreateStoryRequest)`, sin `context`.
  - `CreateStory` valida en este orden: proyecto activo, título, descripción, prioridad, criterios (descarta los
    vacíos y numera desde 1), Story Points (`ValidateStoryPoints`) y horas estimadas. Recorta título, descripción
    y criterios. Cualquier error del almacén se traduce a `ErrPersistence`.
  - El puerto `BacklogStore` tiene `ProjectIsActive(projectID int64)` y `CreateStory(projectID int64, story
    UserStory)`. Lo implementan `PostgreSQLBacklogStore` e `InMemoryBacklogStore` (con `FailOnCreate` y
    `CreateCalls`).
  - Fakes fuera del paquete: `hu22FailingBacklogStore` (`src/web/backlog_handlers_test.go`) y
    `hu22BDDFailingBacklogStore` (`test/web_backlog_bdd_test.go`).
  - Constantes de estado: solo `StatusPending = "Pendiente"`. Las de prioridad son `PriorityHigh`,
    `PriorityMedium` y `PriorityLow`, con `Priority.IsValid()`.
  - Mensajes (`errors.go`), en minúscula:
    - `ErrTitleRequired`: "el título es obligatorio";
    - `ErrDescriptionRequired`: "la descripción es obligatoria";
    - `ErrInvalidPriority`: "la prioridad debe ser Alta, Media o Baja";
    - `ErrAcceptanceCriteriaRequired`: "agregá al menos un criterio de aceptación";
    - `ErrInvalidStoryPoints`: "los Story Points deben ser 1, 2, 3, 5, 8, 13, 21, 34, 55 u 89".
- **`src/web`, la web de HU-22**:
  - `RegisterBacklog(s, creator *backlog.BacklogService, catalog StoryCatalog)` registra `Section{Name:
    "Backlog", Path: "/backlog", RequiresProject: true}` con `GET` y `POST /backlog`.
  - `parseBacklogForm`:
    - lee `titulo`, `descripcion`, `prioridad`, `story_points`, `horas_estimadas`, `criterio` (varios) y `accion`;
    - un valor no numérico en Story Points da `MsgInvalidStoryPoints` ("Los Story Points no son un valor
      válido");
    - un valor inválido en horas da `MsgInvalidEstimatedHours`.
  - `applyAction` maneja `agregar_criterio` y `quitar_criterio:N`: nunca deja menos de un campo y responde 200
    sin guardar.
  - El handler rechaza la falta de criterios antes de llamar al servicio.
  - `mapBacklogDomainError` asigna cada error a su campo. Para Story Points usa `MsgInvalidStoryPoints`; para
    título, descripción, prioridad y criterios, el texto del dominio.
  - Mensajes de HU-22:
    - `MsgStoryCreated`: "Historia creada correctamente";
    - `MsgStoryCreatedMissingPoints`: "Historia creada correctamente. Falta completar la estimación de Story
      Points", que se muestra como `FlashWarning`.
  - `StoryCatalog.ListStories(ctx, projectID)` devuelve `StoryRow{ID, Title, Priority, StoryPoints,
    EstimatedHours, Status}` ordenado por `id DESC`.
  - El único fake es `hu22FailingStoryCatalog`, sin embeber la interfaz (`src/web/backlog_handlers_test.go:769`).
  - `templates/backlog.html` tiene el formulario de alta y la tabla de historias. Si no hay historias, muestra
    "Aún no hay historias" con el formulario.
- **Pruebas de HU-22 que exigen el orden actual**:
  - `TestPostgreSQLStoryCatalogListsOnlyProjectStoriesInDescendingOrder`
    (`src/web/postgres_story_catalog_integration_test.go:17`);
  - `TestBacklogHandler_GetListsActiveProjectStoriesNewestFirst` (`src/web/backlog_handlers_test.go:664`);
  - el subtest "…recientes primero" de `TestWebBacklogBDD_BacklogListAndEmptyState`
    (`test/web_backlog_bdd_test.go:590`).

  HU-13 cambia el orden y tiene que actualizar las tres.
- **`main.go`**: construye `backlog.NewBacklogService(backlog.NewPostgreSQLBacklogStore(database))` y
  `web.NewPostgreSQLStoryCatalog(database)`, y llama a `web.RegisterBacklog`.
- **Esquema**:
  - `user_stories` tiene:
    - `id BIGSERIAL`;
    - `project_id BIGINT`;
    - `priority` con `CHECK` (Alta, Media, Baja);
    - `story_points` con `CHECK` sobre la escala;
    - `status` con `CHECK` (Pendiente, En Sprint, En progreso, Hecho; migración 004);
    - `estimated_hours`, `created_at` y `active_sprint_id`.
  - `acceptance_criteria` tiene `(story_id, position, content)`, con `UNIQUE (story_id, position)` y `ON DELETE
    CASCADE`.
  - `sprint_story_assignments` tiene `(sprint_id, story_id)`, con `UNIQUE (story_id)`: una historia está en un
    solo Sprint como máximo.
  - `sprints.status` puede ser 'Activo' o 'Finalizado'.
- **Bloqueos de otras historias**:
  - HU-04/HU-23 (`AssignStories`, `RemoveStory`) bloquean el Sprint con `FOR UPDATE` y después las historias con
    `FOR UPDATE OF us`.
  - El diseño de HU-15 (`specs/015-sprint-close/data-model.md`) hace lo mismo: primero el Sprint, después todas
    sus historias. Mantiene la asignación de las historias "Hecho" y devuelve las demás a "Pendiente" sin Sprint.
  - HU-14 cambia el estado con un `UPDATE` atómico sin bloquear el Sprint.
- **HU-15 no está en `main`**: hoy ningún flujo de la aplicación deja un Sprint "Finalizado". Las pruebas lo
  preparan en la base.
- **Contratos web**: `parseID` acepta enteros canónicos entre 1 y `MaxID` (`MaxInt32`).
  `Renderer.PageStatus` toma el mensaje pendiente de la cookie si `PageData.Flash` está vacío.
  `RequireContext` vuelve a la dirección pedida con `GET` (con la consulta) y a la sección con `POST`.

## Decisiones

### 1. Extender el puerto `BacklogStore` con `UpdateStory`

- **Decision**: `BacklogStore` suma `UpdateStory(ctx context.Context, projectID int64, story UserStory) error`, y
  `BacklogService` suma `UpdateStory(ctx, req) (*UpdateStoryResult, error)`.
- **Rationale**: es el criterio de HU-11. `main.go` y la firma de `RegisterBacklog` no cambian.
- **Costo**: `InMemoryBacklogStore` suma el método, y los dos fakes de HU-22 suman un método que devuelve error.
- **Alternatives considered**: un puerto y un servicio separados (descartada: obliga a cambiar `RegisterBacklog`,
  `main.go` y los ayudantes de prueba de HU-22).

### 2. Una sola validación para el alta y la edición

- **Decision**: extraer de `CreateStory` la función `validateStoryFields(title, description, priority, criteria,
  storyPoints)`, que devuelve los valores recortados, los criterios numerados o el primer error, en el orden actual
  (título, descripción, prioridad, criterios, Story Points). `CreateStory` y `UpdateStory` la usan.
- **Rationale**:
  - Principio II: no hay dos copias de la regla.
  - R4: misma validación y mismos mensajes que el alta.
  - Las pruebas de HU-02 cubren que el cambio no altere el alta.
- **Alternatives considered**: copiar las validaciones (descartada: pueden divergir).

### 3. Estados de la historia en el dominio

- **Decision**: agregar a `src/backlog/model.go` las constantes `StatusInSprint = "En Sprint"`,
  `StatusInProgress = "En progreso"` y `StatusDone = "Hecho"`, más `StoryStatuses()` con los cuatro estados en ese
  orden.
- **Rationale**:
  - El filtro necesita la lista de estados válidos de un solo lugar, igual que la prioridad.
  - Los valores coinciden con el `CHECK` de la migración 004.
- **Alternatives considered**: usar las constantes de `src/sprint` (descartada: acopla el Backlog al paquete de
  Sprints); escribir la lista en la web (descartada: dos fuentes).

### 4. Filtro aplicado en la web sobre la lista ordenada

- **Decision**: `ListStories` sigue devolviendo todas las historias del proyecto, ya ordenadas, y el handler filtra
  por estado en memoria.
- **Rationale**:
  - Con la lista completa se distingue "proyecto sin historias" ("Aún no hay historias") de "filtro sin
    resultados" ("No hay historias con ese estado") sin una segunda consulta.
  - El orden no cambia al filtrar.
  - El volumen es chico (SC-005: 200 historias).
  - La firma de `ListStories` no cambia.
- **Alternatives considered**: filtrar en SQL con un parámetro (descartada: cambia la firma y obliga a una consulta
  extra para saber si el Backlog está vacío).

### 5. Orden en la consulta

- **Decision**: `ORDER BY CASE us.priority WHEN 'Alta' THEN 1 WHEN 'Media' THEN 2 ELSE 3 END, us.created_at,
  us.id`.
- **Rationale**: decisión 1 del equipo. El `id` desempata historias creadas en el mismo instante.

### 6. "De un Sprint finalizado" en la lista y en la edición

- **Decision**:
  - `StoryRow` suma `InFinishedSprint bool`, calculado con `LEFT JOIN sprint_story_assignments` y `sprints`
    (`COALESCE(s.status = 'Finalizado', false)`).
  - La lectura del formulario devuelve el mismo dato.
- **Rationale**: `UNIQUE (story_id)` garantiza una fila como máximo, así que el `JOIN` no duplica historias.

### 7. Transacción de guardado con el orden de bloqueos de HU-04 y HU-15

- **Decision**: una transacción `READ COMMITTED` (ver `data-model.md`):
  1. leer la asignación de la historia del proyecto sin bloquear (sin filas → `ErrStoryNotFound`);
  2. si tiene Sprint, bloquearlo con `FOR SHARE` (si está 'Finalizado' → `ErrStoryInFinishedSprint`);
  3. bloquear la historia con `FOR NO KEY UPDATE` y volver a leer su asignación. Si cambió entre el paso 1 y el 3
     (otra persona la asignó o la quitó), se deshace y se repite una vez; si vuelve a cambiar, falla y responde
     500;
  4. actualizar título, descripción, prioridad y Story Points;
  5. borrar los criterios y escribirlos con las posiciones 1..n.
- **Rationale**:
  - Primero el Sprint y después la historia, el mismo orden que HU-04, HU-23 y HU-15: así no hay bloqueos
    cruzados.
  - El `FOR SHARE` sobre el Sprint choca con el `FOR UPDATE` del cierre de HU-15. Si el cierre llega primero, la
    edición espera y después ve 'Finalizado' (409). Si la edición llega primero, el cierre espera y la velocidad
    se calcula con los Story Points ya editados.
  - `FOR NO KEY UPDATE` alcanza porque no se cambia la clave, y choca con el `FOR UPDATE OF us` de HU-15 y HU-23.
  - Reemplazar los criterios completos es la decisión 7 del equipo.
- **Alternatives considered**:
  - Bloquear primero la historia (descartada: orden inverso al de HU-15, con riesgo de bloqueo cruzado).
  - Un solo `SELECT … FOR NO KEY UPDATE OF us FOR SHARE OF s` (descartada: PostgreSQL no garantiza el orden de los
    bloqueos dentro de una sentencia).
  - Actualizar los criterios fila por fila (descartada: con `UNIQUE (story_id, position)` las posiciones chocan
    al reordenar).

### 8. Lectura del formulario: `StoryForEdit`

- **Decision**: `StoryCatalog` suma `StoryForEdit(ctx, projectID, storyID int64) (StoryEdit, error)` en
  `PostgreSQLStoryCatalog`.
  - Una consulta con `LEFT JOIN acceptance_criteria`, ordenada por `position`.
  - Sin filas, `NotFound`; si falla la base, `Unavailable`.
- **Rationale**:
  - Es lectura de pantalla, como `ListStories`.
  - El filtro por `project_id` hace que una historia de otro proyecto sea "inexistente".
  - Hay que agregar un método al fake `hu22FailingStoryCatalog`.
- **Alternatives considered**: leer desde `BacklogStore` (descartada: mezcla la lectura de pantalla con el
  dominio, como decidieron HU-21 y HU-11).

### 9. Rutas en el mismo registro

- **Decision**: `GET` y `POST /backlog/{id}/editar` se agregan al mapa de `RegisterBacklog`. La sección sigue
  siendo la de HU-22 (`RequiresProject: true`).
- **Rationale**: `Register` rechaza una segunda sección con la misma dirección. Con `RequireContext`, el
  escenario 14 sale solo: un `GET` sin proyecto vuelve a la edición pedida.

### 10. Orden de las respuestas en el envío

- **Decision**:
  1. `parseID` (0 → 404);
  2. `StoryForEdit` (inexistente o de otro proyecto → 404);
  3. `ParseForm` y `parseStoryFields(r, false)`, solo para leer los datos, sin validarlos;
  4. Sprint finalizado → 409, con los datos leídos en el paso 3;
  5. "Agregar criterio" o "Quitar" → 200;
  6. errores de formato y falta de criterios → 422;
  7. `UpdateStory`, con sus errores mapeados (422, 404, 409 o 500).
- **Rationale**:
  - El 404 se decide antes de validar, según la convención del equipo.
  - Con el Sprint finalizado no tiene sentido validar, pero el 409 muestra el formulario con lo enviado (FR-020), así
    que el formulario se lee antes. Leerlo no cambia nada ni produce errores visibles.
  - La transacción vuelve a revisar el 404 y el 409 por si cambian entre la lectura y el guardado.

### 11. Abrir la edición de una historia de un Sprint finalizado (respuesta de Clarifications)

- **Decision**: el `GET` responde 303 a la lista (con el filtro de la decisión 12) y deja la advertencia
  `FlashWarning` con `MsgStoryInFinishedSprint`.
- **Rationale**: respuesta A de Clarifications. Usa `SetFlash` y `RedirectSeeOther` de HU-20b.

### 12. La edición conserva el filtro (respuesta de Clarifications)

- **Decision**:
  - El enlace "Editar" lleva `?estado=X` cuando hay filtro, y el formulario lo guarda en un campo oculto
    `estado`.
  - La dirección de regreso la arma `backlogListURL(estado)`: `/backlog?estado=` más el valor escapado si el
    estado es válido, o `/backlog` en cualquier otro caso.
- **Rationale**: respuesta A de Clarifications. Como el valor se valida contra `StoryStatuses()`, nunca se
  redirige fuera de `/backlog` y un valor manipulado no llega a la dirección.

### 13. Parseo del formulario de edición sin horas estimadas

- **Decision**: separar de `parseBacklogForm` una función `parseStoryFields(r, withHours bool)`. El alta la llama
  con `true`; la edición, con `false`, así que ignora `horas_estimadas`.
- **Rationale**: el parseo de título, Story Points, criterios y acción no se duplica, y la edición nunca muestra ni
  valida horas estimadas (decisión 3 del equipo).
- **Alternatives considered**: reutilizar `parseBacklogForm` y borrar después el error de horas (descartada: es
  frágil y oculta la intención).

### 14. Estado inválido en el filtro: aviso junto al filtro y 200

- **Decision**: un `estado` inválido responde 200 con la lista completa y "El estado no es válido" junto al
  selector del filtro (`Data.FilterError`), no en el `Flash`.
- **Rationale**:
  - Es una lectura: se muestra lo que se puede.
  - Si el aviso fuera al `Flash`, taparía un mensaje pendiente de un guardado anterior.

### 15. Textos en un solo lugar

- **Decision**: los textos nuevos van en `src/web/backlog_form.go`, junto a los de HU-22: `MsgStoryUpdated`,
  `MsgStoryUpdatedMissingPoints`, `MsgStoryInFinishedSprint`, `MsgInvalidStatusFilter`, `MsgNoStoriesWithStatus`,
  `LabelFinishedSprint`, `LabelEditStory`, `LabelAllStatuses`, `TitleStoryEdit` y `BacklogFieldStatus`. Los
  mensajes de validación salen de `mapBacklogDomainError`, que se reutiliza.
- **Rationale**: FR-023.

### 16. Rendimiento (SC-005)

- **Decision**: una prueba de handler con 200 historias, varias en un Sprint finalizado, mide la lista (con y sin
  filtro) y el formulario de edición contra la base local, con un umbral de 500 ms.

### 17. La edición no revisa el estado guardado del proyecto

- **Decision**: `UpdateStory` no llama a `ProjectIsActive`. Basta con que la historia exista dentro del proyecto
  activo (pasos 1 y 3 de la transacción).
- **Rationale**:
  - `projects.status` vale 'Activo' por defecto y ninguna historia lo cambia; HU-12 calcula el estado con las fechas.
  - El proyecto ya lo valida el middleware de contexto de HU-20b en cada solicitud.
  - Llamar a `ProjectIsActive` sumaría una consulta fuera de la transacción sin cambiar ningún resultado.
- **Alternatives considered**: exigir 'Activo', como el alta (descartada por ahora: no cambia ningún resultado y no
  hay regla del equipo que lo pida). Si un proyecto pudiera dejar de estar "Activo", hay que revisar esta decisión.

## Preguntas abiertas

Ninguna. Las dos dudas se respondieron en Clarifications (decisiones 11 y 12).
