# Data Model: Formularios seguros y contexto de trabajo

Esta feature **no crea tablas ni migraciones**. Lee los proyectos e integrantes creados por HU-01
(migración `001`). La selección y el mensaje de una sola vista viven en cookies del navegador.

>> **Verificado (T001)** contra `001_create_projects.up.sql`:
> - `projects(id SERIAL, name, start_date, end_date, status)`
> - `members(id SERIAL, display_name)`
> - `project_members(project_id, member_id)`, PK compuesta, ON DELETE CASCADE en ambas FK.
> `MemberSummary.name` y `ResolvedContext.memberName` salen de `members.display_name`.
> Los `id` son INTEGER de 32 bits: una cookie con un valor fuera de 1..2147483647 se trata como
> ausencia ANTES de consultar, para que no se confunda con "base no disponible".
## Flash

Mensaje de una sola vista.

### Fields

- `kind`: `success`, `error` o `warning`.
- `text`: texto definido por el servidor.

### Rules

- Se escribe al terminar una acción y se borra al mostrarse.
- Se renderiza siempre escapado y con una etiqueta de texto además del color.
- Viaja en la cookie `flash`, de corta duración; solo el servidor escribe su contenido.

## FormView

### Fields

- `values`: valores enviados por campo, para conservarlos.
- `errors`: mensaje de error por campo.

### Rules

- Un campo con error se muestra con su mensaje junto al campo.
- Un fallo de guardado agrega el mensaje general `MsgOperationFailed` ("La operación no pudo
  completarse"), conserva los valores y no muestra mensaje de éxito.
- `PageData` (HU-20a) recibe `Flash` y `Form` en esta historia.

## Project (datos leídos de HU-01)

### Fields leídos

- `id`: identificador del proyecto (tabla `projects`).
- `name`: nombre del proyecto.

## Member (datos leídos de HU-01)

### Fields leídos

- `id`: identificador del integrante (tabla `members`).
- `name`: nombre del integrante.
- Pertenencia al proyecto: relación entre integrante y proyecto (tabla `project_members`).

## ProjectSummary

Elemento de la lista de selección.

### Fields

- `id`, `name`.

## MemberSummary

Elemento del selector de integrante.

### Fields

- `id`, `name`.

## ResolvedContext

Resultado de validar las cookies contra la base de datos.

### Fields

- `projectExists`: verdadero si el proyecto existe.
- `projectName`: nombre, si existe.
- `memberBelongs`: verdadero si el integrante pertenece al proyecto.
- `memberName`: nombre, si pertenece.

### Rules

- Sin identificador de proyecto, o con proyecto inexistente, no hay contexto.
- Un integrante solo es válido si pertenece al proyecto activo.

## ActiveContext (definido en HU-20a)

Contexto entregado a handlers y plantillas. El tipo, `WithActiveContext` y `ActiveContextFrom`
viven en HU-20a (`src/web/render.go`); esta historia lo completa.

### Fields

- `ProjectID`, `ProjectName`: proyecto activo, o vacíos si no hay.
- `MemberID`, `MemberName`: integrante activo, o vacíos si no hay.

### Rules

- Puede haber proyecto sin integrante; nunca integrante sin proyecto.
- Se reconstruye en cada solicitud a partir de las cookies validadas.

## Cookies

### Fields

- `ctx_proyecto`: identificador numérico del proyecto activo.
- `ctx_integrante`: identificador numérico del integrante activo.
- `flash`: mensaje de una sola vista (tipo y texto).

### Rules

- `ctx_proyecto` y `ctx_integrante` no llevan `Expires` (de sesión); todas llevan `HttpOnly`,
  `SameSite=Lax`, `Path=/` y `Secure` si hay HTTPS.
- Un valor no numérico, negativo, muy grande o inexistente se trata como ausencia y se borra.
- Cambiar a un proyecto distinto borra `ctx_integrante`.
- `flash` se borra al leerse.

## ReturnTo

Destino de regreso del campo `volver_a`.

### Rules

- Válido solo si empieza con una sola `/`, no contiene `//`, `\`, esquema ni caracteres de
  control, y su primer segmento coincide con una sección registrada.
- Si no es válido, el destino es `/`.

## ContextStore

Interfaz interna de solo lectura.

### Operations

- `ListProjects(ctx)`: devuelve los `ProjectSummary` para el selector.
- `ListMembers(ctx, projectID)`: devuelve los `MemberSummary` del proyecto.
- `ResolveContext(ctx, projectID, memberID)`: devuelve el `ResolvedContext` en una consulta.

## PostgreSQLContextStore

Implementación concreta de `ContextStore` contra PostgreSQL.

### Dependencies

- `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` mediante `database/sql`.

### Rules

- Solo realiza lecturas; no modifica proyectos ni integrantes.
- Distingue "no existe" de "base de datos no disponible" para que el middleware no limpie la
  selección ante una caída.
- Debe contar con pruebas de integración contra PostgreSQL antes de integrarse.
