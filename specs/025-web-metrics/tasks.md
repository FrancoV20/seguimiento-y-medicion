# Tasks: Interfaz web de métricas del Sprint

**Input**: Design documents from `specs/025-web-metrics/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-07, HU-15, HU-17, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno, sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Story Points planificados y porcentaje de historias completadas (HU-10), detalle de historias (HU-16), gráficos (HU-08) y exportar (HU-09) quedan fuera. No se modifican archivos de HU-07, HU-15, HU-17, HU-20a ni HU-20b.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza de HU-07, HU-15, HU-20a y HU-20b.

- [x] T001 Revisar HU-07 y HU-15 en `main`: ruta del paquete de HU-07 y firma de su servicio (entrada con Sprint y proyecto, resultado con velocidad, desviación o su falta, X e Y del cálculo parcial), tipo y redondeo de la desviación, textos exactos "No disponible" y "Cálculo parcial: basado en X de Y historias", si exige Sprint finalizado, qué devuelve para un Sprint sin historias completadas y cómo trata las historias devueltas por HU-15; valor exacto del estado de Sprint finalizado en `sprints.status`; identificadores ya definidos en el paquete `web` (HU-14, HU-15, HU-21 a HU-24) y cómo formatean fechas; anotar el resultado en `specs/025-web-metrics/data-model.md` y ajustar `plan.md`
- [x] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register` con rutas `GET`, `Section.RequiresProject`, `ActiveContextFrom`, `Renderer.Page`, `Fail`, `RenderError` y la página 405
- [x] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/025-web-metrics/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [x] T004 [P] Confirmar que las migraciones de Sprints (HU-04) y de las tablas que lee HU-07 en `main` coinciden con el esquema anotado en `specs/025-web-metrics/data-model.md` y avisar al equipo si cambiaron

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Catálogo de solo lectura, adaptador de HU-07 y formato de la vista, compartidos por las historias.

- [x] T005 [P] Definir `FinishedSprint` y la interfaz `MetricsCatalog` con `FinishedSprints` y `ActiveSprintNumber` en `src/web/metrics_catalog.go`
- [x] T006 [P] Escribir la prueba de integración de `PostgreSQLMetricsCatalog` con fixtures propias (dos proyectos; Sprints finalizados 1, 2 y 3 y uno activo en el primero; un Sprint finalizado en el segundo): solo Sprints finalizados del proyecto, más reciente primero, el activo no figura y su número se informa, otro proyecto no se mezcla y proyecto sin Sprints, en `src/web/postgres_metrics_catalog_integration_test.go`
- [x] T007 Implementar `PostgreSQLMetricsCatalog` con `database/sql` y pgx y las consultas de `data-model.md` en `src/web/postgres_metrics_catalog.go`
- [x] T008 [P] Escribir la prueba del adaptador sobre el servicio real de HU-07 y PostgreSQL real, con Sprints cerrados por el servicio de HU-15: velocidad de las historias "Hecho"; desviación con valores conocidos (10 horas estimadas y 12 reales dan 20 %); sin horas estimadas; cálculo parcial con X de Y; Sprint sin historias completadas con velocidad cero; las historias devueltas no cuentan, en `src/web/metrics_source_test.go`
- [x] T009 Definir en `src/web/metrics_source.go` la interfaz `MetricsSource`, el tipo `SprintMetrics` y el adaptador sobre el servicio de HU-07, que traduce tipos sin calcular ni redondear
- [x] T010 [P] Escribir las pruebas unitarias del formato y la selección: desviación positiva, negativa y cero con dos decimales, coma y signo; "No disponible"; `0 Story Points`; alerta de cálculo parcial con X de Y; elección del Sprint (válido, no numérico, inexistente, activo, de otro proyecto y sin lista); aviso del Sprint en curso, en `src/web/metrics_view_test.go`
- [x] T011 Implementar en `src/web/metrics_view.go` los textos en un solo lugar (`MsgMetricsNoSprints`, `MsgMetricsActiveHint`), el formato de la desviación y la velocidad, la selección del Sprint y el armado de `MetricsView`

**Checkpoint**: El catálogo PostgreSQL, el adaptador de HU-07 y el formato están listos y probados con datos reales.

---

## Phase 3: User Story 1 - Ver las métricas de un Sprint finalizado (Priority: P1) 🎯 MVP

**Goal**: Mostrar la velocidad y la desviación del Sprint finalizado más reciente, con los casos sin horas estimadas, parcial y velocidad cero.

**Independent Test**: Con un Sprint cerrado con datos conocidos, abrir la pantalla y verificar velocidad y desviación; repetir sin horas estimadas, con datos incompletos y sin historias completadas.

### Tests for User Story 1

- [x] T012 [P] [US1] Escribir la prueba de handler con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: Sprint finalizado con 10 horas estimadas, 12 reales y 8 Story Points completados muestra `8 Story Points` y `+20,00 %`, es el más reciente por defecto, solo del proyecto activo y con verificación de tiempo de respuesta, en `src/web/metrics_handlers_test.go`
- [x] T013 [P] [US1] Escribir la prueba de handler de un Sprint sin horas estimadas: "No disponible" en la desviación, la velocidad visible y sin error ni valor inválido, en `src/web/metrics_handlers_test.go`
- [x] T014 [P] [US1] Escribir la prueba de handler de un Sprint con datos incompletos: la alerta "Cálculo parcial: basado en X de Y historias" con los valores de HU-07, en `src/web/metrics_handlers_test.go`
- [x] T015 [P] [US1] Escribir la prueba de handler de un Sprint cerrado por HU-15 sin historias en "Hecho": `0 Story Points` y desviación "No disponible", sin error, en `src/web/metrics_handlers_test.go`
- [x] T016 [P] [US1] Escribir los escenarios BDD de métricas de un Sprint finalizado, sin esfuerzo estimado, cálculo parcial y velocidad cero en `test/web_metrics_bdd_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Implementar el handler `GET /metricas` en `src/web/metrics_handlers.go`: lee los Sprints finalizados del proyecto del contexto, elige el Sprint, pide las métricas a HU-07, arma la vista y usa `Fail` si falla la lectura o el cálculo
- [ ] T018 [US1] Crear `metrics.html` con la velocidad, la desviación con su fórmula, la alerta de cálculo parcial y el Sprint elegido, escapando los textos, en `src/web/templates/metrics.html`
- [ ] T019 [US1] Construir el catálogo y el adaptador de HU-07 y registrar la sección "Métricas del Sprint" (`/metricas`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 muestra las métricas de HU-07 desde la web; el MVP cierra el ciclo de punta a punta.

---

## Phase 4: User Story 2 - Elegir el Sprint y casos vacíos (Priority: P1)

**Goal**: Elegir entre los Sprints finalizados y explicar los casos sin Sprints o con un Sprint en curso.

**Independent Test**: Tres Sprints finalizados y uno activo; proyecto sin Sprints finalizados.

### Tests for User Story 2

- [ ] T020 [P] [US2] Escribir la prueba de handler de la lista: Sprints finalizados del más reciente al más antiguo con número, Sprint Goal y fechas, y métricas del más reciente; con `?sprint=ID` las del Sprint elegido, en `src/web/metrics_handlers_test.go`
- [ ] T021 [P] [US2] Escribir la prueba de handler de un `sprint` no numérico, inexistente, activo o de otro proyecto en la dirección: se ignora y se muestra el más reciente sin error, en `src/web/metrics_handlers_test.go`
- [ ] T022 [P] [US2] Escribir la prueba de handler sin Sprints finalizados: "Aún no hay Sprints finalizados", con y sin Sprint activo, y sin pedir nada a HU-07, en `src/web/metrics_handlers_test.go`
- [ ] T023 [P] [US2] Escribir la prueba de handler con un Sprint activo y Sprints finalizados: el activo no figura en la lista y se informa que sus métricas se calculan al cerrarlo, en `src/web/metrics_handlers_test.go`
- [ ] T024 [P] [US2] Escribir los escenarios BDD de elegir otro Sprint, sin Sprints finalizados y Sprint en curso en `test/web_metrics_bdd_test.go`

### Implementation for User Story 2

- [ ] T025 [US2] Completar `metrics.html` con la lista de Sprints (enlace `?sprint=ID` y el elegido marcado), el mensaje de sin Sprints y el aviso del Sprint en curso en `src/web/templates/metrics.html`
- [ ] T026 [US2] Completar el armado de la vista y el handler en `src/web/metrics_view.go` y `src/web/metrics_handlers.go` para la selección, el mensaje vacío y el aviso del Sprint activo

**Checkpoint**: US2 permite recorrer los Sprints finalizados y explica los casos vacíos.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [ ] T027 [P] Documentar en `specs/025-web-metrics/quickstart.md` los pasos para probar a mano con un Sprint cerrado y la nota de puerto
- [ ] T028 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/metrics_handlers.go` y `src/web/metrics_view.go`
- [ ] T029 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T030 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T031 [P] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no una pantalla incompleta, en `src/web/metrics_handlers_test.go`
- [ ] T032 [P] Escribir la prueba de handler con una fuente de métricas que falla al calcular: página de error de HU-20a y no métricas incompletas ni detalles internos, en `src/web/metrics_handlers_test.go`
- [ ] T033 [P] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): redirección de HU-20b, en `src/web/metrics_handlers_test.go`
- [ ] T034 [P] Escribir la prueba de handler de un Sprint Goal con `<script>`: se muestra como texto, en `src/web/metrics_handlers_test.go`
- [ ] T035 [P] Escribir la prueba de que un `POST` a `/metricas` recibe la página 405 de HU-20a y no modifica nada, en `src/web/metrics_handlers_test.go`
- [x] T036 Guardar en `specs/025-web-metrics/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T037 Guardar en `specs/025-web-metrics/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/025-web-metrics/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en los paquetes de reglas y cálculos
- [ ] T038 Coordinar con HU-16 que puede reutilizar `MetricsSource` para mostrar la velocidad en el detalle de un Sprint; con HU-10 que los Story Points planificados y el porcentaje completado se muestran en su propia historia; con HU-08 y HU-09 que consumen los mismos cálculos de HU-07; y con HU-15 que el escenario "Cierre bloqueado" quedó solo en su pantalla
- [ ] T039 Revisar trazabilidad de FR-001 a FR-014 y SC-001 a SC-006 en `specs/025-web-metrics/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T040 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, verificar que el pipeline de GitHub Actions queda en verde y actualizar el manual breve de usuario con la pantalla "Métricas del Sprint"
- [ ] T041 Verificar a mano con la aplicación en ejecución los 7 escenarios del Issue y la lista de `quickstart.md`
- [ ] T042 Abrir el Pull Request desde `HU-25`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T001; T005, T006, T008 y T010 en paralelo; T007 después de T005 y T006; T009 después de T008; T011 después de T010.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T017 y T018.
- **Polish (Phase 5)**: Depende de las historias que se entreguen; T031 a T035 se escriben junto a las pruebas de las historias.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-07 (cálculos), HU-15 (Sprint finalizado), HU-17 (horas estimadas), HU-20a y HU-20b.
- **US2 (P1)**: Depende del handler y de la plantilla de US1.

### Parallel Opportunities

- T003 y T004; T005, T006, T008 y T010.
- T012 a T016 y T020 a T024 pueden escribirse en paralelo antes de implementar.
- T027, T028, T031 a T035 y T040 pueden ejecutarse en paralelo después de la implementación.

## Implementation Strategy

### MVP First (User Story 1 y User Story 2)

1. Completar Setup y Foundational; guardar `evidencia-red.txt` (T036) con las pruebas fallando.
2. Implementar US1 y US2; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
3. Verificar a mano un Sprint cerrado con y sin horas estimadas antes de abrir el Pull Request.

### Incremental Delivery

1. Entregar US1 como MVP funcional: métricas del Sprint más reciente.
2. Agregar US2 para elegir entre Sprints y explicar los casos vacíos.
3. Ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, historias, Sprints, horas y cierres (con el servicio de HU-15); no dependen de otras pantallas.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures borran primero las historias (los Sprints, sus asignaciones y los registros caen en cascada) y después el proyecto.
- Los textos "No disponible" y "Cálculo parcial..." se toman de HU-07 en las pruebas, no se reescriben.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T019, T033 |
| FR-002 | T006, T020 |
| FR-003 | T008, T010, T012 |
| FR-004 | T010, T012 |
| FR-005 | T008, T010, T013 |
| FR-006 | T008, T010, T014 |
| FR-007 | T008, T015 |
| FR-008 | T006, T023 |
| FR-009 | T022 |
| FR-010 | T010, T021 |
| FR-011 | T031, T032 |
| FR-012 | T028, T032 |
| FR-013 | T034 |
| FR-014 | T006, T033, T035 |
| SC-001 | T008, T012 |
| SC-002 | T006, T021, T023 |
| SC-003 | T008, T013, T015 |
| SC-004 | T008, T014 |
| SC-005 | Verificación de tiempo en T012 |
| SC-006 | T035 |