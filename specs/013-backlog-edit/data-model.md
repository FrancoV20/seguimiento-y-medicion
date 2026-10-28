# Modelo de datos: HU-13 – Consulta y Edición del Product Backlog (web)

## Esquema

Sin migración nueva. HU-13 usa estas tablas existentes:

| Tabla | Columnas usadas | Operación de HU-13 |
|---|---|---|
| `user_stories` | `id`, `project_id`, `title`, `description`, `priority`, `story_points`, `estimated_hours`, `status`, `created_at` | lee; actualiza `title`, `description`, `priority` y `story_points` |
| `acceptance_criteria` | `story_id`, `position`, `content` | lee; reemplaza los de la historia |
| `sprint_story_assignments` | `sprint_id`, `story_id` | lee (Sprint de la historia) |
| `sprints` | `id`, `status` | lee con `FOR SHARE` (¿'Finalizado'?) |

`status`, `estimated_hours`, `active_sprint_id` y `created_at` no se escriben (FR-012). Los `CHECK` de
`priority` y `story_points` son la segunda barrera de R4.

## Dominio (`src/backlog`)

### Estados (`model.go`)

```go
const (
    StatusPending    StoryStatus = "Pendiente" // ya existe
    StatusInSprint   StoryStatus = "En Sprint"
    StatusInProgress StoryStatus = "En progreso"
    StatusDone       StoryStatus = "Hecho"
)

func StoryStatuses() []StoryStatus // Pendiente, En Sprint, En progreso, Hecho
```

### Edición (`update.go`)

```go
type UpdateStoryRequest struct {
    ProjectID          int64
    StoryID            int64
    Title              string
    Description        string
    Priority           Priority
    AcceptanceCriteria []string
    StoryPoints        *int // nil = sin estimar
}

type UpdateStoryResult struct {
    Story   *UserStory
    Warning string // WarnEstimationPending si queda sin Story Points
}
```

### Errores nuevos (`update_errors.go`)

| Error | Cuándo |
|---|---|
| `ErrStoryNotFound` | `ProjectID` o `StoryID` menor o igual a 0, o la historia no existe en ese proyecto |
| `ErrStoryInFinishedSprint` | la historia está asignada a un Sprint 'Finalizado' |

Se reutilizan `ErrTitleRequired`, `ErrDescriptionRequired`, `ErrInvalidPriority`, `ErrAcceptanceCriteriaRequired`,
`ErrInvalidStoryPoints` y `ErrPersistence`.

### Validación compartida (`service.go`)

`validateStoryFields(title, description string, priority Priority, criteria []string, storyPoints *int)
(storyFields, error)`, extraída de `CreateStory`:

- recorta título, descripción y criterios;
- descarta los criterios vacíos y numera los demás desde 1;
- devuelve el primer error en este orden: `ErrTitleRequired`, `ErrDescriptionRequired`, `ErrInvalidPriority`,
  `ErrAcceptanceCriteriaRequired`, `ErrInvalidStoryPoints`.

`CreateStory` sigue validando el proyecto antes y las horas estimadas después.

### Servicio (`update_service.go`)

`func (s *BacklogService) UpdateStory(ctx context.Context, req UpdateStoryRequest) (*UpdateStoryResult, error)`:

1. `req.ProjectID <= 0 || req.StoryID <= 0` → `ErrStoryNotFound`. No se llama a `ProjectIsActive`: la edición no
   revisa el estado guardado del proyecto (research, decisión 17).
2. `validateStoryFields(...)` → errores de HU-02.
3. `s.store.UpdateStory(ctx, req.ProjectID, UserStory{ID, Title, Description, Priority, AcceptanceCriteria,
   StoryPoints})`.
4. `ErrStoryNotFound` y `ErrStoryInFinishedSprint` se devuelven sin cambios; cualquier otro error se traduce a
   `ErrPersistence`, como en el alta.
5. Si `StoryPoints == nil`, `Warning = WarnEstimationPending`.

### Puerto (`store.go`)

```go
type BacklogStore interface {
    ProjectIsActive(projectID int64) (bool, error)
    CreateStory(projectID int64, story UserStory) (UserStory, error)
    UpdateStory(ctx context.Context, projectID int64, story UserStory) error // HU-13
}
```

`InMemoryBacklogStore` suma:

- `UpdateStory`, que reemplaza la historia de `Stories` con ese `ID` y `ProjectID`, o devuelve `ErrStoryNotFound`;
- `UpdateErr error`, para devolver un error de dominio en las pruebas;
- `FailOnUpdate bool` y `UpdateCalls int`.

## Transacción de guardado (`PostgreSQLBacklogStore.UpdateStory`)

`BeginTx(ctx, nil)` (`READ COMMITTED`), `defer tx.Rollback()`:

```sql
-- 1. Historia del proyecto y su Sprint, sin bloquear (sin filas → ErrStoryNotFound)
SELECT a.sprint_id
FROM user_stories us
LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
WHERE us.id = $1 AND us.project_id = $2;

-- 2. Solo si sprint_id no es nulo: bloquear el Sprint primero (mismo orden que HU-04, HU-23 y HU-15)
SELECT status FROM sprints WHERE id = $1 FOR SHARE;    -- 'Finalizado' → ErrStoryInFinishedSprint

-- 3. Bloquear la historia y confirmar que la asignación no cambió
SELECT a.sprint_id
FROM user_stories us
LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
WHERE us.id = $1 AND us.project_id = $2
FOR NO KEY UPDATE OF us;
--    sin filas → ErrStoryNotFound (la borraron);
--    sprint_id distinto del paso 1 → ROLLBACK y repetir desde el paso 1 una sola vez;
--    distinto otra vez → errAssignmentChanged (→ ErrPersistence, 500)

-- 4. Datos de la historia
UPDATE user_stories
SET title = $2, description = $3, priority = $4, story_points = $5
WHERE id = $1;

-- 5. Criterios completos, posiciones 1..n
DELETE FROM acceptance_criteria WHERE story_id = $1;
INSERT INTO acceptance_criteria (story_id, position, content) VALUES ($1, $2, $3); -- una por criterio

COMMIT;
```

- Un guardado sin cambios ejecuta todos los pasos y confirma: es un éxito.
- Un error en cualquier paso deshace todo (R8).
- `story_points` nulo se pasa como `sql.NullInt64{}`.

Orden de bloqueos con el cierre de Sprint (HU-15), que hace `FOR UPDATE` sobre el Sprint y `FOR UPDATE OF us` sobre
sus historias:

| Llega primero | Qué pasa |
|---|---|
| La edición | Toma `FOR SHARE` sobre el Sprint. El cierre espera su `COMMIT` y calcula con los datos editados. |
| El cierre | Toma `FOR UPDATE` sobre el Sprint. La edición espera en el paso 2, ve 'Finalizado' y responde 409. |

## Lectura de pantalla (`src/web`)

### Lista (`backlog_catalog.go`, `postgres_story_catalog.go`)

`StoryRow` suma `InFinishedSprint bool`.

```sql
SELECT us.id, us.title, us.priority, us.story_points, us.estimated_hours, us.status,
       COALESCE(s.status = 'Finalizado', false) AS in_finished_sprint
FROM user_stories us
LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
LEFT JOIN sprints s ON s.id = a.sprint_id
WHERE us.project_id = $1
ORDER BY CASE us.priority WHEN 'Alta' THEN 1 WHEN 'Media' THEN 2 ELSE 3 END,
         us.created_at, us.id;
```

### Formulario (`StoryForEdit`)

```go
type StoryEdit struct {
    ID               int64
    Title            string
    Description      string
    Priority         string
    StoryPoints      *int
    Criteria         []string // ordenados por position
    InFinishedSprint bool
}

type StoryCatalog interface {
    ListStories(ctx context.Context, projectID int64) ([]StoryRow, error)
    StoryForEdit(ctx context.Context, projectID, storyID int64) (StoryEdit, error) // HU-13
}
```

```sql
SELECT us.title, us.description, us.priority, us.story_points,
       COALESCE(s.status = 'Finalizado', false), ac.content
FROM user_stories us
LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
LEFT JOIN sprints s ON s.id = a.sprint_id
LEFT JOIN acceptance_criteria ac ON ac.story_id = us.id
WHERE us.id = $1 AND us.project_id = $2
ORDER BY ac.position;
```

Sin filas: `NotFound`. Error de la base: `Unavailable`. Un `ac.content` nulo (historia sin criterios): `Criteria`
vacío, y el formulario muestra un campo vacío.

## Modelo de vista (`src/web`)

```go
// backlogPageData (HU-22) suma:
Filter       string   // estado elegido; "" = Todos
Statuses     []string // Pendiente, En Sprint, En progreso, Hecho
FilterError  string   // MsgInvalidStatusFilter o ""
NoResults    string   // MsgNoStoriesWithStatus o ""

// backlogStoryView (HU-22) suma:
EditURL          string // /backlog/{id}/editar[?estado=…]
InFinishedSprint bool

type backlogEditPageData struct {
    ID                 int64
    Filter             string // campo oculto "estado"
    BackURL            string // backlogListURL(Filter), para "Cancelar"
    Criteria           []backlogCriterionField
    Priorities         []string
    StoryPoints        []int
    InvalidPriority    string
    InvalidStoryPoints string
}
```

`backlogListURL(estado string) string` devuelve `/backlog?estado=` más el valor escapado si `estado` está en
`StoryStatuses()`; si no, `/backlog`.

## Textos (`src/web/backlog_form.go`)

| Constante | Texto |
|---|---|
| `TitleStoryEdit` | "Editar historia" |
| `MsgStoryUpdated` | "Historia modificada correctamente" |
| `MsgStoryUpdatedMissingPoints` | "Historia modificada correctamente. Falta completar la estimación de Story Points" |
| `MsgStoryInFinishedSprint` | "La historia pertenece a un Sprint finalizado; no se puede editar" |
| `MsgInvalidStatusFilter` | "El estado no es válido" |
| `MsgNoStoriesWithStatus` | "No hay historias con ese estado" |
| `LabelEditStory` | "Editar" |
| `LabelFinishedSprint` | "Sprint finalizado" |
| `LabelAllStatuses` | "Todos" |
| `BacklogFieldStatus` | "estado" |

## Mapeo de respuestas

| Caso | Respuesta | Texto o destino |
|---|---|---|
| `GET` con `{id}` inválido, inexistente o de otro proyecto | 404 | `Fail(NotFound(...))` |
| `GET` de una historia de un Sprint finalizado | 303 a `backlogListURL(estado)` | `FlashWarning`, `MsgStoryInFinishedSprint` |
| `POST` con `{id}` inválido, inexistente o de otro proyecto | 404 | `Fail(NotFound(...))` |
| `POST` de una historia de un Sprint finalizado (lectura o transacción) | 409 | formulario con los datos enviados y `Flash{FlashError, MsgStoryInFinishedSprint}` |
| `accion` agregar o quitar | 200 | formulario con el cambio |
| errores de `parseStoryFields`, falta de criterios y errores de HU-02 | 422 | `mapBacklogDomainError`, junto al campo |
| `ErrStoryNotFound` desde la transacción | 404 | `Fail(NotFound(...))` |
| otro error | 500 | formulario y `Flash{FlashError, MsgOperationFailed}` |
| éxito con Story Points | 303 a `backlogListURL(estado)` | `FlashSuccess`, `MsgStoryUpdated` |
| éxito sin Story Points | 303 a `backlogListURL(estado)` | `FlashWarning`, `MsgStoryUpdatedMissingPoints` |
