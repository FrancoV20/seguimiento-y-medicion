# Data Model: Crear y asignar Sprint

## Project

Contexto propietario de los Sprints y del Product Backlog.

### Fields

- `id`: identificador único del proyecto.
- `sprints`: Sprints creados para el proyecto.
- `backlog`: historias disponibles y sus estados.
- `nextSprintNumber`: siguiente número secuencial a utilizar.

### Rules

- El proyecto debe existir para crear un Sprint.
- `nextSprintNumber` comienza en 1 y avanza sin reutilizar identificadores ya emitidos.

## SprintStore

Interfaz interna para persistir Sprints, relaciones con historias y cambios de estado.

### Operations

- `CreateSprint(projectID, sprint)`: persiste un Sprint válido y devuelve un error si falla.
- `AssignStories(sprintID, storyIDs)`: crea asociaciones y actualiza estados como una unidad.
- `RemoveStory(sprintID, storyID)`: elimina una asociación para permitir un movimiento explícito.

### Rules

- La creación y sus asociaciones deben confirmarse en una única operación persistente.
- Un conflicto de exclusividad no debe alterar la asociación activa existente.

## PostgreSQLSprintStore

Implementación concreta de `SprintStore` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` como driver de PostgreSQL.
- `github.com/jackc/pgx/v5/stdlib` para registrar pgx y operar mediante `database/sql`.

### Rules

- Debe persistir identificador secuencial, Goal, fechas, estado y asociaciones de historias.
- Debe ejecutar creación, asignaciones y cambios de estado dentro de transacciones de
	`database/sql`.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse al sistema.

## Sprint

Iteración activa de trabajo con un identificador generado automáticamente.

### Fields

- `id`: identificador técnico único.
- `number`: número secuencial del proyecto.
- `displayID`: etiqueta derivada, por ejemplo `Sprint 1`; no editable.
- `goal`: texto obligatorio, normalizado y no vacío.
- `startDate`: fecha de inicio.
- `endDate`: fecha de finalización.
- `status`: inicialmente `Activo`.
- `storyIDs`: historias asociadas, posiblemente vacío.

### Rules

- `endDate` debe ser posterior a `startDate`.
- Un Sprint válido puede crearse sin historias.
- Un Sprint sin historias produce una advertencia no bloqueante.
- El identificador visible se genera desde `number` y no se recibe en la solicitud.

## UserStory

Historia existente del Product Backlog que puede asignarse.

### Fields

- `id`: identificador de la historia.
- `status`: `Pendiente` o `En Sprint`, entre otros estados definidos por backlog.
- `activeSprintID`: Sprint activo al que pertenece, opcional.

### Rules

- Solo historias `Pendiente` están disponibles para una nueva asignación.
- Una historia puede tener como máximo un `activeSprintID`.
- Asignar una historia válida la cambia a `En Sprint`.
- Quitarla de un Sprint activo elimina `activeSprintID` y la devuelve a `Pendiente`.

## SprintCreationRequest

Datos ingresados por el Scrum Master.

### Fields

- `projectID`: proyecto seleccionado.
- `goal`: Sprint Goal obligatorio.
- `startDate` y `endDate`: período obligatorio.
- `storyIDs`: cero o más historias seleccionadas.

El identificador del Sprint no forma parte de la solicitud.

## SprintCreationResult

Resultado de creación y asignación.

### Fields

- `sprint`: Sprint creado con etiqueta secuencial.
- `warning`: advertencia si `storyIDs` está vacío.
- `assignmentErrors`: historias rechazadas y motivos si la operación no puede completarse.
- `storageError`: error de PostgreSQL sin informar detalles internos sensibles.

### Invariants

- Un resultado exitoso crea un único Sprint.
- Una operación inválida no cambia estados de historias ni crea asociaciones parciales.
- La advertencia por Sprint vacío no convierte el resultado en error.

## Quitar y volver a asignar (FIX-HU-04)

Reemplaza a `StoryMove`: con un solo Sprint activo por proyecto no hay otro Sprint activo al
que mover la historia, y la función `MoveStory` se eliminó.

### Transition

`historia En Sprint del Sprint activo -> quitar -> historia Pendiente -> asignar al Sprint activo -> historia En Sprint`.

## FIX-HU-04: un solo Sprint activo por proyecto

### Revisión de HU-04 en `main` (T001)

- Paquete `src/sprint`. Firmas: `Service.CreateSprint(ctx, request, project, stories)`, la
  función de dominio `CreateSprint(request, project, stories)` y `SprintStore` con
  `CreateSprint(ctx, sprint)`, `AssignStories(ctx, sprintID, storyIDs)` y
  `RemoveStory(ctx, sprintID, storyID)`.
- `projects.next_sprint_number` se lee con `SELECT … FOR UPDATE` y se incrementa en
  `nextSprintNumber` (`sequence.go`), dentro de la transacción de `CreateSprint`.
- Al asignar, la historia queda en "En Sprint" (`AssignStoryToSprint` y
  `persistStoryAssignments`); al quitarla vuelve a "Pendiente" (`RemoveStoryFromSprint`).
  Confirmado.
- Pruebas que creaban dos Sprints activos en el mismo proyecto (corregidas en T007):
  `TestMemorySprintStoreGeneratesSequentialSprintIDs`,
  `TestMemorySprintStoreAssignsExclusivelyAndAllowsRemoveThenAssign`,
  `TestMoveStoryRemovesBeforeAssigningToDestinationSprint` (eliminada junto con `MoveStory`),
  `TestPostgreSQLSprintStoreCreateSprintAndSequence` y
  `TestPostgreSQLSprintStoreAssignmentConflictRollsBack`. Las pruebas de HU-03 y HU-07 insertan
  Sprints directamente con SQL y no pasan por el servicio: no se ven afectadas.

### Campos nuevos

- `Project.ActiveSprintNumber`: número del Sprint activo; 0 si no hay. Si quien llama lo
  informa, la validación de dominio rechaza antes de llegar a la base.
- `UserStory.ActiveSprintNumber`: número visible del Sprint al que está asignada; se usa en
  el mensaje de conflicto.

### Errores (los usa HU-23)

| Error | Cómo reconocerlo | Mensaje |
|---|---|---|
| Ya hay un Sprint activo | `errors.Is(err, sprint.ErrActiveSprintExists)`; `errors.As(err, &*sprint.ActiveSprintExistsError)` da `Number` | `ya hay un Sprint activo (Sprint N)` |
| Historia ya asignada | `errors.Is(err, sprint.ErrActiveSprintConflict)`; `errors.As(err, &*sprint.StoryAlreadyAssignedError)` da `StoryID` y `SprintNumber` | `la historia X ya está asignada al Sprint N` |

### Cómo se aplica la regla

En `PostgreSQLSprintStore.CreateSprint`, dentro de la transacción y antes de tomar el número,
`lockProjectWithoutActiveSprint` bloquea la fila del proyecto (`SELECT … FOR UPDATE`) y busca
un Sprint del proyecto con estado `Activo`. Si existe, devuelve `ActiveSprintExistsError` y la
transacción se revierte sin incrementar `next_sprint_number`. El bloqueo ordena las creaciones
simultáneas: la segunda espera, ve el Sprint de la primera y se rechaza. No hay migraciones ni
índices nuevos.
