# Data Model: Actualización de estado de historias (Tablero)

Esta feature **no crea tablas ni migraciones**. Usa `user_stories`, `sprint_story_assignments`,
`sprints` y `effort_logs`.

## Esquema usado

- `user_stories`: `id`, `project_id`, `title`, `story_points` (opcional), `status` con los valores
  `'Pendiente'`, `'En Sprint'`, `'En progreso'` y `'Hecho'`. Solo se actualiza `status`.
- `sprint_story_assignments(sprint_id, story_id)`: `UNIQUE (story_id)`.
- `sprints(id, project_id, number, goal, start_date, end_date, status)`: estado `'Activo'` o
  `'Finalizado'`.
- `effort_logs`: horas de HU-03, con referencia a la historia.

> **A verificar (T001 y T004)**: nombre de la columna de historia en `effort_logs` (se supone
> `story_id`), que `status` admita los cuatro valores y que no haya otras columnas obligatorias al
> actualizar.

## StoryStatus y transiciones

### Values

- `Pendiente`, `En Sprint`, `En progreso`, `Hecho` (constantes del dominio; se reutilizan las de HU-04 si
  existen).

### Rules

- `CanTransition(origen, destino)` es verdadero solo para En Sprint → En progreso, En progreso → Hecho y
  Hecho → En progreso.
- `NextAction(estado)` devuelve la acción válida de ese estado (`comenzar`, `completar`, `reabrir`) o
  ninguna para `Pendiente`.

## StoryState

Lectura de una historia para decidir el cambio.

### Fields

- `status`: estado actual.
- `sprintID`, `sprintActive`: Sprint al que está asignada y si está activo (vacío si no tiene).
- `hasEffort`: verdadero si tiene horas registradas.

## StatusResult

Resultado de un cambio aplicado.

### Fields

- `from`, `to`: estados.
- `noHoursWarning`: verdadero al completar sin horas registradas.

## Errores del dominio

`ErrStoryNotFound`, `ErrNotInActiveSprint`, `ErrAlreadyInStatus`, `ErrStatusChanged`,
`ErrTransitionNotAllowed`. Cualquier otro error es una falla de almacenamiento.

## StoryStatusStore

Puerto de almacenamiento del dominio.

### Operations

- `FindStory(ctx, projectID, storyID)`: devuelve el `StoryState` o `ErrStoryNotFound`.
- `UpdateStatus(ctx, storyID, from, to)`: devuelve si se cambió una fila (comparación atómica).

### Queries

```sql
-- FindStory
SELECT us.status, s.id, (s.status = 'Activo'), 
       EXISTS (SELECT 1 FROM effort_logs el WHERE el.story_id = us.id)
FROM user_stories us
LEFT JOIN sprint_story_assignments a ON a.story_id = us.id
LEFT JOIN sprints s ON s.id = a.sprint_id
WHERE us.id = $1 AND us.project_id = $2;

-- UpdateStatus (atómico)
UPDATE user_stories us
SET status = $3
WHERE us.id = $1 AND us.status = $2
  AND EXISTS (SELECT 1 FROM sprint_story_assignments a
              JOIN sprints s ON s.id = a.sprint_id
              WHERE a.story_id = us.id AND s.status = 'Activo');
```

### Rules

- Si `UpdateStatus` afecta 0 filas, el servicio responde `ErrStatusChanged`.
- El adaptador tiene pruebas de integración contra PostgreSQL con fixtures propias, que se eliminan al
  terminar borrando primero las historias.

## ChangeStatus (servicio)

`ChangeStatus(ctx, projectID, storyID, target, expected)`: aplica el orden de validación (historia del
proyecto, Sprint activo, ya aplicado, estado esperado, transición permitida, cambio atómico) y devuelve
un `StatusResult` o un error del dominio. `expected` es opcional.

## BoardForm

Datos del envío de `POST /tablero/estado`.

### Fields

- `historia_id`, `accion`, `estado_actual` (opcional).

### Rules

- `accion` válida: `comenzar` (destino `En progreso`), `completar` (`Hecho`) o `reabrir` (`En progreso`).
- `historia_id` numérico; los demás valores se rechazan sin llamar al servicio.

## BoardView

Sprint activo con sus historias.

### Fields

- `id`, `number`, `label` ("Sprint N"), `goal`, `startDate`, `endDate`.
- `groups`: tres grupos (`En Sprint`, `En progreso`, `Hecho`) de `BoardStory`.
- `plannedPoints`, `completedPoints`.

### Rules

- Si hubiera más de un Sprint activo por datos anteriores a FIX-HU-04, es el de mayor `number`.
- `plannedPoints` y `completedPoints` se calculan con una función pura sobre las historias (sin
  Story Points cuenta 0).

## BoardStory

### Fields

- `id`, `title`, `storyPoints` (opcional, "Sin estimar" si falta), `status`, `hasEffort`.
- `action`: acción válida (`NextAction`) y su etiqueta ("Comenzar", "Completar", "Reabrir").

### Rules

- Las historias de cada grupo se ordenan por `id` ascendente.

## BoardCatalog y PostgreSQLBoardCatalog

### Operations

- `ActiveBoard(ctx, projectID)`: devuelve el `BoardView` activo o ninguno; distingue error de lectura de
  "sin Sprint activo".

### Queries

```sql
-- Sprint activo (el de mayor número)
SELECT id, number, goal, start_date, end_date
FROM sprints WHERE project_id = $1 AND status = 'Activo'
ORDER BY number DESC LIMIT 1;

-- Historias del Sprint
SELECT us.id, us.title, us.story_points, us.status,
       EXISTS (SELECT 1 FROM effort_logs el WHERE el.story_id = us.id) AS has_effort
FROM sprint_story_assignments a
JOIN user_stories us ON us.id = a.story_id
WHERE a.sprint_id = $1
ORDER BY us.id;
```

### Rules

- Solo lecturas; `database/sql` y `pgx/v5/stdlib`.
- Pruebas de integración con fixtures propias contra PostgreSQL.

## FieldErrors y mensajes

- Éxito: `MsgStatusStarted` ("La historia pasó a En progreso"), `MsgStatusCompleted` ("La historia pasó
  a Hecho"), `MsgStatusCompletedNoHours` ("La historia pasó a Hecho. Advertencia: no tiene horas reales
  registradas"), `MsgStatusReopened` ("La historia volvió a En progreso").
- Advertencia: `MsgAlreadyInStatus` ("La historia ya está en {estado}").
- Rechazos: `MsgNotInActiveSprint`, `MsgTransitionNotAllowed`, `MsgStatusChanged`, `MsgStoryNotFound`,
  `MsgInvalidAction`.
- Vistas vacías: `MsgNoActiveSprint` ("No hay un Sprint activo"), `MsgSprintWithoutStories` ("El Sprint no
  tiene historias asignadas").
- Falla de guardado: `MsgOperationFailed` de HU-20b.