# Tasks: Actualización de estado de historias (Tablero)

**Input**: Design documents from `specs/014-story-status/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md; HU-04, FIX-HU-04, HU-03, HU-20a y HU-20b (completa) integradas en `main`

**Tests**: TDD/BDD requeridos por la constitución; las pruebas se escriben antes de implementar y ejecutan reglas reales.

**Organization**: Las tareas están agrupadas por historia de usuario y ordenadas por dependencias.

**Migraciones**: Esta feature no crea migraciones. Si el equipo agregara alguna, toma el último número de `main` más uno, sin importar el número de la historia.

**Pruebas**: Se ejecutan con `go test -count=1 -p 1 ./...` (los paquetes de integración comparten una base).

**Alcance**: Asignar y quitar historias (HU-23), cerrar el Sprint (HU-15), velocidad (HU-07), horas (HU-03 y HU-24) e identificación de la persona quedan fuera. No se modifican archivos de HU-04.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirmar lo que se reutiliza.

- [ ] T001 Revisar en `main` HU-02, HU-04, FIX-HU-04, HU-03 y HU-07: ruta del paquete de Sprint, constantes de los cuatro estados, qué asigna HU-04 al incorporar una historia (`sprint_story_assignments`, `active_sprint_id`, estado `'En Sprint'`), columna de historia en `effort_logs`, si `user_stories` exige otras columnas al actualizar, y que HU-07 cuenta como completadas las historias con estado `'Hecho'`; anotar el resultado en `specs/014-story-status/data-model.md` y ajustar `plan.md`
- [ ] T002 Hacer `git merge origin/main` y verificar que HU-20a y HU-20b están integradas: `Server.Register` con varias rutas y métodos, `RequireContext` para `POST` según `Section.RequiresProject`, `ActiveContextFrom`, `SetFlash`, `MsgOperationFailed`, parcial `alert.html`, redirección 303 y protección contra envíos de otro origen
- [ ] T003 [P] Verificar que `docker-compose.yml` levanta PostgreSQL y documentar en `specs/014-story-status/quickstart.md` `DATABASE_URL` con la nota del puerto 5432 o 5433
- [ ] T004 [P] Confirmar que las migraciones `002`, `003` y `004` en `main` coinciden con el esquema anotado en `specs/014-story-status/data-model.md` (cuatro estados admitidos) y avisar al equipo si cambiaron

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Regla de transiciones, servicio, almacenamiento y catálogo, compartidos por las tres historias.

- [ ] T005 [P] Escribir las pruebas unitarias de `CanTransition` (las tres permitidas; rechazadas En Sprint → Hecho, En progreso → En Sprint, Hecho → En Sprint, cualquier destino Pendiente y las repetidas) y de `NextAction` por estado en `src/sprint/story_status_test.go`
- [ ] T006 [P] Escribir las pruebas del servicio `ChangeStatus` contra PostgreSQL real con fixtures propias (dos proyectos; historias en cada estado; Sprint activo, Sprint finalizado y sin Sprint; con y sin horas): cada transición, advertencia de horas solo al completar sin horas, historia inexistente o de otro proyecto, fuera de Sprint activo, estado ya aplicado, estado esperado distinto, transición no permitida y el orden de validación, en `src/sprint/story_status_service_test.go`
- [ ] T007 [P] Escribir la prueba de integración de `PostgreSQLStoryStatusStore`: `FindStory` (estado, Sprint, activo, horas, aislamiento por proyecto) y `UpdateStatus` atómico (cambia con el estado esperado, no cambia con otro, no cambia si el Sprint está finalizado y, con dos cambios simultáneos, solo uno se aplica), en `src/sprint/postgres_story_status_store_integration_test.go`
- [ ] T008 [P] Definir `BoardView`, `BoardStory`, la función pura de Story Points planificados y completados y la interfaz `BoardCatalog` en `src/web/board_catalog.go`
- [ ] T009 [P] Escribir la prueba de integración de `PostgreSQLBoardCatalog` con fixtures propias (dos proyectos, Sprint activo con historias en los tres estados y una sin Story Points, Sprint finalizado, proyecto sin Sprint): Sprint del proyecto correcto, sin datos de otro proyecto, grupos, `hasEffort`, acción de cada historia y Story Points, en `src/web/postgres_board_catalog_integration_test.go`
- [ ] T010 Definir en `src/sprint/story_status.go` los estados (reutilizando las constantes de HU-04 si existen), `CanTransition`, `NextAction`, los errores del dominio y la interfaz `StoryStatusStore`
- [ ] T011 Implementar el servicio `ChangeStatus` con el orden de validación de `data-model.md` y el resultado con la advertencia de horas en `src/sprint/story_status.go`
- [ ] T012 Implementar `PostgreSQLStoryStatusStore` con `database/sql` y pgx y la sentencia atómica de `data-model.md` en `src/sprint/postgres_story_status_store.go`
- [ ] T013 Implementar `PostgreSQLBoardCatalog` en `src/web/postgres_board_catalog.go`

**Checkpoint**: La regla, el servicio y los adaptadores están listos y probados contra PostgreSQL.

---

## Phase 3: User Story 1 - Cambiar el estado de una historia (Priority: P1) 🎯 MVP

**Goal**: Comenzar, completar y reabrir una historia del Sprint activo desde el tablero, con sus mensajes.

**Independent Test**: Con un Sprint activo con historias "En Sprint", comenzar, completar con y sin horas y reabrir una; verificar estado, mensajes y Story Points.

### Tests for User Story 1

- [ ] T014 [P] [US1] Escribir la prueba de handler de comenzar con `cookiejar` y proyecto activo elegido por el flujo de HU-20b: 303 a `/tablero`, "La historia pasó a En progreso" una sola vez, estado nuevo y, con dos proyectos, solo en el activo, en `src/web/board_handlers_test.go`
- [ ] T015 [P] [US1] Escribir la prueba de handler de completar con horas: "La historia pasó a Hecho", estado "Hecho" y Story Points completados actualizados, en `src/web/board_handlers_test.go`
- [ ] T016 [P] [US1] Escribir la prueba de handler de completar sin horas: "La historia pasó a Hecho. Advertencia: no tiene horas reales registradas" en un solo mensaje, en `src/web/board_handlers_test.go`
- [ ] T017 [P] [US1] Escribir la prueba de handler de reabrir: "La historia volvió a En progreso", estado "En progreso" y los Story Points dejan de contarse, en `src/web/board_handlers_test.go`
- [ ] T018 [P] [US1] Escribir los escenarios BDD de comenzar, completar, completar sin horas y reabrir en `test/web_board_bdd_test.go`

### Implementation for User Story 1

- [ ] T019 [US1] Implementar en `src/web/board_form.go` el armado del envío (`historia_id`, `accion`, `estado_actual`), la relación acción → destino y los textos de éxito
- [ ] T020 [US1] Implementar los handlers `GET /tablero` y `POST /tablero/estado` en `src/web/board_handlers.go`: llaman al servicio con el proyecto de `ActiveContext` y usan `SetFlash` (éxito o éxito con advertencia) y la redirección 303
- [ ] T021 [US1] Crear `board.html` con los grupos por estado, una acción por historia (formulario con `historia_id`, `accion` y `estado_actual`) y los Story Points, usando el parcial `alert.html`, en `src/web/templates/board.html`
- [ ] T022 [US1] Construir el servicio, el almacenamiento y el catálogo y registrar la sección "Tablero" (`/tablero`, `RequiresProject: true`) con `Server.Register` en `src/cmd/server/main.go`

**Checkpoint**: US1 cambia el estado desde la web; HU-15, HU-07 y HU-25 ya tienen historias en "Hecho".

---

## Phase 4: User Story 2 - Rechazos y errores (Priority: P1)

**Goal**: Rechazar con motivo los cambios inválidos o en conflicto sin tocar el estado.

**Independent Test**: Cambiar historias sin Sprint, de un Sprint finalizado, ya en "Hecho", de "En Sprint" a "Hecho" y con el estado cambiado por otra persona; simular una falla de guardado.

### Tests for User Story 2

- [ ] T023 [P] [US2] Escribir la prueba de handler de historia sin Sprint o de un Sprint finalizado, y de historia inexistente, no numérica o de otro proyecto: mensajes y códigos de la spec, sin cambios, y un campo de proyecto enviado se ignora, en `src/web/board_handlers_test.go`
- [ ] T024 [P] [US2] Escribir la prueba de handler de marcar dos veces: "La historia ya está en Hecho", sin cambios y Story Points completados sin duplicar, en `src/web/board_handlers_test.go`
- [ ] T025 [P] [US2] Escribir la prueba de handler de transición no permitida enviada directamente ("No se puede pasar de En Sprint a Hecho") y de acción desconocida ("La acción no es válida"), sin cambios, en `src/web/board_handlers_test.go`
- [ ] T026 [P] [US2] Escribir la prueba de handler de estado desactualizado: se cambia la historia por otro camino, se envía la acción con el `estado_actual` anterior y se rechaza con el mensaje y sin cambios, en `src/web/board_handlers_test.go`
- [ ] T027 [P] [US2] Escribir la prueba de handler con un almacenamiento que falla al guardar: "La operación no pudo completarse", sin éxito y sin cambios, en `src/web/board_handlers_test.go`
- [ ] T028 [P] [US2] Escribir los escenarios BDD de fuera de un Sprint activo, marcar dos veces, transición no permitida, estado desactualizado y error al guardar en `test/web_board_bdd_test.go`

### Implementation for User Story 2

- [ ] T029 [US2] Implementar en `src/web/board_form.go` el mapeo de los errores del dominio a mensajes con `{estado}`, `{origen}` y `{destino}`, y `MsgOperationFailed` para cualquier otro
- [ ] T030 [US2] Implementar en `src/web/board_handlers.go` las respuestas 409, 422 y 500 con el tablero y el mensaje, y el 303 con advertencia para el estado ya aplicado
- [ ] T031 [US2] Verificar en `src/web/board_handlers_test.go` que ninguna respuesta de error contiene texto de consultas ni rutas internas

**Checkpoint**: US2 mantiene los estados y los Story Points coherentes ante cualquier error o conflicto.

---

## Phase 5: User Story 3 - Ver el tablero del Sprint activo (Priority: P1)

**Goal**: Mostrar el Sprint activo agrupado por estado, con Story Points y las acciones válidas.

**Independent Test**: Sprint con historias en los tres estados; sin Sprint activo; Sprint sin historias; falla de lectura.

### Tests for User Story 3

- [ ] T032 [P] [US3] Escribir la prueba de handler de la vista: Sprint, grupos por estado, Story Points completados de planificados, historia sin Story Points como "Sin estimar", solo el Sprint activo del proyecto activo y tiempo de respuesta, en `src/web/board_handlers_test.go`
- [ ] T033 [P] [US3] Escribir la prueba de handler de las acciones según el estado: "En Sprint" solo "Comenzar", "En progreso" solo "Completar" y "Hecho" solo "Reabrir", en `src/web/board_handlers_test.go`
- [ ] T034 [P] [US3] Escribir la prueba de handler sin Sprint activo: "No hay un Sprint activo" y ninguna acción, en `src/web/board_handlers_test.go`
- [ ] T035 [P] [US3] Escribir la prueba de handler con un Sprint sin historias: "El Sprint no tiene historias asignadas", en `src/web/board_handlers_test.go`
- [ ] T036 [P] [US3] Escribir la prueba de handler con un catálogo que falla al leer: página de error de HU-20a y no un tablero incompleto, en `src/web/board_handlers_test.go`
- [ ] T037 [P] [US3] Escribir los escenarios BDD de tablero del Sprint activo, acciones según el estado, sin Sprint activo y Sprint sin historias en `test/web_board_bdd_test.go`

### Implementation for User Story 3

- [ ] T038 [US3] Completar `board.html` con los estados vacíos ("No hay un Sprint activo", "El Sprint no tiene historias asignadas") y la presentación del progreso en `src/web/templates/board.html`
- [ ] T039 [US3] Usar `Fail` en `src/web/board_handlers.go` cuando falla la lectura del Sprint o de las historias

**Checkpoint**: US3 muestra el avance del Sprint y las acciones disponibles de cada historia.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validación final, evidencias, coordinación y trazabilidad.

- [ ] T040 [P] Documentar en `specs/014-story-status/quickstart.md` los pasos para probar a mano con un Sprint activo y la nota de puerto
- [ ] T041 [P] Revisar que errores y registros no expongan detalles internos ni valores de configuración en `src/web/board_handlers.go` y `src/web/board_form.go`
- [ ] T042 Ejecutar `docker compose config` y `docker compose up -d postgres` para validar el entorno local
- [ ] T043 Ejecutar `go test -count=1 -p 1 ./...` y completar todos los escenarios de `quickstart.md`
- [ ] T044 [P] Escribir la prueba de handler de recarga después de un cambio: no se repite el cambio ni el mensaje, en `src/web/board_handlers_test.go`
- [ ] T045 [P] Escribir la prueba de handler sin proyecto activo válido (sin selección, proyecto eliminado y cookie inválida): redirección de HU-20b y no se cambia nada, en `src/web/board_handlers_test.go`
- [ ] T046 [P] Verificar que `T036` cubre el error al leer y que el catálogo distingue "sin Sprint" de un error de lectura en `src/web/postgres_board_catalog.go`
- [ ] T047 Guardar en `specs/014-story-status/evidencia-red.txt` la salida de `go test -count=1 -p 1 -v ./...` tomada antes de implementar, con las pruebas nuevas fallando
- [ ] T048 Guardar en `specs/014-story-status/evidencia-pruebas.txt` la salida final de `go test -count=1 -p 1 -v ./...` y en `specs/014-story-status/cobertura.txt` la de `go test -count=1 -p 1 -cover ./...`, verificando al menos 80% en `src/sprint`
- [ ] T049 Coordinar con quien tome HU-15 que las historias "Hecho" definen lo completado y que el cierre decide qué pasa con las no terminadas; con HU-07 y HU-25 que cuentan `'Hecho'`; con HU-23 que ambas pantallas leen el Sprint activo y conviene unificar el catálogo después; y con quien corrija HU-08 y HU-13 que los grupos son los cuatro estados
- [ ] T050 Revisar trazabilidad de FR-001 a FR-018 y SC-001 a SC-007 en `specs/014-story-status/tasks.md` y la tabla de trazabilidad de `spec.md`
- [ ] T051 [P] Ejecutar `go vet ./...` y `gofmt -l .` sin diferencias, verificar que el pipeline de GitHub Actions queda en verde y actualizar el manual breve de usuario con la pantalla "Tablero"
- [ ] T052 Verificar a mano con la aplicación en ejecución los 13 escenarios del Issue y la lista de `quickstart.md`
- [ ] T053 Abrir el Pull Request desde `HU-14`, con revisión y aprobación de otro integrante, y pasar la tarjeta a "Hecho" cuando se integre a `main`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 y T002 primero; T003 y T004 en paralelo.
- **Foundational (Phase 2)**: Depende de T001 y T004; T005 a T009 en paralelo; T010 antes de T011 y T012; T013 requiere T008.
- **User Story 1 (Phase 3)**: Depende de toda la Phase 2 y de T002; es el MVP.
- **User Story 2 (Phase 4)**: Depende de T020; puede avanzar en paralelo con US3.
- **User Story 3 (Phase 5)**: Depende de T013 y T021; puede avanzar en paralelo con US2.
- **Polish (Phase 6)**: Depende de las historias que se entreguen; T044 a T046 se escriben junto a las pruebas de las historias.

### User Story Dependencies

- **US1 (P1)**: Depende de HU-04 y FIX-HU-04, HU-03 (horas), HU-20a y HU-20b.
- **US2 (P1)**: Depende del handler de US1; se verifica con envíos inválidos de forma independiente.
- **US3 (P1)**: Depende del catálogo de Phase 2; verificable con datos de prueba.
- **HU-15, HU-07 y HU-25**: Necesitan historias en "Hecho", que esta historia permite registrar desde la web.

### Parallel Opportunities

- T003 y T004; T005 a T009.
- T014 a T018, T023 a T028 y T032 a T037 pueden escribirse en paralelo antes de implementar.
- T040, T041, T044 a T046 y T051 pueden ejecutarse en paralelo después de la implementación.
- Mientras HU-20a y HU-20b no estén integradas, se pueden adelantar T001, T004 a T013.

## Implementation Strategy

### MVP First (User Story 1, User Story 2 y User Story 3)

1. Completar Setup y Foundational; guardar `evidencia-red.txt` (T047) con las pruebas de las tres historias
   fallando.
2. Implementar US1 a US3; ejecutar `docker compose up -d postgres` y `go test -count=1 -p 1 ./...`.
3. Verificar a mano los tres cambios de estado con un Sprint activo antes de abrir el Pull Request.

### Incremental Delivery

1. Entregar US1 como MVP funcional: permite llevar historias a "Hecho".
2. Agregar US2 para rechazar errores y conflictos.
3. Agregar US3 para ver el avance del Sprint.
4. Ejecutar la validación completa del quickstart.

## Notes

- Todas las tareas siguen el formato `- [ ] T### [P?] [US?] descripción con ruta`.
- Las pruebas deben verificar comportamiento real y observar un estado fallido antes de implementar.
- PostgreSQL debe ejecutarse con Docker; la aplicación Go y las pruebas se ejecutan localmente.
- Las pruebas crean sus propios proyectos, historias, Sprints y horas; no dependen de HU-21, HU-22 ni HU-23.
- `user_stories.project_id` no tiene borrado en cascada: las fixtures borran primero las historias (los Sprints y sus asignaciones caen en cascada) y después el proyecto.
- Para dejar un Sprint finalizado en las pruebas, cambiar su estado directamente en la base (HU-15 todavía no existe).
- Si la base ya aplicó una migración de número mayor, `migrate up` no aplica una nueva de número menor; recrear la base con `docker compose down -v` y volver a migrar.

## Trazabilidad

| Requisito | Pruebas |
|---|---|
| FR-001 | T020, T022, T044 |
| FR-002 | T009, T032 |
| FR-003 | T005, T033 |
| FR-004 | T005, T006, T014 |
| FR-005 | T006, T015 |
| FR-006 | T006, T016 |
| FR-007 | T006, T017 |
| FR-008 | T006, T023 |
| FR-009 | T006, T024 |
| FR-010 | T005, T025 |
| FR-011 | T007, T026 |
| FR-012 | T027 |
| FR-013 | T034, T035 |
| FR-014 | T023, T045 |
| FR-015 | T025 |
| FR-016 | T036 |
| FR-017 | T009, T015, T017, T024 |
| FR-018 | T031, T041 |
| SC-001 | T006, T014, T015, T017 |
| SC-002 | T023, T025, T026 |
| SC-003 | T015, T017, T024 |
| SC-004 | T027 |
| SC-005 | T006, T016 |
| SC-006 | T007 |
| SC-007 | Verificación de tiempo en T032 |