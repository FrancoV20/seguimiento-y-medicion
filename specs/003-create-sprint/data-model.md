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

## StoryMove

Operación explícita para cambiar una historia de Sprint.

### Transition

`Sprint activo A + historia En Sprint -> quitar -> historia Pendiente -> asignar a Sprint B -> historia En Sprint`.

La asignación directa `Sprint activo A -> Sprint activo B` sin quitar primero debe rechazarse.
