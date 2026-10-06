# Tasks: Calcular métricas del Sprint

**Input**: Design documents from `specs/004-calculate-metrics/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: TDD/BDD requeridos por la constitución; las pruebas deben escribirse antes de implementar y ejecutar reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Preparar Go, PostgreSQL/Docker y esquema de métricas.

- [ x] T001 Verificar o inicializar el módulo Go en `go.mod` con Go 1.27.1
- [ x] T002 Verificar o agregar `github.com/jackc/pgx/v5` y `github.com/jackc/pgx/v5/stdlib` en `go.mod` y actualizar `go.sum`
- [x ] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL 16 en la base `seguimiento_y_medicion`
- [x ] T004 [P] Crear la migración SQL `db/migrations/004_create_sprint_metrics.sql` para snapshots de métricas, conteos parciales y alertas; depende de que las migraciones `001_create_projects.sql`, `002_create_backlog_stories.sql` y `003_create_sprints.sql` ya estén aplicadas
- [x ] T005 Instalar la herramienta `github.com/golang-migrate/migrate` si no está instalada y documentar en `specs/004-calculate-metrics/quickstart.md` la aplicación de migraciones en orden numérico

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Crear modelos, aritmética, validaciones y persistencia compartidos por las historias.

- [ ] T006 [P] Definir `SprintData`, `CompletedStory`, `SprintMetrics` y `CalculationError` en `src/metrics/model.go`
- [ ] T007 [P] Definir errores explícitos para Sprint no finalizado, Story Points inválidos, horas reales negativas, esfuerzo no disponible y persistencia en `src/metrics/errors.go`
- [ ] T008 Implementar operaciones decimales y redondeo matemático a dos posiciones en `src/metrics/calculation.go`, evitando división por cero
- [ ] T009 Definir `MetricsStore` con `LoadSprintData(sprintID)` y `SaveMetrics(metrics)` en `src/metrics/store.go`
- [ ] T010 Implementar `PostgreSQLMetricsStore` con `database/sql`, `github.com/jackc/pgx/v5/stdlib` y transacciones en `src/metrics/postgres_metrics_store.go`
- [ ] T011 [P] Crear el store controlado para pruebas unitarias en `src/metrics/test_store.go`, sin sustituir la integración PostgreSQL
- [ ] T012 [P] Definir la validación de cierre de Sprint y el listado de historias afectadas en `src/metrics/closure_validation.go`

**Checkpoint**: El modelo, cálculo, validación de cierre, contrato y adaptador PostgreSQL están listos para las historias.

---

## Phase 3: User Story 1 - Consultar velocidad y desviación (Priority: P1) 🎯 MVP

**Goal**: Calcular y persistir velocidad, totales de esfuerzo y desviación firmada de un Sprint finalizado con datos completos.

**Independent Test**: Con un Sprint finalizado y datos válidos, calcular velocidad y desviación; verificar fórmula, signo, redondeo y snapshot persistido en PostgreSQL.

### Tests for User Story 1

- [ ] T013 [P] [US1] Escribir pruebas unitarias de velocidad como suma de Story Points de historias `Terminada` en `src/metrics/service_test.go`
- [ ] T014 [P] [US1] Escribir pruebas unitarias de totales de horas y fórmula `((real - estimado) / estimado) * 100` en `src/metrics/service_test.go`
- [ ] T015 [P] [US1] Escribir pruebas unitarias de desviación positiva, negativa y redondeo a dos decimales en `src/metrics/calculation_test.go`
- [ ] T016 [P] [US1] Escribir la prueba de integración de carga y persistencia de métricas en PostgreSQL en `src/metrics/postgres_metrics_store_integration_test.go`
- [ ] T017 [P] [US1] Escribir el escenario BDD de cálculo de desviación y velocidad en `test/metrics_bdd_test.go`

### Implementation for User Story 1

- [ ] T018 [US1] Implementar `CalculateMetrics` en `src/metrics/service.go`, cargando el Sprint finalizado y sus historias desde `MetricsStore`
- [ ] T019 [US1] Implementar en `src/metrics/calculation.go` la velocidad, totales de horas, desviación firmada y redondeo matemático a dos decimales
- [ ] T020 [US1] Implementar en `src/metrics/service.go` la persistencia del resultado mediante `MetricsStore.SaveMetrics` solo después de un cálculo válido
- [ ] T021 [US1] Completar `PostgreSQLMetricsStore` en `src/metrics/postgres_metrics_store.go` para cargar fuentes y guardar el snapshot dentro de una transacción
- [ ] T022 [US1] Habilitar la integración PostgreSQL con variables de conexión, fixtures de Sprint finalizado y limpieza en `src/metrics/postgres_metrics_store_integration_test.go`

**Checkpoint**: US1 calcula y persiste métricas completas reproducibles, y permite consultar velocidad y desviación de forma independiente.

---

## Phase 4: User Story 2 - Informar ausencia de esfuerzo estimado (Priority: P2)

**Goal**: Evitar división por cero y calcular parcialmente cuando faltan datos de algunas historias.

**Independent Test**: Ejecutar un Sprint sin horas estimadas y otro con datos completos solo en X de Y historias; verificar `No disponible` o alerta exacta de cálculo parcial.

### Tests for User Story 2

- [ ] T023 [P] [US2] Escribir prueba unitaria de esfuerzo estimado total cero o ausente con desviación `No disponible` en `src/metrics/service_test.go`
- [ ] T024 [P] [US2] Escribir prueba unitaria de cálculo parcial usando únicamente historias con horas estimadas y reales completas en `src/metrics/service_test.go`
- [ ] T025 [P] [US2] Escribir prueba del formato exacto `Cálculo parcial: basado en X de Y historias` en `src/metrics/service_test.go`
- [ ] T026 [P] [US2] Escribir el escenario BDD de ausencia y cálculo parcial en `test/metrics_bdd_test.go`

### Implementation for User Story 2

- [ ] T027 [US2] Implementar en `src/metrics/calculation.go` la selección de historias completas y los conteos `calculationUsedCount`/`calculationTotalCount`
- [ ] T028 [US2] Implementar en `src/metrics/service.go` la alerta exacta `Cálculo parcial: basado en X de Y historias` cuando `X < Y`
- [ ] T029 [US2] Implementar en `src/metrics/calculation.go` el resultado `No disponible` cuando el esfuerzo estimado total sea cero o no exista, sin división
- [ ] T030 [US2] Verificar en `src/metrics/service_test.go` que velocidad y métricas disponibles continúan calculándose aunque la desviación sea `No disponible`

**Checkpoint**: US2 informa ausencia de base y cálculo parcial sin valores inválidos ni métricas inventadas.

---

## Phase 5: User Story 3 - Validar datos antes de cerrar el Sprint (Priority: P2)

**Goal**: Bloquear el cierre del Sprint cuando existan horas reales negativas o Story Points inválidos.

**Independent Test**: Intentar cerrar un Sprint con historias inválidas; verificar que permanece abierto y se identifican todas las historias y campos a corregir.

### Tests for User Story 3

- [ ] T031 [P] [US3] Escribir pruebas unitarias de horas reales negativas y Story Points inválidos en `src/metrics/closure_validation_test.go`
- [ ] T032 [P] [US3] Escribir prueba de múltiples historias inválidas y motivos completos en `src/metrics/closure_validation_test.go`
- [ ] T033 [P] [US3] Escribir el escenario BDD de bloqueo de cierre en `test/metrics_bdd_test.go`

### Implementation for User Story 3

- [ ] T034 [US3] Implementar `ValidateSprintClosure` en `src/metrics/closure_validation.go`, devolviendo todos los errores por historia y campo
- [ ] T035 [US3] Integrar la validación de cierre en `src/metrics/service.go` antes de cambiar el Sprint a `Finalizado` o calcular métricas definitivas
- [ ] T036 [US3] Implementar en `src/metrics/errors.go` mensajes que identifiquen historias con horas reales negativas o Story Points inválidos y soliciten corrección
- [ ] T037 [US3] Verificar en `src/metrics/closure_validation_test.go` y `test/metrics_bdd_test.go` que el cierre bloqueado no persiste estado final ni métricas definitivas

**Checkpoint**: US3 bloquea el cierre con datos inválidos y deja el Sprint corregible sin mutaciones parciales.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, documentación y trazabilidad.

- [ ] T038 [P] Documentar variables de conexión, arranque y apagado de PostgreSQL Docker en `specs/004-calculate-metrics/quickstart.md`
- [ ] T039 [P] Revisar que errores de cálculo y PostgreSQL no expongan detalles internos en `src/metrics/errors.go`
- [ ] T040 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T041 Ejecutar `go test ./...` y completar todos los escenarios del `quickstart.md`
- [ ] T042 Revisar trazabilidad de FR-001 a FR-015 y SC-001 a SC-007 en `specs/004-calculate-metrics/tasks.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 debe preceder T002; T003 y T004 pueden ejecutarse en paralelo; T005 requiere que T004 esté creada.
- **Foundational (Phase 2)**: Depende de Setup; T006-T009, T011 y T012 habilitan el dominio, y T010 requiere T004 y T009 para implementar el store PostgreSQL.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2; es el MVP.
- **User Story 2 (Phase 4)**: Depende de las implementaciones T018-T021 de US1, pero sus pruebas son independientes del cierre.
- **User Story 3 (Phase 5)**: Depende de T012 y T018 para validar el cierre y cargar el Sprint, y puede desarrollarse en paralelo con US2 después de Foundation.
- **Polish (Phase 6)**: Depende de las historias que se decida entregar.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-04 para disponer de un Sprint finalizado y de HU-02 para historias con Story Points.
- **US2 (P2)**: Depende del cálculo de US1, pero es independientemente verificable con sus datos fuente.
- **US3 (P2)**: Depende del modelo de historias y cierre del Sprint, pero es independientemente verificable antes de calcular.

### Parallel Opportunities

- T003 y T004 pueden ejecutarse en paralelo después de confirmar el módulo.
- T006, T007, T008, T011 y T012 pueden ejecutarse en paralelo.
- T013, T014, T015, T016 y T017 pueden escribirse en paralelo antes de implementar US1.
- T023, T024, T025 y T026 pueden escribirse en paralelo con la implementación de US1.
- T031, T032 y T033 pueden escribirse en paralelo con US2.
- T038 y T039 pueden ejecutarse en paralelo después de la implementación.

---

## Parallel Example: User Story 1

```text
T013: Prueba de velocidad en src/metrics/service_test.go
T014: Prueba de fórmula y totales en src/metrics/service_test.go
T015: Prueba de redondeo y signo en src/metrics/calculation_test.go
T016: Integración PostgreSQL en src/metrics/postgres_metrics_store_integration_test.go
T017: Escenario BDD en test/metrics_bdd_test.go
```

## Parallel Example: User Story 2

```text
T023: Prueba No disponible en src/metrics/service_test.go
T024: Prueba de cálculo parcial en src/metrics/service_test.go
T025: Prueba de alerta X de Y en src/metrics/service_test.go
T026: Escenario BDD en test/metrics_bdd_test.go
```

## Parallel Example: User Story 3

```text
T031: Prueba de datos inválidos en src/metrics/closure_validation_test.go
T032: Prueba de múltiples errores en src/metrics/closure_validation_test.go
T033: Escenario BDD en test/metrics_bdd_test.go
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Completar Setup y Foundational.
2. Escribir y hacer fallar las pruebas de velocidad, fórmula, redondeo y persistencia.
3. Implementar US1 y ejecutar `docker compose up -d postgres` seguido de `go test ./...`.
4. Validar snapshots de métricas contra PostgreSQL antes de avanzar a US2 y US3.

### Incremental Delivery

1. Entregar US1 como MVP funcional.
2. Agregar US2 para ausencia de estimación y cálculos parciales.
3. Agregar US3 para proteger el cierre frente a datos inválidos.
4. Ejecutar la validación completa del quickstart antes de integrar.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
