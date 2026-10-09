# Tasks: Registrar horas trabajadas

**Input**: Design documents from `specs/003-timesheet/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: TDD/BDD requeridos por la constitución; las pruebas deben escribirse antes de implementar y ejecutar reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Preparar Go, PostgreSQL/Docker y esquema de registros de esfuerzo.

- [ ] T001 Verificar o inicializar el módulo Go en `go.mod` con Go 1.27.1
- [ ] T002 Verificar o agregar `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` en `go.mod` y actualizar `go.sum`
- [ ] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL 16 en la base `seguimiento_y_medicion`
- [ ] T004 [P] Crear las migraciones SQL `db/migrations/005_create_effort_logs.up.sql` y `db/migrations/005_create_effort_logs.down.sql` para la tabla `effort_logs` (historia, integrante, fecha, actividad, horas `NUMERIC(5,2)` con `CHECK (hours > 0 AND hours <= 24)`); depende de que las migraciones `001_create_projects`, `002_create_backlog_stories` y `003_create_sprints` ya estén aplicadas
- [ ] T005 Instalar la herramienta `github.com/golang-migrate/migrate` si no está instalada y documentar en `specs/003-timesheet/quickstart.md` la aplicación de migraciones en orden numérico

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Crear modelos, aritmética de horas, validaciones y persistencia compartidos por las historias.

- [ ] T006 [P] Definir `EffortEntry`, `LogEffortRequest`, `LogEffortResult` y `AccumulatedEffort` en `src/effort/model.go`
- [ ] T007 [P] Definir errores explícitos y mensajes exactos para horas inválidas, actividad vacía, fecha requerida, integrante no válido, historia sin Sprint activo y persistencia en `src/effort/errors.go`
- [ ] T008 [P] Implementar el tipo `Hours` en centésimas de hora con interpretación de entrada, rango `(0, 24]`, máximo dos decimales y suma exacta en `src/effort/hours.go`
- [ ] T009 Definir `EffortStore` con `StoryAcceptsEffort`, `MemberBelongsToStoryProject`, `SaveEntry` y `AccumulatedHours` en `src/effort/store.go`
- [ ] T010 Implementar `PostgreSQLEffortStore` con `database/sql`, `github.com/jackc/pgx/v5/stdlib` y transacciones en `src/effort/postgres_effort_store.go`
- [ ] T011 [P] Crear el store controlado para pruebas unitarias en `src/effort/test_store.go`, sin sustituir la integración PostgreSQL
- [ ] T012 [P] Definir la validación de campos con acumulación de todos los errores en `src/effort/validation.go`

**Checkpoint**: El modelo, la aritmética, la validación, el contrato y el adaptador PostgreSQL están listos para las historias.

---

## Phase 3: User Story 1 - Registrar esfuerzo diario (Priority: P1) 🎯 MVP

**Goal**: Registrar un esfuerzo válido sobre una historia de un Sprint activo y devolver el esfuerzo real acumulado actualizado.

**Independent Test**: Con una historia asignada a un Sprint activo, registrar 4 horas con actividad "Desarrollo de endpoint en Go" y verificar que el acumulado aumenta 4 horas y que el registro queda persistido en PostgreSQL.

### Tests for User Story 1

- [ ] T013 [P] [US1] Escribir prueba unitaria de registro de 4 horas que suma 4 horas al acumulado de la historia en `src/effort/service_test.go`
- [ ] T014 [P] [US1] Escribir prueba unitaria de registros sucesivos (4 h y 2,5 h) que acumulan 6,5 h sin sobrescribir en `src/effort/service_test.go`
- [ ] T015 [P] [US1] Escribir prueba unitaria de rechazo cuando la historia no está asignada a un Sprint activo en `src/effort/service_test.go`
- [ ] T016 [P] [US1] Escribir la prueba de integración de guardado y acumulado con historia en Sprint activo en `src/effort/postgres_effort_store_integration_test.go`
- [ ] T017 [P] [US1] Escribir el escenario BDD "Carga de esfuerzo diario" en `test/effort_bdd_test.go`

### Implementation for User Story 1

- [ ] T018 [US1] Implementar `LogEffort` en `src/effort/service.go`, cargando el contexto de la historia desde `EffortStore` y usando un reloj inyectable
- [ ] T019 [US1] Implementar en `src/effort/service.go` el resultado con registro creado, `accumulatedHours` y `successMessage` solo después de persistir
- [ ] T020 [US1] Implementar `SaveEntry` en `src/effort/postgres_effort_store.go`, insertando el registro y calculando el acumulado en una misma transacción
- [ ] T021 [US1] Implementar `StoryAcceptsEffort` y `AccumulatedHours` en `src/effort/postgres_effort_store.go`, verificando que el Sprint de la historia esté `Activo`
- [ ] T022 [US1] Habilitar la integración PostgreSQL con variables de conexión, fixtures de proyecto, integrante, historia y Sprint activo, y limpieza en `src/effort/postgres_effort_store_integration_test.go`

**Checkpoint**: US1 registra esfuerzo y acumula horas de forma reproducible, y permite consultar el acumulado de una historia de forma independiente.

---

## Phase 4: User Story 2 - Rechazar horas inválidas (Priority: P2)

**Goal**: Rechazar horas negativas, cero o mayores a 24 sin modificar el acumulado.

**Independent Test**: Intentar registrar -1, 0 y 25 horas; verificar el mensaje "El valor de horas no es válido" y que el acumulado y la base permanecen sin cambios.

### Tests for User Story 2

- [ ] T023 [P] [US2] Escribir pruebas en tabla de `Hours` con -1, 0, 24, 24,01, 25, 0,5, 1,555, vacío y texto no numérico en `src/effort/hours_test.go`
- [ ] T024 [P] [US2] Escribir prueba unitaria de que una carga con horas inválidas no invoca `SaveEntry` ni cambia el acumulado en `src/effort/service_test.go`
- [ ] T025 [P] [US2] Escribir el escenario BDD "Carga de horas inválida" en `test/effort_bdd_test.go`
- [ ] T026 [P] [US2] Escribir prueba de integración de que la restricción `CHECK` rechaza horas fuera de rango escritas directamente en `src/effort/postgres_effort_store_integration_test.go`

### Implementation for User Story 2

- [ ] T027 [US2] Implementar en `src/effort/hours.go` la validación del rango `(0, 24]`, el máximo de dos decimales y el rechazo de valores no numéricos
- [ ] T028 [US2] Implementar en `src/effort/errors.go` el mensaje exacto "El valor de horas no es válido"
- [ ] T029 [US2] Integrar en `src/effort/validation.go` y `src/effort/service.go` la validación de horas antes de guardar
- [ ] T030 [US2] Verificar en `src/effort/service_test.go` que el acumulado de la historia es idéntico antes y después de una carga rechazada

**Checkpoint**: US2 rechaza horas inválidas con el mensaje correcto y sin mutaciones.

---

## Phase 5: User Story 3 - Exigir actividad, fecha e integrante (Priority: P2)

**Goal**: Exigir actividad realizada, fecha e integrante válido en cada registro e informar todos los errores juntos.

**Independent Test**: Enviar por separado un registro sin actividad, otro sin fecha y otro sin integrante válido; verificar cada mensaje y que no se guarda nada.

### Tests for User Story 3

- [ ] T031 [P] [US3] Escribir pruebas unitarias de actividad vacía y solo espacios en `src/effort/service_test.go`
- [ ] T032 [P] [US3] Escribir pruebas unitarias de fecha omitida y fecha posterior a la actual con reloj inyectado en `src/effort/service_test.go`
- [ ] T033 [P] [US3] Escribir pruebas unitarias de integrante omitido, inexistente y ajeno al proyecto de la historia en `src/effort/service_test.go`
- [ ] T034 [P] [US3] Escribir prueba unitaria de varios campos inválidos con todos los errores informados juntos en `src/effort/service_test.go`
- [ ] T035 [P] [US3] Escribir los escenarios BDD "Actividad realizada vacía", "Fecha de carga faltante" e "Integrante no especificado" en `test/effort_bdd_test.go`
- [ ] T036 [P] [US3] Escribir prueba de integración de `MemberBelongsToStoryProject` con integrante del proyecto y ajeno en `src/effort/postgres_effort_store_integration_test.go`

### Implementation for User Story 3

- [ ] T037 [US3] Implementar en `src/effort/validation.go` la normalización de la actividad y el rechazo de actividad vacía
- [ ] T038 [US3] Implementar en `src/effort/validation.go` la fecha obligatoria y el rechazo de fechas futuras
- [ ] T039 [US3] Implementar `MemberBelongsToStoryProject` en `src/effort/postgres_effort_store.go` y su uso en la validación del integrante
- [ ] T040 [US3] Implementar en `src/effort/errors.go` los mensajes exactos de actividad obligatoria, fecha requerida e integrante responsable
- [ ] T041 [US3] Verificar en `src/effort/service_test.go` y `test/effort_bdd_test.go` que ninguna validación fallida persiste registros ni modifica el acumulado

**Checkpoint**: US3 exige los campos obligatorios, informa todos los errores y no deja registros parciales.

---

## Phase 6: User Story 4 - Informar fallas al guardar (Priority: P3)

**Goal**: Informar que la operación no pudo completarse cuando falla el almacenamiento, sin mensaje de éxito ni cambios en el acumulado.

**Independent Test**: Simular una falla del almacenamiento con todos los campos válidos; verificar el mensaje "La operación no pudo completarse", la ausencia de mensaje de éxito y que el acumulado no cambia.

### Tests for User Story 4

- [ ] T042 [P] [US4] Escribir prueba unitaria con un store que falla al guardar: mensaje de no completado, sin `successMessage` y sin cambios en el acumulado en `src/effort/service_test.go`
- [ ] T043 [P] [US4] Escribir prueba de integración de falla transaccional (contexto cancelado o violación forzada) que no deja registros parciales en `src/effort/postgres_effort_store_integration_test.go`
- [ ] T044 [P] [US4] Escribir el escenario BDD "Error al guardar el registro de esfuerzo" en `test/effort_bdd_test.go`

### Implementation for User Story 4

- [ ] T045 [US4] Implementar en `src/effort/service.go` el mapeo de errores de almacenamiento a un `storageError` genérico "La operación no pudo completarse"
- [ ] T046 [US4] Implementar en `src/effort/postgres_effort_store.go` el `ROLLBACK` ante cualquier error de `SaveEntry`
- [ ] T047 [US4] Verificar en `src/effort/service_test.go` que un resultado con `storageError` nunca incluye `successMessage` ni registro

**Checkpoint**: US4 informa fallas sin falso éxito ni datos parciales.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación, integración con HU-07 y trazabilidad.

- [ ] T048 [P] Documentar variables de conexión, arranque y apagado de PostgreSQL Docker en `specs/003-timesheet/quickstart.md`
- [ ] T049 [P] Revisar que los errores de validación y PostgreSQL no expongan detalles internos en `src/effort/errors.go`
- [ ] T050 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T051 Ejecutar `go test ./...` y completar todos los escenarios del `quickstart.md`
- [ ] T052 Coordinar con el responsable de HU-07 que `actualHours` de cada historia se lee como la suma de `effort_logs.hours` y que la migración 005 se aplica antes de sus pruebas de integración
- [ ] T053 Revisar trazabilidad de FR-001 a FR-015 y SC-001 a SC-007 en `specs/003-timesheet/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 debe preceder T002; T003 y T004 pueden ejecutarse en paralelo; T005 requiere que T004 esté creada.
- **Foundational (Phase 2)**: Depende de Setup; T006-T009, T011 y T012 habilitan el dominio, y T010 requiere T004 y T009 para implementar el store PostgreSQL.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T018 de US1 para integrar la validación en el servicio, pero sus pruebas de `Hours` son independientes.
- **User Story 3 (Phase 5)**: Depende de T012 y T018; puede desarrollarse en paralelo con US2 después de Foundation. T039 requiere T010.
- **User Story 4 (Phase 6)**: Depende de T018 y T020; puede desarrollarse en paralelo con US2 y US3.
- **Polish (Phase 7)**: Depende de las historias que se decida entregar; T052 requiere que exista el código de HU-07.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-01 (proyectos e integrantes), HU-02 (historias) y HU-04 (Sprint activo y asignación de historias) para disponer de datos de contexto; en las pruebas unitarias se usan stores controlados.
- **US2 (P2)**: Depende de la validación de US1, pero es independientemente verificable sobre `Hours`.
- **US3 (P2)**: Depende del modelo de integrantes de HU-01, pero es independientemente verificable antes de guardar.
- **US4 (P3)**: Depende del guardado de US1, y es verificable con un store que falla.
- **HU-07**: Depende de US1 para obtener `actualHours`.

### Parallel Opportunities

- T003 y T004 pueden ejecutarse en paralelo después de confirmar el módulo.
- T006, T007, T008, T011 y T012 pueden ejecutarse en paralelo.
- T013, T014, T015, T016 y T017 pueden escribirse en paralelo antes de implementar US1.
- T023, T024, T025 y T026 pueden escribirse en paralelo con la implementación de US1.
- T031 a T036 pueden escribirse en paralelo con US2.
- T042, T043 y T044 pueden escribirse en paralelo con US3.
- T048 y T049 pueden ejecutarse en paralelo después de la implementación.

---

## Parallel Example: User Story 1

```text
T013: Prueba de registro de 4 horas en src/effort/service_test.go
T014: Prueba de acumulación sucesiva en src/effort/service_test.go
T015: Prueba de historia sin Sprint activo en src/effort/service_test.go
T016: Integración PostgreSQL en src/effort/postgres_effort_store_integration_test.go
T017: Escenario BDD en test/effort_bdd_test.go
```

## Parallel Example: User Story 2

```text
T023: Pruebas en tabla de Hours en src/effort/hours_test.go
T024: Prueba de rechazo sin guardar en src/effort/service_test.go
T025: Escenario BDD en test/effort_bdd_test.go
T026: Integración de restricción CHECK en src/effort/postgres_effort_store_integration_test.go
```

## Parallel Example: User Story 3

```text
T031: Prueba de actividad vacía en src/effort/service_test.go
T032: Prueba de fecha faltante o futura en src/effort/service_test.go
T033: Prueba de integrante no válido en src/effort/service_test.go
T035: Escenarios BDD en test/effort_bdd_test.go
```

## Parallel Example: User Story 4

```text
T042: Prueba de falla del store en src/effort/service_test.go
T043: Integración de falla transaccional en src/effort/postgres_effort_store_integration_test.go
T044: Escenario BDD en test/effort_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Completar Setup y Foundational.
2. Escribir y hacer fallar las pruebas de registro, acumulación y persistencia.
3. Implementar US1 y ejecutar `docker compose up -d postgres` seguido de `go test ./...`.
4. Validar el acumulado contra PostgreSQL y avisar al responsable de HU-07 antes de avanzar a US2, US3 y US4.

### Incremental Delivery

1. Entregar US1 como MVP funcional, porque habilita `actualHours` para HU-07.
2. Agregar US2 para proteger el rango de horas.
3. Agregar US3 para exigir actividad, fecha e integrante.
4. Agregar US4 para informar fallas de guardado sin falso éxito.
5. Ejecutar la validación completa del quickstart antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Los nombres de tablas y columnas de las migraciones 001, 002 y 003 deben verificarse en los archivos reales antes de escribir T004 y T010.
