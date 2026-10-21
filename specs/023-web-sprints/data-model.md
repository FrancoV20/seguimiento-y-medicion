# Data Model: Interfaz web de Sprints

Esta feature **no crea tablas ni migraciones**. La creación, la asignación y el quitar usan el esquema
de HU-04 (migración `004`) a través del servicio y el store existentes. Las lecturas de pantalla usan
`sprints`, `sprint_story_assignments`, `user_stories` y `effort_logs`.

## Esquema leído

- `sprints(id BIGSERIAL PK, project_id BIGINT, number INTEGER, goal TEXT, start_date DATE, end_date
  DATE, status VARCHAR(20) DEFAULT 'Activo', created_at)`: estado `'Activo'` o `'Finalizado'`,
  `end_date > start_date`, `UNIQUE (project_id, number)`.
- `projects.next_sprint_number`: contador del próximo número (lo usa HU-04; esta feature no lo toca).
- `sprint_story_assignments(sprint_id, story_id, created_at)`: `PRIMARY KEY (sprint_id, story_id)` y
  `UNIQUE (story_id)`: una historia tiene como máximo una asignación.
- `user_stories`: además de lo de HU-02, `active_sprint_id BIGINT` y estados válidos `'Pendiente'`,
  `'En Sprint'`, `'En progreso'` y `'Hecho'`.
- `effort_logs`: registros de horas de HU-03, con una referencia a la historia.

> **A verificar (T004)**: el nombre exacto de la columna de historia en `effort_logs` (se supone
> `story_id`) y que HU-04 mantenga `sprint_story_assignments` y `user_stories.active_sprint_id`
> consistentes (T001).

Notas:

- Los identificadores son `BIGINT`; en Go se usan como `int64`.
- `user_stories.project_id` no tiene borrado en cascada; los Sprints y sus asignaciones sí se borran
  con el proyecto o con la historia. Las pruebas borran primero las historias y después el proyecto.

## SprintForm

Datos del envío de creación, para conservarlos al volver a mostrar el formulario.

### Fields

- `goal`: texto del objetivo.
- `startDate`, `endDate`: texto de cada fecha tal como llegó.
- `storyIDs`: identificadores de historias marcadas, como texto.

### Rules

- Los valores se devuelven tal como se enviaron cuando hay errores.
- Un campo ausente en el envío se trata como vacío.
- No hay campo de identificador de Sprint: el identificador lo genera el sistema.

## SprintFormInput

Resultado de armar el envío para invocar la creación.

### Fields

- `projectID`: tomado del contexto del servidor.
- `goal`, `startDate`, `endDate`: fechas de calendario (`AAAA-MM-DD`).
- `storyIDs`: identificadores numéricos sin repetidos (puede estar vacío).

### Rules

- Una fecha vacía o con formato distinto de `AAAA-MM-DD` genera un error junto a su campo.
- Un identificador de historia no numérico genera un error junto a la lista.
- Los repetidos se toman una sola vez.
- Las demás reglas (Goal, período, elegibilidad, exclusividad, Sprint activo único) las aplica HU-04.

## StoryOption

Historia del Backlog mostrada en la lista de asignación.

### Fields

- `id`, `title`, `priority`, `storyPoints` (opcional), `status`.
- `assignedSprintNumber`: número del Sprint al que pertenece, si tiene asignación.
- `available`: verdadero si está en estado `'Pendiente'` y sin asignación.
- `unavailableReason`: "Ya está asignada al Sprint N" o "No está pendiente"; vacío si está disponible.

### Rules

- Se ordenan por `id` descendente.
- Solo incluye historias del proyecto pedido.

## SprintView

Sprint activo del proyecto con sus historias.

### Fields

- `id`, `number` y `label` ("Sprint N").
- `goal`, `startDate`, `endDate`.
- `stories`: lista de `SprintStoryRow`.

### Rules

- Si por datos anteriores a FIX-HU-04 hubiera más de un Sprint activo, es el de mayor `number`.

## SprintStoryRow

Historia asignada al Sprint activo.

### Fields

- `id`, `title`, `storyPoints` (opcional), `status`.
- `hasEffort`: verdadero si la historia tiene horas registradas.
- `canRemove`: verdadero si `status` es `'En Sprint'` y no tiene horas registradas.
- `removeBlockedReason`: "Ya está en progreso", "Ya está hecha" o "Tiene horas registradas".

### Rules

- Las filas se ordenan por `id` ascendente.
- `canRemove` y `removeBlockedReason` se obtienen con `CanRemoveStory(status, hasEffort)`, la misma
  función pura del dominio (`src/sprint`) que usa `RemoveStory` para validar.

## FormAction

Acciones de los formularios de la pantalla.

### Values

- Crear: `POST /sprints`.
- Agregar: `POST /sprints/historias` con una o más `historia`.
- Quitar: `POST /sprints/historias/quitar` con `historia_id`.

### Rules

- Agregar sin ninguna `historia` informa "Elija al menos una historia" y no cambia nada.
- Quitar solo actúa sobre una historia del Sprint activo del proyecto activo.

## FieldErrors

Mensajes de error por campo para `FormView` de HU-20b.

### Fields

- Claves: `objetivo`, `fecha_inicio`, `fecha_fin`, `historia`.
- Valores: texto del mensaje.

### Messages

- Goal obligatorio, inconsistencia temporal e historia no disponible: los de HU-04, sin cambios.
- Fecha faltante: "Falta la fecha de inicio" o "Falta la fecha de fin" (`MsgMissingStartDate`,
  `MsgMissingEndDate`). Formato inválido: "La fecha no tiene un formato válido"
  (`MsgInvalidDateFormat`).
- Ya hay un Sprint activo: viene de `ErrActiveSprintExists` (FIX-HU-04), con el número del activo:
  "Ya hay un Sprint activo (Sprint N)".
- Sin Sprint activo: "No hay un Sprint activo" (`MsgNoActiveSprint`).
- Éxito de crear: `MsgSprintCreated`; con advertencia si no hay historias: `MsgSprintWithoutStories`.
- Éxito de agregar: `MsgStoriesAdded`. Éxito de quitar: `MsgStoryRemoved`.
- Sin selección al agregar: `MsgSelectAtLeastOneStory`.
- Historia no removible: el motivo del dominio ("Ya está en progreso", "Ya está hecha" o "Tiene horas
  registradas").
- Falla de guardado: `MsgOperationFailed` de HU-20b.

## SprintCatalog

Interfaz interna de solo lectura para la pantalla.

### Operations

- `ActiveSprint(ctx, projectID)`: devuelve el `SprintView` activo del proyecto, o ninguno.
- `ListStoryOptions(ctx, projectID)`: devuelve los `StoryOption` del proyecto.

### Rules

- Distingue errores de lectura de resultados vacíos: sin Sprint activo o sin historias no es un error.

## PostgreSQLSprintCatalog

Implementación concreta de `SprintCatalog` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` mediante `database/sql`.

### Queries

```sql
-- ActiveSprint: el de mayor número
SELECT id, number, goal, start_date, end_date
FROM sprints
WHERE project_id = $1 AND status = 'Activo'
ORDER BY number DESC
LIMIT 1;

-- Historias del Sprint activo (con indicador de horas)
SELECT us.id, us.title, us.story_points, us.status,
       EXISTS (SELECT 1 FROM effort_logs el WHERE el.story_id = us.id) AS has_effort
FROM sprint_story_assignments a
JOIN user_stories us ON us.id = a.story_id
WHERE a.sprint_id = $1
ORDER BY us.id;

-- ListStoryOptions
SELECT us.id, us.title, us.priority, us.story_points, us.status, s.number
FROM user_stories us
LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
LEFT JOIN sprints s ON s.id = a.sprint_id
WHERE us.project_id = $1
ORDER BY us.id DESC;
```

### Rules

- Solo realiza lecturas; la creación, la asignación y el quitar pasan por HU-04.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse; las pruebas crean sus
  propios proyectos, historias, Sprints y horas y los eliminan al terminar, borrando primero las
  historias.

## SprintActions

Interfaz mínima de escritura que declara HU-23 y satisfacen el servicio y el store de HU-04.

### Operations

- `CreateSprint(ctx, request)`: crea el Sprint y asigna sus historias; rechaza con
  `ErrActiveSprintExists` si el proyecto ya tiene uno activo (FIX-HU-04).
- `AssignStories(ctx, sprintID, storyIDs)`: asigna historias disponibles a un Sprint existente.
- `RemoveStory(ctx, sprintID, storyID)`: quita una historia y la devuelve a `'Pendiente'`; valida
  estado y horas (T047).

### Rules

- Las firmas exactas se toman de HU-04 en T001; los handlers dependen de esta interfaz y no del tipo
  concreto.