# Tasks: Crear y asignar Sprint

**Input**: Design documents from `specs/003-create-sprint/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: TDD/BDD requeridos por la constitución; las pruebas deben escribirse antes de implementar y ejecutar reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Preparar Go, dependencias PostgreSQL/Docker y esquema de Sprints.

- [ ] T001 Verificar o inicializar el módulo Go en `go.mod` con Go 1.27.1
- [ ] T002 Verificar o agregar `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` en `go.mod` y actualizar `go.sum`
- [ ] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL 16 en la base `seguimiento_y_medicion`
- [ ] T004 [P] Crear la migración SQL `db/migrations/003_create_sprints.sql` para Sprints, asociaciones, estados e identificador secuencial por proyecto; depende de que las migraciones `001_create_projects.sql` y `002_create_backlog_stories.sql` ya estén aplicadas
- [ ] T005 Instalar la herramienta `github.com/golang-migrate/migrate` si no está instalada y documentar en `specs/003-create-sprint/quickstart.md` la aplicación de migraciones en orden numérico

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Crear el modelo, errores, store y transacciones compartidos por las tres historias.

- [ ] T006 [P] Definir `Project`, `Sprint`, `UserStory` y `SprintCreationRequest` en `src/sprint/model.go`, incluyendo estado `Activo`, Goal obligatorio e identificador no editable
- [ ] T007 [P] Definir errores explícitos para proyecto inexistente, Goal vacío, fechas inválidas, historia no disponible, conflicto de Sprint activo y persistencia en `src/sprint/errors.go`
- [ ] T008 Definir `SprintStore` con operaciones `CreateSprint`, `AssignStories` y `RemoveStory` en `src/sprint/store.go`
- [ ] T009 Implementar la generación secuencial `Sprint 1`, `Sprint 2` y sucesivos con protección transaccional en `src/sprint/sequence.go`
- [ ] T010 Implementar `PostgreSQLSprintStore` con `database/sql`, `github.com/jackc/pgx/v5/stdlib` y transacciones para Sprint, asociaciones y estados en `src/sprint/postgres_sprint_store.go`
- [ ] T011 [P] Crear el store controlado para pruebas unitarias en `src/sprint/test_store.go`, sin sustituir la integración PostgreSQL
- [ ] T012 [P] Definir helpers de historias `Pendiente`, `En Sprint` y asociación exclusiva en `src/sprint/story_assignment.go`

**Checkpoint**: El modelo, la secuencia, las transiciones y el adaptador PostgreSQL están listos para los casos de uso.

---

## Phase 3: User Story 1 - Crear Sprint y asignar historias (Priority: P1) 🎯 MVP

**Goal**: Crear un Sprint válido con identificador automático, Goal obligatorio y tres historias pendientes asignadas exclusivamente.

**Independent Test**: Con un proyecto y tres historias `Pendiente`, crear un Sprint con Goal y fechas válidas; verificar identificador `Sprint 1`, asociaciones, estado `En Sprint` y persistencia PostgreSQL.

### Tests for User Story 1

- [ ] T013 [P] [US1] Escribir prueba unitaria de creación con tres historias pendientes, estado `En Sprint` e identificador automático en `src/sprint/service_test.go`
- [ ] T014 [P] [US1] Escribir pruebas unitarias de Goal vacío e identificador no editable en `src/sprint/service_test.go`
- [ ] T015 [P] [US1] Escribir prueba unitaria de conflicto cuando una historia ya pertenece a otro Sprint activo en `src/sprint/service_test.go`
- [ ] T016 [P] [US1] Escribir prueba unitaria del movimiento quitar-primero/asignar-después en `src/sprint/service_test.go`
- [ ] T017 [P] [US1] Escribir prueba de integración PostgreSQL para Sprint, tres asociaciones, estados y secuencia en `src/sprint/postgres_sprint_store_integration_test.go`
- [ ] T018 [P] [US1] Escribir escenarios BDD de creación, conflicto, movimiento, identificador automático y Goal obligatorio en `test/sprint_bdd_test.go`

### Implementation for User Story 1

- [ ] T019 [US1] Implementar `CreateSprint` en `src/sprint/service.go`, validando proyecto, Goal, fechas presentes, historias seleccionadas y estados antes de mutar
- [ ] T020 [US1] Implementar en `src/sprint/sequence.go` el identificador automático no editable con secuencia por proyecto y sin consumir números en operaciones inválidas
- [ ] T021 [US1] Implementar en `src/sprint/story_assignment.go` la asignación exclusiva: una historia no puede pertenecer a dos Sprints activos y el conflicto debe bloquearse
- [ ] T022 [US1] Implementar el movimiento explícito en `src/sprint/service.go`, quitando primero la historia del Sprint actual, devolviéndola a `Pendiente` y permitiendo luego la nueva asignación
- [ ] T023 [US1] Completar `PostgreSQLSprintStore` en `src/sprint/postgres_sprint_store.go` con transacción para crear Sprint, asociar historias y actualizar estados sin mutaciones parciales
- [ ] T024 [US1] Habilitar la integración PostgreSQL con variables de conexión, fixtures de proyecto/historias y limpieza en `src/sprint/postgres_sprint_store_integration_test.go`

**Checkpoint**: US1 crea y persiste un Sprint con asignaciones exclusivas, permite movimientos en dos pasos y rechaza Goal vacío.

---

## Phase 4: User Story 2 - Validar el período del Sprint (Priority: P2)

**Goal**: Impedir Sprints con fecha final anterior o igual a la inicial.

**Independent Test**: Intentar crear un Sprint con `endDate <= startDate`; verificar error temporal, ausencia de Sprint y ninguna mutación de historias.

### Tests for User Story 2

- [ ] T025 [P] [US2] Escribir pruebas unitarias para `endDate < startDate` y `endDate == startDate` en `src/sprint/service_test.go`
- [ ] T026 [P] [US2] Escribir prueba de que fechas inválidas no consumen el identificador secuencial ni llaman al store en `src/sprint/service_test.go`
- [ ] T027 [P] [US2] Escribir el escenario BDD de fechas inválidas en `test/sprint_bdd_test.go`

### Implementation for User Story 2

- [ ] T028 [US2] Implementar en `src/sprint/service.go` la validación estricta `endDate > startDate`, rechazando igualdad y anterior con error temporal explícito
- [ ] T029 [US2] Verificar en `src/sprint/service_test.go` y `test/sprint_bdd_test.go` que el rechazo conserva datos corregibles y no persiste cambios

**Checkpoint**: US2 rechaza ambos períodos inválidos sin alterar el comportamiento válido de US1.

---

## Phase 5: User Story 3 - Crear Sprint sin historias (Priority: P3)

**Goal**: Permitir crear un Sprint válido sin historias y mostrar una advertencia no bloqueante.

**Independent Test**: Crear un Sprint con Goal y fechas válidas y una selección vacía; verificar persistencia, colección vacía y advertencia.

### Tests for User Story 3

- [ ] T030 [P] [US3] Escribir prueba unitaria de Sprint válido sin historias, advertencia y estado `Activo` en `src/sprint/service_test.go`
- [ ] T031 [P] [US3] Escribir prueba de integración PostgreSQL para Sprint sin asociaciones en `src/sprint/postgres_sprint_store_integration_test.go`
- [ ] T032 [P] [US3] Escribir el escenario BDD de Sprint sin historias en `test/sprint_bdd_test.go`

### Implementation for User Story 3

- [ ] T033 [US3] Implementar en `src/sprint/service.go` la creación con `storyIDs` vacío sin tratarla como error y devolviendo advertencia explícita
- [ ] T034 [US3] Completar en `src/sprint/postgres_sprint_store.go` la persistencia de Sprints sin asociaciones dentro de una transacción válida
- [ ] T035 [US3] Verificar en `src/sprint/service_test.go` y `src/sprint/postgres_sprint_store_integration_test.go` que una advertencia no impide crear el Sprint

**Checkpoint**: US3 permite registrar la estructura de la iteración sin historias y deja el Sprint listo para asignaciones posteriores.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación y trazabilidad.

- [ ] T036 [P] Documentar variables de conexión, arranque y apagado de PostgreSQL Docker en `specs/003-create-sprint/quickstart.md`
- [ ] T037 [P] Revisar que los errores de Sprint no expongan detalles internos de PostgreSQL en `src/sprint/errors.go`
- [ ] T038 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T039 Ejecutar `go test ./...` y completar todos los escenarios del `quickstart.md`
- [ ] T040 Revisar trazabilidad de FR-001 a FR-015 y SC-001 a SC-008 en `specs/003-create-sprint/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 debe preceder T002; T003 y T004 pueden ejecutarse en paralelo; T005 requiere que T004 esté creada.
- **Foundational (Phase 2)**: Depende de Setup; T006-T009 habilitan el servicio y T010 requiere T004 y T008.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2; es el MVP.
- **User Story 2 (Phase 4)**: Depende de las implementaciones `CreateSprint` y de secuencia T019/T020, y puede probarse de forma independiente.
- **User Story 3 (Phase 5)**: Depende de las implementaciones `CreateSprint` y `PostgreSQLSprintStore` T019/T023, pero no requiere historias asignadas.
- **Polish (Phase 6)**: Depende de las historias que se decida entregar.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-02 para disponer de un proyecto con historias `Pendiente`.
- **US2 (P2)**: Depende del servicio de creación de US1, pero es independientemente verificable.
- **US3 (P3)**: Depende del servicio de creación de US1, pero no depende de historias del backlog.

### Parallel Opportunities

- T003 y T004 pueden ejecutarse en paralelo después de confirmar el módulo.
- T006, T007, T011 y T012 pueden ejecutarse en paralelo.
- T013, T014, T015, T016, T017 y T018 pueden escribirse en paralelo antes de implementar US1.
- T025, T026 y T027 pueden escribirse en paralelo con la preparación de US1.
- T030, T031 y T032 pueden escribirse en paralelo con US2.
- T036 y T037 pueden ejecutarse en paralelo después de la implementación.

---

## Parallel Example: User Story 1

```text
T013: Prueba unitaria de creación y asignación en src/sprint/service_test.go
T014: Prueba de Goal vacío e identificador automático en src/sprint/service_test.go
T015: Prueba de exclusividad en src/sprint/service_test.go
T016: Prueba de movimiento en src/sprint/service_test.go
T017: Integración PostgreSQL en src/sprint/postgres_sprint_store_integration_test.go
T018: Escenarios BDD en test/sprint_bdd_test.go
```

## Parallel Example: User Story 2

```text
T025: Pruebas de fechas inválidas en src/sprint/service_test.go
T026: Prueba de secuencia sin mutación en src/sprint/service_test.go
T027: Escenario BDD en test/sprint_bdd_test.go
```

## Parallel Example: User Story 3

```text
T030: Prueba unitaria de Sprint vacío en src/sprint/service_test.go
T031: Integración PostgreSQL sin asociaciones en src/sprint/postgres_sprint_store_integration_test.go
T032: Escenario BDD en test/sprint_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Completar Setup y Foundational.
2. Escribir y hacer fallar las pruebas de creación, secuencia, exclusividad, movimiento y Goal.
3. Implementar US1 y ejecutar `docker compose up -d postgres` seguido de `go test ./...`.
4. Validar persistencia y asociaciones en PostgreSQL antes de avanzar a US2 y US3.

### Incremental Delivery

1. Entregar US1 como MVP funcional.
2. Agregar US2 para proteger el período temporal.
3. Agregar US3 para permitir Sprints sin historias.
4. Ejecutar la validación completa del quickstart antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
