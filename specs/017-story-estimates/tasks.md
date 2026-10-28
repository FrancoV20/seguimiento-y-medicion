# Tasks: Horas estimadas por historia (Estimaciones)

**Input**: Design documents from `specs/017-story-estimates/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-02, HU-03, HU-22, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno, sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Alta de la historia con horas (HU-22), edición de otros datos (HU-13), horas reales (HU-03 y HU-24) y desviación (HU-07) quedan fuera. No se modifican archivos de HU-02 ni de HU-22.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza.

- [x] T001 Revisar en `main` HU-02, HU-22, HU-03 y HU-07: ruta del paquete de historias, nombres, tipo y errores de la regla de horas que dejó HU-22 (mayor que 0, hasta dos decimales, coma o punto, máximo 9999,99), mensajes y texto "Sin estimar"; que HU-07 lee `user_stories.estimated_hours`; columnas de historia y de horas en `effort_logs`; y si `user_stories` exige otras columnas al actualizar; anotar el resultado en `specs/017-story-estimates/data-model.md` y ajustar `plan.md`
- [x] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register` con varias rutas y métodos, `RequireContext` para `POST` según `Section.RequiresProject`, `ActiveContextFrom`, `SetFlash`, `MsgOperationFailed`, parciales `alert.html` y `field.html`, redirección 303 y protección contra envíos de otro origen
- [x] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/017-story-estimates/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [x] T004 [P] Confirmar que las migraciones `002`, `003` y `004` en `main` coinciden con el esquema anotado en `specs/017-story-estimates/data-model.md` y avisar al equipo si cambiaron

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Servicio, almacenamiento y catálogo compartidos por las tres historias.

- [x] T005 [P] Escribir las pruebas del servicio `UpdateEstimatedHours` contra PostgreSQL real con fixtures propias (dos proyectos; historias con y sin estimación, con horas reales y de un Sprint finalizado): registrar y modificar, valores válidos exactos (`8`, `1,5`, `1.5`, `0,01`, `9999,99`) y rechazados (`0`, `-1`, `abc`, vacío, `1,555`, `10000`, `9999,995`) con el valor anterior intacto, Sprint finalizado, historia inexistente o de otro proyecto, mismo valor, y que solo cambia `estimated_hours`, en `src/story/estimated_hours_test.go`
- [x] T006 [P] Escribir la prueba de integración de `PostgreSQLEstimateStore`: `FindEstimate` (horas, Sprint finalizado, aislamiento por proyecto) y `UpdateEstimate` atómico (cambia solo `estimated_hours`, no cambia si el Sprint está finalizado), en `src/story/postgres_estimate_store_integration_test.go`
- [x] T007 [P] Definir `EstimateRow` y la interfaz `EstimateCatalog` en `src/web/estimates_catalog.go`
- [x] T008 [P] Escribir la prueba de integración de `PostgreSQLEstimateCatalog` con fixtures propias (dos proyectos; historias estimadas, sin estimar, con y sin horas reales, de un Sprint finalizado; proyecto sin historias): solo las del proyecto pedido, orden descendente por `id`, horas reales (0 sin registros) y bloqueo, en `src/web/postgres_estimate_catalog_integration_test.go`
- [x] T009 Implementar en `src/story/estimated_hours.go` el servicio `UpdateEstimatedHours` (usa la regla de horas de HU-22), `EstimateResult`, los errores `ErrStoryNotFound` y `ErrStoryInFinishedSprint` y la interfaz `EstimateStore`
- [x] T010 Implementar `PostgreSQLEstimateStore` con `database/sql` y pgx y las sentencias de `data-model.md` en `src/story/postgres_estimate_store.go`
- [x] T011 Implementar `PostgreSQLEstimateCatalog` en `src/web/postgres_estimate_catalog.go`

**Checkpoint**: El servicio y los adaptadores están listos y probados contra PostgreSQL.

---

## Phase 3: User Story 1 - Registrar y modificar la estimación (Priority: P1) 🎯 MVP

**Goal**: Guardar y cambiar las horas estimadas de una historia desde la pantalla.

**Independent Test**: Guardar horas en una historia sin estimar, modificarlas y verificar mensaje, valor y horas reales; repetir con coma y punto.

### Tests for User Story 1

- [ ] T012 [P] [US1] Escribir la prueba de handler de registrar con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: 303 a `/estimaciones`, "Horas estimadas registradas" una sola vez, valor mostrado y, con dos proyectos, solo en el activo, en `src/web/estimates_handlers_test.go`
- [ ] T013 [P] [US1] Escribir la prueba de handler de modificar: "Horas estimadas actualizadas", el valor reemplaza al anterior y las horas reales no cambian, en `src/web/estimates_handlers_test.go`
- [ ] T014 [P] [US1] Escribir la prueba de handler con horas `8`, `1,5`, `1.5` y `9999,99`: se guardan exactas, sin redondeo y se muestran con coma, en `src/web/estimates_handlers_test.go`
- [ ] T015 [P] [US1] Escribir los escenarios BDD de estimación registrada, modificar la estimación y horas con coma decimal en `test/web_estimates_bdd_test.go`

### Implementation for User Story 1

- [ ] T016 [US1] Implementar en `src/web/estimates_form.go` el armado del envío (`historia_id`, `horas_estimadas`), los textos propios en un solo lugar (`MsgHoursRegistered`, `MsgHoursUpdated`, `MsgStoryNotFound`, `MsgHoursLockedSprint`) y el formato de horas con coma
- [ ] T017 [US1] Implementar los handlers `GET /estimaciones` y `POST /estimaciones` en `src/web/estimates_handlers.go`: llaman al servicio con el proyecto de `ActiveContext` y usan `SetFlash` y la redirección 303
- [ ] T018 [US1] Crear `estimates.html` con la tabla de historias (título, estado, Story Points, horas estimadas, horas reales) y un formulario por fila (`historia_id`, campo de horas y botón "Guardar"), usando los parciales `alert.html` y `field.html`, en `src/web/templates/estimates.html`
- [ ] T019 [US1] Construir el servicio, el almacenamiento y el catálogo y registrar la sección "Estimaciones" (`/estimaciones`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 registra y modifica estimaciones desde la web; HU-07 y HU-25 ya tienen datos de horas estimadas.

---

## Phase 4: User Story 2 - Rechazos y errores (Priority: P1)

**Goal**: Rechazar con motivo los valores inválidos y los cambios no permitidos sin tocar el valor anterior.

**Independent Test**: Enviar horas inválidas y por encima del máximo; enviar el cambio de una historia de un Sprint finalizado; simular una falla de guardado.

### Tests for User Story 2

- [ ] T020 [P] [US2] Escribir la prueba de handler con horas inválidas (`0`, `-1`, `abc`, vacío y `1,555`): "El valor de horas no es válido" junto al campo de esa fila, lo escrito conservado, valor anterior igual y 422, en `src/web/estimates_handlers_test.go`
- [ ] T021 [P] [US2] Escribir la prueba de handler con horas por encima del máximo (`10000` y `9999,995`): "Las horas estimadas no pueden superar 9999,99" junto al campo y valor anterior igual, en `src/web/estimates_handlers_test.go`
- [ ] T022 [P] [US2] Escribir la prueba de handler de una historia de un Sprint finalizado con un envío directo: 409 con el mensaje y valor sin cambios, en `src/web/estimates_handlers_test.go`
- [ ] T023 [P] [US2] Escribir la prueba de handler con historia no numérica, inexistente o de otro proyecto: "La historia no existe" y 422, y un campo de proyecto enviado se ignora, en `src/web/estimates_handlers_test.go`
- [ ] T024 [P] [US2] Escribir la prueba de handler con un almacenamiento que falla al guardar: "La operación no pudo completarse", sin éxito y valor anterior intacto, en `src/web/estimates_handlers_test.go`
- [ ] T025 [P] [US2] Escribir los escenarios BDD de horas inválidas, horas por encima del máximo, historia de un Sprint finalizado y error al guardar en `test/web_estimates_bdd_test.go`

### Implementation for User Story 2

- [ ] T026 [US2] Implementar en `src/web/estimates_form.go` el mapeo de los errores del dominio a mensajes y a la fila afectada, con `MsgOperationFailed` para cualquier otro, y en `src/web/estimates_handlers.go` las respuestas 422, 409 y 500 con la lista y lo escrito conservado
- [ ] T027 [US2] Verificar en `src/web/estimates_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 mantiene la estimación confiable ante valores inválidos, cierres de Sprint y fallas.

---

## Phase 5: User Story 3 - Ver las estimaciones (Priority: P1)

**Goal**: Mostrar las historias con sus horas estimadas y reales, "Sin estimar", el bloqueo y el estado vacío.

**Independent Test**: Historias estimadas, sin estimar y de un Sprint finalizado; sin historias; falla de lectura.

### Tests for User Story 3

- [ ] T028 [P] [US3] Escribir la prueba de handler de la lista: título, estado, Story Points, horas estimadas y reales, solo las historias del proyecto activo, más recientes primero y tiempo de respuesta, en `src/web/estimates_handlers_test.go`
- [ ] T029 [P] [US3] Escribir la prueba de handler de "Sin estimar": una historia sin horas estimadas (o sin Story Points) nunca se muestra como 0 y una sin horas reales muestra 0, en `src/web/estimates_handlers_test.go`
- [ ] T030 [P] [US3] Escribir la prueba de handler de una historia de un Sprint finalizado: campo deshabilitado con el motivo "Sprint finalizado", en `src/web/estimates_handlers_test.go`
- [ ] T031 [P] [US3] Escribir la prueba de handler sin historias: "Aún no hay historias", en `src/web/estimates_handlers_test.go`
- [ ] T032 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no una lista incompleta, en `src/web/estimates_handlers_test.go`
- [ ] T033 [P] [US3] Escribir los escenarios BDD de lista de estimaciones, historia sin estimar, historia bloqueada en la lista y sin historias en `test/web_estimates_bdd_test.go`

### Implementation for User Story 3

- [ ] T034 [US3] Completar `estimates.html` con "Sin estimar", el campo deshabilitado con el motivo y el estado vacío "Aún no hay historias" en `src/web/templates/estimates.html`
- [ ] T035 [US3] Usar `Fail` en `src/web/estimates_handlers.go` cuando falla la lectura de las historias

**Checkpoint**: US3 muestra qué falta estimar, qué se puede cambiar y cómo se compara con lo real.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [ ] T036 [P] Documentar en `specs/017-story-estimates/quickstart.md` los pasos para probar a mano con historias estimadas, horas reales y un Sprint finalizado, y la nota de puerto
- [ ] T037 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/estimates_handlers.go` y `src/web/estimates_form.go`
- [ ] T038 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T039 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T040 [P] Escribir la prueba de handler de recarga después de guardar: no se repite el cambio ni el mensaje, en `src/web/estimates_handlers_test.go`
- [ ] T041 [P] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): redirección de HU-20b y no se guarda nada, en `src/web/estimates_handlers_test.go`
- [ ] T042 Guardar en `specs/017-story-estimates/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T043 Guardar en `specs/017-story-estimates/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/017-story-estimates/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en el paquete de historias
- [ ] T044 Coordinar con quien tome HU-07 y HU-25 que la estimación está en `user_stories.estimated_hours` y que una historia sin estimar es ausente y no 0; con HU-15 que las "Hecho" quedan en el Sprint finalizado (bloqueadas) y las no terminadas pierden la asignación (editables); con HU-13 que la lista de Estimaciones es independiente y la edición de la historia no toca las horas; y con HU-09 que las horas estimadas del reporte salen de la misma columna
- [ ] T045 Revisar trazabilidad de FR-001 a FR-016 y SC-001 a SC-007 en `specs/017-story-estimates/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T046 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, verificar que el pipeline de GitHub Actions queda en verde y actualizar el manual breve de usuario con la pantalla "Estimaciones"
- [ ] T047 Verificar a mano con la aplicación en ejecución los 11 escenarios del Issue y la lista de `quickstart.md`
- [ ] T048 Abrir el Pull Request desde `HU-17`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005 a T008 en paralelo; T009 antes de T010; T011 requiere T007.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T017; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T011 y T018; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen; T040 y T041 se escriben junto a las pruebas de las historias.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-02 (esquema), HU-22 (regla de horas), HU-03 (horas reales), HU-20a y HU-20b.
- **US2 (P1)**: Depende del handler de US1; se verifica con envíos inválidos de forma independiente.
- **US3 (P1)**: Depende del catálogo de Phase 2; verificable con datos de prueba.
- **HU-07, HU-25 y HU-09**: Necesitan horas estimadas, que esta historia permite registrar y modificar desde la web.

### Parallel Opportunities

- T003 y T004; T005 a T008.
- T012 a T015, T020 a T025 y T028 a T033 pueden escribirse en paralelo antes de implementar.
- T036, T037, T040, T041 y T046 pueden ejecutarse en paralelo después de la implementación.
- Mientras HU-22 no esté integrada, se pueden adelantar T001 a T004 y T007 a T008.

## Implementation Strategy

### MVP First (User Story 1, User Story 2 y User Story 3)

1. Completar Setup y Foundational; guardar `evidencia-red.txt` (T042) con las pruebas de las tres historias fallando.
2. Implementar US1 a US3; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
3. Verificar a mano registrar y modificar una estimación antes de abrir el Pull Request.

### Incremental Delivery

1. Entregar US1 como MVP funcional: permite registrar y cambiar horas estimadas.
2. Agregar US2 para rechazar valores inválidos y cambios sobre Sprints finalizados.
3. Agregar US3 para ver la lista con horas estimadas y reales.
4. Ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, historias, Sprints y horas; no dependen de HU-21, HU-22 ni HU-14.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures borran primero las historias (los Sprints y sus asignaciones caen en cascada) y después el proyecto.
- Para dejar un Sprint finalizado en las pruebas, cambiar su estado directamente en la base (HU-15 todavía no existe).
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base con `docker compose down -v` y volver a migrar.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T017, T019, T040 |
| FR-002 | T008, T028 |
| FR-003 | T029 |
| FR-004 | T005, T012 |
| FR-005 | T005, T006, T013 |
| FR-006 | T005, T014 |
| FR-007 | T005, T020 |
| FR-008 | T005, T021 |
| FR-009 | T005, T006, T022 |
| FR-010 | T008, T030 |
| FR-011 | T024 |
| FR-012 | T031 |
| FR-013 | T023, T041 |
| FR-014 | T001, T005, T006 |
| FR-015 | T032 |
| FR-016 | T027, T037 |
| SC-001 | T005, T014 |
| SC-002 | T005, T020, T021 |
| SC-003 | T005, T006, T022 |
| SC-004 | T005, T006, T013 |
| SC-005 | T024 |
| SC-006 | T029 |
| SC-007 | Verificación de tiempo en T028 |