# Data Model: Cierre de Sprint

Esta feature agrega **una tabla** (`sprint_story_results`) con una migración nueva. El resto usa
`sprints`, `sprint_story_assignments` y `user_stories`. El setup confirmó que el paquete de dominio y
almacenamiento de Sprint es `src/sprint`.

## Migración

Número: `014`. La migración `013` ya está ocupada por FIX-HU-21; el equipo
confirmó que esa migración pertenece a otra historia. La migración de HU-15 usa el siguiente número y no
edita migraciones ajenas.

```sql
-- 014_sprint_story_results.up.sql
CREATE TABLE sprint_story_results (
    sprint_id    BIGINT      NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
    story_id     BIGINT      NOT NULL REFERENCES user_stories(id) ON DELETE CASCADE,
    final_status VARCHAR(20) NOT NULL
                 CHECK (final_status IN ('Pendiente', 'En Sprint', 'En progreso', 'Hecho')),
    story_points INTEGER,
    PRIMARY KEY (sprint_id, story_id)
);

-- 014_sprint_story_results.down.sql
DROP TABLE IF EXISTS sprint_story_results;
```

La columna `user_stories.story_points` es `INTEGER` nullable y restringe los valores a la secuencia
Fibonacci permitida. La migración nueva debe probarse en ambos sentidos (`up`, `down 1`, `up`) como parte
del DoD.

## Esquema usado

- `sprints`: `id BIGSERIAL`, `project_id BIGINT` con FK a `projects` y `ON DELETE CASCADE`, `number
  INTEGER`, `goal TEXT`, `start_date DATE`, `end_date DATE`, `status VARCHAR(20)` y `created_at
  TIMESTAMPTZ`. El estado acepta `'Activo'` o `'Finalizado'`; `end_date` debe ser posterior a `start_date`
  y `(project_id, number)` es único. FIX-HU-04 impide crear un segundo Sprint activo para el mismo
  proyecto (`ErrActiveSprintExists`).
- `sprint_story_assignments`: `sprint_id BIGINT` y `story_id BIGINT`, ambos FK con borrado en cascada;
  PK compuesta `(sprint_id, story_id)` y `UNIQUE (story_id)`, por lo que una historia tiene como máximo
  una asignación vigente.
- `user_stories`: `status TEXT` acepta `'Pendiente'`, `'En Sprint'`, `'En progreso'` o `'Hecho'`;
  `story_points INTEGER` admite `NULL` y los valores Fibonacci configurados; `active_sprint_id BIGINT`
  referencia `sprints` con `ON DELETE SET NULL`.
- La asignación de HU-04 inserta en `sprint_story_assignments` y cambia la historia de `'Pendiente'` a
  `'En Sprint'`, asignando `active_sprint_id`. Al quitarla, borra la asociación y deja la historia en
  `'Pendiente'` con `active_sprint_id = NULL`; esa operación solo aplica a una historia en `'En Sprint'`.
- El cierre conserva las asignaciones de las historias `'Hecho'`. Para las no completadas, las elimina y
  las devuelve a `'Pendiente'` con `active_sprint_id = NULL`.
- `sprint_story_results`: lo que el Sprint tenía al cerrar; nunca se actualiza después.

## Hallazgos de dependencias (T001 y T002)

### HU-07: validación y métricas

- La validación reutilizable tiene la firma
  `metrics.ValidateSprintClosure(sprint metrics.SprintData) error`. `SprintData` contiene `ID`, `Status` y
  `Stories []CompletedStory`; cada `CompletedStory` lleva `ID`, `Status`, `StoryPoints *int`,
  `EstimatedHours *big.Rat` y `ActualHours *big.Rat`.
- La función solo valida historias `'Hecho'`: rechaza horas reales negativas y Story Points nulos o
  fuera de la secuencia permitida. Devuelve un `ClosureValidationError` cuyo `Error()` concatena motivos
  separados por saltos de línea, con formato `Historia {id}: horas reales negativas` o
  `Historia {id}: Story Points inválidos`. Las pruebas ejercitan la función directamente con
  `SprintData` inválido y verifican el ID, el motivo y el formato de múltiples motivos.
- No debe usarse `(*metrics.Service).CalculateMetrics(sprintID int64) (SprintMetrics, error)` para
  validar antes del cierre: el servicio exige que el Sprint esté `'Finalizado'`, ejecuta la misma
  validación, calcula y persiste las métricas. El cálculo de velocidad suma `StoryPoints` (`int`) solo de
  historias `'Hecho'`; con cero historias completadas, la velocidad es `0`.
- HU-07 carga las historias a través de `sprint_story_assignments` y suma `effort_logs.hours` por
  `story_id`, sin filtrar por fechas ni por Sprint. Como una historia solo conserva una asignación vigente,
  al validar el Sprint activo entran todas las horas históricas registradas para sus historias; el modelo
  actual no atribuye horas a cada Sprint en el que una historia haya participado ni permite reconstruir la
  asignación anterior una vez quitada.

### HU-03: registro de horas

`PostgreSQLEffortStore.StoryAcceptsEffort` exige que la historia tenga `active_sprint_id` que apunte a un
Sprint con estado `'Activo'`. El servicio rechaza los demás casos con `ErrStoryNotInSprint`
(`"La historia no está asignada a un Sprint activo"`). Por lo tanto, HU-15 no necesita agregar esa regla
al dominio de HU-03 para bloquear horas en una historia devuelta o de un Sprint finalizado.

### HU-14: tablero

`web.BoardCatalog` expone `ActiveBoard(ctx, projectID) (BoardView, found, error)`. `found == false` indica
que no hay Sprint activo; los errores representan fallas de lectura. `BoardView` contiene el ID, número,
etiqueta, objetivo, fechas, grupos de historias y Story Points planificados/completados. Cada `BoardStory`
contiene ID, título, `StoryPoints *int` (nil significa sin estimar), estado, si tiene esfuerzo y la acción
válida. Los grupos están ordenados `'En Sprint'`, `'En progreso'`, `'Hecho'`; `BoardPoints` suma todos los
puntos planificados y solo suma como completados los de historias `'Hecho'`.

### HU-20a y HU-20b: integración web

- `Server.Register(section, routes)` agrega la sección y registra cada patrón/método de `Routes`; aplica
  `RequireContext(section)` a las rutas registradas. Con `Section.RequiresProject`, las solicitudes sin
  proyecto activo reciben un flash y una redirección 303 al selector, también para los envíos `POST`.
- `ActiveContextFrom(ctx)` devuelve el `*ActiveContext` o nil. `SetFlash` guarda el mensaje para la
  siguiente vista y `RedirectSeeOther` emite el 303 de Post/Redirect/Get. El mensaje genérico ante fallas
  de guardado es `MsgOperationFailed` (`"La operación no pudo completarse"`).
- El parcial se llama `"alert"` y está definido en `src/web/templates/layout.html`; no existe un archivo
  separado `alert.html`.
- `main.go` instala `CrossOriginGuard` antes de registrar las rutas. `http.CrossOriginProtection` rechaza
  solicitudes no seguras reconocidas como originadas en otro sitio con 403; solicitudes sin cabeceras de
  origen (por ejemplo, curl o las pruebas) pasan.

### Verificación de base de datos (T003 y T004)

`docker-compose.yml` publica PostgreSQL como `5433:5432`: el host usa el puerto `5433` y el contenedor
escucha en `5432`. FIX-HU-21 ocupa la migración `013`; la migración de HU-15 es la `014`.

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