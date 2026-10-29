# Tasks: Cierre de Sprint

**Input**: Design documents from `specs/015-sprint-close/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-14, HU-07, HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Una migración nueva (`sprint_story_results`). Toma el último número de `main` más uno, sin importar el número de la historia, con `.up.sql` y `.down.sql` probados en ambos sentidos. Nadie edita una migración ajena.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Velocidad y desviación (HU-07), métricas en pantalla (HU-25), Sprints anteriores (HU-16) y reabrir un Sprint quedan fuera. No se modifican archivos de HU-04, HU-14, HU-20a ni HU-20b.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza.

- [x] T001 Revisar en `main` HU-03, HU-04, FIX-HU-04, HU-07 y HU-14: ruta del paquete de Sprint; validación previa al cierre de HU-07 (firma, datos que devuelve, mensaje y cómo se prueba el caso inválido) y su cálculo de velocidad; qué escribe HU-04 al asignar y quitar (`sprint_story_assignments`, `active_sprint_id`, estados); si HU-03 ya rechaza horas en historias fuera del Sprint activo; cómo cuenta HU-07 las horas de una historia que estuvo en dos Sprints; `BoardCatalog` de HU-14; y el tipo de `user_stories.story_points`; hallazgos documentados en `data-model.md` y planificación ajustada en `plan.md`.
- [x] T002 Integrar/verificar `origin/main` y HU-20a/HU-20b: `Server.Register` registra varias rutas y métodos y aplica `RequireContext` según `Section.RequiresProject`; se verificaron `ActiveContextFrom`, `SetFlash`, `MsgOperationFailed`, la redirección 303 y la protección contra envíos de otro origen. El parcial `"alert"` está definido en `layout.html` (no existe `alert.html` como archivo separado).
- [x] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `quickstart.md` `DATABASE_URL`; el puerto publicado es `5433` en el host hacia `5432` en el contenedor.
- [x] T004 [P] Confirmar el esquema de `sprints`, `sprint_story_assignments` y `user_stories` y el último número de migración integrado: `012` está ocupado por HU-06; la migración HU-15 es `013`; hallazgos documentados en `data-model.md` y `plan.md`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Migración, servicio y almacenamiento del cierre, compartidos por las tres historias.

- [x] T005 Escribir la migración `013_sprint_story_results.up.sql` y `.down.sql` en `db/migrations/` y probarla en ambos sentidos (`up`, `down 1`, `up`)
- [x] T006 [P] Escribir las pruebas del servicio `CloseSprint` contra PostgreSQL real con fixtures propias (dos proyectos; Sprint con historias "Hecho", "En progreso" y "En Sprint"; Sprint sin historias; Sprint sin "Hecho"; Sprint finalizado): cierre válido, orden de validación (Sprint del proyecto, activo, validación de HU-07), `ErrSprintNotFound`, `ErrSprintAlreadyClosed`, `ErrCloseBlocked` con datos inválidos armados como en HU-07 y velocidad cero de HU-07 cuando no hay "Hecho", en `src/sprint/close_sprint_test.go`
- [x] T007 [P] Escribir la prueba de integración de `PostgreSQLSprintCloser`: transacción completa (Sprint "Finalizado", no completadas en "Pendiente" sin asignación ni `active_sprint_id`, "Hecho" conservadas, una fila en `sprint_story_results` por historia con estado final y Story Points), el registro no cambia al reasignar una devuelta o editar sus puntos, una falla a mitad del cierre revierte todo, y dos cierres simultáneos aplican solo uno, en `src/sprint/postgres_sprint_closer_integration_test.go`
- [x] T008 [P] Escribir la prueba del armado del resumen a partir de `BoardView` de HU-14 (completadas, a devolver, Story Points planificados y completados) en `src/web/close_form_test.go`
- [x] T009 Definir en `src/sprint/close_sprint.go` el servicio `CloseSprint`, `CloseResult`, los errores `ErrSprintNotFound`, `ErrSprintAlreadyClosed` y `ErrCloseBlocked` y la interfaz `SprintCloseStore`
- [x] T010 Implementar el servicio `CloseSprint` con el orden de validación y la invocación de la validación de HU-07 en `src/sprint/close_sprint.go`
- [x] T011 Implementar `PostgreSQLSprintCloser` con `database/sql` y pgx y la transacción de `data-model.md` (bloqueo del Sprint y sus historias, registro, devolución y finalización) en `src/sprint/postgres_sprint_closer.go`

**Checkpoint**: La migración, el servicio y el almacenamiento están listos y probados contra PostgreSQL.

---

## Phase 3: User Story 1 - Cerrar el Sprint (Priority: P1) 🎯 MVP

**Goal**: Cerrar el Sprint activo desde la pantalla, devolviendo las no completadas y congelando el registro.

**Independent Test**: Con un Sprint con historias "Hecho", "En progreso" y "En Sprint", cerrarlo y verificar Sprint, historias, registro, asignación a un Sprint nuevo y velocidad de HU-07.

### Tests for User Story 1

- [x] T012 [P] [US1] Escribir la prueba de handler de cierre exitoso con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: 303 a `/cierre`, "Sprint N cerrado correctamente. Historias devueltas al Backlog: Y" una sola vez, Sprint "Finalizado" y, con dos proyectos, solo cierra el activo, en `src/web/close_handlers_test.go`
- [x] T013 [P] [US1] Escribir la prueba de handler de las historias después del cierre: las "Hecho" siguen en el Sprint finalizado y las demás quedan "Pendiente" y disponibles en el Backlog, en `src/web/close_handlers_test.go`
- [x] T014 [P] [US1] Escribir la prueba de handler del Sprint sin historias completadas: se cierra y HU-07 calcula velocidad cero, en `src/web/close_handlers_test.go`
- [x] T015 [P] [US1] Escribir la prueba de que las historias devueltas se pueden asignar a un Sprint nuevo con el servicio de HU-04 y de que el registro del Sprint cerrado no cambia, en `src/sprint/close_sprint_test.go`
- [x] T016 [US1] Si T001 muestra que HU-03 no rechaza horas en historias fuera del Sprint activo, escribir su prueba (Sprint finalizado y historia devuelta) y agregar la regla en el dominio de HU-03 junto a sus reglas; si ya existe, escribir solo la prueba y avisar a quien mantiene HU-03
- [x] T017 [P] [US1] Escribir los escenarios BDD de cierre exitoso, historias completadas conservadas, datos del Sprint congelados, Sprint sin historias completadas e historias devueltas disponibles en `test/web_close_bdd_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Implementar en `src/web/close_form.go` el armado del envío (`sprint_id`), el resumen a partir de `BoardCatalog` y los textos propios en un solo lugar (`MsgSprintClosed`, `MsgSprintAlreadyClosed`, `MsgSprintNotFound`, `MsgCloseBlocked`, `MsgNoSprintToClose`)
- [x] T019 [US1] Implementar los handlers `GET /cierre` y `POST /cierre` en `src/web/close_handlers.go`: llaman al servicio con el proyecto de `ActiveContext` y usan `SetFlash` y la redirección 303
- [x] T020 [US1] Crear `close.html` con el resumen (Sprint, completadas, a devolver, Story Points), el aviso de que no se puede deshacer y el formulario con `sprint_id` y el botón "Cerrar Sprint", usando el parcial `"alert"` definido en `layout.html`, en `src/web/templates/close.html`
- [x] T021 [US1] Construir el servicio, el almacenamiento y el catálogo y registrar la sección "Cierre de Sprint" (`/cierre`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 cierra Sprints desde la web; HU-07, HU-10, HU-16 y HU-25 ya tienen un Sprint finalizado con su registro.

---

## Phase 4: User Story 2 - Rechazos y errores (Priority: P1)

**Goal**: Bloquear el cierre ante datos inválidos, Sprints ya cerrados o fallas, sin cambios parciales.

**Independent Test**: Cierre con datos inválidos, Sprint ya cerrado, Sprint de otro proyecto, carga de horas posterior y falla de guardado.

### Tests for User Story 2

- [x] T022 [P] [US2] Escribir la prueba de handler de cierre bloqueado por HU-07: 409 con el mensaje y la lista de historias a corregir, Sprint activo y sin cambios, en `src/web/close_handlers_test.go`
- [x] T023 [P] [US2] Escribir la prueba de handler de Sprint ya finalizado, incluido un `sprint_id` viejo con un Sprint nuevo activo: "El Sprint ya está cerrado", 409 y el Sprint nuevo sigue activo, en `src/web/close_handlers_test.go`
- [x] T024 [P] [US2] Escribir la prueba de handler con `sprint_id` no numérico, inexistente o de otro proyecto: "El Sprint no existe" y 422, y un campo de proyecto enviado se ignora, en `src/web/close_handlers_test.go`
- [x] T025 [P] [US2] Escribir la prueba de handler con un almacenamiento que falla al guardar: "La operación no pudo completarse", sin éxito, Sprint activo y sin cambios parciales, en `src/web/close_handlers_test.go`
- [x] T026 [P] [US2] Escribir la prueba de handler de la carga de esfuerzo después del cierre: se rechaza (regla de HU-03), en `src/web/close_handlers_test.go`
- [x] T027 [P] [US2] Escribir los escenarios BDD de cierre bloqueado por HU-07, Sprint ya finalizado, carga de esfuerzo posterior y error al cerrar en `test/web_close_bdd_test.go`

### Implementation for User Story 2

- [x] T028 [US2] Implementar en `src/web/close_form.go` el mapeo de los errores del dominio a mensajes y la lista de historias a corregir, con `MsgOperationFailed` para cualquier otro
- [x] T029 [US2] Implementar en `src/web/close_handlers.go` las respuestas 409, 422 y 500 con la pantalla y el mensaje
- [x] T030 [US2] Verificar en `src/web/close_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 evita cierres inválidos, duplicados o parciales y mantiene los datos del Sprint coherentes.

---

## Phase 5: User Story 3 - Ver el resumen antes de cerrar (Priority: P1)

**Goal**: Mostrar el resumen del cierre o el estado sin Sprint activo.

**Independent Test**: Sprint con historias en distintos estados; sin Sprint activo; falla de lectura.

### Tests for User Story 3

- [x] T031 [P] [US3] Escribir la prueba de handler del resumen: Sprint, completadas, a devolver, Story Points completados sobre planificados y botón, solo del proyecto activo, con verificación de tiempo de respuesta, en `src/web/close_handlers_test.go`
- [x] T032 [P] [US3] Escribir la prueba de handler sin Sprint activo: "No hay un Sprint activo para cerrar" y sin botón, en `src/web/close_handlers_test.go`
- [ ] T033 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no una pantalla incompleta, en `src/web/close_handlers_test.go`
- [x] T034 [P] [US3] Escribir la prueba de handler de un título con `<script>`: se muestra como texto, en `src/web/close_handlers_test.go`
- [x] T035 [P] [US3] Completar `close.html` con el estado "No hay un Sprint activo para cerrar" y sin botón en `src/web/templates/close.html`
- [x] T036 [P] [US3] Escribir los escenarios BDD de resumen antes de cerrar y sin Sprint activo en `test/web_close_bdd_test.go`
- [x] T037 [US3] Usar `Fail` en `src/web/close_handlers.go` cuando falla la lectura del Sprint o de las historias

**Checkpoint**: US3 muestra el efecto del cierre antes de confirmarlo.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [x] T038 [P] Documentar en `specs/015-sprint-close/quickstart.md` los pasos para probar a mano con un Sprint con historias en distintos estados y la nota de puerto
- [x] T039 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/close_handlers.go` y `src/web/close_form.go`
- [ ] T040 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T041 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T042 [P] Escribir la prueba de handler de recarga después de cerrar: no se cierra nada más ni se repite el mensaje, en `src/web/close_handlers_test.go`
- [ ] T043 [P] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): redirección de HU-20b y no se cierra nada, en `src/web/close_handlers_test.go`
- [ ] T044 [P] Verificar que `T033` cubre el error al leer y que el catálogo distingue "sin Sprint" de un error de lectura
- [ ] T045 Guardar en `specs/015-sprint-close/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T046 Guardar en `specs/015-sprint-close/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/015-sprint-close/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en `src/sprint`
- [ ] T047 Coordinar con quien tome HU-10 que los Story Points planificados y el porcentaje de historias completadas de un Sprint finalizado salen de `sprint_story_results`; con HU-16 que las historias no completadas salen de ahí (las devueltas ya no están asignadas); con HU-25 que el escenario "Cierre bloqueado" ya está cubierto acá y que lee la velocidad de HU-07; con HU-13 y HU-17 que las historias "Hecho" quedan en el Sprint finalizado (bloqueadas) y las devueltas son editables; y con HU-07 que una historia devuelta puede entrar a otro Sprint con horas ya registradas
- [ ] T048 Revisar trazabilidad de FR-001 a FR-016 y SC-001 a SC-007 en `specs/015-sprint-close/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T049 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, verificar que el pipeline de GitHub Actions queda en verde y actualizar el manual breve de usuario con la pantalla "Cierre de Sprint"
- [ ] T050 Verificar a mano con la aplicación en ejecución los 11 escenarios del Issue y la lista de `quickstart.md`
- [ ] T051 Abrir el Pull Request desde `HU-15`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005 puede hacerse junto a T006 a T008; T009 antes de T010 y T011.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T019; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T018 y T020; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen; T042 a T044 se escriben junto a las pruebas de las historias.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-14 (estados y `BoardCatalog`), HU-07 (validación), HU-04 y FIX-HU-04, HU-03, HU-20a y HU-20b.
- **US2 (P1)**: Depende del handler de US1; se verifica con envíos inválidos de forma independiente.
- **US3 (P1)**: Depende del resumen de US1.
- **HU-10, HU-16 y HU-25**: Necesitan un Sprint finalizado y su registro, que esta historia permite crear desde la web.

### Parallel Opportunities

- T003 y T004; T006 a T008.
- T012 a T017, T022 a T027 y T031 a T036 pueden escribirse en paralelo antes de implementar.
- T038, T039, T042 a T044 y T049 pueden ejecutarse en paralelo después de la implementación.

## Implementation Strategy

### MVP First (User Story 1, User Story 2 y User Story 3)

1. Completar Setup y Foundational; guardar `evidencia-red.txt` (T045) con las pruebas de las tres historias fallando.
2. Implementar US1 a US3; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
3. Verificar a mano un cierre completo y la creación del Sprint siguiente antes de abrir el Pull Request.

### Incremental Delivery

1. Entregar US1 como MVP funcional: cierra el Sprint y devuelve las historias.
2. Agregar US2 para bloquear datos inválidos, cierres duplicados y fallas.
3. Agregar US3 para ver el resumen antes de cerrar.
4. Ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, historias, Sprints y horas; no dependen de HU-21, HU-22 ni HU-23.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures borran primero las historias (los Sprints, sus asignaciones y los registros caen en cascada) y después el proyecto.
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base con `docker compose down -v` y volver a migrar. Si dos ramas crean el mismo número de migración, la segunda en integrarse renombra la suya.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T019, T021, T042 |
| FR-002 | T008, T031 |
| FR-003 | T032 |
| FR-004 | T006, T012 |
| FR-005 | T006, T007, T013 |
| FR-006 | T007, T013 |
| FR-007 | T005, T007, T015 |
| FR-008 | T006, T022 |
| FR-009 | T006, T014 |
| FR-010 | T006, T023 |
| FR-011 | T016, T026 |
| FR-012 | T007, T025 |
| FR-013 | T006, T024, T043 |
| FR-014 | T033 |
| FR-015 | T030, T039 |
| FR-016 | T015 |
| SC-001 | T006, T007, T012, T013 |
| SC-002 | T007, T015 |
| SC-003 | T006, T007, T022, T023, T025 |
| SC-004 | T015 |
| SC-005 | T007 |
| SC-006 | T016, T026 |
| SC-007 | Verificación de tiempo en T031 |