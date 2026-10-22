# Data Model: Cierre de Sprint

Esta feature agrega **una tabla** (`sprint_story_results`) con una migración nueva. El resto usa
`sprints`, `sprint_story_assignments` y `user_stories`.

## Migración

Número: el último de `main` más uno (hoy `009`, después de la `008` del fix de HU-01); nadie edita una
migración ajena. Se verifica en T001 y T004.

```sql
-- NNN_create_sprint_story_results.up.sql
CREATE TABLE sprint_story_results (
    sprint_id    BIGINT      NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    story_id     BIGINT      NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    final_status VARCHAR(20) NOT NULL CHECK (final_status IN ('En Sprint', 'En progreso', 'Hecho')),
    story_points INTEGER,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (sprint_id, story_id)
);
CREATE INDEX idx_sprint_story_results_story ON sprint_story_results (story_id);

-- NNN_create_sprint_story_results.down.sql
DROP TABLE sprint_story_results;
```

> **A verificar (T001 y T004)**: el tipo de `user_stories.story_points` (se supone entero) y que la
> migración se prueba en ambos sentidos (DoD).

## Esquema usado

- `sprints(id, project_id, number, goal, start_date, end_date, status)`: estado `'Activo'` o
  `'Finalizado'`. El cierre lo pasa a `'Finalizado'`.
- `sprint_story_assignments(sprint_id, story_id)`: `UNIQUE (story_id)`. El cierre elimina las filas de las
  historias no completadas.
- `user_stories`: `status`, `story_points`, `active_sprint_id`. El cierre pone las no completadas en
  `'Pendiente'` y `active_sprint_id` en `NULL`.
- `sprint_story_results`: lo que el Sprint tenía al cerrar; nunca se actualiza después.

## SprintResult (registro del Sprint)

### Fields

- `sprintID`, `storyID`, `finalStatus` (`En Sprint`, `En progreso` o `Hecho`), `storyPoints` (opcional).

### Rules

- Se crea una fila por cada historia que tenía el Sprint en el momento de cerrar.
- Los Story Points son los que la historia tenía al cerrar; una edición posterior no los cambia.
- Para un Sprint finalizado: planificadas = todas las filas; completadas = filas con `final_status =
  'Hecho'`; devueltas = las demás.

## CloseSprint (servicio)

`CloseSprint(ctx, projectID, sprintID)`: comprueba que el Sprint exista en el proyecto (`ErrSprintNotFound`),
que esté activo (`ErrSprintAlreadyClosed`), invoca la validación de HU-07 (`ErrCloseBlocked` con las
historias a corregir) y cierra de forma transaccional. Devuelve un `CloseResult`.

### CloseResult

- `sprintNumber`, `completedCount`, `returnedCount`.

## Errores del dominio

`ErrSprintNotFound`, `ErrSprintAlreadyClosed`, `ErrCloseBlocked`. Cualquier otro error es una falla de
almacenamiento.

## SprintCloseStore

Puerto de almacenamiento del dominio.

### Operations

- `FindSprint(ctx, projectID, sprintID)`: devuelve número y estado, o `ErrSprintNotFound`.
- `CloseSprint(ctx, sprintID)`: cierra en una transacción y devuelve los conteos, o `ErrSprintAlreadyClosed`
  si ya no estaba activo.

### Transacción de cierre

```sql
BEGIN;
-- 1. Bloquear el Sprint y sus historias
SELECT status FROM sprints WHERE id = $1 FOR UPDATE;           -- 'Finalizado' => ErrSprintAlreadyClosed
SELECT us.id FROM user_stories us
JOIN sprint_story_assignments a ON a.story_id = us.id
WHERE a.sprint_id = $1 FOR UPDATE OF us;

-- 2. Registro del Sprint (todas las historias)
INSERT INTO sprint_story_results (sprint_id, story_id, final_status, story_points)
SELECT a.sprint_id, us.id, us.status, us.story_points
FROM sprint_story_assignments a JOIN user_stories us ON us.id = a.story_id
WHERE a.sprint_id = $1;

-- 3. Devolver las no completadas
DELETE FROM sprint_story_assignments a USING user_stories us
WHERE a.sprint_id = $1 AND us.id = a.story_id AND us.status <> 'Hecho'
RETURNING a.story_id;                                          -- ids devueltos
UPDATE user_stories SET status = 'Pendiente', active_sprint_id = NULL
WHERE id = ANY($2);                                            -- $2 = ids devueltos

-- 4. Finalizar
UPDATE sprints SET status = 'Finalizado' WHERE id = $1;
COMMIT;
```

### Rules

- Cualquier error revierte la transacción: el Sprint sigue activo y las historias no cambian.
- Pruebas de integración contra PostgreSQL con fixtures propias, que se eliminan al terminar borrando
  primero las historias.

## CloseForm

### Fields

- `sprint_id`: identificador numérico del Sprint que se veía.

### Rules

- Un valor no numérico se rechaza sin llamar al servicio ("El Sprint no existe").

## CloseSummary (vista)

Se arma con `BoardCatalog.ActiveBoard` de HU-14.

### Fields

- `sprintID`, `label` ("Sprint N"), `goal`, `startDate`, `endDate`.
- `completed`: historias en "Hecho".
- `toReturn`: historias en "En Sprint" o "En progreso".
- `plannedPoints`, `completedPoints`.

### Rules

- Sin Sprint activo no hay resumen ni botón.

## Mensajes

- Éxito: `MsgSprintClosed` ("Sprint {N} cerrado correctamente. Historias devueltas al Backlog: {Y}").
- Rechazos: `MsgSprintAlreadyClosed` ("El Sprint ya está cerrado"), `MsgSprintNotFound` ("El Sprint no
  existe"), `MsgCloseBlocked` (el de HU-07 con las historias a corregir).
- Vista vacía: `MsgNoSprintToClose` ("No hay un Sprint activo para cerrar").
- Aviso en el resumen: las historias no completadas vuelven al Backlog y el cierre no se puede deshacer.
- Falla de guardado: `MsgOperationFailed` de HU-20b.