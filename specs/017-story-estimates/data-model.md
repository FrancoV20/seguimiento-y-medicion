# Data Model: Horas estimadas por historia (Estimaciones)

Esta feature **no crea tablas ni migraciones**. Usa `user_stories`, `effort_logs`,
`sprint_story_assignments` y `sprints`.

## Esquema usado

- `user_stories`: `id`, `project_id`, `title`, `status`, `story_points` (opcional) y `estimated_hours
  NUMERIC(6,2)` (opcional, `CHECK` mayor que 0). Solo se actualiza `estimated_hours`.
- `effort_logs`: registros de horas de HU-03, con referencia a la historia y las horas.
- `sprint_story_assignments(sprint_id, story_id)` y `sprints(id, status)`: estado `'Activo'` o
  `'Finalizado'`.

> **Verificado en T001 y T004 (2026-10-28, `main` en `d12067f`)**: `effort_logs` usa `story_id` y
> `hours NUMERIC(5,2)`; HU-07 lee `user_stories.estimated_hours`; actualizar solo `estimated_hours` no
> choca con otras columnas obligatorias. Detalle en "Contrato de HU-02, HU-22, HU-03 y HU-07 confirmado
> (T001)" y "Verificación de migraciones 002, 003 y 004 (T004)".

## Hours (regla de HU-22)

La representación y la validación de las horas son las que HU-22 deja en el dominio (junto a HU-02).

### Rules

- Mayor que 0, hasta dos decimales, máximo 9999,99, coma o punto, sin separador de miles; no se redondea.
- Errores y mensajes: "El valor de horas no es válido" y "Las horas estimadas no pueden superar 9999,99".

### Lo que existe en `main` (T001)

- Paquete: `src/backlog` (HU-02 y HU-22); no existe `src/story`.
- Regla numérica: `validEstimatedHours(h *float64) bool` y `maxEstimatedHours = 9999.99` en
  `src/backlog/service.go`, **sin exportar**. Acepta `nil`; rechaza NaN, infinito, `<= 0`, `> 9999.99` y
  más de dos decimales (`math.Round(v*100)/100 != v`).
- Error del dominio: un solo sentinel, `backlog.ErrInvalidEstimatedHours` ("las horas estimadas deben ser
  mayores a 0 y menores a 10000"); el dominio no distingue "inválido" de "por encima del máximo".
- El texto **no** se interpreta en el dominio: lo hace `src/web/backlog_form.go` con
  `strings.ReplaceAll(",", ".")` y `strconv.ParseFloat`, que también acepta `1e3`, `+5`, `0x1p3` y
  `1.000` (este último como 1). La web elige el mensaje: `MsgHoursTooLarge` si el valor supera 9999,99,
  si no `MsgInvalidEstimatedHours`.
- Mensajes en `src/web/backlog_form.go`: `MsgInvalidEstimatedHours` ("El valor de horas no es válido") y
  `MsgHoursTooLarge` ("Las horas estimadas no pueden superar 9999,99").

### Decisión para HU-17

Sin modificar archivos de HU-22, HU-17 agrega en `src/backlog` (archivo nuevo) la interpretación estricta
del texto, `ParseEstimatedHours(text string) (float64, error)`:

1. Quita espacios al inicio y al final; vacío es inválido.
2. Acepta solo `^[0-9]+([.,][0-9]+)?$` (sin signo, exponente, hexadecimal ni separador de miles).
3. Más de dos decimales es inválido (`1,555`, `9999,995`); se verifica antes que el máximo.
4. Mayor que 9999,99 devuelve `ErrEstimatedHoursTooLarge`, un error que `errors.Is` reconoce también
   como `ErrInvalidEstimatedHours`.
5. El valor resultante se confirma con `validEstimatedHours` (misma regla numérica de HU-22).

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

-- UpdateEstimate (atómico y solo esa columna), en una transacción con dos sentencias:
-- 1) bloquear la historia: espera a un cierre de Sprint en curso (HU-15 bloquea las historias FOR UPDATE)
SELECT 1 FROM user_stories WHERE id = $1 AND project_id = $2 FOR UPDATE;
-- 2) guardar: esta sentencia ya ve el cierre confirmado
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
- El bloqueo previo es necesario (revisión del PR, FR-009): en una sola sentencia, si el `UPDATE` espera
  a que HU-15 termine de cerrar el Sprint, al destrabarse vuelve a evaluar la fila de la historia pero
  el `NOT EXISTS` sigue viendo el Sprint como estaba antes del cierre, y modificaba una historia de un
  Sprint ya finalizado. Lo cubre `TestPostgreSQLEstimateStore_GuardarDuranteElCierreDelSprintNoModificaLaHistoria`.
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

- `id`, `title`, `status`, `storyPoints` (opcional, "Sin estimar" si falta). "Sin estimar" no es una
  constante: HU-22 lo escribe literal en `backlog_handlers.go` y `backlog.html`; HU-17 define su propia
  constante en `estimates_form.go`.
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
- Errores de horas: los de HU-22 (`MsgInvalidEstimatedHours`, `MsgHoursTooLarge`, en
  `src/web/backlog_form.go`).
- Rechazos: `MsgStoryNotFound` ("La historia no existe"), `MsgHoursLockedSprint` ("La historia pertenece a
  un Sprint finalizado; sus horas estimadas no se pueden modificar") y el motivo de la lista ("Sprint
  finalizado").
- Estado vacío: "Aún no hay historias" (HU-22 lo escribe literal en `backlog.html`; existe la constante
  `MsgPokerNoStories` con el mismo texto en `poker_messages.go`).
- Falla de guardado: `MsgOperationFailed` de HU-20b.

## Contrato de HU-02, HU-22, HU-03 y HU-07 confirmado (T001)

- **Historias (HU-02 y HU-22)**: paquete `src/backlog`. `UserStory.EstimatedHours *float64` (`nil` = sin
  horas). El alta (`PostgreSQLBacklogStore.CreateStory`) inserta `estimated_hours` como
  `sql.NullFloat64`. No hay operación de actualización de historias: HU-17 agrega la suya.
- **Archivos existentes con nombres parecidos**: `src/backlog/estimated_hours_test.go` y
  `src/backlog/postgres_estimated_hours_integration_test.go` son de HU-22 (alta con horas). El paquete de
  prueba ya define `floatPtr`; las pruebas de HU-17 no deben redeclararlo.
- **Horas reales (HU-03)**: `effort_logs(story_id BIGINT, hours NUMERIC(5,2))`; HU-03 suma `hours`
  por `story_id`.
- **Desviación (HU-07)**: `src/metrics/postgres_metrics_store.go` comprueba en `information_schema` que
  exista `user_stories.estimated_hours` y la lee como `us.estimated_hours::text` junto a
  `SUM(el.hours)::text`, por las historias asignadas al Sprint. Cualquier valor que guarde HU-17 lo ve HU-07
  sin cambios.
- **Columnas obligatorias**: `title`, `description` y `priority` son `NOT NULL` pero el `UPDATE` de HU-17
  solo toca `estimated_hours`; el `CHECK (estimated_hours > 0)` es la última defensa en la base.
- **Sprint finalizado**: HU-15 (cierre) no está en `main`. Un Sprint solo queda `'Finalizado'` por SQL y su
  asignación en `sprint_story_assignments` se conserva (`UNIQUE (story_id)`), así que las consultas de
  este documento detectan el bloqueo. Cuando se integre HU-15 hay que confirmar que el cierre no borra la
  asignación de las historias terminadas.

## Verificación de migraciones 002, 003 y 004 (T004)

Revisadas en `main` (`d12067f`); coinciden con el esquema anotado, con estos detalles:

- **002**: `user_stories(id BIGSERIAL, project_id BIGINT NOT NULL, title, description, priority NOT NULL,
  story_points INTEGER` con escala Fibonacci, `status`, `estimated_hours NUMERIC(6,2) CHECK (> 0)`,
  `created_at)`. `NUMERIC(6,2)` admite hasta 9999,99, que es el máximo de la regla.
- **003**: `effort_logs(id, story_id BIGINT NOT NULL REFERENCES user_stories, member_id INTEGER,
  work_date, activity, hours NUMERIC(5,2) CHECK (> 0 AND <= 24), created_at)` e índice por `story_id`.
- **004**: `sprints(status 'Activo' | 'Finalizado')`, `sprint_story_assignments(sprint_id, story_id)` con
  `PRIMARY KEY (sprint_id, story_id)` y `UNIQUE (story_id)`, `user_stories.active_sprint_id` y el estado
  `'En Sprint'` agregado al `CHECK` de `user_stories.status`.
- Ninguna migración posterior (007, 008, 011) cambia estas tablas. Si HU-17 necesitara una migración, el
  próximo número es **012**. No hace falta avisar al equipo de cambios.
