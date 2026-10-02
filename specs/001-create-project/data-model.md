# Data Model: Crear nuevo proyecto

## Project

Unidad de gestión que habilita backlog, Sprints, esfuerzo y métricas.

### Fields

- `id`: identificador único asignado por el almacenamiento.
- `name`: nombre obligatorio, normalizado y no vacío.
- `memberIDs`: uno o más identificadores de integrantes sin duplicados.
- `startDate`: fecha de inicio de calendario.
- `endDate`: fecha de finalización de calendario.
- `status`: estado inicial `Activo`.

### Invariants

- `name` contiene al menos un carácter no vacío.
- `memberIDs` tiene al menos un elemento.
- `endDate >= startDate`.
- Un proyecto creado y persistido se encuentra disponible para seleccionar en HU-02.

## Member

Persona disponible para asociarse a un proyecto.

### Fields

- `id`: identificador existente del integrante.
- `displayName`: datos de presentación gestionados fuera de HU-01.

### Rules

- Cada `memberID` de una solicitud debe referir a un integrante disponible.
- Un integrante no debe repetirse en la misma solicitud; cualquier duplicado debe rechazarse
	con un error de validación explícito y no puede normalizarse ni ignorarse en silencio.

## ProjectCreationRequest

Datos ingresados desde Gestión de Proyectos.

### Fields

- `name`: texto del proyecto.
- `memberIDs`: integrantes seleccionados.
- `startDate` y `endDate`: período del proyecto.

El `id` y el `status` no son editables por el usuario durante el alta.

## ProjectStore

Frontera interna de persistencia.

### Operations

- `Create(project)`: registra un proyecto válido y devuelve su identificador o un error.
- `MemberExists(id)`: permite verificar que los integrantes seleccionados estén disponibles.

### Rules

- `Create` no debe confirmarse antes de que el almacenamiento acepte el proyecto.
- Un error de `Create` no debe producir un mensaje de éxito.

## PostgreSQLProjectStore

Implementación concreta de `ProjectStore` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` como driver de PostgreSQL.
- `github.com/jackc/pgx/v5/stdlib` para registrar el driver y operar mediante `database/sql`.

### Rules

- Debe persistir el proyecto, su estado `Activo`, su período y sus relaciones con integrantes.
- Debe ejecutar las operaciones de creación dentro de una transacción de `database/sql` para
	evitar proyectos o asociaciones parciales.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse al sistema.

## CreateProjectResult

Resultado para la capa de presentación.

### Fields

- `project`: proyecto creado cuando la operación es exitosa.
- `successMessage`: confirmación posterior a la persistencia.
- `validationErrors`: errores de campos o período.
- `storageError`: error de persistencia sin exponer detalles internos sensibles.

Un integrante duplicado produce un `validationError` explícito y evita llamar al almacenamiento.

### State Transition

`Solicitud -> validación -> ProjectStore.Create -> Proyecto Activo`.

Si falla la validación o el almacenamiento, el proyecto no se considera creado.
