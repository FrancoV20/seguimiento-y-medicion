# Data Model: Base de la aplicación web

Esta feature **no crea tablas ni migraciones**. Los modelos son estructuras en memoria; el único
dato leído de PostgreSQL es el control de versiones de la herramienta de migraciones.

## Config

Valores leídos del entorno al iniciar.

### Fields

- `databaseURL`: obligatoria (`DATABASE_URL`). Nunca se imprime completa.
- `addr`: opcional (`ADDR`), por defecto `:8080`.

### Rules

- Si `DATABASE_URL` falta o está vacía, el arranque falla con un mensaje que nombra la variable.
- Cualquier mensaje o registro usa la versión redactada de la URL (sin clave).
- Un `ADDR` no válido hace fallar el arranque con un mensaje que nombra el puerto afectado.

## Section

Apartado del menú.

### Fields

- `name`: texto visible en el menú.
- `path`: dirección base de la sección.
- `requiresProject`: verdadero si la sección necesita un proyecto activo (lo exige HU-20b).
- `requiresMember`: verdadero si necesita integrante identificado (lo exige HU-20b).

### Rules

- Solo las secciones registradas aparecen en el menú.
- Dos secciones con la misma `path` hacen fallar el arranque.

## PageData

Datos comunes entregados a cada plantilla.

### Fields

- `title`, `section` (la actual, para resaltarla), `menu` (secciones registradas).
- `context`: `*ActiveContext` opcional; nulo muestra "Sin proyecto seleccionado".
- `data`: contenido específico de la página.

### Rules

- HU-20b agrega `flash` (mensaje de una sola vista) y `form` (valores y errores de formulario).

## ActiveContext

Proyecto e integrante activos, que completa HU-20b. El tipo y sus funciones de acceso viven en
esta historia para que el layout compile sin depender de HU-20b.

### Fields

- `ProjectID`, `ProjectName`: proyecto activo, o vacíos si no hay.
- `MemberID`, `MemberName`: integrante activo, o vacíos si no hay.

### Operations

- `WithActiveContext(ctx, ac)`: devuelve un contexto que lleva el `ActiveContext`.
- `ActiveContextFrom(ctx)`: devuelve el `ActiveContext` o nulo si no hay.

## AppError

Clasificación de fallos para elegir la respuesta.

### Kinds

- `NotFound` → 404.
- `Unavailable` → 503 ("El servicio no está disponible temporalmente").
- `Internal` → 500 genérico.

### Rules

- El detalle técnico solo va al registro del servidor.
- Ninguna respuesta incluye rutas, consultas, trazas ni credenciales.
- Otros estados (405, y el 403 de HU-20b) se emiten con `RenderError(status, title, message)`.

## SchemaStatus

Resultado de leer `schema_migrations` (tabla de golang-migrate).

### Fields

- `version`: última versión aplicada.
- `dirty`: indica un estado interrumpido.

### Rules

- `dirty` o `version` menor a la esperada impide el arranque.
- `version` mayor a la esperada permite el arranque con una advertencia en el registro.

## SchemaVersionReader

Interfaz interna para leer el estado del esquema.

### Operations

- `ReadSchemaStatus(ctx)`: devuelve `SchemaStatus` o un error de disponibilidad.

## PostgreSQLSchemaVersionReader

Implementación concreta de `SchemaVersionReader` contra PostgreSQL.

### Rules

- Lee la fila única de `schema_migrations` mediante `database/sql` y pgx.
- Si la tabla no existe, se interpreta como "sin migraciones aplicadas" (versión 0).
- Debe contar con prueba de integración contra PostgreSQL antes de integrarse.
