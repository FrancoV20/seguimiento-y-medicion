# Tasks: Crear nuevo proyecto

**Input**: Design documents from `specs/001-create-project/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: TDD/BDD requeridos por la constitución; las pruebas deben escribirse antes de la implementación y comprobar comportamiento real.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Inicializar Go, dependencias y PostgreSQL local para desarrollo e integración.

- [ ] T001 Inicializar el módulo Go en `go.mod` con Go 1.27.1 y el módulo del repositorio
- [ ] T002 Agregar `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` como dependencias de persistencia en `go.mod` y actualizar `go.sum`
- [ ] T003 [P] Configurar PostgreSQL 16, la base `seguimiento_y_medicion`, el volumen, el puerto 5432 y el healthcheck en `docker-compose.yml`
- [ ] T004 [P] Crear la migración SQL inicial en `db/migrations/001_create_projects.sql` para proyectos, integrantes y la relación entre ambos
- [ ] T005 Instalar la herramienta `github.com/golang-migrate/migrate` y documentar en `specs/001-create-project/quickstart.md` la aplicación de migraciones en orden numérico

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Crear el modelo, errores y contratos compartidos que bloquean ambas historias.

- [ ] T006 [P] Definir `Project`, `Member` y `ProjectCreationRequest` en `src/project/model.go`, incluyendo estado inicial `Activo`, fechas de calendario y la regla `endDate >= startDate`
- [ ] T007 [P] Definir errores de validación explícitos para nombre vacío, integrantes ausentes, integrante inexistente, integrante duplicado y período inválido en `src/project/errors.go`
- [ ] T008 Definir la interfaz `ProjectStore` y sus operaciones `Create(project)` y `MemberExists(id)` en `src/project/store.go`
- [ ] T009 Implementar `PostgreSQLProjectStore` con `database/sql`, `github.com/jackc/pgx/v5/stdlib` y transacciones para persistir proyecto, estado, fechas e integrantes en `src/project/postgres_store.go`
- [ ] T010 [P] Crear el store controlado para pruebas unitarias en `src/project/test_store.go`, sin reemplazar las pruebas de integración contra PostgreSQL

**Checkpoint**: La base Go, el esquema PostgreSQL, el contrato de almacenamiento y el adaptador concreto están listos para implementar los casos de uso.

---

## Phase 3: User Story 1 - Registrar un proyecto (Priority: P1) 🎯 MVP

**Goal**: Crear y persistir un proyecto válido con al menos un integrante, estado `Activo` y confirmación posterior a la persistencia.

**Independent Test**: Con Docker Compose levantado, enviar nombre, integrantes existentes y fechas válidas; verificar un proyecto persistido en PostgreSQL y un mensaje de éxito.

### Tests for User Story 1

- [ ] T011 [P] [US1] Escribir pruebas unitarias de creación válida, proyecto `Activo`, al menos un integrante y fecha de inicio igual a fecha de fin en `src/project/service_test.go`
- [ ] T012 [P] [US1] Escribir pruebas unitarias de campos obligatorios, integrante inexistente e integrante duplicado con error explícito y sin llamar al store en `src/project/service_test.go`
- [ ] T013 [P] [US1] Escribir la prueba de integración de persistencia exitosa en PostgreSQL, incluyendo proyecto e integrantes relacionados, en `src/project/postgres_store_integration_test.go`
- [ ] T014 [P] [US1] Escribir el escenario BDD de creación exitosa y confirmación en `test/project_bdd_test.go`

### Implementation for User Story 1

- [ ] T015 [US1] Implementar el caso de uso `CreateProject` en `src/project/service.go`, validando nombre no vacío, al menos un integrante, integrantes existentes, duplicados rechazados explícitamente y fechas obligatorias antes de persistir
- [ ] T016 [US1] Implementar el resultado de creación en `src/project/model.go`, devolviendo confirmación solo después de que `ProjectStore.Create` confirme la persistencia y alerta/error sin éxito cuando falle
- [ ] T017 [US1] Completar la integración de `PostgreSQLProjectStore` en `src/project/postgres_store.go`, usando una transacción `database/sql` para evitar proyectos o relaciones parciales
- [ ] T018 [US1] Habilitar la prueba de integración de PostgreSQL con variables de conexión y limpieza de datos en `src/project/postgres_store_integration_test.go`

**Checkpoint**: US1 permite crear un proyecto válido, persistirlo en PostgreSQL y verificarlo de forma independiente con `go test ./...`.

---

## Phase 4: User Story 2 - Evitar fechas inconsistentes (Priority: P2)

**Goal**: Rechazar un proyecto cuya fecha de finalización sea anterior a la fecha de inicio sin persistirlo.

**Independent Test**: Enviar un proyecto con `endDate < startDate`; verificar error de inconsistencia temporal, ausencia de mensaje de éxito y cero llamadas al store.

### Tests for User Story 2

- [ ] T019 [P] [US2] Escribir la prueba unitaria de fecha de finalización anterior a la inicial en `src/project/service_test.go`, verificando error temporal y store sin mutación
- [ ] T020 [P] [US2] Escribir el escenario BDD de fechas inválidas en `test/project_bdd_test.go`

### Implementation for User Story 2

- [ ] T021 [US2] Completar la validación de calendario en `src/project/service.go` para aceptar `endDate == startDate` y rechazar únicamente `endDate < startDate` con el error explícito correspondiente
- [ ] T022 [US2] Verificar en `src/project/service_test.go` y `test/project_bdd_test.go` que una fecha inválida conserva los datos corregibles, no registra proyectos y no muestra éxito

**Checkpoint**: US2 rechaza períodos inválidos de forma independiente sin alterar el comportamiento válido de US1.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación y calidad transversal.

- [ ] T023 [P] Documentar variables de conexión, arranque y apagado de PostgreSQL Docker en `specs/001-create-project/quickstart.md`
- [ ] T024 [P] Revisar que los errores de `src/project` no expongan detalles internos de PostgreSQL a la capa de presentación
- [ ] T025 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T026 Ejecutar `go test ./...` y completar todos los escenarios del `quickstart.md`
- [ ] T027 Revisar trazabilidad de FR-001 a FR-009 y SC-001 a SC-004 en `specs/001-create-project/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 debe preceder T002; T003 y T004 pueden ejecutarse en paralelo con T001; T005 requiere la migración T004.
- **Foundational (Phase 2)**: Depende de Setup; T006-T008 habilitan el servicio y T009 requiere T004 y T008.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2; es el MVP.
- **User Story 2 (Phase 4)**: Depende de la implementación `CreateProject` de T015; sus pruebas pueden escribirse en paralelo con las de US1.
- **Polish (Phase 5)**: Depende de las historias que se decida entregar.

### User Story Dependencies

- **US1 (P1)**: Depende de la infraestructura y contratos fundacionales; no depende de otra historia.
- **US2 (P2)**: Depende del mismo servicio de creación de US1, pero es independientemente verificable.

### Parallel Opportunities

- T003 y T004 pueden ejecutarse en paralelo después de iniciar el módulo.
- T005, T006 y T009 pueden ejecutarse en paralelo.
- T010, T011, T012 y T013 son pruebas independientes y pueden escribirse en paralelo antes de implementar.
- T018 y T019 pueden escribirse en paralelo con el trabajo de pruebas de US1.
- T022 y T023 pueden ejecutarse en paralelo después de la implementación.

---

## Parallel Example: User Story 1

```text
T010: Pruebas unitarias de creación válida y fecha igual en src/project/service_test.go
T011: Pruebas unitarias de obligatorios, integrante inexistente y duplicado en src/project/service_test.go
T012: Prueba de integración PostgreSQL en src/project/postgres_store_integration_test.go
T013: Escenario BDD en test/project_bdd_test.go
```

## Parallel Example: User Story 2

```text
T018: Prueba unitaria de endDate < startDate en src/project/service_test.go
T019: Escenario BDD de fechas inválidas en test/project_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Completar Setup y Foundational.
2. Escribir y hacer fallar las pruebas de creación válida, persistencia y validaciones.
3. Implementar US1 y ejecutar `docker compose up -d postgres` seguido de `go test ./...`.
4. Validar la creación real en PostgreSQL antes de avanzar a US2.

### Incremental Delivery

1. Entregar US1 como MVP funcional.
2. Agregar US2 para proteger la consistencia temporal.
3. Ejecutar la validación completa del quickstart antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
