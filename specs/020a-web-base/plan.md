# Implementation Plan: Base de la aplicación web

**Branch**: `020a-web-base` | **Date**: 2026-10-17 | **Updated**: 2026-10-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from [spec.md](spec.md)

## Summary

Implementar la base web de la aplicación como Server-Side Rendering en Go: un servidor
`net/http` con plantillas `html/template` embebidas, registro de secciones que alimenta el menú,
arranque seguro (configuración, reintentos de conexión, verificación de migraciones, puerto,
apagado ordenado) y manejo uniforme de errores 404/405/500/503 con la estructura común. No agrega
tablas ni migraciones. Los mensajes y formularios, la protección contra envíos ajenos y el
contexto de proyecto e integrante son de HU-20b.

> **Supuestos de estructura a verificar en el repositorio** (ajustar rutas si difieren):
> paquete `src/web`, punto de entrada `src/cmd/server/main.go`, migraciones en `db/migrations/`.

## Technical Context

**Language/Version**: Go 1.27.1

**Primary Dependencies**: Biblioteca estándar de Go (`net/http`, `html/template`, `embed`,
`log/slog`, `os/signal`) y `github.com/jackc/pgx/v5/stdlib` ya incorporado en HU-01/HU-07. No se
agregan dependencias nuevas.

**Storage**: PostgreSQL con Docker Compose; esta feature solo lee la tabla `schema_migrations`
(versión y estado `dirty`) que mantiene golang-migrate, mediante `PostgreSQLSchemaVersionReader`.

**Testing**: `go test -count=1 -p 1 ./...` con pruebas unitarias, `net/http/httptest` para
handlers, escenarios BDD en `test/web_base_bdd_test.go` y pruebas de integración contra
PostgreSQL real. `-p 1` evita que los paquetes de integración se pisen al compartir una base.

**Target Platform**: Aplicación Go multiplataforma; desarrollo en Windows con Docker Desktop.

**Project Type**: Aplicación web SSR (capa de entrada HTTP sobre el núcleo de dominio en Go).

**Performance Goals**: Página de inicio en menos de 500 ms bajo uso normal.

**Constraints**: Sin recursos externos (sin CDN), sin dependencias nuevas, errores sin detalles
internos, mensajes en español, respuestas en UTF-8, ninguna página a medias ante un fallo.

**Scale/Scope**: Un servidor con una base de datos; incluye la estructura común y los errores,
no las pantallas de HU-21 a HU-25, ni los formularios y mensajes ni el contexto de proyecto e
integrante (HU-20b).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Value-Driven Product Scope**: PASS. Habilita HU-20b y HU-21 a HU-25 y la demostración del
  producto; no agrega funcionalidades fuera de la historia.
- **II. Quality by Construction**: PASS. Configuración, esquema, errores y render son
  responsabilidades separadas; las historias posteriores reutilizan los componentes.
- **III. Test-First and Evidence-Driven Delivery**: PASS. Cada escenario tiene pruebas unitarias o
  de handler y un escenario BDD; las pruebas se escriben antes de implementar.
- **IV. Agile, Collaborative Execution**: PASS. Alcance acotado y demostrable; la división de
  HU-20 en dos historias permite que dos personas trabajen sin pisarse.
- **V. Measurement and Continuous Improvement**: PASS. No altera las métricas; la decisión de
  dividir HU-20 queda para la retrospectiva.
- **VI. PostgreSQL Persistence**: PASS. La interfaz `SchemaVersionReader` tiene su adaptador
  `PostgreSQLSchemaVersionReader` con prueba de integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Go y PostgreSQL se mantienen; las plantillas SSR son
  una decisión de arquitectura interna que la constitución no restringe y queda registrada en
  `research.md`.

## Project Structure

### Documentation (this feature)

```text
specs/020a-web-base/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── tasks.md
└── checklists/requirements.md
```

No se crea `contracts/`: la interfaz externa son páginas HTML; sus reglas están en `spec.md`. Los
contratos internos que consumen otras historias se describen abajo.

### Source Code (repository root)

```text
db/
└── migrations.go                       # embed de migrations/*.sql (versión esperada)
src/
├── cmd/server/main.go                  # arranque: config, DB, esquema, rutas, apagado
└── web/
    ├── config.go                       # lectura de variables y redacción de valores
    ├── database.go                     # apertura de conexión con reintentos
    ├── schema.go                       # SchemaVersionReader y verificación de esquema
    ├── postgres_schema_version_reader.go
    ├── server.go                       # construcción del servidor, Use() y apagado ordenado
    ├── sections.go                     # registro de secciones y menú
    ├── render.go                       # plantillas embebidas, PageData, ActiveContext, render a buffer
    ├── errors.go                       # errores de aplicación, Fail, RenderError y páginas de error
    ├── messages.go                     # textos compartidos de las páginas de error
    ├── middleware.go                   # recover (y, opcional, tiempo máximo por solicitud)
    ├── templates/
    │   ├── layout.html
    │   ├── home.html
    │   └── error.html
    ├── static/app.css
    └── *_test.go
test/
└── web_base_bdd_test.go
```

**Structure Decision**: Se mantiene `src/` para el código y `test/` para BDD, agregando el paquete
`src/web`. El punto de entrada vive en `src/cmd/server`. `db/migrations.go` solo expone los
archivos de migración para calcular la versión esperada; no contiene SQL.

### Contratos internos para HU-20b y HU-21 a HU-25

- `Section{Name, Path, RequiresProject, RequiresMember}` y `Server.Register(section, routes)`:
  cada historia registra su sección y sus rutas; el menú se arma solo con las registradas.
  HU-20b usa los dos indicadores para exigir contexto.
- `Renderer.Page(w, r, template, data)`: aplica el layout, renderiza a buffer y envía la página
  completa o la de error.
- `PageData{Title, Section, Menu, Context, Data}`: `Context` es un `*ActiveContext` opcional; si
  es nulo se muestra "Sin proyecto seleccionado". HU-20b agrega `Flash` y `Form` a `PageData`.
- `ActiveContext{ProjectID, ProjectName, MemberID, MemberName}` con `WithActiveContext(ctx, ac)` y
  `ActiveContextFrom(ctx)`: 20a define el tipo y las funciones; HU-20b es quien lo completa desde
  un middleware y `Renderer.Page` lo lee del contexto de la solicitud.
- `Fail(w, r, err)`: traduce errores de aplicación a página 404, 503 o 500 sin detalles internos.
- `RenderError(w, r, status, title, message)`: página de error con la estructura común para
  cualquier estado (HU-20b la usa para el 403).
- `Server.Use(mw)`: agrega un middleware a la cadena; `recover` siempre queda primero. HU-20b la
  usa para la protección de envíos y el contexto.

## Complexity Tracking

No hay violaciones de la constitución que requieran justificación. `db/migrations.go` agrega un
paquete mínimo para embeber las migraciones y evitar dos fuentes de verdad sobre la versión
esperada.

## Post-Design Constitution Check

- **I. Value-Driven Product Scope**: PASS. El diseño se limita a la estructura base y no adelanta
  pantallas de otras historias.
- **II. Quality by Construction**: PASS. Cada archivo tiene una responsabilidad; las historias
  posteriores solo consumen los contratos internos.
- **III. Test-First and Evidence-Driven Delivery**: PASS. `quickstart.md` define pruebas
  unitarias, de handler, BDD y de integración, incluida una prueba con caída y recuperación real
  de la conexión a la base de datos.
- **IV. Agile, Collaborative Execution**: PASS. Las tareas permiten entregar US1 y US2 primero, de
  modo que HU-20b pueda empezar con el layout y el servidor.
- **V. Measurement and Continuous Improvement**: PASS. Sin cambios en métricas.
- **VI. PostgreSQL Persistence**: PASS. El único almacenamiento nuevo (lectura del esquema) tiene
  adaptador e integración contra PostgreSQL.
- **Technology & Quality Constraints**: PASS. Sin dependencias ni tecnologías nuevas.
