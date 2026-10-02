# Data Model: Agregar historia al Product Backlog

## Project

Representa el contexto al que pertenece el Product Backlog.

### Fields

- `id`: identificador único del proyecto.
- `status`: estado del proyecto; debe ser `Activo` para aceptar nuevas historias.
- `backlog`: Product Backlog asociado.

### Rules

- Un proyecto que no está `Activo` rechaza el alta de historias.
- Cada historia creada queda asociada a un único proyecto.

## ProductBacklog

Colección ordenada de historias de un proyecto.

### Fields

- `projectID`: proyecto propietario.
- `stories`: historias registradas en el backlog.

### Rules

- La operación de alta valida todos los datos antes de agregar la historia.
- Un error de validación o persistencia no deja una historia parcialmente agregada.
- El backlog conserva la prioridad y el orden de cada historia para su planificación posterior.

## BacklogStore

Interfaz interna que abstrae la persistencia del Product Backlog.

### Operations

- `CreateStory(projectID, story)`: registra una historia y su asociación al proyecto o devuelve
	un error.
- `ProjectIsActive(projectID)`: verifica que el proyecto pueda recibir historias.

### Rules

- Las operaciones de persistencia deben confirmar la historia y su asociación como una unidad.
- Los errores de almacenamiento no producen un mensaje de éxito ni mutaciones parciales.

## PostgreSQLBacklogStore

Implementación concreta de `BacklogStore` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` como driver de PostgreSQL.
- `github.com/jackc/pgx/v5/stdlib` para registrar pgx y operar mediante `database/sql`.

### Rules

- Debe persistir la historia, su estado inicial `Pendiente`, su proyecto, prioridad, criterios y
	Story Points cuando estén informados.
- Debe ejecutar la creación y asociación dentro de una transacción de `database/sql`.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse al sistema.

## UserStory

Unidad de trabajo que se incorpora al backlog.

### Fields

- `id`: identificador único generado por el sistema.
- `title`: título obligatorio, no vacío después de normalizar espacios.
- `description`: descripción obligatoria, no vacía después de normalizar espacios.
- `priority`: valor obligatorio del conjunto `Alta`, `Media`, `Baja`.
- `acceptanceCriteria`: uno o más criterios con contenido no vacío.
- `storyPoints`: valor opcional; si existe debe pertenecer a `1, 2, 3, 5, 8, 13, 21, 34, 55, 89`.
- `status`: siempre `Pendiente` al crearse por este caso de uso.
- `estimationAlert`: alerta presente cuando `storyPoints` está ausente.

### State Transition

`Nueva solicitud válida -> Pendiente`.

Los cambios posteriores de estado quedan fuera de HU-02.

## CreateStoryRequest

Datos ingresados por el Scrum Master.

### Fields

- `projectID`: proyecto activo seleccionado.
- `title` y `description`: texto obligatorio.
- `priority`: prioridad seleccionada.
- `acceptanceCriteria`: colección de criterios.
- `storyPoints`: opcional.

El estado y el identificador no son campos editables de la solicitud.

## CreateStoryResult

Resultado de la operación de alta.

### Fields

- `story`: historia creada cuando la operación tiene éxito.
- `successMessage`: confirmación de registro.
- `warning`: alerta de estimación pendiente si no hay Story Points.
- `validationErrors`: errores que impiden crear la historia.

Un error de `PostgreSQLBacklogStore` se informa como error de persistencia y no como creación
exitosa.

### Invariants

- Éxito implica una única historia visible en el backlog.
- Ausencia de Story Points produce advertencia, no error.
- Story Points fuera de la escala produce error y no muta el backlog.
