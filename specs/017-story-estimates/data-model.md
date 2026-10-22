# Data Model: Horas estimadas por historia (Estimaciones)

Esta feature **no crea tablas ni migraciones**. Usa `user_stories`, `effort_logs`,
`sprint_story_assignments` y `sprints`.

## Esquema usado

- `user_stories`: `id`, `project_id`, `title`, `status`, `story_points` (opcional) y `estimated_hours
  NUMERIC(6,2)` (opcional, `CHECK` mayor que 0). Solo se actualiza `estimated_hours`.
- `effort_logs`: registros de horas de HU-03, con referencia a la historia y las horas.
- `sprint_story_assignments(sprint_id, story_id)` y `sprints(id, status)`: estado `'Activo'` o
  `'Finalizado'`.

> **A verificar (T001 y T004)**: nombres de la columna de historia y de horas en `effort_logs` (se
> suponen `story_id` y `hours`), que HU-07 lee `estimated_hours` y que no hay otras columnas obligatorias
> al actualizar.

## Hours (regla de HU-22)

La representación y la validación de las horas son las que HU-22 deja en el dominio (junto a HU-02).

### Rules

- Mayor que 0, hasta dos decimales, máximo 9999,99, coma o punto, sin separador de miles; no se redondea.
- Errores y mensajes: "El valor de horas no es válido" y "Las horas estimadas no pueden superar 9999,99".

## EstimateState

Lectura de una historia para decidir el cambio.

### Fields

- `hours`: horas estimadas actuales (ausentes si no tiene).
- `inFinishedSprint`: verdadero si está asignada a un Sprint finalizado.

## EstimateResult

Resultado de un guardado.

### Fields

- `previous`: horas anteriores (ausentes si no tenía).
- `current`: horas guardadas.

### Rules

- `previous` ausente indica "Horas estimadas registradas"; presente, "Horas estimadas actualizadas".

## Errores del dominio

`ErrStoryNotFound`, `ErrStoryInFinishedSprint` y los de la regla de horas (inválidas y por encima del
máximo). Cualquier otro error es una falla de almacenamiento.

## EstimateStore

Puerto de almacenamiento del dominio.

### Operations

- `FindEstimate(ctx, projectID, storyID)`: devuelve el `EstimateState` o `ErrStoryNotFound`.
- `UpdateEstimate(ctx, projectID, storyID, hours)`: devuelve si se cambió una fila.

### Queries

```sql
-- FindEstimate
SELECT us.estimated_hours,
       EXISTS (SELECT 1 FROM sprint_story_assignments a
               JOIN sprints s ON s.id = a.sprint_id
               WHERE a.story_id = us.id AND s.status = 'Finalizado')
FROM user_stories us
WHERE us.id = $1 AND us.project_id = $2;

-- UpdateEstimate (atómico y solo esa columna)
UPDATE user_stories us
SET estimated_hours = $3
WHERE us.id = $1 AND us.project_id = $2
  AND NOT EXISTS (SELECT 1 FROM sprint_story_assignments a
                  JOIN sprints s ON s.id = a.sprint_id
                  WHERE a.story_id = us.id AND s.status = 'Finalizado');
```

### Rules

- Si `UpdateEstimate` afecta 0 filas, el servicio relee con `FindEstimate` para responder
  `ErrStoryNotFound` o `ErrStoryInFinishedSprint`.
- El adaptador tiene pruebas de integración contra PostgreSQL con fixtures propias, que se eliminan al
  terminar borrando primero las historias.

## UpdateEstimatedHours (servicio)

`UpdateEstimatedHours(ctx, projectID, storyID, input)`: valida el texto con la regla de HU-22, lee la
historia, guarda de forma atómica y devuelve un `EstimateResult` o un error del dominio.

## EstimateForm

Datos del envío de `POST /estimaciones`.

### Fields

- `historia_id`, `horas_estimadas` (texto tal como llegó).

### Rules

- `historia_id` numérico; el valor de horas se entrega sin cambios al dominio.
- El texto escrito se conserva en la fila cuando hay error.

## EstimateRow

Historia mostrada en la lista.

### Fields

- `id`, `title`, `status`, `storyPoints` (opcional, "Sin estimar" si falta).
- `estimatedHours` (opcional, "Sin estimar" si falta), `actualHours` (0 si no hay registros).
- `locked`, `lockedReason`: verdadero y "Sprint finalizado" si la historia está en un Sprint finalizado.

### Rules

- Orden por `id` descendente; solo las historias del proyecto pedido.
- Las horas se muestran con coma decimal.

## EstimateCatalog y PostgreSQLEstimateCatalog

### Operations

- `ListEstimates(ctx, projectID)`: devuelve las `EstimateRow` del proyecto; sin historias no es un error.

### Queries

```sql
SELECT us.id, us.title, us.status, us.story_points, us.estimated_hours,
       COALESCE((SELECT SUM(el.hours) FROM effort_logs el WHERE el.story_id = us.id), 0) AS actual_hours,
       EXISTS (SELECT 1 FROM sprint_story_assignments a
               JOIN sprints s ON s.id = a.sprint_id
               WHERE a.story_id = us.id AND s.status = 'Finalizado') AS in_finished_sprint
FROM user_stories us
WHERE us.project_id = $1
ORDER BY us.id DESC;
```

### Rules

- Solo lecturas; `database/sql` y `pgx/v5/stdlib`.
- Pruebas de integración con fixtures propias contra PostgreSQL.

## Mensajes

- Éxito: `MsgHoursRegistered` ("Horas estimadas registradas"), `MsgHoursUpdated` ("Horas estimadas
  actualizadas").
- Errores de horas: los de HU-22 (`MsgInvalidHours`, `MsgHoursTooLarge`).
- Rechazos: `MsgStoryNotFound` ("La historia no existe"), `MsgHoursLockedSprint` ("La historia pertenece a
  un Sprint finalizado; sus horas estimadas no se pueden modificar") y el motivo de la lista ("Sprint
  finalizado").
- Estado vacío: "Aún no hay historias" (HU-22).
- Falla de guardado: `MsgOperationFailed` de HU-20b.