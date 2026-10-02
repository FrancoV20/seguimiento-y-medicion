# Tasks: Agregar historia al Product Backlog

**Input**: Design documents from `specs/002-add-backlog-story/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: TDD/BDD requeridos por la constitución; las pruebas deben escribirse antes de la implementación y verificar comportamiento real.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Preparar módulo Go, dependencias PostgreSQL y esquema del backlog.

- [ ] T001 Verificar o inicializar el módulo Go en `go.mod` con Go 1.22 o posterior
- [ ] T002 Verificar o agregar `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` en `go.mod` y actualizar `go.sum`
- [ ] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL 16 en la base `seguimiento` para desarrollo e integración
- [ ] T004 [P] Crear la migración SQL `db/migrations/002_create_backlog_stories.sql` para historias, criterios de aceptación, relación con proyectos y campos de estimación

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Crear entidades, validadores y persistencia compartidos por las historias de HU-02.

- [ ] T005 [P] Definir `UserStory`, `AcceptanceCriterion`, `ProductBacklog` y `CreateStoryRequest` en `src/backlog/model.go`, con estado inicial `Pendiente` y Story Points opcionales
- [ ] T006 [P] Definir errores explícitos para proyecto inactivo, campos obligatorios, criterios vacíos, Fibonacci inválido y persistencia en `src/backlog/errors.go`
- [ ] T007 [P] Implementar la validación centralizada de la escala `1, 2, 3, 5, 8, 13, 21, 34, 55, 89` y ausencia de estimación en `src/backlog/story_points.go`
- [ ] T008 Definir `BacklogStore` con `CreateStory(projectID, story)` y `ProjectIsActive(projectID)` en `src/backlog/store.go`
- [ ] T009 Implementar `PostgreSQLBacklogStore` con `database/sql`, `github.com/jackc/pgx/v5/stdlib` y transacciones para historia, criterios y asociación en `src/backlog/postgres_backlog_store.go`
- [ ] T010 [P] Crear el store controlado para pruebas unitarias en `src/backlog/test_store.go`, sin sustituir la integración PostgreSQL

**Checkpoint**: El modelo, validadores, contrato de almacenamiento y adaptador PostgreSQL están disponibles para las historias de usuario.

---

## Phase 3: User Story 1 - Registrar historia en el backlog (Priority: P1) 🎯 MVP

**Goal**: Agregar una historia válida al Product Backlog de un proyecto activo en estado `Pendiente`, con Story Points opcionales y alerta cuando falte la estimación.

**Independent Test**: Con Docker Compose levantado y un proyecto `Activo`, registrar una historia con título, descripción, prioridad y criterios; verificar persistencia, estado `Pendiente`, confirmación y alerta si no hay Story Points.

### Tests for User Story 1

- [ ] T011 [P] [US1] Escribir pruebas unitarias de alta válida, asociación al proyecto y estado `Pendiente` en `src/backlog/service_test.go`
- [ ] T012 [P] [US1] Escribir pruebas unitarias de Story Points ausentes: creación permitida, ausencia explícita y alerta de estimación pendiente en `src/backlog/service_test.go`
- [ ] T013 [P] [US1] Escribir pruebas unitarias de valores Fibonacci válidos e inválidos en `src/backlog/story_points_test.go`
- [ ] T014 [P] [US1] Escribir la prueba de integración de `PostgreSQLBacklogStore` para historia, criterios y asociación persistidos en `src/backlog/postgres_backlog_store_integration_test.go`
- [ ] T015 [P] [US1] Escribir el escenario BDD de registro exitoso y estado inicial en `test/backlog_bdd_test.go`

### Implementation for User Story 1

- [ ] T016 [US1] Implementar el caso de uso `CreateStory` en `src/backlog/service.go`, validando proyecto `Activo`, título, descripción, prioridad y datos de la solicitud antes de mutar el backlog
- [ ] T017 [US1] Implementar en `src/backlog/service.go` el comportamiento de Story Points opcionales: permitir ausencia, conservarla explícitamente y devolver la alerta de estimación pendiente
- [ ] T018 [US1] Implementar en `src/backlog/story_points.go` el rechazo de valores fuera de `1, 2, 3, 5, 8, 13, 21, 34, 55, 89`
- [ ] T019 [US1] Completar `PostgreSQLBacklogStore` en `src/backlog/postgres_backlog_store.go` con transacción `database/sql` para persistir historia, criterios, prioridad, estado `Pendiente` y Story Points cuando estén informados
- [ ] T020 [US1] Habilitar la prueba de integración PostgreSQL con variables de conexión, fixtures de proyecto activo y limpieza en `src/backlog/postgres_backlog_store_integration_test.go`

**Checkpoint**: US1 permite registrar una historia válida, persistirla contra PostgreSQL y manejar Story Points ausentes o válidos de forma independiente.

---

## Phase 4: User Story 2 - Exigir criterios de aceptación (Priority: P2)

**Goal**: Impedir el registro de historias sin al menos un criterio de aceptación con contenido.

**Independent Test**: Intentar guardar una historia sin criterios o con criterios compuestos solo por espacios; verificar solicitud de corrección y ausencia de registro.

### Tests for User Story 2

- [ ] T021 [P] [US2] Escribir pruebas unitarias de criterios ausentes, colección vacía y criterios solo con espacios en `src/backlog/service_test.go`
- [ ] T022 [P] [US2] Escribir la prueba de que un error de criterios no llama a `BacklogStore.Create` ni deja persistencia parcial en `src/backlog/service_test.go`
- [ ] T023 [P] [US2] Escribir el escenario BDD de criterios de aceptación obligatorios en `test/backlog_bdd_test.go`

### Implementation for User Story 2

- [ ] T024 [US2] Implementar la validación de al menos un criterio no vacío en `src/backlog/service.go`, conservando los demás datos ingresados para corregirlos
- [ ] T025 [US2] Completar los errores y mensajes de validación de criterios en `src/backlog/errors.go`, solicitando agregar al menos un criterio
- [ ] T026 [US2] Verificar en `src/backlog/service_test.go` y `test/backlog_bdd_test.go` que una historia sin criterios no se agrega ni se muestra como creada

**Checkpoint**: US2 rechaza criterios ausentes de forma independiente sin afectar el alta válida de US1.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación y trazabilidad.

- [ ] T027 [P] Documentar variables de conexión, arranque y apagado de PostgreSQL Docker en `specs/002-add-backlog-story/quickstart.md`
- [ ] T028 [P] Revisar que los errores del backlog no expongan detalles internos de PostgreSQL en `src/backlog/errors.go`
- [ ] T029 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T030 Ejecutar `go test ./...` y completar todos los escenarios del `quickstart.md`
- [ ] T031 Revisar trazabilidad de FR-001 a FR-013 y SC-001 a SC-007 en `specs/002-add-backlog-story/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 debe preceder T002; T003 y T004 pueden ejecutarse en paralelo con T001.
- **Foundational (Phase 2)**: Depende de Setup; T005-T008 habilitan el servicio y T009 requiere T004 y T008.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T016 del servicio y puede agregar sus pruebas sin bloquear la validación de US1.
- **Polish (Phase 5)**: Depende de las historias que se decida entregar.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-01 para disponer de un proyecto `Activo`, pero es independiente dentro del backlog una vez creado el proyecto.
- **US2 (P2)**: Depende del caso de uso de creación de US1, pero es independientemente verificable.

### Parallel Opportunities

- T003 y T004 pueden ejecutarse en paralelo después de confirmar el módulo.
- T005, T006, T007 y T010 pueden ejecutarse en paralelo.
- T011, T012, T013, T014 y T015 son pruebas independientes y pueden escribirse en paralelo antes de implementar US1.
- T021, T022 y T023 pueden escribirse en paralelo con la preparación de US1.
- T027 y T028 pueden ejecutarse en paralelo después de la implementación.

---

## Parallel Example: User Story 1

```text
T011: Pruebas unitarias de alta válida en src/backlog/service_test.go
T012: Pruebas de Story Points opcionales y alerta en src/backlog/service_test.go
T013: Pruebas de escala Fibonacci en src/backlog/story_points_test.go
T014: Integración de PostgreSQL en src/backlog/postgres_backlog_store_integration_test.go
T015: Escenario BDD en test/backlog_bdd_test.go
```

## Parallel Example: User Story 2

```text
T021: Pruebas unitarias de criterios vacíos en src/backlog/service_test.go
T022: Prueba de ausencia de mutación en src/backlog/service_test.go
T023: Escenario BDD de criterios obligatorios en test/backlog_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Completar Setup y Foundational.
2. Escribir y hacer fallar las pruebas de alta, persistencia, Story Points y Fibonacci.
3. Implementar US1 y ejecutar `docker compose up -d postgres` seguido de `go test ./...`.
4. Validar la historia persistida en PostgreSQL antes de avanzar a US2.

### Incremental Delivery

1. Entregar US1 como MVP funcional.
2. Agregar US2 para garantizar criterios de aceptación verificables.
3. Ejecutar la validación completa del quickstart antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
