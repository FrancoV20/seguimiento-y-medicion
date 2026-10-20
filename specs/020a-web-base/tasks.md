# Tasks: Base de la aplicación web

**Input**: Design documents from `specs/020a-web-base/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno (hoy la próxima es la 008), sin importar el número de la historia, y se agrega después de que sus dependencias estén en `main`.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Mensajes y formularios, protección contra envíos ajenos y contexto de proyecto e integrante son de HU-20b. Las tareas marcadas `[Extra]` no son criterios de aceptación; se hacen solo si sobra tiempo.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Preparar estructura, embebido de migraciones y entorno.

- [X] T001 Verificar `go.mod` con Go 1.27.1 y que `github.com/jackc/pgx/v5/stdlib` ya esté incorporado
- [X] T002 [P] Crear la estructura `src/web/`, `src/web/templates/`, `src/web/static/` y `src/cmd/server/`
- [X] T003 [P] Crear `db/migrations.go` con `//go:embed migrations/*.sql` y una función que devuelva la mayor versión de los archivos `.up.sql`
- [X] T004 [P] Verificar que `docker-compose.yml` expone PostgreSQL y documentar `DATABASE_URL` (con la nota del puerto 5432 o 5433) y `ADDR` en `specs/020a-web-base/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tipos y utilidades compartidos por todas las historias.

- [X] T005 [P] Definir los textos compartidos de las páginas de error (no encontrado, método no permitido, error inesperado, "El servicio no está disponible temporalmente") en `src/web/messages.go`
- [X] T006 [P] Definir `AppError` (`NotFound`, `Unavailable`, `Internal`) en `src/web/errors.go`
- [X] T007 [P] Definir `Section` y `Registry` con rechazo de direcciones duplicadas en `src/web/sections.go`
- [X] T008 [P] Definir `PageData`, `ActiveContext`, `WithActiveContext` y `ActiveContextFrom` en `src/web/render.go`
- [X] T009 [P] Crear los estilos locales base (estructura, menú, sección resaltada, páginas de error) en `src/web/static/app.css`

**Checkpoint**: Los tipos compartidos existen; las historias pueden escribir sus pruebas.

---

## Phase 3: User Story 1 - Arrancar la aplicación y ver la página de inicio (Priority: P1) 🎯 MVP

**Goal**: Servidor que arranca de forma confiable, muestra la página de inicio y falla con mensajes claros.

**Independent Test**: Arrancar con base disponible y abrir la dirección; repetir con configuración ausente, base caída, base lenta, migraciones pendientes, esquema `dirty` y puerto ocupado.

### Tests for User Story 1

- [X] T010 [P] [US1] Escribir pruebas de `LoadConfig` (variable faltante nombrada sin valores, valor por defecto de `ADDR`, redacción de la URL) en `src/web/config_test.go`
- [X] T011 [P] [US1] Escribir pruebas de la apertura con reintentos (éxito al N-ésimo intento, agotamiento sin exponer la clave ni la URL completa) en `src/web/database_test.go`
- [X] T012 [P] [US1] Escribir pruebas por tabla de la verificación de esquema (versión menor, `dirty`, igual, mayor con advertencia, sin tabla) en `src/web/schema_test.go`
- [X] T013 [P] [US1] Escribir la prueba de integración de `PostgreSQLSchemaVersionReader` contra PostgreSQL real en `src/web/postgres_schema_version_reader_integration_test.go`
- [X] T014 [P] [US1] Escribir la prueba de apagado ordenado con una solicitud en curso (completa en hasta 10 s y libera la base) en `src/web/server_test.go`
- [X] T015 [P] [US1] Escribir la prueba de puerto en uso o no válido: informa el puerto afectado sin detalles internos y no queda en ejecución, en `src/web/server_test.go`
- [X] T016 [P] [US1] Escribir los escenarios BDD de arranque y página de inicio en `test/web_base_bdd_test.go`

### Implementation for User Story 1

- [X] T017 [US1] Implementar `LoadConfig` y la redacción de valores en `src/web/config.go`
- [X] T018 [US1] Implementar la apertura de conexión con reintentos (10 intentos, 2 s) en `src/web/database.go`
- [X] T019 [US1] Implementar `SchemaVersionReader` y `CheckSchema` en `src/web/schema.go`
- [X] T020 [US1] Implementar `PostgreSQLSchemaVersionReader` en `src/web/postgres_schema_version_reader.go`
- [X] T021 [US1] Implementar la construcción del servidor, la ruta `GET /`, `Server.Use` y el apagado ordenado en `src/web/server.go`
- [X] T022 [US1] Implementar el arranque (configuración, base, esquema, rutas, señales) en `src/cmd/server/main.go`, informando puerto en uso sin detalles internos

**Checkpoint**: US1 arranca o falla con mensajes claros y muestra la página de inicio; HU-20b puede empezar a usar el servidor.

---

## Phase 4: User Story 2 - Navegar entre secciones con una estructura consistente (Priority: P1)

**Goal**: Layout común con menú dinámico, sección actual, espacio de contexto y recursos locales.

**Independent Test**: Registrar secciones de prueba y verificar menú, resaltado, "Sin proyecto seleccionado", ausencia de recursos externos y caracteres especiales.

### Tests for User Story 2

- [X] T023 [P] [US2] Escribir pruebas del menú: solo secciones registradas, sección actual resaltada y nueva sección sin tocar el layout en `src/web/sections_test.go`
- [X] T024 [P] [US2] Escribir la prueba de que registrar dos secciones con la misma dirección hace fallar el arranque con un mensaje claro en `src/web/sections_test.go`
- [X] T025 [P] [US2] Escribir pruebas de la página renderizada: espacio de contexto "Sin proyecto seleccionado", `Content-Type` UTF-8 y texto con tildes, ñ y símbolos en `src/web/render_test.go`
- [X] T026 [P] [US2] Escribir la prueba que verifica que el HTML no referencia otros orígenes y que `/static/app.css` se sirve localmente en `src/web/render_test.go`
- [X] T027 [P] [US2] Escribir los escenarios BDD de estructura y navegación en `test/web_base_bdd_test.go`

### Implementation for User Story 2

- [X] T028 [US2] Implementar plantillas embebidas, render a buffer y funciones auxiliares en `src/web/render.go`
- [X] T029 [US2] Crear `layout.html` y `home.html` en `src/web/templates/` con menú, resaltado y espacio de contexto
- [X] T030 [US2] Servir `/static/` desde los archivos embebidos en `src/web/server.go`

**Checkpoint**: US2 ofrece la estructura que HU-20b y HU-21 a HU-25 consumen.

---

## Phase 5: User Story 3 - Recibir errores claros sin exponer detalles internos (Priority: P2)

**Goal**: Páginas 404, 405, 500 y 503 consistentes, con estructura común, sin detalles internos y con recuperación automática.

**Independent Test**: Solicitar rutas inexistentes, parámetros malformados y métodos no soportados; provocar un pánico; detener y reanudar la base con el servidor activo.

### Tests for User Story 3

- [X] T031 [P] [US3] Escribir pruebas de 404 con estructura común y enlace al inicio para dirección inexistente y parámetro malformado, negativo o muy grande en `src/web/errors_test.go`
- [X] T032 [P] [US3] Escribir la prueba de que una plantilla que falla a mitad del render produce la página de error completa y nunca HTML parcial en `src/web/render_test.go`
- [X] T033 [P] [US3] Escribir la prueba de 405 con estructura común y servidor activo en `src/web/errors_test.go`
- [X] T034 [P] [US3] Escribir la prueba de pánico: página 500 genérica con estructura común, servidor activo, detalle solo en el log en `src/web/middleware_test.go`
- [X] T035 [P] [US3] Escribir la prueba que verifica que ninguna página de error contiene rutas, SQL, trazas ni el valor de `DATABASE_URL` en `src/web/errors_test.go`
- [X] T036 [P] [US3] Escribir la prueba de integración con un proxy TCP local que se pausa y reanuda delante de PostgreSQL: 503 con la base caída (estructura común sin consultar la base) y respuesta normal al volver, sin reiniciar, en `src/web/database_outage_integration_test.go`
- [X] T037 [P] [US3] Escribir los escenarios BDD de errores en `test/web_base_bdd_test.go`

### Implementation for User Story 3

- [X] T038 [US3] Implementar `Fail`, `RenderError` y las páginas 404, 405, 500 y 503 con `error.html` en `src/web/errors.go` y `src/web/templates/error.html`
- [X] T039 [US3] Implementar el middleware de recuperación de pánicos con registro de la traza en `src/web/middleware.go`
- [X] T040 [US3] Detectar indisponibilidad de la base (error de conexión, `driver.ErrBadConn`, `context.DeadlineExceeded`) y mapearla a 503 en `src/web/errors.go`
- [X] T041 [US3] Envolver el `ServeMux` para que 404 y 405 usen la estructura común, verificando antes el comportamiento de `mux.Handler(r)` en Go 1.27.1 (ver Decision 10 de `research.md`), en `src/web/server.go`
- [ ] T042 [US3] [Extra] Agregar un tiempo máximo por solicitud con contexto en `src/web/middleware.go` (no es criterio de aceptación)

**Checkpoint**: US3 mantiene el servidor en pie y sin filtrar detalles ante cualquier fallo.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación y trazabilidad.

- [X] T043 [P] Escribir una prueba de integración que cree una base descartable, aplique todos los `.up.sql` de `db/migrations/` en orden y la borre al terminar (nunca usa la base de desarrollo), para detectar huecos u orden incorrecto, en `db/migrations_integration_test.go`
- [X] T044 [P] Documentar variables, arranque, apagado y la verificación manual de caída de la base en `specs/020a-web-base/quickstart.md`
- [X] T045 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/errors.go` y `src/web/config.go`
- [X] T046 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [X] T047 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [X] T048 Coordinar con quien hace HU-20b que cada sección se registra con `Server.Register`, que `ActiveContext` y `Server.Use` son sus puntos de extensión y que HU-20b puede comenzar al cerrar US1 y US2
- [X] T049 Guardar la salida de `go test -count=1 -p 1 -v ./...` en `specs/020a-web-base/evidencia-pruebas.txt` y la cobertura de `go test -count=1 -p 1 -cover ./...` para el informe de métricas y cobertura
- [X] T050 Revisar trazabilidad de FR-001 a FR-017 y SC-001 a SC-007 en `specs/020a-web-base/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 antes de T003; T002, T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T002; T005 a T009 en paralelo.
- **User Story 1 (Phase 3)**: Depende de Phase 2 y T003; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T021 (servidor); desbloquea la parte visual de HU-20b y HU-22 a HU-25.
- **User Story 3 (Phase 5)**: Depende de T021 y T028; T036 requiere la prueba de proxy sobre PostgreSQL real.
- **Polish (Phase 6)**: Depende de las historias que se entreguen.

### User Story Dependencies

- **US1 (P1)**: Depende de las migraciones existentes solo para verificar el esquema; las pruebas de configuración y reintentos son independientes de la base de datos.
- **US2 (P1)**: Depende del servidor de US1, pero es verificable con secciones de prueba.
- **US3 (P2)**: Depende del servidor y del renderizado; es independiente de US2 salvo por la estructura común de las páginas de error (T029).
- **HU-20b**: Depende de US1 a US3 (usa `Fail`, `RenderError`, `Server.Use`, `ActiveContext` y el layout).
- **HU-21 a HU-25**: Dependen de HU-20a y HU-20b completas.

### Cross-feature Dependencies

- **HU-20b** consume `Section`, `ActiveContext`, `RenderError`, `Fail`, `Server.Use` y el renderizado; requiere US1 a US3.
- **HU-21 a HU-25** consumen `Server.Register`; HU-21 puede comenzar al cerrar US1 y US2 de HU-20a y la parte de formularios de HU-20b.

### Parallel Opportunities

- T002, T003 y T004; T005 a T009; las pruebas de cada historia son [P] entre sí.
- US2 y US3 pueden desarrollarse en paralelo después de US1 (T021).

## Parallel Example: User Story 1

```text
T010: Pruebas de configuración en src/web/config_test.go
T011: Pruebas de reintentos en src/web/database_test.go
T012: Verificación de esquema en src/web/schema_test.go
T013: Integración PostgreSQL en src/web/postgres_schema_version_reader_integration_test.go
T014: Apagado ordenado en src/web/server_test.go
T015: Puerto en uso en src/web/server_test.go
T016: Escenarios BDD en test/web_base_bdd_test.go
```

## Parallel Example: User Story 2

```text
T023: Pruebas del menú en src/web/sections_test.go
T024: Dirección duplicada en src/web/sections_test.go
T025: Pruebas de la página renderizada en src/web/render_test.go
T026: Recursos locales en src/web/render_test.go
T027: Escenarios BDD en test/web_base_bdd_test.go
```

## Parallel Example: User Story 3

```text
T031: Pruebas de 404 en src/web/errors_test.go
T033: Prueba de 405 en src/web/errors_test.go
T034: Prueba de pánico en src/web/middleware_test.go
T035: Prueba de que no se exponen detalles en src/web/errors_test.go
T036: Caída y recuperación de la base en src/web/database_outage_integration_test.go
T037: Escenarios BDD en test/web_base_bdd_test.go
```

## Implementation Strategy

### MVP First (US1 y US2)

1. Completar Setup y Foundational.
2. Escribir y observar fallar las pruebas de US1 y US2.
3. Implementar US1 y US2; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
4. Avisar al equipo que HU-20b puede empezar con el servidor y el layout.

### Incremental Delivery

1. Entregar US1 y US2 (base mínima para HU-20b).
2. Agregar US3 (errores).
3. Ejecutar la validación completa de `quickstart.md` antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- El nombre y las columnas de la tabla `schema_migrations` deben verificarse contra la versión instalada de golang-migrate antes de escribir T013 y T020.
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base local con `docker compose down -v` y volver a migrar.

## Trazabilidad

Estado al cierre: todos los requisitos tienen al menos una prueba automatizada. T042 (tiempo
máximo por solicitud) es `[Extra]`, no cubre ningún requisito y queda pendiente.

| Requisito | Pruebas |
|---|---|
| FR-001 | T016, T025 |
| FR-002 | T010, T016 |
| FR-003 | T011, T016 |
| FR-004 | T011, T016, T035 |
| FR-005 | T012, T013, T016, T043 |
| FR-006 | T015, T016 |
| FR-007 | T014 |
| FR-008 | T025, T027 |
| FR-009 | T023, T024, T027 (incluye `Server.Register` con ruta en conflicto) |
| FR-010 | T026 |
| FR-011 | T025 |
| FR-012 | T031, T037 |
| FR-013 | T033, T037 |
| FR-014 | T034, T037 |
| FR-015 | T036, T037 |
| FR-016 | T035 |
| FR-017 | T032 |
| SC-001 | T010, T016 |
| SC-002 | T011, T016 |
| SC-003 | T031, T037 |
| SC-004 | T035 |
| SC-005 | T036 |
| SC-006 | T026 |
| SC-007 | T016 (`TestEscenarioWeb_PaginaDeInicioConMenu` falla si la página tarda más de 500 ms) |
